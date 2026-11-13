package remote

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// When Claude cannot say whether a session still runs, session stop does not call it
// closed: the record stays, and with it the worktree (R-10).
func TestStopWhenClaudeCannotTell(t *testing.T) {
	f, repo := launchFake(t)
	c, set := f.load()
	f.trust(repo)
	if err := f.sys.Start(c, set, Launch{Task: "task-site", Name: "box · Site", Folder: repo}, "go"); err != nil {
		t.Fatal(err)
	}
	f.sys.Run = func(name string, args ...string) (string, error) {
		if filepath.Base(name) == "claude" && len(args) > 0 {
			switch args[0] {
			case "agents":
				return "claude: update available\n[]", nil // not JSON
			case "daemon":
				return "", nil
			}
			return "", errors.New("exit status 1")
		}
		return "", nil
	}
	err := f.sys.Stop(c.Dir, "task-site")
	if !errors.Is(err, ErrCannotTell) {
		t.Fatalf("Stop = %v, want ErrCannotTell", err)
	}
	if !HasLaunch(c.Dir, "task-site") {
		t.Fatal("the record went although the session may still run")
	}
}

// A record whose id is not a session id is never passed to claude (R-10).
func TestStopRefusesOddID(t *testing.T) {
	f, repo := launchFake(t)
	c, set := f.load()
	f.trust(repo)
	f.bgOut = "backgrounded · ab12cd34 · x\n"
	if err := f.sys.Start(c, set, Launch{Task: "task-site", Name: "box · Site", Folder: repo}, "go"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(launchRecordPath(c.Dir, "task-site"), []byte(`{"id":"--all","task":"task-site"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	f.calls = nil
	if err := f.sys.Stop(c.Dir, "task-site"); !errors.Is(err, ErrCannotTell) {
		t.Fatalf("Stop = %v", err)
	}
	if len(f.calls) != 0 {
		t.Errorf("claude was called with it: %v", f.calls)
	}
}

// Two starts of one task at once: one session, on record; the other start finds it
// running (R-11).
func TestStartOneAtATime(t *testing.T) {
	f, repo := launchFake(t)
	c, set := f.load()
	f.trust(repo)
	f.bgOut = "backgrounded · ab12cd34 · x\n"
	l := Launch{Task: "task-site", Name: "box · Site", Folder: repo}
	var mu sync.Mutex
	run, runIn := f.sys.Run, f.sys.RunIn
	f.sys.Run = func(name string, args ...string) (string, error) {
		mu.Lock()
		defer mu.Unlock()
		return run(name, args...)
	}
	entered, release := make(chan struct{}), make(chan struct{})
	launches := 0
	f.sys.RunIn = func(dir, name string, args ...string) (string, error) {
		mu.Lock()
		launches++
		first := launches == 1
		mu.Unlock()
		if first {
			close(entered)
			<-release
		}
		mu.Lock()
		defer mu.Unlock()
		return runIn(dir, name, args...)
	}
	errs := make(chan error, 2)
	go func() { errs <- f.sys.Start(c, set, l, "go") }()
	<-entered
	go func() { errs <- f.sys.Start(c, set, l, "go") }()
	time.Sleep(300 * time.Millisecond) // the second start reaches the lock
	close(release)
	e1, e2 := <-errs, <-errs
	if !((e1 == nil && errors.Is(e2, ErrRunning)) || (e2 == nil && errors.Is(e1, ErrRunning))) {
		t.Fatalf("starts: %v and %v, want one nil and one ErrRunning", e1, e2)
	}
	mu.Lock()
	defer mu.Unlock()
	if launches != 1 {
		t.Fatalf("claude --bg ran %d times", launches)
	}
}
