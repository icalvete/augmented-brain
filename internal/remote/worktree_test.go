package remote

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/paths"
	"github.com/icalvete/augmented-brain/internal/testenv"
)

// gitWorld is an origin repository and a bare clone kept in <root>/site/.git, the
// layout of the 9-nov machine.
type gitWorld struct {
	t                  *testing.T
	root, origin, repo string
	cfgDir             string
}

func (w *gitWorld) git(dir string, args ...string) string {
	w.t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=Lucia", "-c", "user.email=lucia@ejemplo.com", "-c", "init.defaultBranch=main"}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		w.t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func (w *gitWorld) commit(text string) string {
	w.t.Helper()
	if err := os.WriteFile(filepath.Join(w.origin, "a.txt"), []byte(text+"\n"), 0o644); err != nil {
		w.t.Fatal(err)
	}
	w.git(w.origin, "add", "a.txt")
	w.git(w.origin, "commit", "-q", "-m", text)
	return w.git(w.origin, "rev-parse", "HEAD")
}

func newGitWorld(t *testing.T) *gitWorld {
	t.Helper()
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
	w := &gitWorld{t: t, root: root, origin: filepath.Join(root, "origin"), repo: filepath.Join(root, "site"), cfgDir: filepath.Join(root, "config")}
	if err := os.MkdirAll(w.origin, 0o755); err != nil {
		t.Fatal(err)
	}
	w.git(w.origin, "init", "-q")
	w.commit("one")
	w.git(root, "clone", "-q", "--bare", w.origin, filepath.Join(w.repo, ".git"))
	w.git(w.repo, "config", "remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*")
	return w
}

// A task of a bare repository gets a worktree of its own, <repo>-<task>, on a new branch
// from origin's default branch as it is now; a second start reuses it; removing it
// keeps it while it holds work of its own. Decided on 9-nov: two tasks of one
// repository must not share a folder, nor start on old code.
func TestTaskWorktree(t *testing.T) {
	w := newGitWorld(t)
	newest := w.commit("two") // origin moves on after the clone
	wt, base, made, fetched, err := TaskWorktree(w.cfgDir, w.repo, "task-new-website")
	if err != nil || !made || !fetched || base != "origin/main" {
		t.Fatalf("TaskWorktree = %q %q made=%v fetched=%v %v", wt, base, made, fetched, err)
	}
	if want := filepath.Join(w.root, "site-new-website"); wt != want {
		t.Errorf("worktree at %q, want %q", wt, want)
	}
	if got := w.git(wt, "rev-parse", "HEAD"); got != newest {
		t.Errorf("worktree at %s, want origin's newest %s", got, newest)
	}
	if got := w.git(wt, "rev-parse", "--abbrev-ref", "HEAD"); got != "task-new-website" {
		t.Errorf("branch %q", got)
	}
	if out, _ := exec.Command("git", "-C", wt, "rev-parse", "--abbrev-ref", "@{upstream}").CombinedOutput(); !strings.Contains(string(out), "no upstream") {
		t.Errorf("the task's branch tracks something: %s", out)
	}
	if !paths.Same(WorktreeRoot(wt), w.repo) {
		t.Errorf("WorktreeRoot(%s) = %q", wt, WorktreeRoot(wt))
	}
	if again, _, made, _, err := TaskWorktree(w.cfgDir, w.repo, "task-new-website"); err != nil || made || again != wt {
		t.Errorf("second start: %q made=%v %v", again, made, err)
	}
	if err := os.WriteFile(filepath.Join(wt, "b.txt"), []byte("draft\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, removed, why := RemoveTaskWorktree(w.cfgDir, "task-new-website"); removed || why != KeptChanges {
		t.Errorf("with a change: removed=%v why=%q", removed, why)
	}
	w.git(wt, "add", "b.txt")
	w.git(wt, "commit", "-q", "-m", "draft")
	if _, removed, why := RemoveTaskWorktree(w.cfgDir, "task-new-website"); removed || why != KeptCommits {
		t.Errorf("with a commit not pushed: removed=%v why=%q", removed, why)
	}
	w.git(wt, "reset", "-q", "--hard", "origin/main")
	if _, removed, why := RemoveTaskWorktree(w.cfgDir, "task-new-website"); !removed {
		t.Fatalf("clean worktree not removed: %q", why)
	}
	if _, err := os.Stat(wt); !os.IsNotExist(err) {
		t.Errorf("the worktree is still there: %v", err)
	}
	if out := w.git(w.repo, "branch", "--list", "task-new-website"); out != "" {
		t.Errorf("the branch is still there: %s", out)
	}
	if err := os.MkdirAll(wt, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, _, _, _, err := TaskWorktree(w.cfgDir, w.repo, "task-new-website"); err != ErrNotOurWorktree {
		t.Errorf("a folder in the way: %v", err)
	}
	if f, removed, why := RemoveTaskWorktree(w.cfgDir, "task-other"); f != "" || removed || why != "" {
		t.Errorf("no record: %q %v %q", f, removed, why)
	}
}

// The sweep removes the worktrees of sessions that ended without session stop when
// they hold nothing of their own, keeps the rest, never touches one whose session runs
// nor a worktree made by hand; closing a task does the same for its own.
func TestSweepWorktrees(t *testing.T) {
	w := newGitWorld(t)
	running := map[string]bool{}
	sys := System{GOOS: "linux", Run: func(name string, args ...string) (string, error) {
		if name != "systemctl" {
			return "", nil
		}
		switch {
		case len(args) > 1 && args[1] == "list-units":
			var b strings.Builder
			for task := range running {
				b.WriteString(LaunchUnit(task) + " loaded active running x\n")
			}
			return b.String(), nil
		case len(args) > 2 && args[1] == "show":
			for task := range running {
				if args[2] == LaunchUnit(task) {
					return "ActiveState=active\nEnvironment=PATH=/usr/bin " + TaskEnv + "=" + task + "\n", nil
				}
			}
		}
		return "", nil
	}}
	mk := func(task string) string {
		wt, _, _, _, err := TaskWorktree(w.cfgDir, w.repo, task)
		if err != nil {
			t.Fatal(err)
		}
		return wt
	}
	clean, busy, live := mk("task-clean"), mk("task-busy"), mk("task-live")
	if err := os.WriteFile(filepath.Join(busy, "notes.txt"), []byte("half done\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	byHand := filepath.Join(w.root, "site-by-hand")
	w.git(w.repo, "worktree", "add", "-q", byHand, "origin/main")
	running["task-live"] = true

	removed, kept := sys.SweepWorktrees(w.cfgDir)
	if len(removed) != 1 || removed[0] != clean {
		t.Errorf("removed %v, want only %s", removed, clean)
	}
	if len(kept) != 1 || kept[0].Task != "task-busy" || kept[0].Why != KeptChanges {
		t.Errorf("kept %+v, want task-busy with changes", kept)
	}
	for _, d := range []string{busy, live, byHand} {
		if _, err := os.Stat(d); err != nil {
			t.Errorf("%s was touched: %v", d, err)
		}
	}
	// Closing a task whose session still runs leaves its worktree for later.
	if _, removed, _ := sys.CloseTaskWorktree(w.cfgDir, "task-live"); removed {
		t.Error("closing removed the worktree of a running session")
	}
	delete(running, "task-live")
	if f, removed, why := sys.CloseTaskWorktree(w.cfgDir, "task-live"); !removed || f != live {
		t.Errorf("closing after the session ended: %q %v %q", f, removed, why)
	}
	// Without systemd to ask, nothing is swept.
	sys.GOOS = "darwin"
	if removed, kept := sys.SweepWorktrees(w.cfgDir); removed != nil || kept != nil {
		t.Errorf("swept without knowing what runs: %v %v", removed, kept)
	}
}
