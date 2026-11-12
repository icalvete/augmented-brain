// Package watch follows a pull request until something worth telling happens: it is
// merged or closed, a required check fails, every required check has finished, or the
// time allowed runs out. It replaces the loops each session used to write by hand,
// which hung on an answer they did not expect and never said so.
//
// It reads gh's plain output (a line per check, its fields separated by tabs), which
// every version of gh writes: `gh pr checks --json` came late. gh's exit code is not
// trusted: it is 8 while a check is pending and 1 when one failed, so only what it
// printed is.
package watch

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Kind is how a watch ended.
type Kind int

const (
	Green    Kind = iota // every required check finished and none failed; the PR is still open
	Failed               // a required check failed or was cancelled
	Merged               // the PR was merged
	Closed               // the PR was closed without merging
	TimedOut             // Max passed with checks still pending
	Conflict             // the PR conflicts with its base: no CI runs until it is rebased
	Broken               // gh could not tell: missing, logged out, or an answer not understood
)

// Result is what a watch found.
type Result struct {
	Kind    Kind
	PR      int
	Failed  []string      // the required checks that failed (Failed)
	Pending []string      // the required checks still pending (TimedOut)
	Others  []string      // the checks not required still pending or failed, "name (bucket)"
	Problem string        // what gh said (Broken)
	Elapsed time.Duration // from start to end
}

// Options say what to watch and how. Run, Sleep and Now are there so tests never run
// gh nor wait.
type Options struct {
	PR    int
	Repo  string        // owner/name; empty: the repository of the working folder
	Every time.Duration // between two looks
	Max   time.Duration // the longest it watches
	// Grace is how long a PR may report no required check before that is a problem:
	// right after a push GitHub has not listed them yet, but a filter that matches
	// nothing must not look like "still waiting" for ever.
	Grace time.Duration
	// Run runs gh with these arguments and returns what it printed, and the error of
	// running it (an exit code other than 0 included).
	Run   func(args ...string) (stdout, stderr string, err error)
	Sleep func(time.Duration)
	Now   func() time.Time
}

// tries is how many looks in a row may fail before the watch gives up and says so.
const tries = 3

// buckets are the states gh gives a check in its second column.
var buckets = []string{"pass", "fail", "pending", "skipping", "cancel"}

// Check is one line of gh pr checks.
type Check struct{ Name, Bucket string }

// PR watches a pull request until it has something to tell. It never waits for ever:
// after Max it ends TimedOut, and after tries failed looks in a row it ends Broken.
func PR(o Options) Result {
	start := o.Now()
	res := func(r Result) Result {
		r.PR = o.PR
		r.Elapsed = o.Now().Sub(start)
		return r
	}
	failures := 0
	var last error
	for {
		req, err := o.look()
		if err == nil {
			failures = 0
			switch {
			case req.state == "MERGED":
				return res(Result{Kind: Merged})
			case req.state == "CLOSED":
				return res(Result{Kind: Closed})
			case req.state == "CONFLICTING":
				return res(Result{Kind: Conflict})
			case len(req.checks) == 0:
				if o.Now().Sub(start) >= o.Grace {
					return res(Result{Kind: Broken, Problem: "no required checks reported"})
				}
			default:
				if failed := names(req.checks, "fail", "cancel"); len(failed) > 0 {
					return res(Result{Kind: Failed, Failed: failed, Others: o.others(req.checks)})
				}
				if len(names(req.checks, "pending")) == 0 {
					return res(Result{Kind: Green, Others: o.others(req.checks)})
				}
			}
		} else {
			failures++
			last = err
			if failures >= tries {
				return res(Result{Kind: Broken, Problem: last.Error()})
			}
		}
		if o.Now().Sub(start) >= o.Max {
			r := Result{Kind: TimedOut}
			if err == nil {
				r.Pending = names(req.checks, "pending")
			} else {
				r.Problem = err.Error()
			}
			return res(r)
		}
		o.Sleep(o.Every)
	}
}

type look struct {
	state  string
	checks []Check
}

// look is one look: the PR's state and, while it is open, its required checks.
func (o Options) look() (look, error) {
	args := []string{"pr", "view", strconv.Itoa(o.PR), "--json", "state,mergeable", "-q", `.state + " " + .mergeable`}
	out, errOut, err := o.Run(o.repo(args)...)
	state, mergeable, _ := strings.Cut(strings.TrimSpace(out), " ")
	switch state {
	case "OPEN", "MERGED", "CLOSED":
	default:
		return look{}, problem(errOut, err, "gh pr view gave "+strconv.Quote(strings.TrimSpace(out)))
	}
	if state != "OPEN" {
		return look{state: state}, nil
	}
	// A PR in conflict gets no CI at all: waiting for its checks would wait for nothing.
	if mergeable == "CONFLICTING" {
		return look{state: "CONFLICTING"}, nil
	}
	checks, err := o.checks(true)
	return look{state: state, checks: checks}, err
}

// checks are the PR's checks, the required ones only or all.
func (o Options) checks(required bool) ([]Check, error) {
	args := []string{"pr", "checks", strconv.Itoa(o.PR)}
	if required {
		args = append(args, "--required")
	}
	out, errOut, err := o.Run(o.repo(args)...)
	if strings.TrimSpace(out) == "" {
		// gh says so in its error output when GitHub has listed none yet.
		if strings.Contains(errOut, "no checks reported") || strings.Contains(errOut, "no required checks reported") {
			return nil, nil
		}
		return nil, problem(errOut, err, "gh pr checks printed nothing")
	}
	return Parse(out)
}

// others are the checks not required that are still pending or failed: they do not
// decide the watch, but the person should know.
func (o Options) others(required []Check) []string {
	all, err := o.checks(false)
	if err != nil {
		return nil
	}
	var out []string
	for _, c := range all {
		if slices.ContainsFunc(required, func(r Check) bool { return r.Name == c.Name }) {
			continue
		}
		if c.Bucket == "pending" || c.Bucket == "fail" || c.Bucket == "cancel" {
			out = append(out, c.Name+" ("+c.Bucket+")")
		}
	}
	return out
}

func (o Options) repo(args []string) []string {
	if o.Repo != "" {
		args = append(args, "-R", o.Repo)
	}
	return args
}

// Parse reads gh pr checks' plain output. A line it does not understand is an error,
// never skipped: a check it cannot read could be the one that failed.
func Parse(out string) ([]Check, error) {
	var checks []Check
	for line := range strings.SplitSeq(strings.TrimRight(out, "\n"), "\n") {
		f := strings.Split(line, "\t")
		if len(f) < 2 || !slices.Contains(buckets, f[1]) {
			return nil, fmt.Errorf("gh pr checks gave a line not understood: %q", line)
		}
		checks = append(checks, Check{Name: strings.TrimSpace(f[0]), Bucket: f[1]})
	}
	return checks, nil
}

func names(checks []Check, in ...string) []string {
	var out []string
	for _, c := range checks {
		if slices.Contains(in, c.Bucket) {
			out = append(out, c.Name)
		}
	}
	return out
}

// problem is what gh said when it failed, or why its answer was not understood.
func problem(errOut string, err error, why string) error {
	if s := strings.TrimSpace(errOut); s != "" {
		if i := strings.IndexByte(s, '\n'); i > 0 {
			s = s[:i]
		}
		return errors.New(s)
	}
	if err != nil {
		return err
	}
	return errors.New(why)
}
