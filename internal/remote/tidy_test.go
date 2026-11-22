package remote

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/icalvete/augmented-brain/internal/filelock"
)

// Records that outlived what they named go; what still names something, or a lock in
// use or too recent, stays. Nothing but records is touched.
func TestTidyRecords(t *testing.T) {
	cfgDir := t.TempDir()
	here := t.TempDir() // a worktree folder that exists
	gone := filepath.Join(t.TempDir(), "removed-by-hand")
	write := func(path, text string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(taskWorktreeRecords(cfgDir), "task-a"), gone+"\ttask-a\n"+here+"\t\n")
	write(filepath.Join(taskWorktreeRecords(cfgDir), "task-b"), gone+"\ttask-b\n")
	write(filepath.Join(worktreeRecords(cfgDir), "task-c"), gone+"\n")
	write(filepath.Join(worktreeRecords(cfgDir), "task-d"), here+"\n")
	locks := filepath.Join(cfgDir, "state", "session-locks")
	old := time.Now().Add(-48 * time.Hour)
	for _, name := range []string{"task-old.lock", "task-recent.lock", "task-held.lock", "task-live.lock"} {
		write(filepath.Join(locks, name), "")
		if name != "task-recent.lock" {
			if err := os.Chtimes(filepath.Join(locks, name), old, old); err != nil {
				t.Fatal(err)
			}
		}
	}
	launchOnRecord(t, cfgDir, "task-live")
	release, ok := HoldStop(cfgDir, "task-x") // a stop's lock, held
	if !ok {
		t.Fatal("hold")
	}
	defer release()
	unlock, err := filelock.LockWithin(filepath.Join(locks, "task-held.lock"), 0)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	// Taking the lock wrote nothing, but put its time back all the same.
	if err := os.Chtimes(filepath.Join(locks, "task-held.lock"), old, old); err != nil {
		t.Fatal(err)
	}

	if n := TidyRecords(cfgDir, time.Now()); n != 4 {
		t.Errorf("took off %d records, want 4 (two noted lines, a session worktree, an old lock)", n)
	}
	if got := readNoted(cfgDir, "task-a"); len(got) != 1 || got[0].folder != here {
		t.Errorf("task-a keeps %v, want only the folder that exists", got)
	}
	if _, err := os.Stat(filepath.Join(taskWorktreeRecords(cfgDir), "task-b")); !os.IsNotExist(err) {
		t.Error("task-b's record, now empty, stays")
	}
	if _, err := os.Stat(filepath.Join(worktreeRecords(cfgDir), "task-c")); !os.IsNotExist(err) {
		t.Error("a session worktree record of a folder gone stays")
	}
	if _, err := os.Stat(filepath.Join(worktreeRecords(cfgDir), "task-d")); err != nil {
		t.Error("a session worktree record of a folder that exists went")
	}
	left, _ := os.ReadDir(locks)
	var names []string
	for _, e := range left {
		names = append(names, e.Name())
	}
	if got := strings.Join(names, " "); got != "task-held.lock task-live.lock task-recent.lock" {
		t.Errorf("locks left: %s", got)
	}
	if _, err := os.Stat(here); err != nil {
		t.Error("a folder was touched")
	}
}

// Closing a task takes off the record of a noted worktree someone removed by hand: it
// stayed for good before (22-nov: the release worktree of a task closed that day).
func TestRemoveNotedWorktreesForgetsVanished(t *testing.T) {
	cfgDir := t.TempDir()
	gone := filepath.Join(t.TempDir(), "removed-by-hand")
	if err := addNoted(cfgDir, "task-r", noted{gone, "task-r"}); err != nil {
		t.Fatal(err)
	}
	removed, kept := RemoveNotedWorktrees(cfgDir, "task-r")
	if len(removed) != 0 || len(kept) != 0 {
		t.Errorf("removed %v, kept %v: a folder gone is neither", removed, kept)
	}
	if got := readNoted(cfgDir, "task-r"); len(got) != 0 {
		t.Errorf("still noted: %v", got)
	}
}
