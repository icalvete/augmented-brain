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
	self, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(self, "session", "stop", task, "--when-idle")
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
		out += "\n" + i18n.T("en", "session.worktree.closed.kept", paths.Tilde(folder), i18n.T("en", "session.worktree.why."+why))
	}
	gone, kept := RemoveNotedWorktrees(cfgDir, task, skip...)
	for _, f := range gone {
		out += "\n" + i18n.T("en", "session.worktree.closed", paths.Tilde(f))
	}
	for _, k := range kept {
		out += "\n" + i18n.T("en", "session.worktree.closed.kept", paths.Tilde(k.Folder), i18n.T("en", "session.worktree.why."+k.Why))
	}
	return out
}
