package remote

import (
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
		if strings.HasSuffix(call, "agents --json") {
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
	if err := f.sys.Stop(c.Dir, "task-site"); err != nil {
		t.Fatal(err)
	}
	if len(f.calls) == 0 || f.calls[0] != "systemctl --user stop "+LaunchUnit("task-site") {
		t.Errorf("calls: %v", f.calls)
	}
}
