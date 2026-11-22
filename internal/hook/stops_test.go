package hook

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/icalvete/augmented-brain/internal/brain"
	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/remote"
	"github.com/icalvete/augmented-brain/internal/state"
)

// A session start of any space starts again the stop of a task session that a reboot
// cut short (22-nov: the parked task's session stayed until clean was run by hand).
func TestSessionStartResumesStops(t *testing.T) {
	cfgDir, work := setupSpace(t)
	c, err := config.Load(cfgDir)
	if err != nil {
		t.Fatal(err)
	}
	b, err := brain.Open(c, c.Spaces["work"])
	if err != nil {
		t.Fatal(err)
	}
	n, err := b.NewTask(brain.NewTaskInput{Title: "Ghost session"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.Park(brain.ParkInput{ID: n.Header.ID, Next: "later"}); err != nil {
		t.Fatal(err)
	}
	record := filepath.Join(cfgDir, state.Dir, "sessions", n.Header.ID+".json")
	if err := os.MkdirAll(filepath.Dir(record), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(record, []byte(`{"id":"9b1817a2","task":"`+n.Header.ID+`"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := remote.MarkStop(cfgDir, n.Header.ID, true); err != nil {
		t.Fatal(err)
	}
	var started []string
	old := StartStop
	StartStop = func(_, task string, keep bool) error {
		started = append(started, task)
		if !keep {
			t.Error("a parked task's stop must keep its worktrees")
		}
		return nil
	}
	t.Cleanup(func() { StartStop = old })
	run(t, func(in *bytes.Reader, out *bytes.Buffer) error { return SessionStartPart(in, out, cfgDir, 1) },
		map[string]any{"cwd": work, "session_id": "s9"})
	if len(started) != 1 || started[0] != n.Header.ID {
		t.Errorf("stops started: %v", started)
	}
}
