package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/icalvete/augmented-brain/internal/brain"
	"github.com/icalvete/augmented-brain/internal/config"
)

// fakeWatch makes watch run against a pretend gh that answers each look with the next
// of checks (the last one repeats), and never sleeps.
func fakeWatch(t *testing.T, state string, checks ...string) *[][]string {
	t.Helper()
	var calls [][]string
	now := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)
	look := 0
	gh, sleep, clock, lookPath := watchGh, watchSleep, watchNow, watchLookPath
	t.Cleanup(func() { watchGh, watchSleep, watchNow, watchLookPath = gh, sleep, clock, lookPath })
	watchLookPath = func(string) (string, error) { return "/usr/bin/gh", nil }
	watchSleep = func(d time.Duration) { now = now.Add(d) }
	watchNow = func() time.Time { return now }
	watchGh = func(args ...string) (string, string, error) {
		calls = append(calls, args)
		if args[1] == "view" {
			return state + "\n", "", nil
		}
		if !strings.Contains(strings.Join(args, " "), "--required") {
			return "install-linux\tpending\t0\thttps://x\n", "", nil
		}
		out := checks[min(look, len(checks)-1)]
		look++
		return out, "", nil
	}
	return &calls
}

func TestWatchWritesTheTask(t *testing.T) {
	h, cfg := home(t)
	c, err := config.Load(cfg)
	if err != nil {
		t.Fatal(err)
	}
	s := c.SpaceFor(filepath.Join(h, "augmented-brain", "trabajo"))
	b, err := brain.Open(c, s)
	if err != nil {
		t.Fatal(err)
	}
	task, err := b.NewTask(brain.NewTaskInput{Title: "Un cambio"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.Update(brain.UpdateInput{ID: task.Header.ID, Next: "esperar la CI"}); err != nil {
		t.Fatal(err)
	}
	wd, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(wd) })
	if err := os.Chdir(filepath.Join(h, "augmented-brain", "trabajo")); err != nil {
		t.Fatal(err)
	}
	calls := fakeWatch(t, "OPEN", "test\tpending\t0\tu\n", "test\tpass\t3m\tu\n")
	var out, errOut bytes.Buffer
	code := run([]string{"watch", "pr", "#104", "--task", task.Header.ID, "--repo", "o/r"}, nil, &out, &errOut)
	if code != 0 {
		t.Fatalf("watch: %d\n%s%s", code, out.String(), errOut.String())
	}
	want := "PR #104: todos los checks obligatorios en verde (1 min). Sigue abierta: falta mezclarla. No obligatorios, aún en marcha o fallidos: install-linux (pending)."
	if !strings.Contains(out.String(), want) || !strings.Contains(out.String(), "Para el asistente: díselo ahora") {
		t.Errorf("output:\n%s", out.String())
	}
	if !strings.Contains(strings.Join((*calls)[0], " "), "-R o/r") {
		t.Errorf("calls: %v", *calls)
	}
	b, _ = brain.Open(c, s)
	n, err := b.Task(task.Header.ID)
	if err != nil {
		t.Fatal(err)
	}
	if n.Header.Next != want {
		t.Errorf("next: %q", n.Header.Next)
	}
	if !strings.Contains(n.Body, "Vigilancia de la PR #104: "+want+" El siguiente paso de antes era: esperar la CI") {
		t.Errorf("journal:\n%s", n.Body)
	}
}

// A task that does not exist is said before the wait, not three hours later.
func TestWatchChecksTheTaskFirst(t *testing.T) {
	h, _ := home(t)
	wd, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(wd) })
	if err := os.Chdir(filepath.Join(h, "augmented-brain", "trabajo")); err != nil {
		t.Fatal(err)
	}
	calls := fakeWatch(t, "OPEN", "test\tpass\t1m\tu\n")
	var out, errOut bytes.Buffer
	if code := run([]string{"watch", "pr", "5", "--task", "task-no-existe"}, nil, &out, &errOut); code != 1 {
		t.Fatalf("watch: %d", code)
	}
	if len(*calls) != 0 || !strings.Contains(errOut.String(), "task-no-existe") {
		t.Errorf("calls %v, errOut %s", *calls, errOut.String())
	}
}

func TestWatchFailedEndsWithOne(t *testing.T) {
	home(t)
	fakeWatch(t, "OPEN", "test\tfail\t1m\tu\n")
	var out, errOut bytes.Buffer
	if code := run([]string{"watch", "pr", "5"}, nil, &out, &errOut); code != 1 || !strings.Contains(out.String(), "ha fallado un check obligatorio: test") {
		t.Fatalf("watch: %d\n%s%s", code, out.String(), errOut.String())
	}
}

func TestWatchUsage(t *testing.T) {
	home(t)
	for _, args := range [][]string{{"watch"}, {"watch", "pr"}, {"watch", "pr", "x"}, {"watch", "issue", "3"}, {"watch", "pr", "3", "--max", "0s"}} {
		var out, errOut bytes.Buffer
		if code := run(args, nil, &out, &errOut); code != 2 || !strings.Contains(errOut.String(), "watch pr") {
			t.Errorf("%v: %d %s", args, code, errOut.String())
		}
	}
}
