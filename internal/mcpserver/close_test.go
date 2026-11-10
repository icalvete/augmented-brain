package mcpserver

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/remote"
)

// Closing a task with an open watch date says so: it keeps coming up at the start of
// the session (the case of 7-nov: a check for the 8th in a task closed on the 7th).
func TestCloseTellsOpenWatches(t *testing.T) {
	cs := connect(t, space(t))
	mustCall(t, cs, "task_new", map[string]any{"title": "Rollback"})
	mustCall(t, cs, "task_update", map[string]any{"id": "task-rollback", "watch_date": "2026-11-08", "watch_what": "descriptions still in Jenkins"})
	mustCall(t, cs, "task_update", map[string]any{"id": "task-rollback", "watch_date": "2026-11-01", "watch_what": "already checked"})
	mustCall(t, cs, "task_update", map[string]any{"id": "task-rollback", "watch_done": "2026-11-01"})
	out := mustCall(t, cs, "task_close", map[string]any{"id": "task-rollback", "result": "done"})
	if !strings.Contains(out, "1 open watch date") || !strings.Contains(out, "2026-11-08: descriptions still in Jenkins") || strings.Contains(out, "already checked") {
		t.Fatalf("task_close:\n%s", out)
	}
	// it still comes up: pending lists the watch of the closed task
	if p := mustCall(t, cs, "pending", map[string]any{"within_days": 400}); !strings.Contains(p, "descriptions still in Jenkins") {
		t.Fatalf("pending:\n%s", p)
	}
	mustCall(t, cs, "task_new", map[string]any{"title": "Plain"})
	if out := mustCall(t, cs, "task_close", map[string]any{"id": "task-plain", "result": "done"}); strings.Contains(out, "watch") {
		t.Fatalf("no watches, yet:\n%s", out)
	}
}

// Closing a task whose own session runs has a process of its own close that session
// once it has finished its answer (often the closing one): never at once.
func TestCloseStopsTheTaskSession(t *testing.T) {
	cs := connect(t, space(t))
	cfgDir := filepath.Join(os.Getenv("HOME"), "cfg")
	mustCall(t, cs, "task_new", map[string]any{"title": "Site"})
	if err := os.MkdirAll(filepath.Join(cfgDir, "state", "sessions"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "state", "sessions", "task-site.json"), []byte(`{"id":"abc12345","task":"task-site"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	oldSys, oldStop := NewRemoteSystem, StopSessionWhenIdle
	t.Cleanup(func() { NewRemoteSystem, StopSessionWhenIdle = oldSys, oldStop })
	NewRemoteSystem = func() (remote.System, error) {
		return remote.System{GOOS: "linux",
			LookPath: func(string) (string, error) { return "/usr/bin/claude", nil },
			Run: func(name string, args ...string) (string, error) {
				if len(args) > 0 && args[0] == "agents" {
					return `[{"id":"abc12345","status":"busy"}]`, nil
				}
				return "", nil
			}}, nil
	}
	var stopped []string
	StopSessionWhenIdle = func(dir, task string) error { stopped = append(stopped, dir+" "+task); return nil }
	out := mustCall(t, cs, "task_close", map[string]any{"id": "task-site", "result": "done"})
	if len(stopped) != 1 || stopped[0] != cfgDir+" task-site" || !strings.Contains(out, "closes when it finishes this answer") {
		t.Fatalf("stopped %v:\n%s", stopped, out)
	}
	// A task with no session of its own starts nothing.
	mustCall(t, cs, "task_new", map[string]any{"title": "Plain"})
	mustCall(t, cs, "task_close", map[string]any{"id": "task-plain", "result": "done"})
	if len(stopped) != 1 {
		t.Errorf("stopped %v", stopped)
	}
}
