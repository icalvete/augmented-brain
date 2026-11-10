package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/paths"
	"github.com/icalvete/augmented-brain/internal/remote"
)

// A working copy of a task's repository wins over a bare one of the same name; a bare
// one alone is still the task's repository (it then gets a worktree of its own).
func TestTaskFolderSkipsBare(t *testing.T) {
	wf := t.TempDir()
	bare := filepath.Join(wf, "site")
	if err := os.MkdirAll(filepath.Join(bare, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bare, ".git", "config"), []byte("[core]\n\trepositoryformatversion = 0\n\tbare = true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := &config.Space{WorkFolders: []string{wf}}
	if got := taskRepo(s, []string{"site"}); got != bare {
		t.Errorf("taskRepo = %q, want the bare repository %s when it is the only one", got, bare)
	}
	// The same repository found deeper, as a working tree, is the folder.
	work := filepath.Join(wf, "code", "site")
	if err := os.MkdirAll(filepath.Join(work, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(work, ".git", "config"), []byte("[core]\n\tbare = false\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := taskRepo(s, []string{"site"}); got != work {
		t.Errorf("taskRepo = %q, want the working copy %q over the bare one", got, work)
	}
}

// A bare repository given with --folder gets the task's own worktree, as one found by
// the task's repositories does: on 10-nov it reused one of its worktrees, and the
// session opened on another branch's code.
func TestSessionFolderBareGetsTaskWorktree(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git")
	}
	if runtime.GOOS == "windows" {
		t.Skip("session start is Linux only")
	}
	root, err := paths.Resolve(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	git := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "init.defaultBranch=main"}, args...)...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	origin, repo := filepath.Join(root, "origin"), filepath.Join(root, "work", "site")
	if err := os.MkdirAll(origin, 0o755); err != nil {
		t.Fatal(err)
	}
	git(origin, "init", "-q")
	git(origin, "commit", "-q", "--allow-empty", "-m", "one")
	git(root, "clone", "-q", "--bare", origin, filepath.Join(repo, ".git"))
	git(repo, "config", "remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*")
	// A worktree of the repository that is not the task's.
	other := filepath.Join(root, "work", "site-other")
	git(repo, "worktree", "add", "-q", "-b", "other", other)

	c := &config.Config{Dir: filepath.Join(root, "config"), Language: "en"}
	s := &config.Space{Name: "personal", WorkFolders: []string{filepath.Join(root, "work")}}
	sys := remote.System{GOOS: "linux", Run: func(string, ...string) (string, error) { return "", nil }}
	var out, errOut bytes.Buffer
	dir, code := sessionFolder(c, s, sys, "task-new-website", nil, repo, &out, &errOut)
	want, _ := remote.TaskWorktreePath(repo, "task-new-website")
	if code != 0 || dir != want {
		t.Fatalf("folder %q (code %d, %s), want the task's worktree %q", dir, code, errOut.String(), want)
	}
	if !isGitRepo(dir) {
		t.Errorf("the task's worktree %s was not made", dir)
	}
	// A plain folder given with --folder is used as it is.
	if dir, code := sessionFolder(c, s, sys, "task-new-website", nil, other, &out, &errOut); code != 0 || dir != other {
		t.Errorf("a working copy: %q %d", dir, code)
	}
}
