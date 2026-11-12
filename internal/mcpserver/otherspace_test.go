package mcpserver

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/icalvete/augmented-brain/internal/brain"
	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/setup"
	"github.com/icalvete/augmented-brain/internal/state"
)

// From a session of personal, the task tools with space read and write a task of
// work: what is written says where it came from, nothing lands in personal, work is
// marked for its backup, and closing is left to work's own sessions.
func TestTasksOfAnotherSpaceThroughTools(t *testing.T) {
	openWork := space(t)
	home := os.Getenv("HOME")
	cfgDir := filepath.Join(home, "cfg")
	other := filepath.Join(home, "me")
	if err := os.MkdirAll(other, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := setup.AddSpace(cfgDir, setup.Options{Space: "personal", WorkFolder: other, Memory: "~/brain-personal"}); err != nil {
		t.Fatal(err)
	}
	openPersonal := func() (*brain.Brain, error) {
		c, err := config.Load(cfgDir)
		if err != nil {
			return nil, err
		}
		b, err := brain.Open(c, c.Spaces["personal"])
		if b != nil {
			b.Now = func() time.Time { return time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC) }
		}
		return b, err
	}
	p, w := connect(t, openPersonal), connect(t, openWork)
	mustCall(t, w, "task_new", map[string]any{"title": "Reload the history"})
	const id = "task-reload-the-history"

	if out, isErr := call(t, p, "task_journal", map[string]any{"id": id, "entry": "x"}); !isErr || !strings.Contains(out, "no task") {
		t.Fatalf("without space it must look in personal only: %v %s", isErr, out)
	}
	if out, isErr := call(t, p, "task_get", map[string]any{"id": id, "space": "nowhere"}); !isErr || !strings.Contains(out, "personal, work") {
		t.Fatalf("unknown space: %v %s", isErr, out)
	}
	if out := mustCall(t, p, "pending", map[string]any{"space": "work"}); !strings.Contains(out, "Space work") || !strings.Contains(out, id) {
		t.Fatalf("pending of work:\n%s", out)
	}
	out := mustCall(t, p, "task_journal", map[string]any{"id": id, "space": "work", "entry": "The user runs the reload himself, with care"})
	if !strings.Contains(out, "space work, not in personal") {
		t.Fatalf("task_journal: %s", out)
	}
	mustCall(t, p, "task_update", map[string]any{"id": id, "space": "work", "next": "Run the reload", "status": "analysis"})
	mustCall(t, p, "task_step", map[string]any{"id": id, "space": "work", "add": "Check the counts"})
	if out, isErr := call(t, p, "task_update", map[string]any{"id": id, "space": "work", "status": "closed"}); !isErr || !strings.Contains(out, "closed from a session of that space") {
		t.Fatalf("closing from another space: %v %s", isErr, out)
	}

	got := mustCall(t, p, "task_get", map[string]any{"id": id, "space": "work"})
	for _, want := range []string{
		"The user runs the reload himself, with care (from personal)",
		"next: Run the reload",
		"- [ ] Check the counts",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("task_get from personal lacks %q:\n%s", want, got)
		}
	}
	if own := mustCall(t, w, "task_get", map[string]any{"id": id}); !strings.Contains(own, "with care (from personal)") || strings.Contains(own, "not in") {
		t.Errorf("work's own session reads it plainly:\n%s", own)
	}
	if out := mustCall(t, p, "pending", nil); strings.Contains(out, id) {
		t.Errorf("the task showed up in personal:\n%s", out)
	}
	if !state.TakeOutside(cfgDir, "work", time.Now()) {
		t.Error("work was not marked for its backup")
	}
	if state.TakeOutside(cfgDir, "personal", time.Now()) {
		t.Error("personal was marked, though nothing was written there from outside")
	}
}
