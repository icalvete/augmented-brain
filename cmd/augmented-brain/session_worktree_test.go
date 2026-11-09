package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/paths"
	"github.com/icalvete/augmented-brain/internal/remote"
	"github.com/icalvete/augmented-brain/internal/testenv"
)

// A task of a bare repository gets a worktree of its own, <repo>-<task>, on a new branch
// from origin's default branch as it is now; a second start reuses it; stop removes it
// with its branch when nothing would be lost, and keeps it otherwise. Decided on 9-nov:
// two tasks of one repository must not share a folder, nor start on old code.
func TestTaskWorktree(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git")
	}
	if runtime.GOOS == "windows" {
		t.Skip("session start is Linux only")
	}
	testenv.CleanGit()
	// Resolved: on macOS the temporary folder is behind a link, and git writes real paths.
	root, err := paths.Resolve(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	g := func(dir string, args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=Lucia", "-c", "user.email=lucia@ejemplo.com", "-c", "init.defaultBranch=main"}, args...)...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	origin := filepath.Join(root, "origin")
	if err := os.MkdirAll(origin, 0o755); err != nil {
		t.Fatal(err)
	}
	g(origin, "init", "-q")
	if err := os.WriteFile(filepath.Join(origin, "a.txt"), []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	g(origin, "add", "a.txt")
	g(origin, "commit", "-q", "-m", "one")
	// The layout of the 9-nov machine: a folder whose .git is a bare repository.
	repo := filepath.Join(root, "site")
	g(root, "clone", "-q", "--bare", origin, filepath.Join(repo, ".git"))
	g(repo, "config", "remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*")
	// origin moves on after the clone: the worktree must start from the new commit.
	if err := os.WriteFile(filepath.Join(origin, "a.txt"), []byte("two\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	g(origin, "commit", "-q", "-am", "two")
	newest := g(origin, "rev-parse", "HEAD")

	c := &config.Config{Dir: filepath.Join(root, "config")}
	wt, base, made, fetched, err := taskWorktree(c, repo, "task-new-website")
	if err != nil || !made || !fetched || base != "origin/main" {
		t.Fatalf("taskWorktree = %q %q made=%v fetched=%v %v", wt, base, made, fetched, err)
	}
	if want := filepath.Join(root, "site-new-website"); wt != want {
		t.Errorf("worktree at %q, want %q", wt, want)
	}
	if got := g(wt, "rev-parse", "HEAD"); got != newest {
		t.Errorf("worktree at %s, want origin's newest %s", got, newest)
	}
	if got := g(wt, "rev-parse", "--abbrev-ref", "HEAD"); got != "task-new-website" {
		t.Errorf("branch %q", got)
	}
	if out, _ := exec.Command("git", "-C", wt, "rev-parse", "--abbrev-ref", "@{upstream}").CombinedOutput(); !strings.Contains(string(out), "no upstream") {
		t.Errorf("the task's branch tracks something: %s", out)
	}
	if !paths.Same(remote.WorktreeRoot(wt), repo) {
		t.Errorf("WorktreeRoot(%s) = %q", wt, remote.WorktreeRoot(wt))
	}
	// A second start reuses it.
	if again, _, made, _, err := taskWorktree(c, repo, "task-new-website"); err != nil || made || again != wt {
		t.Errorf("second start: %q made=%v %v", again, made, err)
	}
	// Stop keeps it while it holds something of its own.
	if err := os.WriteFile(filepath.Join(wt, "b.txt"), []byte("draft\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, removed, why := removeTaskWorktree(c, "task-new-website"); removed || why != "changes" {
		t.Errorf("with a change: removed=%v why=%q", removed, why)
	}
	g(wt, "add", "b.txt")
	g(wt, "commit", "-q", "-m", "draft")
	if _, removed, why := removeTaskWorktree(c, "task-new-website"); removed || why != "commits" {
		t.Errorf("with a commit not pushed: removed=%v why=%q", removed, why)
	}
	g(wt, "reset", "-q", "--hard", "origin/main")
	if _, removed, why := removeTaskWorktree(c, "task-new-website"); !removed {
		t.Fatalf("clean worktree not removed: %q", why)
	}
	if _, err := os.Stat(wt); !os.IsNotExist(err) {
		t.Errorf("the worktree is still there: %v", err)
	}
	if out, _ := exec.Command("git", "-C", repo, "branch", "--list", "task-new-website").Output(); strings.TrimSpace(string(out)) != "" {
		t.Errorf("the branch is still there: %s", out)
	}
	// A folder of that name that is not ours is never used.
	if err := os.MkdirAll(wt, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, _, _, _, err := taskWorktree(c, repo, "task-new-website"); err != errNotOurWorktree {
		t.Errorf("a folder in the way: %v", err)
	}
	// Stop with no worktree of ours says nothing.
	if f, removed, why := removeTaskWorktree(c, "task-other"); f != "" || removed || why != "" {
		t.Errorf("no record: %q %v %q", f, removed, why)
	}
}
