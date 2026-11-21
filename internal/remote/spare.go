package remote

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Spare sessions are background sessions left over in Claude's list, for `augmented-brain
// clean` to remove (decided on 21-nov). Claude retires a background session left unused
// for hours but keeps it in its list, and its daemon brings every retired one back on
// its next start (measured on 21-nov, 2.1.330: eight came back within 15 seconds, all of
// them of tasks parked or put off). Only `claude stop` plus `claude rm` takes one out
// for good, out of Claude Desktop and claude.ai too.
//
// Two kinds:
//   - a task's own session (`session start`) whose task is parked, put off, closed or
//     gone: removed as `session stop` does, its worktrees left;
//   - a background session no task opened that has ended or that the user archived
//     (done or stopped, Ended): someone else started it, so clean asks for each one,
//     never all at once.
//
// A session answering now (busy, or waiting for a permission) is never one: it may be
// the very session running clean.

// Spare is one session left over.
type Spare struct {
	ID     string // Claude's id of the background session
	Name   string // its name in Claude Desktop and claude.ai
	Folder string
	State  string // done, stopped, working, blocked; "" when Claude no longer lists it
	// Task is the task whose own session it is, with TaskStatus its status now ("" for
	// a task no space has). Empty for a session no task opened.
	Task       string
	TaskStatus string
}

// OfTask says it is a task's own session.
func (sp Spare) OfTask() bool { return sp.Task != "" }

// InWorktree says it works in a worktree Claude Code made (.claude/worktrees), which
// `claude rm` removes too when nothing would be lost, and keeps otherwise.
func (sp Spare) InWorktree() bool {
	return strings.Contains(filepath.ToSlash(sp.Folder), "/.claude/worktrees/")
}

// TaskLive says whether a task still needs its session: status is its status now, and
// found false means no space has it.
type TaskLive func(task string) (status string, found bool)

// liveStatus are the statuses of a task whose session is not left over.
func liveStatus(status string) bool {
	switch status {
	case "parked", "someday", "closed":
		return false
	}
	return true
}

// SpareSessions are the sessions left over (Spare): first the tasks' own, then the
// others, each in Claude's order.
func (s System) SpareSessions(cfgDir string, live TaskLive) ([]Spare, error) {
	if s.GOOS != "linux" {
		return nil, ErrUnsupported
	}
	claude, err := s.FindClaude()
	if err != nil {
		return nil, errors.New("claude is not installed or not in the PATH")
	}
	list, err := s.agents(claude)
	if err != nil {
		return nil, err
	}
	byID := map[string]claudeAgent{}
	for _, a := range list {
		if a.ID != "" {
			byID[a.ID] = a
		}
	}
	answering := func(a claudeAgent) bool { return a.Pid > 0 && a.Status != "" && a.Status != "idle" }
	var tasks, others []Spare
	launched := map[string]bool{}
	entries, _ := os.ReadDir(launchRecords(cfgDir))
	for _, e := range entries {
		task, ok := strings.CutSuffix(e.Name(), ".json")
		if !ok {
			continue
		}
		r, ok := readLaunch(cfgDir, task)
		if !ok {
			continue
		}
		launched[r.ID] = true
		status, found := live(task)
		if found && liveStatus(status) {
			continue
		}
		a, listed := byID[r.ID]
		if listed && answering(a) {
			continue
		}
		sp := Spare{ID: r.ID, Name: r.Name, Folder: r.Folder, Task: task, TaskStatus: status}
		if listed {
			sp.State = a.State
		}
		tasks = append(tasks, sp)
	}
	for _, a := range list {
		if a.ID == "" || launched[a.ID] || a.Kind != "background" || (a.State != "done" && a.State != "stopped") || answering(a) {
			continue
		}
		others = append(others, Spare{ID: a.ID, Name: a.Name, Folder: a.Cwd, State: a.State})
	}
	return append(tasks, others...), nil
}

// RemoveSpare takes a spare session out of Claude's list, and so out of Claude Desktop
// and claude.ai: a task's as `session stop --keep-worktrees` does, any other with
// `claude stop` and `claude rm`. When rm refuses (a worktree with work of its own, a
// session that runs again), its words come back in the error.
func (s System) RemoveSpare(cfgDir string, sp Spare) error {
	if sp.OfTask() {
		return s.Stop(cfgDir, sp.Task)
	}
	if !sessionID.MatchString(sp.ID) {
		return fmt.Errorf("%q is not a session id", sp.ID)
	}
	claude, err := s.FindClaude()
	if err != nil {
		return errors.New("claude is not installed or not in the PATH")
	}
	_, _ = s.Run(claude, "stop", sp.ID) // fails when it had ended already
	if _, err := s.Run(claude, "rm", sp.ID); err != nil {
		s.wakeDaemon(claude)
		if out, err := s.Run(claude, "rm", sp.ID); err != nil {
			if msg := strings.TrimSpace(cleanTerminal(out)); msg != "" {
				return fmt.Errorf("claude rm %s: %s", sp.ID, msg)
			}
			return fmt.Errorf("claude rm %s: %w", sp.ID, err)
		}
	}
	return nil
}
