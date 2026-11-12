package watch

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"
)

// fake is gh as a list of looks: each look is the state and the required checks'
// output, and all is what gh pr checks without --required prints.
type fake struct {
	states  []string // one per look; the last one repeats
	checks  []string // gh pr checks --required, one per look; the last one repeats
	errOut  string   // what gh writes to its error output on every call
	all     string
	now     time.Time
	looks   int
	calls   [][]string
	runErr  error
	checked int
}

func (f *fake) options() Options {
	f.now = time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)
	return Options{
		PR: 7, Repo: "o/r", Every: 30 * time.Second, Max: time.Hour, Grace: 5 * time.Minute,
		Now:   func() time.Time { return f.now },
		Sleep: func(d time.Duration) { f.now = f.now.Add(d) },
		Run: func(args ...string) (string, string, error) {
			f.calls = append(f.calls, args)
			switch {
			case args[1] == "view":
				i := min(f.looks, len(f.states)-1)
				f.looks++
				return f.states[i], f.errOut, f.runErr
			case slices.Contains(args, "--required"):
				i := min(f.checked, len(f.checks)-1)
				f.checked++
				out := f.checks[i]
				var err error
				if strings.Contains(out, "\tpending") {
					err = errors.New("exit status 8") // what gh does: never trusted
				}
				return out, f.errOut, err
			default:
				return f.all, "", nil
			}
		},
	}
}

func TestGreenAfterPending(t *testing.T) {
	f := &fake{
		states: []string{"OPEN MERGEABLE\n"},
		checks: []string{"test\tpending\t0\thttps://x\n", "test\tpass\t3m\thttps://x\nprivacy\tskipping\t0\thttps://x\n"},
		all:    "test\tpass\t3m\thttps://x\ninstall-linux\tpending\t0\thttps://x\n",
	}
	r := PR(f.options())
	if r.Kind != Green || r.PR != 7 || r.Elapsed != 30*time.Second {
		t.Fatalf("got %+v", r)
	}
	if !slices.Equal(r.Others, []string{"install-linux (pending)"}) {
		t.Errorf("others: %v", r.Others)
	}
	if !slices.Contains(f.calls[0], "-R") {
		t.Errorf("no repo given: %v", f.calls[0])
	}
}

func TestFailedNamesTheCheck(t *testing.T) {
	f := &fake{states: []string{"OPEN"}, checks: []string{"test\tpass\t1m\tu\ncompat\tfail\t1m\tu\nwin\tcancel\t1m\tu"}}
	r := PR(f.options())
	if r.Kind != Failed || !slices.Equal(r.Failed, []string{"compat", "win"}) {
		t.Fatalf("got %+v", r)
	}
}

func TestMergedAndClosed(t *testing.T) {
	for state, kind := range map[string]Kind{"MERGED": Merged, "CLOSED": Closed, "OPEN CONFLICTING": Conflict} {
		f := &fake{states: []string{"OPEN", state}, checks: []string{"test\tpending\t0\tu"}}
		if r := PR(f.options()); r.Kind != kind {
			t.Errorf("%s: got %+v", state, r)
		}
	}
}

// The loop that hung on 12-nov: a filter that finds nothing must end, not wait.
func TestNoRequiredChecksEndsAfterGrace(t *testing.T) {
	f := &fake{states: []string{"OPEN"}, checks: []string{""}, errOut: "no required checks reported on the 'x' branch"}
	r := PR(f.options())
	if r.Kind != Broken || r.Elapsed < 5*time.Minute || r.Elapsed > 6*time.Minute {
		t.Fatalf("got %+v", r)
	}
}

func TestAnswerNotUnderstoodIsBroken(t *testing.T) {
	f := &fake{states: []string{"null"}, checks: []string{"x"}}
	r := PR(f.options())
	if r.Kind != Broken || !strings.Contains(r.Problem, "null") {
		t.Fatalf("got %+v", r)
	}
	f = &fake{states: []string{"OPEN"}, checks: []string{"something else entirely"}}
	if r := PR(f.options()); r.Kind != Broken || !strings.Contains(r.Problem, "not understood") {
		t.Fatalf("got %+v", r)
	}
}

func TestGhFailingSaysWhat(t *testing.T) {
	f := &fake{states: []string{""}, checks: []string{""}, errOut: "To get started with GitHub CLI, please run:  gh auth login\nmore", runErr: errors.New("exit status 4")}
	r := PR(f.options())
	if r.Kind != Broken || r.Problem != "To get started with GitHub CLI, please run:  gh auth login" || f.looks != tries {
		t.Fatalf("got %+v after %d looks", r, f.looks)
	}
}

// One failed look among good ones is not the end.
func TestOneBadLookIsForgiven(t *testing.T) {
	f := &fake{states: []string{"OPEN", "", "OPEN"}, checks: []string{"t\tpending\t0\tu", "t\tpass\t1m\tu"}}
	if r := PR(f.options()); r.Kind != Green {
		t.Fatalf("got %+v", r)
	}
}

func TestTimesOut(t *testing.T) {
	f := &fake{states: []string{"OPEN"}, checks: []string{"slow\tpending\t0\tu\nfast\tpass\t1m\tu"}}
	o := f.options()
	r := PR(o)
	if r.Kind != TimedOut || r.Elapsed != time.Hour || !slices.Equal(r.Pending, []string{"slow"}) {
		t.Fatalf("got %+v", r)
	}
}
