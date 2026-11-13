package remote

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// What the revision of nov-2026 found the cleanup losing (R-02, R-04, N-06, R-16): each
// case is a worktree whose work exists nowhere else, and it must stay.

// A worktree left on another commit (a review, a bisect) with commits of its own on the
// task's branch: the branch is what would be deleted, so it is what counts (R-02).
func TestRemoveKeepsBranchCommits(t *testing.T) {
	w := newGitWorld(t)
	wt, _, made, _, err := TaskWorktree(w.cfgDir, w.repo, "task-feature")
	if err != nil || !made {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wt, "work.txt"), []byte("three days of work\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	w.git(wt, "add", "work.txt")
	w.git(wt, "commit", "-q", "-m", "work never pushed")
	w.git(wt, "checkout", "-q", "--detach", "origin/main")
	if _, removed, why := RemoveTaskWorktree(w.cfgDir, "task-feature"); removed || why != KeptCommits {
		t.Fatalf("removed=%v why=%q, want kept with %q", removed, why, KeptCommits)
	}
	if w.git(w.repo, "branch", "--list", "task-feature") == "" {
		t.Fatal("the task's branch, with a commit on no remote, was deleted")
	}
	if _, err := os.Stat(wt); err != nil {
		t.Fatalf("the worktree went: %v", err)
	}
}

// Files git ignores (a .env, a key) stay with their worktree; what a build makes again
// (node_modules) does not keep it (R-04).
func TestRemoveKeepsIgnoredFiles(t *testing.T) {
	w := newGitWorld(t)
	if err := os.WriteFile(filepath.Join(w.origin, ".gitignore"), []byte(".env\nnode_modules/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	w.git(w.origin, "add", ".gitignore")
	w.git(w.origin, "commit", "-q", "-m", "ignore")
	wt, _, made, _, err := TaskWorktree(w.cfgDir, w.repo, "task-secret")
	if err != nil || !made {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(wt, "node_modules", "left-pad"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wt, "node_modules", "left-pad", "index.js"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	secret := filepath.Join(wt, ".env")
	if err := os.WriteFile(secret, []byte("API_KEY=only-copy\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, removed, why := RemoveTaskWorktree(w.cfgDir, "task-secret"); removed || why != KeptIgnored {
		t.Fatalf("removed=%v why=%q, want kept with %q", removed, why, KeptIgnored)
	}
	if _, err := os.Stat(secret); err != nil {
		t.Fatalf(".env went: %v", err)
	}
	// Without the .env, only node_modules is left: it goes.
	if err := os.Remove(secret); err != nil {
		t.Fatal(err)
	}
	if _, removed, why := RemoveTaskWorktree(w.cfgDir, "task-secret"); !removed {
		t.Fatalf("a worktree with only node_modules ignored stays: %q", why)
	}
}

// A file git status hides by the user's settings, or ignores, is work all the same
// (N-06).
func TestRemoveSeesHiddenWork(t *testing.T) {
	for _, mode := range []string{"plain", "showUntrackedFiles=no", "global-ignore", "info-exclude"} {
		t.Run(mode, func(t *testing.T) {
			w := newGitWorld(t)
			w.git(w.repo, "fetch", "-q", "origin")
			wt := filepath.Join(w.root, "site-feature")
			w.git(w.repo, "worktree", "add", "-q", "-b", "feature", wt, "origin/main")
			file := filepath.Join(wt, "new-module.go")
			want := KeptChanges
			cfg := filepath.Join(w.root, "gitconfig")
			switch mode {
			case "showUntrackedFiles=no":
				if err := os.WriteFile(cfg, []byte("[status]\n\tshowUntrackedFiles = no\n"), 0o644); err != nil {
					t.Fatal(err)
				}
				t.Setenv("GIT_CONFIG_GLOBAL", cfg)
			case "global-ignore":
				ign := filepath.Join(w.root, "globalignore")
				if err := os.WriteFile(ign, []byte(".env\n"), 0o644); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(cfg, []byte("[core]\n\texcludesFile = "+filepath.ToSlash(ign)+"\n"), 0o644); err != nil {
					t.Fatal(err)
				}
				t.Setenv("GIT_CONFIG_GLOBAL", cfg)
				file, want = filepath.Join(wt, ".env"), KeptIgnored
			case "info-exclude":
				if err := os.MkdirAll(filepath.Join(w.repo, ".git", "info"), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(w.repo, ".git", "info", "exclude"), []byte("notes-*.md\n"), 0o644); err != nil {
					t.Fatal(err)
				}
				file, want = filepath.Join(wt, "notes-of-three-days.md"), KeptIgnored
			}
			if err := os.WriteFile(file, []byte("work that exists nowhere else\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := NoteTaskWorktree(w.cfgDir, "task-x", wt); err != nil {
				t.Fatal(err)
			}
			removed, kept := RemoveNotedWorktrees(w.cfgDir, "task-x")
			if len(removed) != 0 || len(kept) != 1 || kept[0].Why != want {
				t.Errorf("removed=%v kept=%+v, want kept with %q", removed, kept, want)
			}
			if _, err := os.Stat(file); err != nil {
				t.Errorf("the file went: %v", err)
			}
		})
	}
}

// A record that names some other worktree is dropped, and that worktree and a branch
// of the user's named like the task stay (R-16).
func TestRemoveIgnoresForeignRecord(t *testing.T) {
	w := newGitWorld(t)
	mine := filepath.Join(w.root, "my-own-worktree")
	w.git(w.repo, "fetch", "-q", "origin")
	w.git(w.repo, "worktree", "add", "-q", "--detach", mine, "origin/main")
	w.git(w.repo, "branch", "task-x", "origin/main")
	rec := filepath.Join(worktreeRecords(w.cfgDir), "task-x")
	if err := os.MkdirAll(filepath.Dir(rec), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rec, []byte(mine+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if folder, removed, why := RemoveTaskWorktree(w.cfgDir, "task-x"); removed || folder != "" || why != "" {
		t.Fatalf("RemoveTaskWorktree = %q %v %q, want nothing of ours", folder, removed, why)
	}
	if _, err := os.Stat(mine); err != nil {
		t.Fatalf("the user's worktree went: %v", err)
	}
	if w.git(w.repo, "branch", "--list", "task-x") == "" {
		t.Fatal("the user's branch task-x went")
	}
	if _, err := os.Stat(rec); !os.IsNotExist(err) {
		t.Errorf("the record that pointed elsewhere stays: %v", err)
	}
}

// A folder with a tab in its path cannot be noted: read back, it would name another
// worktree and a branch (R-16).
func TestNoteRefusesOddPath(t *testing.T) {
	w := newGitWorld(t)
	w.git(w.repo, "fetch", "-q", "origin")
	odd := filepath.Join(w.root, "wt\tmain")
	w.git(w.repo, "worktree", "add", "-q", "--detach", odd, "origin/main")
	if err := NoteTaskWorktree(w.cfgDir, "task-t", odd); !errors.Is(err, ErrOddPath) {
		t.Fatalf("NoteTaskWorktree = %v, want ErrOddPath", err)
	}
	if got := readNoted(w.cfgDir, "task-t"); len(got) != 0 {
		t.Errorf("noted %+v", got)
	}
}

// A folder that has a .git file but is not a worktree git knows of is never removed
// (N-06).
func TestRemoveNeedsRegisteredWorktree(t *testing.T) {
	w := newGitWorld(t)
	wt, _, _, _, err := TaskWorktree(w.cfgDir, w.repo, "task-a")
	if err != nil {
		t.Fatal(err)
	}
	fake := filepath.Join(w.root, "not-a-worktree")
	if err := os.MkdirAll(fake, 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(wt, ".git"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fake, ".git"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	if done, why := RemoveWorktree(fake, ""); done || why != "" {
		t.Fatalf("RemoveWorktree = %v %q", done, why)
	}
	if _, err := os.Stat(fake); err != nil {
		t.Fatalf("the folder went: %v", err)
	}
	if !strings.Contains(w.git(w.repo, "worktree", "list"), wt) {
		t.Error("the real worktree was touched")
	}
}

// A worktree made a moment ago, whose session has no record yet, is not swept (R-05).
func TestSweepLeavesFreshWorktree(t *testing.T) {
	w := newGitWorld(t)
	sys := System{GOOS: "linux", Run: func(string, ...string) (string, error) { return "", nil },
		LookPath: func(string) (string, error) { return "/x/claude", nil }}
	wt, _, made, _, err := TaskWorktree(w.cfgDir, w.repo, "task-a")
	if err != nil || !made {
		t.Fatal(err)
	}
	if removed, _ := sys.SweepWorktrees(w.cfgDir); len(removed) != 0 {
		t.Fatalf("swept %v", removed)
	}
	if _, err := os.Stat(wt); err != nil {
		t.Fatal(err)
	}
	// An old record reused by a new session start counts as new again.
	ageRecords(t, w.cfgDir)
	if _, _, made, _, err := TaskWorktree(w.cfgDir, w.repo, "task-a"); err != nil || made {
		t.Fatalf("reuse: made=%v %v", made, err)
	}
	if removed, _ := sys.SweepWorktrees(w.cfgDir); len(removed) != 0 {
		t.Fatalf("swept a reused worktree: %v", removed)
	}
}
