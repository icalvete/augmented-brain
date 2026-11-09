package remote

import (
	"errors"
	"os"
	"path/filepath"
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
		if got := worktreeRoot(wt); got != repo {
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
	if got := worktreeRoot(f.root); got != "" {
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
	if err := f.sys.Start(c, set, l); !errors.Is(err, ErrUntrusted) {
		t.Fatalf("untrusted folder: %v", err)
	}
	// The space's service folder is served already.
	own := filepath.Join(f.root, "p")
	f.trust(own)
	if err := f.sys.Start(c, set, Launch{Task: "task-site", Name: "x", Folder: own}); !errors.Is(err, ErrServed) {
		t.Fatalf("service folder: %v", err)
	}
	for _, call := range f.calls {
		if strings.HasPrefix(call, "systemd-run") {
			t.Fatalf("started although refused: %s", call)
		}
	}
	f.sys.GOOS = "darwin"
	if err := f.sys.Start(c, set, l); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("macOS: %v", err)
	}
}

func TestStartRuns(t *testing.T) {
	f, repo := launchFake(t)
	c, set := f.load()
	f.trust(repo)
	unit := LaunchUnit("task-site")
	f.active[unit] = "active"
	if err := f.sys.Start(c, set, Launch{Task: "task-site", Name: "box · Site", Folder: repo}); err != nil {
		t.Fatal(err)
	}
	var run string
	for _, call := range f.calls {
		if strings.HasPrefix(call, "systemd-run") {
			run = call
		}
	}
	for _, want := range []string{"--user", "--unit=" + unit, "--working-directory=" + repo,
		"--setenv=" + TaskEnv + "=task-site", "--setenv=PATH=" + filepath.Join(f.root, ".local", "bin"), "remote-control --name box · Site", "--permission-mode default", "--spawn session"} {
		if !strings.Contains(run, want) {
			t.Errorf("systemd-run lacks %q: %s", want, run)
		}
	}
}

// A server that exits at once (or writes an error and waits) is reported, and stopped.
func TestStartReportsFailure(t *testing.T) {
	f, repo := launchFake(t)
	c, set := f.load()
	f.trust(repo)
	err := f.sys.Start(c, set, Launch{Task: "task-site", Name: "x", Folder: repo})
	if err == nil || !strings.Contains(err.Error(), "did not start") {
		t.Fatalf("a unit that is not active: %v", err)
	}
	stopped := false
	for _, call := range f.calls {
		if call == "systemctl --user stop "+LaunchUnit("task-site") {
			stopped = true
		}
	}
	if !stopped {
		t.Error("the failed unit was not stopped")
	}
}

func TestStop(t *testing.T) {
	f, _ := launchFake(t)
	if err := f.sys.Stop("task-site"); err != nil {
		t.Fatal(err)
	}
	if len(f.calls) == 0 || f.calls[0] != "systemctl --user stop "+LaunchUnit("task-site") {
		t.Errorf("calls: %v", f.calls)
	}
}
