package remote

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/icalvete/augmented-brain/internal/detach"
	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/paths"
	"github.com/icalvete/augmented-brain/internal/state"
)

// StopLog is where the detached `session stop --when-idle` writes how it went, in the
// configuration's state folder: nobody reads its output otherwise.
const StopLog = "session-stop.log"

// StopDetached is set in the environment of the detached `session stop --when-idle`,
// which does the waiting: without it the command starts that copy and returns, so a
// session can close itself from its own Bash (issue #99: waiting there, it kept itself
// busy, and the two waited for each other).
const StopDetached = "AB_STOP_DETACHED"

// StopWhenIdle starts, detached, the process that closes a task's session once it has
// finished its answer (`session stop <task> --when-idle`), its output appended to
// StopLog (started afresh past 64 KB).
func StopWhenIdle(cfgDir, task string) error {
	return stopDetached(cfgDir, task, false)
}

// StopWhenIdleKeep is StopWhenIdle for a task that was parked: the detached process
// leaves the task's worktrees (`session stop <task> --when-idle --keep-worktrees`).
func StopWhenIdleKeep(cfgDir, task string) error {
	return stopDetached(cfgDir, task, true)
}

func stopDetached(cfgDir, task string, keepWorktrees bool) error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	// On record first: if the machine goes down while the copy waits, the next session
	// start finds it and starts it again (ResumeStops).
	if err := MarkStop(cfgDir, task, keepWorktrees); err != nil {
		return err
	}
	args := []string{"session", "stop", task, "--when-idle"}
	if keepWorktrees {
		args = append(args, "--keep-worktrees")
	}
	cmd := exec.Command(self, args...)
	cmd.Env = append(os.Environ(), "AB_CONFIG_DIR="+cfgDir, StopDetached+"=1")
	cmd.Stdin, cmd.Stdout, cmd.Stderr = nil, nil, nil
	logPath := filepath.Join(cfgDir, state.Dir, StopLog)
	if fi, err := os.Stat(logPath); err == nil && fi.Size() > 64<<10 {
		_ = os.Remove(logPath)
	}
	if err := os.MkdirAll(filepath.Dir(logPath), 0o700); err == nil {
		if f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600); err == nil {
			defer f.Close() // the child keeps its own copy
			cmd.Stdout, cmd.Stderr = f, f
		}
	}
	detach.Cmd(cmd)
	_, err = detach.Start(cmd)
	return err
}

// ClosedTask does what closing a task does beyond its note, whichever way it was closed
// (task_close, task_update with status closed, the panel's Close): it ends the task's
// own session, at once when Claude retired it, through stop (StopWhenIdle) once it has
// finished its answer when it runs, and removes the worktrees made or noted for the
// task that hold nothing of their own, but those in skip (the folder the closing
// session works in). It says what it did in English, like every tool's answer: the
// assistant tells the user in their language.
func (s System) ClosedTask(cfgDir, task string, stop func(cfgDir, task string) error, skip ...string) string {
	// The task has a session of its own: often the one closing it, in the middle of its
	// answer. A process of its own closes it once it is idle, worktrees and all.
	if s.LaunchRunning(cfgDir, task) {
		if err := stop(cfgDir, task); err == nil {
			return "\n" + i18n.T("en", "session.closing")
		}
		return ""
	}
	var out string
	// One Claude retired after hours unused answers nothing: closed now, so it leaves
	// Claude Desktop and claude.ai and its worktree can go below.
	if HasLaunch(cfgDir, task) {
		var left *LeftInClaude
		switch err := s.Stop(cfgDir, task); {
		case err == nil:
			out += "\n" + i18n.T("en", "session.closed.retired")
		case errors.As(err, &left):
			out += "\n" + i18n.T("en", "session.left", task, left.ID)
		}
	}
	folder, removed, why := s.CloseTaskWorktree(cfgDir, task)
	switch {
	case removed:
		out += "\n" + i18n.T("en", "session.worktree.closed", paths.Tilde(folder))
	case why != "":
		out += "\n" + i18n.T("en", "session.worktree.closed.kept", paths.Tilde(folder), WhyKept("en", folder, why))
	}
	gone, kept := RemoveNotedWorktrees(cfgDir, task, skip...)
	for _, f := range gone {
		out += "\n" + i18n.T("en", "session.worktree.closed", paths.Tilde(f))
	}
	for _, k := range kept {
		out += "\n" + i18n.T("en", "session.worktree.closed.kept", paths.Tilde(k.Folder), WhyKept("en", k.Folder, k.Why))
	}
	return out
}

// ParkedTask does what parking a task does beyond its note (task_park, task_update with
// status parked or someday, the panel's Park): it ends the task's own session, so
// Claude's daemon does not bring it back on its next start (it does, for a session it
// only retired: measured on 21-nov, 2.1.330), and leaves its worktrees, since the task
// goes on later. A running session is closed once it has finished its answer, through
// stop (StopWhenIdleKeep); one Claude retired, at once. It says what it did in English,
// like every tool's answer.
func (s System) ParkedTask(cfgDir, task string, stop func(cfgDir, task string) error) string {
	if s.LaunchRunning(cfgDir, task) {
		if err := stop(cfgDir, task); err == nil {
			return "\n" + i18n.T("en", "session.parking")
		}
		return ""
	}
	if !HasLaunch(cfgDir, task) {
		return ""
	}
	var left *LeftInClaude
	switch err := s.Stop(cfgDir, task); {
	case err == nil:
		return "\n" + i18n.T("en", "session.parked.retired")
	case errors.As(err, &left):
		return "\n" + i18n.T("en", "session.left", task, left.ID)
	}
	return ""
}
