package remote

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The worktrees a task's sessions make, through task_worktree or by hand and then noted,
// go when the task is closed: those with nothing of their own, with the branch made for
// them; one with work stays, and so does the one a session still works in. On 10-nov a
// session left two behind.
func TestNotedWorktrees(t *testing.T) {
	w := newGitWorld(t)
	made, branch, base, ok, _, err := MakeTaskWorktree(w.cfgDir, w.repo, "task-site")
	if err != nil || !ok || base != "origin/main" || branch != "task-site" {
		t.Fatalf("make: %q %q %q %v %v", made, branch, base, ok, err)
	}
	if again, _, _, ok, _, err := MakeTaskWorktree(w.cfgDir, w.repo, "task-site"); err != nil || ok || again != made {
		t.Fatalf("again: %q %v %v", again, ok, err)
	}
	byHand := filepath.Join(w.root, "site-by-hand")
	w.git(w.repo, "worktree", "add", "-q", "-b", "by-hand", byHand, "origin/main")
	busy := filepath.Join(w.root, "site-busy")
	w.git(w.repo, "worktree", "add", "-q", "-b", "busy", busy, "origin/main")
	inUse := filepath.Join(w.root, "site-here")
	w.git(w.repo, "worktree", "add", "-q", "-b", "here", inUse, "origin/main")
	for _, f := range []string{byHand, busy, inUse} {
		if err := NoteTaskWorktree(w.cfgDir, "task-site", f); err != nil {
			t.Fatal(err)
		}
	}
	if err := NoteTaskWorktree(w.cfgDir, "task-site", w.root); err == nil {
		t.Error("noted a folder that is no worktree")
	}
	if err := os.WriteFile(filepath.Join(busy, "half.txt"), []byte("half done\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	removed, kept := RemoveNotedWorktrees(w.cfgDir, "task-site", filepath.Join(inUse, "sub"))
	if strings.Join(removed, " ") != strings.Join([]string{made, byHand}, " ") && strings.Join(removed, " ") != strings.Join([]string{byHand, made}, " ") {
		t.Errorf("removed %v, want %s and %s", removed, made, byHand)
	}
	whys := map[string]string{}
	for _, k := range kept {
		whys[k.Folder] = k.Why
	}
	if whys[busy] != KeptChanges || whys[inUse] != KeptInUse || len(kept) != 2 {
		t.Errorf("kept %+v", kept)
	}
	branches := w.git(w.repo, "branch", "--list")
	if strings.Contains(branches, "task-site") || !strings.Contains(branches, "by-hand") {
		t.Errorf("branches: the one made for the task goes, a noted one's stays:\n%s", branches)
	}
	// What stays is still noted, for the next try.
	if got := readNoted(w.cfgDir, "task-site"); len(got) != 2 {
		t.Errorf("still noted: %+v", got)
	}
}
