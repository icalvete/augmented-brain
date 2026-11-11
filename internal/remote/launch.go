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
	// ErrLoggedOut: Claude Code has no login, or it has expired; a session would fail
	// its first turn (`claude auth login` gives it one).
	ErrLoggedOut = errors.New("claude is not logged in")
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

// Retired is the state of a launched session Claude retired: no process, still listed
// in Claude Desktop and claude.ai, and back when opened there.
const Retired = "retired"

// Failed is the state of a launched session whose last turn ended in an error the API
// gave (an expired login, measured on 11-nov): its process may still run, yet it does
// no work until someone writes to it.
const Failed = "failed"

// LaunchStatus is a launched session: Active is "active" while it runs, Retired when
// Claude retired it, Failed when its last turn ended in an API error (Problem says
// which), "ended" when Claude no longer lists it (or systemd's word, for a unit of an
// earlier version).
type LaunchStatus struct {
	Unit    string // Claude's id of the session, or the systemd unit of an earlier version
	Task    string
	Folder  string
	Active  string
	Problem string // with Failed, what the session said: Claude Code's own error text
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

// LaunchedTask is the task this session was launched for (session start), or "" for
// any other session. Its record names the session by Claude's short id, which begins
// the session's id (the hooks' session_id). After /clear the session id changes but
// the job folder Claude gives the session's processes (CLAUDE_JOB_DIR, ending in the
// short id) stays: it counts only when the session runs in the record's folder, since
// a variable may also come from whoever started Claude's daemon.
func LaunchedTask(cfgDir, sessionID, jobDir, cwd string) string {
	entries, _ := os.ReadDir(launchRecords(cfgDir))
	for _, e := range entries {
		task, ok := strings.CutSuffix(e.Name(), ".json")
		if !ok {
			continue
		}
		r, ok := readLaunch(cfgDir, task)
		if !ok || len(r.ID) < 8 {
			continue
		}
		if strings.HasPrefix(sessionID, r.ID) {
			return r.Task
		}
		if jobDir == "" || filepath.Base(jobDir) != r.ID {
			continue
		}
		here, err1 := paths.Resolve(cwd)
		folder, err2 := paths.Resolve(r.Folder)
		if err1 == nil && err2 == nil && cwd != "" && r.Folder != "" && paths.Within(here, folder) {
			return r.Task
		}
	}
	return ""
}

// claudeAgent is one entry of `claude agents --json`.
type claudeAgent struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Cwd    string `json:"cwd"`
	Status string `json:"status"` // "busy" while it answers (or waits for a permission), "idle" when it waits
	// Pid is the session's process. Claude retires a background session left idle for
	// hours (8 on 2.1.322: "bg retire <id>: idle-prompt, idle 8h" in its daemon.log):
	// its process ends, yet it stays in the list, with no pid and no status, until it is
	// opened again from Claude Desktop or claude.ai or removed (measured on 11-nov).
	Pid int `json:"pid"`
	// SessionID names the session's transcript: projects/<folder>/<SessionID>.jsonl in
	// Claude's folder.
	SessionID string `json:"sessionId"`
}

// agents are Claude's background sessions: those that run and those it retired
// (`claude agents --json` lists both; --all would add the ended ones).
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

// listedAgents are the ids of Claude's background sessions, running or retired: what
// Claude Desktop and claude.ai still show.
func (s System) listedAgents(claude string) (map[string]bool, error) {
	return s.agentIDs(claude, false)
}

// runningAgents are the ids of the background sessions whose process runs now.
func (s System) runningAgents(claude string) (map[string]bool, error) {
	return s.agentIDs(claude, true)
}

func (s System) agentIDs(claude string, running bool) (map[string]bool, error) {
	list, err := s.agents(claude)
	if err != nil {
		return nil, err
	}
	ids := map[string]bool{}
	for _, a := range list {
		if a.ID != "" && (!running || a.Pid > 0) {
			ids[a.ID] = true
		}
	}
	return ids, nil
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
		list, err := s.agents(claude)
		if err != nil {
			return nil, err
		}
		states := map[string]string{}
		sessionOf := map[string]string{}
		for _, a := range list {
			if a.ID == "" {
				continue
			}
			states[a.ID] = Retired
			if a.Pid > 0 {
				states[a.ID] = "active"
			}
			sessionOf[a.ID] = a.SessionID
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
			if a := states[r.ID]; a != "" {
				st.Active = a
			}
			if st.Active == "active" {
				if _, problem := state.LastAnswer(s.transcript(sessionOf[r.ID])); problem != "" {
					st.Active, st.Problem = Failed, problem
				}
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
	if !s.loggedIn(claude) {
		return ErrLoggedOut
	}
	launched, err := s.Launches(c.Dir)
	if err != nil {
		return err
	}
	for _, x := range launched {
		// A retired or failed session of the task gives way to the new one (and leaves
		// Claude's lists below); another task's still holds its folder: it comes back
		// there when opened again.
		if x.Task == l.Task && x.Active == "active" {
			return ErrRunning
		}
		if x.Task != l.Task && (x.Active == "active" || x.Active == Failed || x.Active == Retired) && !l.Shared && paths.Same(x.Folder, l.Folder) {
			return ErrServed
		}
	}
	if err := os.MkdirAll(launchRecords(c.Dir), 0o700); err != nil {
		return err
	}
	// An earlier session of the task, ended or retired, leaves Claude's lists once the
	// new one runs: by then Claude's daemon is up, and `claude rm` needs it.
	old, hadOld := readLaunch(c.Dir, l.Task)
	before, _ := s.listedAgents(claude)
	// In a scope of its own: a daemon it starts must not live in the group of a remote
	// service this may run from (scope.go).
	bg, bgArgs := s.scoped(claude,
		"--bg", "--remote-control", l.Name, "-n", l.Name, "--permission-mode", set.PermissionMode, first)
	out, err := s.RunIn(l.Folder, bg, bgArgs...)
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
	if err := s.firstTurn(claude, id, c.Dir, l.Task); err != nil {
		return err
	}
	if hadOld && old.ID != id {
		// A failed one still runs: it is stopped first.
		_, _ = s.Run(claude, "stop", old.ID)
		_, _ = s.Run(claude, "rm", old.ID)
	}
	return nil
}

// FirstTurnWait is how long Start waits for a launched session's first answer. A
// session whose login has expired writes its error within seconds (5 on 11-nov); a
// healthy one answers within a few more, and Start returns as soon as it does.
const FirstTurnWait = 60 * time.Second

// startupWait is how long Start watches for a session that ends at once (no login, a
// broken setting) when it cannot read its transcript.
const startupWait = 5 * time.Second

// FirstTurnFailed: the launched session's first turn ended in an error the API gave;
// Cause is Claude Code's own text, such as "Login reached its time limit · Please run
// /login". The session was stopped and taken out of Claude's lists.
type FirstTurnFailed struct {
	Cause string
	// Login is true when the cause is the login: `claude auth login` mends it.
	Login bool
}

func (e *FirstTurnFailed) Error() string {
	return "the session's first turn failed: " + e.Cause
}

// firstTurn watches a session just launched until its first answer: an error when it
// ends at once or its first turn fails (it is then removed, with its record), nil when
// it answers, or when FirstTurnWait passes, or when its transcript cannot be found
// (an earlier claude gives no session id: then only the first startupWait counts).
func (s System) firstTurn(claude, id, cfgDir, task string) error {
	fail := func(err error) error {
		_, _ = s.Run(claude, "stop", id)
		_, _ = s.Run(claude, "rm", id)
		_ = os.Remove(launchRecordPath(cfgDir, task))
		return err
	}
	for waited := time.Duration(0); waited < FirstTurnWait; waited += time.Second {
		s.sleep(time.Second)
		list, err := s.agents(claude)
		if err != nil {
			continue
		}
		a, listed := findAgent(list, id)
		if !listed {
			log, _ := s.Run(claude, "logs", id)
			_, _ = s.Run(claude, "rm", id)
			_ = os.Remove(launchRecordPath(cfgDir, task))
			return fmt.Errorf("the session did not start: %s", tail(cleanTerminal(log), 300))
		}
		path := s.transcript(a.SessionID)
		if path == "" {
			if waited+time.Second >= startupWait {
				return nil
			}
			continue
		}
		answered, problem := state.LastAnswer(path)
		if problem != "" {
			return fail(&FirstTurnFailed{Cause: problem, Login: LoginProblem(problem)})
		}
		if answered {
			return nil
		}
	}
	return nil
}

// LoginProblem says whether Claude Code's error is about the login (its text on
// 2.1.322: "Login reached its time limit · Please run /login"; also "Invalid API key ·
// Please run /login" and "OAuth token has expired").
func LoginProblem(text string) bool {
	t := strings.ToLower(text)
	return strings.Contains(t, "/login") || strings.Contains(t, "oauth") || strings.Contains(t, "api key") || strings.Contains(t, "log in")
}

// transcript is the transcript of Claude's session with this id: the one file
// projects/*/<id>.jsonl in Claude's folder (the folder's name follows where the session
// works, which may change when it enters a worktree), or "" when there is none or the
// id is not a plain one.
func (s System) transcript(sessionID string) string {
	if s.ClaudeDir == "" || sessionID == "" || strings.ContainsAny(sessionID, `/\*?[`) || strings.Contains(sessionID, "..") {
		return ""
	}
	found, _ := filepath.Glob(filepath.Join(s.ClaudeDir, "projects", "*", sessionID+".jsonl"))
	if len(found) != 1 {
		return ""
	}
	return found[0]
}

// loggedIn says whether Claude Code has a login that has not expired, from `claude
// auth status` (JSON on 2.1.322: loggedIn, loginExpiresAt). When that cannot be read
// (an earlier claude, another output), it says yes: the first turn tells anyway.
func (s System) loggedIn(claude string) bool {
	out, _ := s.Run(claude, "auth", "status")
	var st struct {
		LoggedIn  *bool  `json:"loggedIn"`
		ExpiresAt string `json:"loginExpiresAt"`
	}
	if json.Unmarshal([]byte(strings.TrimSpace(out)), &st) != nil || st.LoggedIn == nil {
		return true
	}
	if !*st.LoggedIn {
		return false
	}
	if t, err := time.Parse(time.RFC3339, st.ExpiresAt); err == nil && t.Before(time.Now()) {
		return false
	}
	return true
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

// ErrNoSession: the task has no session to close.
var ErrNoSession = errors.New("this task has no session")

// LeftInClaude: the task's session no longer runs and is no longer the task's, but
// Claude could not remove it from its lists (Claude Desktop, claude.ai): `claude rm ID`
// does it later.
type LeftInClaude struct{ ID string }

func (e *LeftInClaude) Error() string {
	return "claude could not remove session " + e.ID + "; run `claude rm " + e.ID + "` later"
}

// Stop ends a task's session and takes it out of Claude Desktop and claude.ai. For a
// session of an earlier version, it stops its systemd unit.
//
// `claude stop` and `claude rm` need Claude's daemon, which runs only while something
// uses it (it exits on its own about a minute after the last client goes): without it
// both fail with "the background service may be restarting" however often they are
// tried, as they do after Claude retired the last session (measured on 11-nov,
// 2.1.322). So when rm fails for a session that does not run, Stop starts the daemon
// and tries once more; if Claude still cannot, the record goes anyway and the error is
// a *LeftInClaude.
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
			list, lerr := s.agents(claude)
			a, listed := findAgent(list, r.ID)
			if lerr == nil && listed && a.Pid > 0 {
				return fmt.Errorf("claude rm %s: %w", r.ID, err)
			}
			if lerr != nil || listed {
				s.wakeDaemon(claude)
				if _, err := s.Run(claude, "rm", r.ID); err != nil {
					if err := os.Remove(launchRecordPath(cfgDir, task)); err != nil {
						return err
					}
					return &LeftInClaude{ID: r.ID}
				}
			}
		}
		return os.Remove(launchRecordPath(cfgDir, task))
	}
	if !s.hasUnit(task) {
		return ErrNoSession
	}
	if _, err := s.Run("systemctl", "--user", "stop", LaunchUnit(task)); err != nil {
		return err
	}
	_, _ = s.Run("systemctl", "--user", "reset-failed", LaunchUnit(task))
	return nil
}

// hasUnit says whether an earlier version launched the task's session as a systemd unit
// that systemd still knows of.
func (s System) hasUnit(task string) bool {
	for _, u := range s.unitLaunches() {
		if u.Unit == LaunchUnit(task) {
			return true
		}
	}
	return false
}

// wakeDaemon starts Claude's daemon when it does not run, the way `claude --bg` does
// (transient: it exits on its own once unused), and waits up to ten seconds for it.
func (s System) wakeDaemon(claude string) {
	if _, err := s.Run(claude, "daemon", "status"); err == nil {
		return
	}
	if s.StartDaemon == nil {
		return
	}
	if err := s.StartDaemon(claude); err != nil {
		return
	}
	for i := 0; i < 10; i++ {
		s.sleep(time.Second)
		if _, err := s.Run(claude, "daemon", "status"); err == nil {
			return
		}
	}
}

// LaunchRunning says whether the task has a session of its own whose process runs now.
func (s System) LaunchRunning(cfgDir, task string) bool {
	r, ok := readLaunch(cfgDir, task)
	if !ok || s.GOOS != "linux" {
		return false
	}
	claude, err := s.FindClaude()
	if err != nil {
		return false
	}
	running, err := s.runningAgents(claude)
	return err == nil && running[r.ID]
}

// HasLaunch says whether the task has a session of its own on record, running or not.
func HasLaunch(cfgDir, task string) bool {
	_, ok := readLaunch(cfgDir, task)
	return ok
}

// ErrStillBusy: the session kept answering for as long as StopWhenIdle waits.
var ErrStillBusy = errors.New("the session is still answering")

// findAgent is the session with this id in Claude's list, if it is there.
func findAgent(list []claudeAgent, id string) (claudeAgent, bool) {
	for _, a := range list {
		if a.ID == id {
			return a, true
		}
	}
	return claudeAgent{}, false
}

// StopWhenIdle closes a task's session once it has finished its answer (seen idle
// twice in a row, a couple of seconds apart), then removes the task's worktree when
// nothing would be lost. Closing a task is often done by its own session, in the middle
// of an answer: stopping it there would cut that answer off. A session Claude retired
// or no longer lists answers nothing, so it counts as idle; one waiting for a
// permission is "busy" (measured on 11-nov) and is waited for. It waits at most max,
// and leaves a session that is still answering by then. A session of an earlier
// version (a systemd unit) tells nothing of its answers: it is stopped at once, as
// `session stop` does. With no session at all it says so (ErrNoSession).
func (s System) StopWhenIdle(cfgDir, task string, max time.Duration) error {
	r, ok := readLaunch(cfgDir, task)
	if !ok {
		return s.Stop(cfgDir, task)
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
		a, listed := findAgent(list, r.ID)
		if !listed || a.Pid == 0 || a.Status == "idle" {
			idle++
		} else {
			idle = 0
		}
	}
	stopErr := s.Stop(cfgDir, task)
	var left *LeftInClaude
	if stopErr != nil && !errors.As(stopErr, &left) {
		return stopErr
	}
	_, _, _ = RemoveTaskWorktree(cfgDir, task)
	_, _ = RemoveNotedWorktrees(cfgDir, task)
	return stopErr
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
