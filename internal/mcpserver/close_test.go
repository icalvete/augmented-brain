package mcpserver

import (
	"strings"
	"testing"
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
