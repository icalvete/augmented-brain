package remote

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/icalvete/augmented-brain/internal/filelock"
	"github.com/icalvete/augmented-brain/internal/paths"
	"github.com/icalvete/augmented-brain/internal/state"
)

// Pending stops. Parking, closing or validating a task closes its session through a
// detached `session stop --when-idle`, which waits up to 30 minutes for the session to
// end its answer. When the machine goes down in that time the waiting process dies and
// nothing tried again: on 22-nov a parked task's session stayed on record, and in
// Claude's lists, until someone ran clean.
//
// So the stop is written down before the waiting process starts (MarkStop) and cleared
// when it has done its work (ClearStop). The waiting process holds the stop's lock
// while it waits (HoldStop): a second stop of the same task finds it held and leaves
// the first one to it. A stop on record whose lock nobody holds was cut short, and the
// next session start starts it again (ResumeStops).

func stopsDir(cfgDir string) string { return filepath.Join(cfgDir, state.Dir, "session-stops") }

func stopFile(cfgDir, task, ext string) string {
	return filepath.Join(stopsDir(cfgDir), unitSafe.ReplaceAllString(task, "-")+ext)
}

// MarkStop writes down that task's session is to be closed; keep says its worktrees
// stay (a parked task).
func MarkStop(cfgDir, task string, keep bool) error {
	text := task + "\n"
	if keep {
		text += "keep\n"
	}
	if err := os.MkdirAll(stopsDir(cfgDir), 0o700); err != nil {
		return err
	}
	return paths.WriteAtomic(stopFile(cfgDir, task, ".pending"), []byte(text), 0o600)
}

// ClearStop takes the stop of task off the record: it was done, or found nothing to do.
func ClearStop(cfgDir, task string) {
	_ = os.Remove(stopFile(cfgDir, task, ".pending"))
}

// HoldStop takes the lock of task's stop for the process that waits; ok is false when
// another process holds it already, waiting for the same session.
func HoldStop(cfgDir, task string) (release func(), ok bool) {
	release, err := filelock.LockWithin(stopFile(cfgDir, task, ".lock"), 0)
	if err != nil {
		return func() {}, false
	}
	return release, true
}

// PendingStop is a stop on record that no process is waiting on.
type PendingStop struct {
	Task string
	Keep bool
}

// PendingStops are the stops on record whose waiting process is gone.
func PendingStops(cfgDir string) []PendingStop {
	entries, err := os.ReadDir(stopsDir(cfgDir))
	if err != nil {
		return nil
	}
	var out []PendingStop
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".pending") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(stopsDir(cfgDir), e.Name()))
		if err != nil {
			continue
		}
		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		task := strings.TrimSpace(lines[0])
		if task == "" {
			continue
		}
		release, ok := HoldStop(cfgDir, task)
		if !ok {
			continue // its process is still waiting
		}
		release()
		out = append(out, PendingStop{Task: task, Keep: len(lines) > 1 && strings.TrimSpace(lines[1]) == "keep"})
	}
	return out
}

// stopStatus are the statuses whose task's session is closed: closing and parking close
// it; validation keeps it, for the proof with the user.
func stopStatus(status string) bool {
	switch status {
	case "parked", "someday", "closed":
		return true
	}
	return false
}

// ResumeStops starts again (through start) each stop cut short whose task still wants
// its session closed, and takes off the record those that no longer apply: no session
// on record any more, a task that no space has, or one taken up again since. It returns
// the tasks whose stop it started.
func ResumeStops(cfgDir string, live TaskLive, start func(cfgDir, task string, keep bool) error) []string {
	var out []string
	for _, p := range PendingStops(cfgDir) {
		status, found := live(p.Task)
		if !HasLaunch(cfgDir, p.Task) || !found || !stopStatus(status) {
			ClearStop(cfgDir, p.Task)
			continue
		}
		if start(cfgDir, p.Task, p.Keep) == nil {
			out = append(out, p.Task)
		}
	}
	return out
}

// StartStop starts the detached stop of a task, keeping its worktrees or not.
func StartStop(cfgDir, task string, keep bool) error {
	return stopDetached(cfgDir, task, keep)
}
