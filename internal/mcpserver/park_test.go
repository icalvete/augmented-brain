package mcpserver

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/remote"
)

// Parking a task, or sending it to someday, ends its own session, so Claude's daemon
// does not bring it back on its next start (21-nov), and leaves its worktrees: a
// running one through the detached stop that keeps them, one Claude retired at once.
func TestParkStopsTheTaskSession(t *testing.T) {
	cs := connect(t, space(t))
	cfgDir := filepath.Join(os.Getenv("HOME"), "cfg")
	if err := os.MkdirAll(filepath.Join(cfgDir, "state", "sessions"), 0o700); err != nil {
		t.Fatal(err)
	}
	record := func(task, id string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(cfgDir, "state", "sessions", task+".json"), []byte(`{"id":"`+id+`","task":"`+task+`"}`), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	oldSys, oldStop, oldKeep := NewRemoteSystem, StopSessionWhenIdle, StopSessionWhenIdleKeep
	t.Cleanup(func() { NewRemoteSystem, StopSessionWhenIdle, StopSessionWhenIdleKeep = oldSys, oldStop, oldKeep })
	var calls []string
	agents := ""
	NewRemoteSystem = func() (remote.System, error) {
		return remote.System{GOOS: "linux",
			LookPath: func(string) (string, error) { return "/usr/bin/claude", nil },
			Run: func(name string, args ...string) (string, error) {
				calls = append(calls, strings.Join(args, " "))
				if len(args) > 0 && args[0] == "agents" {
					return agents, nil
				}
				return "", nil
			}}, nil
	}
	var kept, closed []string
	StopSessionWhenIdle = func(dir, task string) error { closed = append(closed, task); return nil }
	StopSessionWhenIdleKeep = func(dir, task string) error { kept = append(kept, dir+" "+task); return nil }

	mustCall(t, cs, "task_new", map[string]any{"title": "Site"})
	record("task-site", "abc12345")
	agents = `[{"pid":4242,"id":"abc12345","status":"busy","state":"working"}]`
	out := mustCall(t, cs, "task_park", map[string]any{"id": "task-site", "next": "go on"})
	if len(kept) != 1 || kept[0] != cfgDir+" task-site" || len(closed) != 0 || !strings.Contains(out, "its worktrees stay") {
		t.Fatalf("kept %v closed %v:\n%s", kept, closed, out)
	}

	// task_update to someday does the same.
	mustCall(t, cs, "task_new", map[string]any{"title": "Blog"})
	record("task-blog", "bcd23456")
	agents = `[{"pid":4243,"id":"bcd23456","status":"idle","state":"working"}]`
	mustCall(t, cs, "task_update", map[string]any{"id": "task-blog", "status": "someday"})
	if len(kept) != 2 || kept[1] != cfgDir+" task-blog" {
		t.Fatalf("kept %v", kept)
	}

	// task_update to validation too: it waits on the user, often longer than the 8
	// hours after which Claude retires a session and leaves it disconnected in claude.ai.
	mustCall(t, cs, "task_new", map[string]any{"title": "Shop"})
	record("task-shop", "cde34567")
	agents = `[{"pid":4245,"id":"cde34567","status":"busy","state":"working"}]`
	out = mustCall(t, cs, "task_update", map[string]any{"id": "task-shop", "status": "validation"})
	if len(kept) != 3 || kept[2] != cfgDir+" task-shop" || !strings.Contains(out, "its worktrees stay") {
		t.Fatalf("kept %v:\n%s", kept, out)
	}
	kept = kept[:2]

	// One Claude retired: closed at once, with claude rm, so the daemon cannot bring it back.
	mustCall(t, cs, "task_new", map[string]any{"title": "Docs"})
	record("task-docs", "def67890")
	agents = `[{"id":"def67890","state":"done"}]`
	calls = nil
	out = mustCall(t, cs, "task_park", map[string]any{"id": "task-docs", "next": "go on"})
	if len(kept) != 2 || !slices.Contains(calls, "rm def67890") || !strings.Contains(out, "Claude had stopped") {
		t.Fatalf("kept %v calls %v:\n%s", kept, calls, out)
	}
	if _, err := os.Stat(filepath.Join(cfgDir, "state", "sessions", "task-docs.json")); !os.IsNotExist(err) {
		t.Errorf("the record of the retired session stays: %v", err)
	}

	// Another status change, or a task with no session of its own, starts nothing.
	mustCall(t, cs, "task_new", map[string]any{"title": "Plain"})
	mustCall(t, cs, "task_park", map[string]any{"id": "task-plain", "next": "go on"})
	record("task-live", "efa78901")
	mustCall(t, cs, "task_new", map[string]any{"title": "Live"})
	agents = `[{"pid":4244,"id":"efa78901","status":"busy","state":"working"}]`
	mustCall(t, cs, "task_update", map[string]any{"id": "task-live", "status": "execution"})
	if len(kept) != 2 || len(closed) != 0 {
		t.Errorf("kept %v closed %v", kept, closed)
	}
}

// The weekly review names the tasks' own sessions whose task no longer goes on, and
// sends to augmented-brain clean; with none on record it asks Claude nothing.
func TestWeeklyReviewNamesSpareSessions(t *testing.T) {
	cs := connect(t, space(t))
	cfgDir := filepath.Join(os.Getenv("HOME"), "cfg")
	oldSys := NewRemoteSystem
	t.Cleanup(func() { NewRemoteSystem = oldSys })
	asked := 0
	NewRemoteSystem = func() (remote.System, error) {
		return remote.System{GOOS: "linux",
			LookPath: func(string) (string, error) { return "/usr/bin/claude", nil },
			Run: func(name string, args ...string) (string, error) {
				asked++
				return `[{"id":"abc12345","kind":"background","state":"done"},{"id":"bcd23456","kind":"background","pid":5,"status":"idle","state":"working"}]`, nil
			}}, nil
	}
	if out := mustCall(t, cs, "weekly_review", nil); strings.Contains(out, "no longer going on") || asked != 0 {
		t.Fatalf("asked %d:\n%s", asked, out)
	}
	mustCall(t, cs, "task_new", map[string]any{"title": "Going"})
	if err := os.MkdirAll(filepath.Join(cfgDir, "state", "sessions"), 0o700); err != nil {
		t.Fatal(err)
	}
	for task, id := range map[string]string{"task-gone": "abc12345", "task-going": "bcd23456"} {
		if err := os.WriteFile(filepath.Join(cfgDir, "state", "sessions", task+".json"), []byte(`{"id":"`+id+`","task":"`+task+`"}`), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	out := mustCall(t, cs, "weekly_review", nil)
	if !strings.Contains(out, "Sessions of tasks no longer going on (1): task-gone.") || !strings.Contains(out, "augmented-brain clean") {
		t.Errorf("weekly_review:\n%s", out)
	}
}
