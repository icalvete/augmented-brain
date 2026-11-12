package remote

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestLaunchUnit(t *testing.T) {
	for in, want := range map[string]string{
		"task-new-website": "augmented-brain-session-new-website.service",
		"task-a b/c":       "augmented-brain-session-a-b-c.service",
	} {
		if got := LaunchUnit(in); got != want {
			t.Errorf("LaunchUnit(%q) = %q, want %q", in, got, want)
		}
	}
}

// The folder must be trusted on its own: a trusted parent is not enough (measured on
// claude 2.1.319), unlike what the space's service needs.
func TestFolderTrustedIsExact(t *testing.T) {
	f := newFake(t)
	repo := filepath.Join(f.root, "p", "site")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	f.trust(filepath.Join(f.root, "p"))
	if f.sys.FolderTrusted(repo) {
		t.Error("a trusted parent made the folder trusted")
	}
	f.trust(repo)
	if !f.sys.FolderTrusted(repo) {
		t.Error("the trusted folder was not trusted")
	}
}

// A git worktree is trusted when its main repository is: Claude Code keeps the answer
// there (measured on 9-nov), for a repository with a .git folder and for a bare one.
func TestFolderTrustedWorktree(t *testing.T) {
	f := newFake(t)
	for _, bare := range []bool{false, true} {
		repo := filepath.Join(f.root, "p", "site")
		if bare {
			repo = filepath.Join(f.root, "p", "site.git")
		}
		common := filepath.Join(repo, ".git")
		if bare {
			common = repo
		}
		wt := repo + "-main"
		rec := filepath.Join(common, "worktrees", "site-main")
		for _, d := range []string{rec, wt} {
			if err := os.MkdirAll(d, 0o755); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.WriteFile(filepath.Join(wt, ".git"), []byte("gitdir: "+rec+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(rec, "commondir"), []byte("../..\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if got := WorktreeRoot(wt); got != repo {
			t.Errorf("bare=%v: worktreeRoot = %q, want %q", bare, got, repo)
		}
		f.trust(filepath.Join(f.root, "p"))
		if f.sys.FolderTrusted(wt) {
			t.Errorf("bare=%v: a trusted parent made the worktree trusted", bare)
		}
		f.trust(repo)
		if !f.sys.FolderTrusted(wt) {
			t.Errorf("bare=%v: the worktree of a trusted repository was not trusted", bare)
		}
	}
	// A plain folder has no main repository.
	if got := WorktreeRoot(f.root); got != "" {
		t.Errorf("worktreeRoot of a plain folder = %q", got)
	}
}

func launchFake(t *testing.T) (*fake, string) {
	f := newFake(t)
	f.sys.Sleep = func(time.Duration) {}
	repo := filepath.Join(f.root, "p", "site")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	return f, repo
}

func TestStartRefuses(t *testing.T) {
	f, repo := launchFake(t)
	c, set := f.load()
	l := Launch{Task: "task-site", Name: "box · Site", Folder: repo}
	if err := f.sys.Start(c, set, l, "go"); !errors.Is(err, ErrUntrusted) {
		t.Fatalf("untrusted folder: %v", err)
	}
	f.trust(repo)
	if err := f.sys.Start(c, set, l, "go"); err != nil {
		t.Fatal(err)
	}
	// The same task again, and another task in the same folder.
	if err := f.sys.Start(c, set, l, "go"); !errors.Is(err, ErrRunning) {
		t.Fatalf("the same task twice: %v", err)
	}
	if err := f.sys.Start(c, set, Launch{Task: "task-other", Name: "x", Folder: repo}, "go"); !errors.Is(err, ErrServed) {
		t.Fatalf("two tasks in one folder: %v", err)
	}
	if len(f.ranIn) != 1 {
		t.Fatalf("started although refused: %v", f.ranIn)
	}
	// Tasks with no repository share the space's folder.
	f.bgOut = "backgrounded · ef56ab78 · y\n"
	if err := f.sys.Start(c, set, Launch{Task: "task-notes", Name: "y", Folder: repo, Shared: true}, "go"); err != nil {
		t.Fatalf("a task with no repository in the space's folder: %v", err)
	}
	f.bgOut = "backgrounded · ab12cd34 · x\n"
	f.sys.GOOS = "darwin"
	if err := f.sys.Start(c, set, l, "go"); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("macOS: %v", err)
	}
}

// A session starts in the background, in its folder, with a first message that names
// its task, so it gets going without anyone writing to it; which session it
// is gets recorded, and `session stop` takes it out of Claude's lists.
func TestStartRunsAndStops(t *testing.T) {
	f, repo := launchFake(t)
	c, set := f.load()
	f.trust(repo)
	if err := f.sys.Start(c, set, Launch{Task: "task-site", Name: "box · Site", Folder: repo}, "Start with your task, task-site."); err != nil {
		t.Fatal(err)
	}
	if len(f.ranIn) != 1 {
		t.Fatalf("runs: %v", f.ranIn)
	}
	for _, want := range []string{repo + " | ", "claude --bg --remote-control box · Site -n box · Site",
		"--permission-mode default", "Start with your task, task-site."} {
		if !strings.Contains(f.ranIn[0], want) {
			t.Errorf("the run lacks %q: %s", want, f.ranIn[0])
		}
	}
	got, err := f.sys.Launches(c.Dir)
	if err != nil || len(got) != 1 || got[0].Task != "task-site" || got[0].Unit != "ab12cd34" || got[0].Active != "active" || got[0].Folder != repo {
		t.Fatalf("launches: %+v %v", got, err)
	}
	f.calls = nil
	if err := f.sys.Stop(c.Dir, "task-site"); err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(f.calls, func(c string) bool { return strings.HasSuffix(c, "claude rm ab12cd34") }) {
		t.Errorf("stop did not remove the session from Claude: %v", f.calls)
	}
	if got, _ := f.sys.Launches(c.Dir); len(got) != 0 {
		t.Errorf("after stop: %+v", got)
	}
	// A session that ended by itself is listed as ended, and the task can start again.
	if err := f.sys.Start(c, set, Launch{Task: "task-site", Name: "x", Folder: repo}, "go"); err != nil {
		t.Fatal(err)
	}
	delete(f.live, "ab12cd34")
	if got, _ := f.sys.Launches(c.Dir); len(got) != 1 || got[0].Active != "ended" {
		t.Errorf("an ended session: %+v", got)
	}
	if err := f.sys.Start(c, set, Launch{Task: "task-site", Name: "x", Folder: repo}, "go"); err != nil {
		t.Errorf("starting again after it ended: %v", err)
	}
}

// A session that ends at once is reported with what it wrote, and leaves no record.
func TestStartReportsFailure(t *testing.T) {
	f, repo := launchFake(t)
	c, set := f.load()
	f.trust(repo)
	f.bgOut = "\x1b[2JError: not logged in\r\n"
	err := f.sys.Start(c, set, Launch{Task: "task-site", Name: "x", Folder: repo}, "go")
	if err == nil || !strings.Contains(err.Error(), "did not start: Error: not logged in") {
		t.Fatalf("no id printed: %v", err)
	}
	// It printed its id, then died.
	f.bgOut = "backgrounded · dead0001 · x\n"
	f.sys.RunIn = func(string, string, ...string) (string, error) { return f.bgOut, nil }
	err = f.sys.Start(c, set, Launch{Task: "task-site", Name: "x", Folder: repo}, "go")
	if err == nil || !strings.Contains(err.Error(), "did not start") {
		t.Fatalf("a session that died: %v", err)
	}
	if got, _ := f.sys.Launches(c.Dir); len(got) != 0 {
		t.Errorf("a failed start left %+v", got)
	}
}

// While Claude starts its daemon, `claude --bg` may print no id: the session is then
// found in `claude agents` by its name and folder.
func TestStartFindsSessionWithoutID(t *testing.T) {
	f, repo := launchFake(t)
	c, set := f.load()
	f.trust(repo)
	f.live["old00001"] = true // another session, running already
	f.named = map[string][2]string{"old00001": {"box · Site", repo}, "new00002": {"box · Site", repo}}
	f.sys.RunIn = func(string, string, ...string) (string, error) {
		f.live["new00002"] = true
		return "Starting background service…\n", nil
	}
	if err := f.sys.Start(c, set, Launch{Task: "task-site", Name: "box · Site", Folder: repo}, "go"); err != nil {
		t.Fatal(err)
	}
	if got, _ := f.sys.Launches(c.Dir); len(got) != 1 || got[0].Unit != "new00002" {
		t.Errorf("launches: %+v", got)
	}
}

// Closing a task from its own session must not cut off its answer: the session is
// closed only once it is idle, and one that keeps answering is left alone.
func TestStopWhenIdle(t *testing.T) {
	f, repo := launchFake(t)
	c, set := f.load()
	f.trust(repo)
	if err := f.sys.Start(c, set, Launch{Task: "task-site", Name: "x", Folder: repo}, "go"); err != nil {
		t.Fatal(err)
	}
	if !f.sys.LaunchRunning(c.Dir, "task-site") {
		t.Fatal("the session does not run")
	}
	f.busy["ab12cd34"] = 3
	f.calls = nil
	if err := f.sys.StopWhenIdle(c.Dir, "task-site", time.Minute); err != nil {
		t.Fatal(err)
	}
	looks := 0
	for _, call := range f.calls {
		if strings.Contains(call, "agents --json") {
			looks++
		}
		if strings.HasSuffix(call, "rm ab12cd34") && looks < 5 {
			t.Errorf("removed after %d looks, while it was still answering", looks)
		}
	}
	if f.sys.LaunchRunning(c.Dir, "task-site") {
		t.Error("still running")
	}
	// One that keeps answering is left.
	if err := f.sys.Start(c, set, Launch{Task: "task-site", Name: "x", Folder: repo}, "go"); err != nil {
		t.Fatal(err)
	}
	f.busy["ab12cd34"] = 1000
	if err := f.sys.StopWhenIdle(c.Dir, "task-site", 10*time.Second); !errors.Is(err, ErrStillBusy) {
		t.Errorf("a busy session: %v", err)
	}
	if !f.sys.LaunchRunning(c.Dir, "task-site") {
		t.Error("a busy session was closed")
	}
}

// A session an earlier version launched as a systemd unit is still listed and stopped.
func TestStopEarlierUnit(t *testing.T) {
	f, _ := launchFake(t)
	c, _ := f.load()
	// No session at all: said, never a silent success.
	if err := f.sys.Stop(c.Dir, "task-site"); !errors.Is(err, ErrNoSession) {
		t.Fatalf("no session: %v", err)
	}
	if err := f.sys.StopWhenIdle(c.Dir, "task-site", time.Minute); !errors.Is(err, ErrNoSession) {
		t.Fatalf("no session, when idle: %v", err)
	}
	// A unit of an earlier version: stopped, by stop and by --when-idle alike (it tells
	// nothing of its answers; on 10-nov --when-idle left it running and said nothing).
	f.units = []string{LaunchUnit("task-site")}
	for _, when := range []bool{false, true} {
		f.calls = nil
		var err error
		if when {
			err = f.sys.StopWhenIdle(c.Dir, "task-site", time.Minute)
		} else {
			err = f.sys.Stop(c.Dir, "task-site")
		}
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Contains(f.calls, "systemctl --user stop "+LaunchUnit("task-site")) {
			t.Errorf("when idle %v: calls: %v", when, f.calls)
		}
	}
}

// retire makes the task's session one Claude retired after hours idle: its process
// ends, it stays listed with no pid (measured on 11-nov, 2.1.322).
func (f *fake) retire(id string) {
	delete(f.live, id)
	f.retired[id] = true
}

// A session Claude retired is listed as retired, not active; --when-idle closes it
// at once (it answers nothing), and a new session of the task replaces it.
func TestRetiredSession(t *testing.T) {
	f, repo := launchFake(t)
	c, set := f.load()
	f.trust(repo)
	l := Launch{Task: "task-site", Name: "x", Folder: repo}
	if err := f.sys.Start(c, set, l, "go"); err != nil {
		t.Fatal(err)
	}
	f.retire("ab12cd34")
	got, err := f.sys.Launches(c.Dir)
	if err != nil || len(got) != 1 || got[0].Active != Retired {
		t.Fatalf("launches: %+v %v", got, err)
	}
	if f.sys.LaunchRunning(c.Dir, "task-site") {
		t.Error("a retired session counted as running")
	}
	if !HasLaunch(c.Dir, "task-site") {
		t.Error("a retired session has no record")
	}
	// Another task may not take its folder: it comes back there when opened.
	if err := f.sys.Start(c, set, Launch{Task: "task-other", Name: "y", Folder: repo}, "go"); !errors.Is(err, ErrServed) {
		t.Errorf("another task in a retired session's folder: %v", err)
	}
	// The task itself starts again; the retired one leaves Claude's lists.
	f.bgOut = "backgrounded · ef56ab78 · x\n"
	if err := f.sys.Start(c, set, l, "go"); err != nil {
		t.Fatalf("starting over a retired session: %v", err)
	}
	if f.retired["ab12cd34"] {
		t.Error("the retired session is still in Claude's lists")
	}
	// --when-idle does not wait for a retired session.
	f.retire("ef56ab78")
	f.calls = nil
	if err := f.sys.StopWhenIdle(c.Dir, "task-site", 10*time.Second); err != nil {
		t.Fatal(err)
	}
	if f.retired["ef56ab78"] || HasLaunch(c.Dir, "task-site") {
		t.Errorf("not closed: retired %v, calls %v", f.retired, f.calls)
	}
}

// session start brings back a session Claude stopped, with its id and conversation
// and no first message (12-nov, 2.1.322: claude respawn); only in its own folder, and
// when Claude cannot, a new one replaces it as before.
func TestWakeRetiredSession(t *testing.T) {
	f, repo := launchFake(t)
	c, _ := f.load()
	f.trust(repo)
	l := Launch{Task: "task-site", Name: "x", Folder: repo}
	if woke, err := f.sys.Wake(c.Dir, l); woke || err != nil {
		t.Fatalf("no session to bring back: %v %v", woke, err)
	}
	_, set := f.load()
	if err := f.sys.Start(c, set, l, "go"); err != nil {
		t.Fatal(err)
	}
	if woke, err := f.sys.Wake(c.Dir, l); woke || err != nil {
		t.Errorf("a running session: %v %v", woke, err)
	}
	f.retire("ab12cd34")
	other := t.TempDir()
	if woke, err := f.sys.Wake(c.Dir, Launch{Task: "task-site", Name: "x", Folder: other}); woke || err != nil || !f.retired["ab12cd34"] {
		t.Errorf("another folder: %v %v", woke, err)
	}
	f.ranIn = nil
	woke, err := f.sys.Wake(c.Dir, l)
	if !woke || err != nil || !f.live["ab12cd34"] || f.retired["ab12cd34"] || len(f.ranIn) > 0 {
		t.Fatalf("woke %v %v, live %v, launched %v", woke, err, f.live, f.ranIn)
	}
	if !containsSuffix(f.calls, " respawn ab12cd34") {
		t.Errorf("calls: %v", f.calls)
	}
	// Claude refuses: nothing woke, Start replaces it.
	f.retire("ab12cd34")
	f.noRespawn = true
	if woke, err := f.sys.Wake(c.Dir, l); woke || err != nil {
		t.Errorf("a refused respawn: %v %v", woke, err)
	}
	// Logged out: said, not tried.
	f.noRespawn = false
	f.auth = `{"loggedIn": false}`
	if _, err := f.sys.Wake(c.Dir, l); !errors.Is(err, ErrLoggedOut) {
		t.Errorf("logged out: %v", err)
	}
}

// A claude that does not know agents --all is asked without it.
func TestAgentsWithoutAll(t *testing.T) {
	f, repo := launchFake(t)
	c, set := f.load()
	f.trust(repo)
	if err := f.sys.Start(c, set, Launch{Task: "task-site", Name: "x", Folder: repo}, "go"); err != nil {
		t.Fatal(err)
	}
	f.oldAgents = true
	got, err := f.sys.Launches(c.Dir)
	if err != nil || len(got) != 1 || got[0].Active != "active" {
		t.Fatalf("launches: %+v %v", got, err)
	}
}

func containsSuffix(list []string, suffix string) bool {
	for _, x := range list {
		if strings.HasSuffix(x, suffix) {
			return true
		}
	}
	return false
}

// A session that waits for a permission is busy (measured on 11-nov): --when-idle
// waits for it, and a stop that cannot remove a running session says so.
func TestWaitingForPermissionIsBusy(t *testing.T) {
	f, repo := launchFake(t)
	c, set := f.load()
	f.trust(repo)
	if err := f.sys.Start(c, set, Launch{Task: "task-site", Name: "x", Folder: repo}, "go"); err != nil {
		t.Fatal(err)
	}
	f.busy["ab12cd34"] = 1000
	if err := f.sys.StopWhenIdle(c.Dir, "task-site", 10*time.Second); !errors.Is(err, ErrStillBusy) {
		t.Errorf("a session waiting for a permission: %v", err)
	}
	f.noDaemon = true // rm fails, and the session runs
	if err := f.sys.Stop(c.Dir, "task-site"); err == nil || !HasLaunch(c.Dir, "task-site") {
		t.Errorf("a running session that claude could not remove: %v", err)
	}
}

// Claude's daemon exits once unused, and then claude stop and rm fail however often
// they are tried (11-nov): stop starts it, then removes the session.
func TestStopWakesDaemon(t *testing.T) {
	f, repo := launchFake(t)
	c, set := f.load()
	f.trust(repo)
	if err := f.sys.Start(c, set, Launch{Task: "task-site", Name: "x", Folder: repo}, "go"); err != nil {
		t.Fatal(err)
	}
	f.retire("ab12cd34")
	f.noDaemon = true
	f.calls = nil
	if err := f.sys.Stop(c.Dir, "task-site"); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(f.calls, "start daemon") || f.retired["ab12cd34"] || HasLaunch(c.Dir, "task-site") {
		t.Errorf("calls %v, retired %v", f.calls, f.retired)
	}
	// A daemon that does not come up: the record goes, and the error names the session
	// Claude still shows.
	if err := f.sys.Start(c, set, Launch{Task: "task-site", Name: "x", Folder: repo}, "go"); err != nil {
		t.Fatal(err)
	}
	f.retire("ab12cd34")
	f.noDaemon, f.daemonStays = true, false
	err := f.sys.Stop(c.Dir, "task-site")
	var left *LeftInClaude
	if !errors.As(err, &left) || left.ID != "ab12cd34" || HasLaunch(c.Dir, "task-site") {
		t.Errorf("a daemon that does not start: %v, record %v", err, HasLaunch(c.Dir, "task-site"))
	}
}

// Lines of a transcript as Claude Code 2.1.322 writes them (11-nov).
const (
	tPrompt  = `{"type":"user","message":{"role":"user","content":"Start with your task."}}` + "\n"
	tLogin   = `{"type":"assistant","message":{"model":"<synthetic>","role":"assistant","content":[{"type":"text","text":"Login reached its time limit · Please run /login"}]},"error":"authentication_failed","isApiErrorMessage":true}` + "\n"
	tAnswer  = `{"type":"assistant","message":{"model":"claude-opus-5-5","role":"assistant","content":[{"type":"text","text":"Reading the task."}]}}` + "\n"
	tOverrun = `{"type":"assistant","message":{"model":"<synthetic>","role":"assistant","content":[{"type":"text","text":"API Error: 529 Overloaded"}]},"error":"unknown","isApiErrorMessage":true}` + "\n"
)

// With no login, or an expired one, nothing is launched: the session would only fail.
// When claude auth status says nothing readable, the launch goes ahead.
func TestStartNeedsLogin(t *testing.T) {
	f, repo := launchFake(t)
	c, set := f.load()
	f.trust(repo)
	l := Launch{Task: "task-site", Name: "x", Folder: repo}
	for _, out := range []string{`{"loggedIn": false}`, `{"loggedIn": true, "loginExpiresAt": "2020-01-01T00:00:00.000Z"}`} {
		f.auth = out
		if err := f.sys.Start(c, set, l, "go"); !errors.Is(err, ErrLoggedOut) {
			t.Fatalf("%s: %v", out, err)
		}
	}
	if len(f.ranIn) != 0 {
		t.Fatalf("launched with no login: %v", f.ranIn)
	}
	f.auth = `{"loggedIn": true, "loginExpiresAt": "2999-01-01T00:00:00.000Z"}`
	if err := f.sys.Start(c, set, l, "go"); err != nil {
		t.Fatal(err)
	}
	if err := f.sys.Stop(c.Dir, "task-site"); err != nil {
		t.Fatal(err)
	}
	f.auth = "Not a JSON answer"
	if err := f.sys.Start(c, set, l, "go"); err != nil {
		t.Fatalf("an unreadable auth status: %v", err)
	}
}

// A session whose first turn fails (its login expired, as on 11-nov) is reported with
// Claude's words, stopped, taken out of Claude's lists, and leaves no record.
func TestStartFirstTurnFails(t *testing.T) {
	f, repo := launchFake(t)
	c, set := f.load()
	f.trust(repo)
	f.sys.RunIn = func(string, string, ...string) (string, error) {
		f.live["ab12cd34"] = true
		f.transcript("ab12cd34", tPrompt+tLogin)
		return f.bgOut, nil
	}
	err := f.sys.Start(c, set, Launch{Task: "task-site", Name: "x", Folder: repo}, "go")
	var failed *FirstTurnFailed
	if !errors.As(err, &failed) || !failed.Login || failed.Cause != "Login reached its time limit · Please run /login" {
		t.Fatalf("got %v", err)
	}
	for _, want := range []string{"claude stop ab12cd34", "claude rm ab12cd34"} {
		if !slices.ContainsFunc(f.calls, func(c string) bool { return strings.HasSuffix(c, want) }) {
			t.Errorf("no %q: %v", want, f.calls)
		}
	}
	if got, _ := f.sys.Launches(c.Dir); len(got) != 0 {
		t.Errorf("a failed first turn left %+v", got)
	}
	// Another API error is not about the login.
	f.sys.RunIn = func(string, string, ...string) (string, error) {
		f.live["ab12cd34"] = true
		f.transcript("ab12cd34", tPrompt+tOverrun)
		return f.bgOut, nil
	}
	err = f.sys.Start(c, set, Launch{Task: "task-site", Name: "x", Folder: repo}, "go")
	if !errors.As(err, &failed) || failed.Login || failed.Cause != "API Error: 529 Overloaded" {
		t.Fatalf("overloaded: %v", err)
	}
}

// Start waits for the first answer, and returns as soon as it comes; a session that
// does not answer within FirstTurnWait is left running.
func TestStartWaitsForFirstAnswer(t *testing.T) {
	f, repo := launchFake(t)
	c, set := f.load()
	f.trust(repo)
	sleeps := 0
	f.sys.Sleep = func(time.Duration) {
		sleeps++
		if sleeps == 12 {
			f.transcript("ab12cd34", tPrompt+tAnswer)
		}
	}
	f.sys.RunIn = func(string, string, ...string) (string, error) {
		f.live["ab12cd34"] = true
		f.transcript("ab12cd34", tPrompt)
		return f.bgOut, nil
	}
	if err := f.sys.Start(c, set, Launch{Task: "task-site", Name: "x", Folder: repo}, "go"); err != nil {
		t.Fatal(err)
	}
	if sleeps != 12 {
		t.Errorf("waited %d seconds, want 12", sleeps)
	}
	if err := f.sys.Stop(c.Dir, "task-site"); err != nil {
		t.Fatal(err)
	}
	sleeps = -1000 // it never answers
	if err := f.sys.Start(c, set, Launch{Task: "task-site", Name: "x", Folder: repo}, "go"); err != nil {
		t.Fatal(err)
	}
	if got := sleeps + 1000; got != int(FirstTurnWait/time.Second) {
		t.Errorf("waited %d seconds, want %d", got, int(FirstTurnWait/time.Second))
	}
	if got, _ := f.sys.Launches(c.Dir); len(got) != 1 || got[0].Active != "active" {
		t.Errorf("a slow session: %+v", got)
	}
}

// A session whose last turn ended in an API error is listed as failed, with Claude's
// words; it still holds its folder against another task, and its own task can start
// again, which stops it first.
func TestLaunchesFailed(t *testing.T) {
	f, repo := launchFake(t)
	c, set := f.load()
	f.trust(repo)
	f.transcript("ab12cd34", tPrompt+tAnswer)
	if err := f.sys.Start(c, set, Launch{Task: "task-site", Name: "x", Folder: repo}, "go"); err != nil {
		t.Fatal(err)
	}
	f.transcript("ab12cd34", tPrompt+tAnswer+tPrompt+tLogin)
	got, err := f.sys.Launches(c.Dir)
	if err != nil || len(got) != 1 || got[0].Active != Failed || got[0].Problem != "Login reached its time limit · Please run /login" {
		t.Fatalf("launches: %+v %v", got, err)
	}
	if err := f.sys.Start(c, set, Launch{Task: "task-other", Name: "y", Folder: repo}, "go"); !errors.Is(err, ErrServed) {
		t.Fatalf("another task in its folder: %v", err)
	}
	f.bgOut = "backgrounded · ef56ab78 · x\n"
	f.transcript("ef56ab78", tPrompt+tAnswer)
	f.calls = nil
	if err := f.sys.Start(c, set, Launch{Task: "task-site", Name: "x", Folder: repo}, "go"); err != nil {
		t.Fatalf("starting its task again: %v", err)
	}
	for _, want := range []string{"claude stop ab12cd34", "claude rm ab12cd34"} {
		if !slices.ContainsFunc(f.calls, func(c string) bool { return strings.HasSuffix(c, want) }) {
			t.Errorf("no %q: %v", want, f.calls)
		}
	}
	if got, _ := f.sys.Launches(c.Dir); len(got) != 1 || got[0].Unit != "ef56ab78" || got[0].Active != "active" {
		t.Errorf("after starting again: %+v", got)
	}
}

// A session knows the task it was launched for: by its session id, which begins with
// the record's short id, or, after /clear, by its job folder, only when it runs in the
// record's folder (a variable may be the daemon's). Any other session has none.
func TestLaunchedTask(t *testing.T) {
	cfgDir := t.TempDir()
	folder := filepath.Join(t.TempDir(), "site")
	if err := os.MkdirAll(filepath.Join(folder, "sub"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(launchRecords(cfgDir), 0o700); err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(launchRecord{ID: "f770712a", Task: "task-new-website", Folder: folder})
	if err := os.WriteFile(launchRecordPath(cfgDir, "task-new-website"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	job := filepath.Join(t.TempDir(), "jobs", "f770712a")
	for _, c := range []struct {
		name, session, job, cwd, want string
	}{
		{"its session id", "f770712a-11c6-4dea-9446-5cc5820b347b", "", "", "task-new-website"},
		{"after /clear, in its folder", "0c0c0c0c-0000-4000-8000-000000000000", job, filepath.Join(folder, "sub"), "task-new-website"},
		{"its job folder elsewhere", "0c0c0c0c-0000-4000-8000-000000000000", job, t.TempDir(), ""},
		{"another session", "0c0c0c0c-0000-4000-8000-000000000000", "", folder, ""},
		{"no session id", "", "", "", ""},
	} {
		if got := LaunchedTask(cfgDir, c.session, c.job, c.cwd); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
	if got := LaunchedTask(t.TempDir(), "f770712a-11c6", "", ""); got != "" {
		t.Errorf("no records: %q", got)
	}
}
