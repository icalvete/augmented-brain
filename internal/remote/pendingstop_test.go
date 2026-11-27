package remote

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func launchOnRecord(t *testing.T, cfgDir, task string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(launchRecordPath(cfgDir, task)), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(launchRecordPath(cfgDir, task), []byte(`{"id":"9b1817a2","task":"`+task+`"}`), 0o600); err != nil {
		t.Fatal(err)
	}
}

// A stop cut short by the machine going down (its record there, its lock free) is
// started again at the next session start; one still waiting is left alone; one that no
// longer applies leaves the record. Measured on 22-nov: a parked task's session stayed
// on record after a reboot cut its stop, until someone ran clean.
func TestResumeStops(t *testing.T) {
	cfgDir := t.TempDir()
	tasks := map[string]string{
		"task-parked":    "parked",
		"task-closed":    "closed",
		"task-validated": "validation", // in validation: its session stays
		"task-resumed":   "execution",  // taken up again since: its session must stay
		"task-waiting":   "parked",     // its stop is still waiting
		"task-gone":      "",           // no space has it
		"task-norecord":  "parked",     // its session is gone already
	}
	for task := range tasks {
		if task != "task-norecord" {
			launchOnRecord(t, cfgDir, task)
		}
		if err := MarkStop(cfgDir, task, task == "task-parked"); err != nil {
			t.Fatal(err)
		}
	}
	release, ok := HoldStop(cfgDir, "task-waiting")
	if !ok {
		t.Fatal("the first hold was refused")
	}
	if _, again := HoldStop(cfgDir, "task-waiting"); again {
		t.Error("a second process got the stop's lock: two would wait for the same session")
	}
	live := func(task string) (string, bool) {
		s := tasks[task]
		return s, s != ""
	}
	started := map[string]bool{}
	got := ResumeStops(cfgDir, live, func(_, task string, keep bool) error {
		started[task] = keep
		return nil
	})
	slices.Sort(got)
	if want := []string{"task-closed", "task-parked"}; !slices.Equal(got, want) {
		t.Errorf("started %v, want %v", got, want)
	}
	if !started["task-parked"] || started["task-closed"] {
		t.Errorf("worktrees: a parked task keeps them, a closed one does not: %v", started)
	}
	for _, task := range []string{"task-resumed", "task-validated", "task-gone", "task-norecord"} {
		if _, err := os.Stat(stopFile(cfgDir, task, ".pending")); !os.IsNotExist(err) {
			t.Errorf("%s: a stop that no longer applies is still on record", task)
		}
	}
	if _, err := os.Stat(stopFile(cfgDir, "task-waiting", ".pending")); err != nil {
		t.Error("the stop still waiting lost its record")
	}
	release()
	if p := PendingStops(cfgDir); !slices.ContainsFunc(p, func(s PendingStop) bool { return s.Task == "task-waiting" }) {
		t.Errorf("once its process is gone the stop is pending: %v", p)
	}
	ClearStop(cfgDir, "task-waiting")
	if p := PendingStops(cfgDir); slices.ContainsFunc(p, func(s PendingStop) bool { return s.Task == "task-waiting" }) {
		t.Error("a cleared stop is still pending")
	}
}
