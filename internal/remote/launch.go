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
	"github.com/icalvete/augmented-brain/internal/state"
)

// A launched session is a background Claude session of its own for one task, started
// by `augmented-brain session start` with `claude --bg --remote-control`, so a session
// the user reaches from another computer can open one per task: it shows up in Claude
// Desktop and claude.ai with the task's title, in the task's folder, and starts on its
// own with a first message that names its task. Claude's own daemon keeps it running,
// not the session that asked for it. `session stop` ends it with `claude stop` and removes it with `claude rm`,
// which also takes it out of Claude Desktop and claude.ai (measured on 10-nov-2026,
// 2.1.322). Which session belongs to which task is kept in state/sessions/<task>.json.
// A reboot ends it too: launched sessions are not brought back.
//
// Measured on 2.1.322: with --bg, Claude asks nothing about trust and lets two sessions
// work in one folder. A session does not get the environment of whoever asks for it:
// `claude --bg` starts Claude's daemon when none runs, with the environment of that
// first caller, and every later session comes from the daemon. So the task travels in
// the first message, never in a variable, and TaskEnv is taken out of what the daemon
// may get (RunIn): set there, every later session would believe it is that task's. The launcher still asks for a trusted folder (a repository's own
// hooks and settings must not run unasked) and keeps one task per folder.
//
// Sessions of earlier versions were transient systemd units (LaunchUnit): they are
// still listed and stopped.

// TaskEnv is the variable the sessions of earlier versions (systemd units) carried: the
// id of their task. Only read now, to list those units.
const TaskEnv = "AUGMENTED_BRAIN_TASK"

const launchPrefix = "augmented-brain-session-"

// Errors of Start the command explains to the user.
var (
	// ErrUntrusted: the folder has not accepted Claude's trust question itself.
	ErrUntrusted = errors.New("claude does not trust this folder")
	// ErrServed: another task's session works in the folder.
	ErrServed = errors.New("another task's session works in this folder")
	// ErrRunning: the task has a launched session already.
	ErrRunning = errors.New("this task has a session already")
)

// Launch is one task's session.
type Launch struct {
	Task   string // the task's id
	Name   string // the session's name in Claude Desktop and claude.ai
	Folder string
	// Shared: the folder is the space's own, for tasks with no repository, which may
	// share it (no code of theirs to step on).
	Shared bool
}

// Unit is the transient unit's name for a task.
func LaunchUnit(task string) string {
	return launchPrefix + unitSafe.ReplaceAllString(strings.TrimPrefix(task, "task-"), "-") + ".service"
}

var unitSafe = regexp.MustCompile(`[^A-Za-z0-9_.-]+`)

// LaunchStatus is a launched session: Active is "active" while it runs.
type LaunchStatus struct {
	Unit   string // Claude's id of the session, or the systemd unit of an earlier version
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
	if root := WorktreeRoot(folder); root != "" {
		cands = append(cands, root)
	}
	for _, p := range cands {
		if f.Projects[p].Accepted || f.Projects[filepath.ToSlash(p)].Accepted {
			return true
		}
	}
	return false
}

// WorktreeRoot is the main repository of a git worktree: its .git is a file
// ("gitdir: <repo>/.git/worktrees/<name>") whose folder holds commondir, the path of the
// repository's git folder. The root is that folder's parent when it is called .git,
// and the folder itself for a bare repository. "" for anything else. Read without
// running git.
func WorktreeRoot(folder string) string {
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

// launchRecord is what state/sessions/<task>.json keeps of a launched session.
type launchRecord struct {
	ID      string    `json:"id"`
	Task    string    `json:"task"`
	Name    string    `json:"name"`
	Folder  string    `json:"folder"`
	Started time.Time `json:"started"`
}

func launchRecords(cfgDir string) string { return filepath.Join(cfgDir, state.Dir, "sessions") }

func launchRecordPath(cfgDir, task string) string {
	return filepath.Join(launchRecords(cfgDir), unitSafe.ReplaceAllString(task, "-")+".json")
}

func readLaunch(cfgDir, task string) (launchRecord, bool) {
	var r launchRecord
	data, err := os.ReadFile(launchRecordPath(cfgDir, task))
	if err != nil || json.Unmarshal(data, &r) != nil || r.ID == "" {
		return r, false
	}
	return r, true
}

// claudeAgent is one entry of `claude agents --json`.
type claudeAgent struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Cwd    string `json:"cwd"`
	Status string `json:"status"` // "busy" while it answers, "idle" when it waits
}

// agents are Claude's background sessions that run now.
func (s System) agents(claude string) ([]claudeAgent, error) {
	out, err := s.Run(claude, "agents", "--json")
	if err != nil {
		return nil, err
	}
	var list []claudeAgent
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &list); err != nil {
		return nil, fmt.Errorf("claude agents --json: %w", err)
	}
	return list, nil
}

// liveAgents are the ids of Claude's background sessions that run now (`claude agents
// --json` lists only those; --all would add the stopped ones).
func (s System) liveAgents(claude string) (map[string]bool, error) {
	list, err := s.agents(claude)
	if err != nil {
		return nil, err
	}
	live := map[string]bool{}
	for _, a := range list {
		if a.ID != "" {
			live[a.ID] = true
		}
	}
	return live, nil
}

// Launches are the launched sessions known on this machine, running or not, sorted by
// task: those in state/sessions, and the systemd units of earlier versions.
func (s System) Launches(cfgDir string) ([]LaunchStatus, error) {
	if s.GOOS != "linux" {
		return nil, ErrUnsupported
	}
	var res []LaunchStatus
	entries, _ := os.ReadDir(launchRecords(cfgDir))
	if len(entries) > 0 {
		claude, err := s.FindClaude()
		if err != nil {
			return nil, errors.New("claude is not installed or not in the PATH")
		}
		live, err := s.liveAgents(claude)
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			task, ok := strings.CutSuffix(e.Name(), ".json")
			if !ok {
				continue
			}
			r, ok := readLaunch(cfgDir, task)
			if !ok {
				continue
			}
			st := LaunchStatus{Unit: r.ID, Task: r.Task, Folder: r.Folder, Active: "ended"}
			if live[r.ID] {
				st.Active = "active"
			}
			res = append(res, st)
		}
	}
	res = append(res, s.unitLaunches()...)
	sort.Slice(res, func(i, j int) bool { return res[i].Task+res[i].Unit < res[j].Task+res[j].Unit })
	return res, nil
}

// unitLaunches are the sessions earlier versions launched as transient systemd units.
// Without systemd there are none.
func (s System) unitLaunches() []LaunchStatus {
	out, err := s.Run("systemctl", "--user", "list-units", "--all", "--plain", "--no-legend", "--type=service", launchPrefix+"*")
	if err != nil {
		return nil
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
				if v == "activating" {
					st.Active = "active"
				}
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
	return res
}

// backgrounded is the line `claude --bg` prints: "backgrounded · <id> · <name>".
var backgrounded = regexp.MustCompile(`backgrounded\s+·\s+([0-9A-Za-z]+)`)

// Start launches a task's session with a first message, so it starts on its own, and
// waits a few seconds to see it stay up; when it ends at once, the error holds the
// last lines it wrote.
func (s System) Start(c *config.Config, set Settings, l Launch, first string) error {
	if s.GOOS != "linux" {
		return ErrUnsupported
	}
	claude, err := s.FindClaude()
	if err != nil {
		return errors.New("claude is not installed or not in the PATH")
	}
	if fi, err := os.Stat(l.Folder); err != nil || !fi.IsDir() {
		return fmt.Errorf("%s is not a folder", l.Folder)
	}
	if !s.FolderTrusted(l.Folder) {
		return ErrUntrusted
	}
	launched, err := s.Launches(c.Dir)
	if err != nil {
		return err
	}
	for _, x := range launched {
		if x.Active != "active" {
			continue
		}
		if x.Task == l.Task {
			return ErrRunning
		}
		if !l.Shared && paths.Same(x.Folder, l.Folder) {
			return ErrServed
		}
	}
	if err := os.MkdirAll(launchRecords(c.Dir), 0o700); err != nil {
		return err
	}
	if old, ok := readLaunch(c.Dir, l.Task); ok {
		// An earlier session of the task that ended: out of Claude's lists too.
		_, _ = s.Run(claude, "rm", old.ID)
	}
	before, _ := s.liveAgents(claude)
	out, err := s.RunIn(l.Folder, claude,
		"--bg", "--remote-control", l.Name, "-n", l.Name, "--permission-mode", set.PermissionMode, first)
	id := ""
	if m := backgrounded.FindStringSubmatch(out); m != nil {
		id = m[1]
	} else if err == nil {
		// While Claude starts its daemon it may print only "Starting background
		// service…" (seen on 10-nov with two launches at once): the new session is then
		// the one with this name and folder that was not running before.
		id = s.newAgent(claude, before, l)
	}
	if err != nil || id == "" {
		msg := tail(cleanTerminal(out), 300)
		if err != nil {
			msg = strings.TrimSpace(msg + " " + err.Error())
		}
		return fmt.Errorf("the session did not start: %s", msg)
	}
	data, _ := json.MarshalIndent(launchRecord{ID: id, Task: l.Task, Name: l.Name, Folder: l.Folder, Started: time.Now().UTC()}, "", "  ")
	if err := paths.WriteAtomic(launchRecordPath(c.Dir, l.Task), append(data, '\n'), 0o600); err != nil {
		_, _ = s.Run(claude, "stop", id)
		_, _ = s.Run(claude, "rm", id)
		return err
	}
	// A session that cannot start (no login, a broken setting) ends within seconds.
	for i := 0; i < 5; i++ {
		s.sleep(time.Second)
		live, err := s.liveAgents(claude)
		if err != nil || live[id] {
			continue
		}
		log, _ := s.Run(claude, "logs", id)
		_, _ = s.Run(claude, "rm", id)
		_ = os.Remove(launchRecordPath(c.Dir, l.Task))
		return fmt.Errorf("the session did not start: %s", tail(cleanTerminal(log), 300))
	}
	return nil
}

// newAgent looks for a few seconds for the session just launched for l among Claude's
// background sessions: its name and folder, and not running before.
func (s System) newAgent(claude string, before map[string]bool, l Launch) string {
	for i := 0; i < 5; i++ {
		list, _ := s.agents(claude)
		for _, a := range list {
			if a.ID != "" && !before[a.ID] && a.Name == l.Name && paths.Same(a.Cwd, l.Folder) {
				return a.ID
			}
		}
		s.sleep(time.Second)
	}
	return ""
}

// Stop ends a task's session and takes it out of Claude Desktop and claude.ai. For a
// session of an earlier version, it stops its systemd unit.
func (s System) Stop(cfgDir, task string) error {
	if s.GOOS != "linux" {
		return ErrUnsupported
	}
	if r, ok := readLaunch(cfgDir, task); ok {
		claude, err := s.FindClaude()
		if err != nil {
			return errors.New("claude is not installed or not in the PATH")
		}
		_, _ = s.Run(claude, "stop", r.ID) // fails when it had ended already
		if _, err := s.Run(claude, "rm", r.ID); err != nil {
			if live, lerr := s.liveAgents(claude); lerr != nil || live[r.ID] {
				return fmt.Errorf("claude rm %s: %w", r.ID, err)
			}
		}
		return os.Remove(launchRecordPath(cfgDir, task))
	}
	if _, err := s.Run("systemctl", "--user", "stop", LaunchUnit(task)); err != nil {
		return err
	}
	_, _ = s.Run("systemctl", "--user", "reset-failed", LaunchUnit(task))
	return nil
}

// LaunchRunning says whether the task has a session of its own that runs now.
func (s System) LaunchRunning(cfgDir, task string) bool {
	r, ok := readLaunch(cfgDir, task)
	if !ok || s.GOOS != "linux" {
		return false
	}
	claude, err := s.FindClaude()
	if err != nil {
		return false
	}
	live, err := s.liveAgents(claude)
	return err == nil && live[r.ID]
}

// ErrStillBusy: the session kept answering for as long as StopWhenIdle waits.
var ErrStillBusy = errors.New("the session is still answering")

// StopWhenIdle closes a task's session once it has finished its answer (seen idle
// twice in a row, a couple of seconds apart), then removes the task's worktree when
// nothing would be lost. Closing a task is often done by its own session, in the middle
// of an answer: stopping it there would cut that answer off. It waits at most max, and
// leaves a session that is still answering by then.
func (s System) StopWhenIdle(cfgDir, task string, max time.Duration) error {
	r, ok := readLaunch(cfgDir, task)
	if !ok {
		return nil
	}
	claude, err := s.FindClaude()
	if err != nil {
		return err
	}
	idle := 0
	for waited := time.Duration(0); idle < 2; waited += 2 * time.Second {
		if waited > max {
			return ErrStillBusy
		}
		s.sleep(2 * time.Second)
		list, err := s.agents(claude)
		if err != nil {
			continue
		}
		state := "gone"
		for _, a := range list {
			if a.ID == r.ID {
				state = a.Status
			}
		}
		switch state {
		case "idle", "gone":
			idle++
		default:
			idle = 0
		}
	}
	if err := s.Stop(cfgDir, task); err != nil {
		return err
	}
	_, _, _ = RemoveTaskWorktree(cfgDir, task)
	_, _ = RemoveNotedWorktrees(cfgDir, task)
	return nil
}

// cleanTerminal takes the terminal's control sequences out of what a session drew,
// which is a screen redrawn in place, not lines: what is left is its words.
func cleanTerminal(s string) string {
	return strings.Join(strings.Fields(terminalCodes.ReplaceAllString(s, " ")), " ")
}

var terminalCodes = regexp.MustCompile(`\x1b\[[0-9;?]*[A-Za-z]|\x1b\][^\x07]*\x07|[\r\x07]`)

// tail is the end of a text, at most n characters.
func tail(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return "…" + string(r[len(r)-n:])
	}
	return s
}

func (s System) sleep(d time.Duration) {
	if s.Sleep != nil {
		s.Sleep(d)
		return
	}
	time.Sleep(d)
}
