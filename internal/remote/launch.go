package remote

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/paths"
)

// A launched session is a `claude remote-control` of its own for one task, started by
// `augmented-brain session start` as a transient systemd user unit, so a session the
// user reaches from another computer (the space's service, which cannot open sessions
// itself) can open one per task: it shows up in Claude Desktop and claude.ai with the
// task's title, in the task's folder, and starts by reading its task (TaskEnv, read by
// the SessionStart hook). It ends when its session ends (--spawn session), or with
// `session stop`. A reboot ends it too: launched sessions are not brought back.
//
// Two limits of claude remote-control shape it (measured on 2.1.319, 8-nov-2026):
// the folder must itself be trusted (a trusted parent folder does not count: the server
// exits with "Workspace not trusted"), and one server per folder ("This folder is
// already served"), so the space's own service folder cannot take a task.

// TaskEnv is the variable a launched session carries: the id of its task.
const TaskEnv = "AUGMENTED_BRAIN_TASK"

const launchPrefix = "augmented-brain-session-"

// Errors of Start the command explains to the user.
var (
	// ErrUntrusted: the folder has not accepted Claude's trust question itself.
	ErrUntrusted = errors.New("claude does not trust this folder")
	// ErrServed: another remote-control server already serves the folder.
	ErrServed = errors.New("this folder is already served")
	// ErrRunning: the task has a launched session already.
	ErrRunning = errors.New("this task has a session already")
)

// Launch is one task's session.
type Launch struct {
	Task   string // the task's id
	Name   string // the session's name in Claude Desktop and claude.ai
	Folder string
}

// Unit is the transient unit's name for a task.
func LaunchUnit(task string) string {
	return launchPrefix + unitSafe.ReplaceAllString(strings.TrimPrefix(task, "task-"), "-") + ".service"
}

var unitSafe = regexp.MustCompile(`[^A-Za-z0-9_.-]+`)

// LaunchStatus is a launched session as systemd sees it.
type LaunchStatus struct {
	Unit   string
	Task   string
	Folder string
	Active string
}

// FolderTrusted says whether Claude's server would start in folder: the folder itself
// has accepted the trust question (its own entry in .claude.json), as measured.
func (s System) FolderTrusted(folder string) bool {
	var f struct {
		Projects map[string]struct {
			Accepted bool `json:"hasTrustDialogAccepted"`
		} `json:"projects"`
	}
	data, err := os.ReadFile(s.ClaudeJSON)
	if err != nil || json.Unmarshal(data, &f) != nil {
		return false
	}
	cands := []string{filepath.Clean(folder)}
	if real, err := paths.Resolve(folder); err == nil {
		cands = append(cands, real)
	}
	// Claude Code keeps the answer for a git worktree under its main repository
	// (measured on 9-nov, 2.1.319: accepted in ~/x-main, written as ~/x), and its
	// server then starts in the worktree.
	if root := worktreeRoot(folder); root != "" {
		cands = append(cands, root)
	}
	for _, p := range cands {
		if f.Projects[p].Accepted || f.Projects[filepath.ToSlash(p)].Accepted {
			return true
		}
	}
	return false
}

// worktreeRoot is the main repository of a git worktree: its .git is a file
// ("gitdir: <repo>/.git/worktrees/<name>") whose folder holds commondir, the path of the
// repository's git folder. The root is that folder's parent when it is called .git,
// and the folder itself for a bare repository. "" for anything else. Read without
// running git.
func worktreeRoot(folder string) string {
	data, err := os.ReadFile(filepath.Join(folder, ".git"))
	if err != nil {
		return "" // a folder .git, or no repository
	}
	line, ok := strings.CutPrefix(strings.TrimSpace(string(data)), "gitdir:")
	if !ok {
		return ""
	}
	gitdir := strings.TrimSpace(line)
	if !filepath.IsAbs(gitdir) {
		gitdir = filepath.Join(folder, gitdir)
	}
	common, err := os.ReadFile(filepath.Join(gitdir, "commondir"))
	if err != nil {
		return "" // a submodule, not a worktree
	}
	dir := strings.TrimSpace(string(common))
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(gitdir, dir)
	}
	dir = filepath.Clean(dir)
	if filepath.Base(dir) == ".git" {
		return filepath.Dir(dir)
	}
	return dir
}

// Launches are the launched sessions systemd knows, running or not, sorted.
func (s System) Launches() ([]LaunchStatus, error) {
	if s.GOOS != "linux" {
		return nil, ErrUnsupported
	}
	out, err := s.Run("systemctl", "--user", "list-units", "--all", "--plain", "--no-legend", "--type=service", launchPrefix+"*")
	if err != nil {
		return nil, err
	}
	var res []LaunchStatus
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 || !strings.HasPrefix(fields[0], launchPrefix) {
			continue
		}
		st := LaunchStatus{Unit: fields[0]}
		props, _ := s.Run("systemctl", "--user", "show", fields[0], "-p", "ActiveState", "-p", "WorkingDirectory", "-p", "Environment")
		for _, p := range strings.Split(props, "\n") {
			k, v, _ := strings.Cut(strings.TrimSpace(p), "=")
			switch k {
			case "ActiveState":
				st.Active = v
			case "WorkingDirectory":
				st.Folder = v
			case "Environment":
				for _, e := range strings.Fields(v) {
					if t, ok := strings.CutPrefix(e, TaskEnv+"="); ok {
						st.Task = t
					}
				}
			}
		}
		res = append(res, st)
	}
	sort.Slice(res, func(i, j int) bool { return res[i].Unit < res[j].Unit })
	return res, nil
}

// Start launches a task's session and waits a few seconds to see it stay up; when the
// server exits at once, the error holds what it wrote in the journal.
func (s System) Start(c *config.Config, set Settings, l Launch) error {
	if s.GOOS != "linux" {
		return ErrUnsupported
	}
	claude, err := s.FindClaude()
	if err != nil {
		return errors.New("claude is not installed or not in the PATH")
	}
	if _, err := s.Run("systemctl", "--user", "show-environment"); err != nil {
		return fmt.Errorf("no systemd user manager: %w", err)
	}
	if fi, err := os.Stat(l.Folder); err != nil || !fi.IsDir() {
		return fmt.Errorf("%s is not a folder", l.Folder)
	}
	if !s.FolderTrusted(l.Folder) {
		return ErrUntrusted
	}
	for _, svc := range Services(c, set) {
		if paths.Same(svc.Folder, l.Folder) {
			return ErrServed
		}
	}
	unit := LaunchUnit(l.Task)
	launched, err := s.Launches()
	if err != nil {
		return err
	}
	for _, x := range launched {
		if x.Active != "active" && x.Active != "activating" {
			continue
		}
		if x.Unit == unit {
			return ErrRunning
		}
		if paths.Same(x.Folder, l.Folder) {
			return ErrServed
		}
	}
	// A unit of an earlier try that failed stays known to systemd under its name.
	_, _ = s.Run("systemctl", "--user", "reset-failed", unit)
	if _, err := s.Run("systemd-run", "--user", "--unit="+unit, "--collect",
		"--description=augmented brain: session for "+l.Task,
		"--working-directory="+l.Folder, "--setenv="+TaskEnv+"="+l.Task, "--setenv=PATH="+s.UnitPath(claude),
		"-p", "Restart=no", "-p", "StandardOutput=null", "-p", "StandardError=journal",
		claude, "remote-control", "--name", l.Name, "--permission-mode", set.PermissionMode, "--spawn", "session"); err != nil {
		return fmt.Errorf("systemd-run: %w", err)
	}
	// The server says what is wrong within a second (untrusted, already served) and then
	// waits about a minute before it exits: read the journal, not only its state.
	for i := 0; i < 6; i++ {
		s.sleep(time.Second)
		state, _ := s.Run("systemctl", "--user", "is-active", unit)
		log, _ := s.Run("journalctl", "--user", "-u", unit, "-o", "cat", "--no-pager", "-q", "-n", "5")
		if strings.Contains(log, "Error:") || (strings.TrimSpace(state) != "active" && strings.TrimSpace(state) != "activating") {
			_, _ = s.Run("systemctl", "--user", "stop", unit)
			msg := strings.TrimSpace(log)
			if msg == "" {
				msg = strings.TrimSpace(state)
			}
			return fmt.Errorf("the session did not start: %s", msg)
		}
	}
	return nil
}

// Stop ends a task's session.
func (s System) Stop(task string) error {
	if s.GOOS != "linux" {
		return ErrUnsupported
	}
	if _, err := s.Run("systemctl", "--user", "stop", LaunchUnit(task)); err != nil {
		return err
	}
	_, _ = s.Run("systemctl", "--user", "reset-failed", LaunchUnit(task))
	return nil
}

func (s System) sleep(d time.Duration) {
	if s.Sleep != nil {
		s.Sleep(d)
		return
	}
	time.Sleep(d)
}
