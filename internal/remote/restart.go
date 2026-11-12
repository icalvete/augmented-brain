package remote

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/icalvete/augmented-brain/internal/state"
)

// Restarting Claude sessions (`session restart`; update runs it with --old).
//
// An open session keeps the augmented brain it started with: after an update its
// tools may fail on what the new program writes. Measured on 12-nov with claude
// 2.1.322, there are three kinds of session, and each restarts its own way:
//
//   - a background session (`claude --bg`, kind "background" with an id):
//     `claude respawn <id>` gives it a new process with the same id and the whole
//     conversation (asked after a respawn what it had said, it said it);
//   - a session of a remote service (kind "interactive", its parent the service's main
//     process): restarting the service brings the same cloud session back in about ten
//     seconds (measured on 11-nov, same session ids). The service serves every session
//     it holds, so it restarts only when all of them are idle, and after KeepOutside,
//     so nothing else in its group dies with it (scope.go);
//   - any other (Claude Desktop, a terminal): its app or its person started it, and
//     killing it from outside leaves a broken window. Only its augmented brain server
//     (an `augmented-brain mcp` below it) is stopped: Claude Code starts it again, from
//     the program of now, at the session's next call to it (measured on 12-nov with a
//     Claude Desktop session: the window stayed, and doctor stopped counting it). Where
//     this program cannot see which server is whose (Windows), it is left alone, and
//     said why.
//
// Each session is looked at again just before its restart, and only an idle one is
// restarted (busy also means waiting for a permission): never one in the middle of an
// answer. One that fails leaves the others going. Never the session, or the service,
// this process runs in: that one waits for its answer to end (RestartWhenIdle).

// SessionKind is how a session is restarted.
type SessionKind int

const (
	// KindOther is a session of Claude Desktop or a terminal: only its augmented brain
	// server is restarted.
	KindOther SessionKind = iota
	// KindBackground is a `claude --bg` session: claude respawn.
	KindBackground
	// KindRemote is a session of a remote service: the service is restarted.
	KindRemote
)

// Session is an open Claude session on this machine.
type Session struct {
	PID       int
	ID        string // a background session's id; "" for the others
	SessionID string
	Name      string
	Status    string // idle, busy, waiting…
	Kind      SessionKind
	Unit      string // the remote service, for KindRemote
	// Old says its augmented brain is an earlier one; Version is that one's ("" when
	// it did not say).
	Old     bool
	Version string
	// Servers are its augmented brain servers to stop, for KindOther: the old ones
	// (OldSessions); none means every one below it.
	Servers []int
}

// Key is how `session restart` names it: a background session's id, or the first
// characters of the session's id.
func (se Session) Key() string {
	if se.ID != "" {
		return se.ID
	}
	if len(se.SessionID) > 8 {
		return se.SessionID[:8]
	}
	return se.SessionID
}

// Label is the name people see, or its key when it has none.
func (se Session) Label() string {
	if se.Name != "" {
		return se.Name
	}
	return se.Key()
}

// Outcome is what a restart did with a session.
type Outcome int

const (
	Restarted       Outcome = iota
	ServerRestarted         // Claude Desktop or a terminal: its augmented brain server only
	Busy                    // answering (or waiting for a permission): left as it is
	Self                    // the session, or the service, this process runs in
	SharedBusy              // its remote service holds another session that is answering
	Unsupported             // Claude Desktop or a terminal whose server cannot be found
	Gone                    // it no longer runs: it starts with the new program when opened
	RestartFailed
)

// RestartResult is what happened to one session.
type RestartResult struct {
	Session Session
	Outcome Outcome
	Err     error
}

// ErrAmbiguous: more than one session answers to the name given.
var ErrAmbiguous = errors.New("more than one session answers to that name")

// Sessions are the Claude sessions running on this machine, each with its kind.
func (s System) Sessions(claude string) ([]Session, error) {
	list, err := s.agents(claude)
	if err != nil {
		return nil, err
	}
	mains := s.serviceMains()
	var out []Session
	for _, a := range list {
		if a.Pid <= 0 {
			continue // retired: no process, so nothing runs an old program
		}
		se := Session{PID: a.Pid, ID: a.ID, SessionID: a.SessionID, Name: a.Name, Status: a.Status}
		switch {
		case a.Kind == "background" || a.ID != "":
			se.Kind = KindBackground
		case s.Parent != nil && mains[s.Parent(a.Pid)] != "":
			se.Kind, se.Unit = KindRemote, mains[s.Parent(a.Pid)]
		}
		out = append(out, se)
	}
	return out, nil
}

// serviceMains maps the main process of every running remote service to its unit.
// Only systemd's services: on macOS and Windows a remote session counts as KindOther.
func (s System) serviceMains() map[int]string {
	out := map[int]string{}
	if s.GOOS != "linux" {
		return out
	}
	units, _ := s.installedUnits()
	for _, u := range units {
		if pid, active := s.mainPID(u); active && pid > 0 {
			out[pid] = u
		}
	}
	return out
}

// OldSessions are the sessions whose augmented brain server is one of old
// (state.OldServers): the server's parent, or the first ancestor that is a session.
// unknown counts the servers with no session above them.
func (s System) OldSessions(claude string, old []state.Server) (out []Session, unknown int, err error) {
	list, err := s.Sessions(claude)
	if err != nil {
		return nil, 0, err
	}
	at := map[int]int{}
	for i, se := range list {
		at[se.PID] = i
	}
	var order []int
	for _, srv := range old {
		found := false
		for _, p := range s.ancestors(srv.PID) {
			if i, ok := at[p]; ok {
				if !list[i].Old {
					list[i].Old, list[i].Version = true, srv.Version
					order = append(order, i)
				}
				list[i].Servers = append(list[i].Servers, srv.PID)
				found = true
				break
			}
		}
		if !found {
			unknown++
		}
	}
	for _, i := range order {
		out = append(out, list[i])
	}
	return out, unknown, nil
}

// FindSession is the running session that key names: a task with a launched session,
// a background session's id, a session id (or its first eight characters or more), or
// the session's name.
func (s System) FindSession(claude, cfgDir, key string) (Session, error) {
	if r, ok := readLaunch(cfgDir, key); ok && r.ID != "" {
		key = r.ID
	}
	list, err := s.Sessions(claude)
	if err != nil {
		return Session{}, err
	}
	var found []Session
	for _, se := range list {
		if se.ID == key || se.SessionID == key || se.Name == key ||
			(len(key) >= 8 && strings.HasPrefix(se.SessionID, key)) {
			found = append(found, se)
		}
	}
	switch len(found) {
	case 0:
		return Session{}, ErrNoSession
	case 1:
		return found[0], nil
	}
	return Session{}, ErrAmbiguous
}

// ancestors are pid and the processes above it, nearest first (at most 16).
func (s System) ancestors(pid int) []int {
	out := []int{pid}
	if s.Parent == nil {
		return out
	}
	for len(out) < 16 {
		p := s.Parent(out[len(out)-1])
		if p <= 1 {
			break
		}
		out = append(out, p)
	}
	return out
}

// isSelf says whether restarting se would kill this process: se is one of its
// ancestors, or se's service's main process is.
func (s System) isSelf(se Session, mains map[int]string) bool {
	if s.PID <= 0 {
		return false
	}
	for _, p := range s.ancestors(s.PID) {
		if p == se.PID || (se.Kind == KindRemote && mains[p] == se.Unit) {
			return true
		}
	}
	return false
}

// Restart restarts each of these sessions as the package comment says, one at a time,
// and says what happened to each. A remote service is restarted once for all its
// sessions given.
func (s System) Restart(claude string, targets []Session) []RestartResult {
	var out []RestartResult
	doneUnit := map[string]RestartResult{}
	for _, t := range targets {
		if t.Kind == KindRemote {
			if r, ok := doneUnit[t.Unit]; ok {
				r.Session = t
				out = append(out, r)
				continue
			}
		}
		r := s.restartOne(claude, t)
		if t.Kind == KindRemote {
			doneUnit[t.Unit] = r
		}
		out = append(out, r)
	}
	return out
}

func (s System) restartOne(claude string, t Session) RestartResult {
	r := RestartResult{Session: t}
	var servers []int
	if t.Kind == KindOther {
		if servers = s.serversOf(t); len(servers) == 0 {
			r.Outcome = Unsupported
			return r
		}
	}
	mains := s.serviceMains()
	if s.isSelf(t, mains) {
		r.Outcome = Self
		return r
	}
	// Looked at again now: it may have started answering since the list was made.
	now, err := s.Sessions(claude)
	if err != nil {
		r.Outcome, r.Err = RestartFailed, err
		return r
	}
	cur, ok := Session{}, false
	for _, se := range now {
		if (t.ID != "" && se.ID == t.ID) || (t.ID == "" && se.PID == t.PID) {
			cur, ok = se, true
			break
		}
	}
	if !ok {
		r.Outcome = Gone
		return r
	}
	if cur.Status != "idle" {
		r.Outcome = Busy
		return r
	}
	switch t.Kind {
	case KindOther:
		// Its window stays; Claude Code starts the server again at its next call.
		for _, pid := range servers {
			if err := s.StopServer(pid); err != nil {
				r.Outcome, r.Err = RestartFailed, err
				return r
			}
		}
		r.Outcome = ServerRestarted
		return r
	case KindBackground:
		if out, err := s.Run(claude, "respawn", t.ID); err != nil {
			r.Outcome, r.Err = RestartFailed, errors.New(strings.TrimSpace(cleanTerminal(out)+" "+err.Error()))
			return r
		}
	case KindRemote:
		for _, se := range now {
			if se.Kind == KindRemote && se.Unit == t.Unit && se.Status != "idle" {
				r.Outcome = SharedBusy
				return r
			}
		}
		// What the server did not start (Claude's daemon above all) must not die with it.
		if _, err := s.KeepOutside(t.Unit); err != nil {
			r.Outcome, r.Err = RestartFailed, err
			return r
		}
		if _, err := s.Run("systemctl", "--user", "restart", t.Unit); err != nil {
			r.Outcome, r.Err = RestartFailed, fmt.Errorf("systemctl --user restart %s: %w", t.Unit, err)
			return r
		}
	}
	r.Outcome = Restarted
	return r
}

// serversOf are the augmented brain servers to stop for a session of Claude Desktop
// or a terminal: those OldSessions found, or else every one running below it. None when
// this program cannot see them.
func (s System) serversOf(t Session) []int {
	if s.StopServer == nil {
		return nil
	}
	if len(t.Servers) > 0 {
		return t.Servers
	}
	if s.MCPServers == nil || s.Parent == nil {
		return nil
	}
	var out []int
	for _, pid := range s.MCPServers() {
		if pid != s.PID && slices.Contains(s.ancestors(pid)[1:], t.PID) {
			out = append(out, pid)
		}
	}
	return out
}

// RestartWhenIdle restarts t once it, and for a remote service every session it
// holds, has been seen idle twice in a row, a couple of seconds apart: what a session
// runs to restart itself after parking its task, detached so that its own answer can
// end. It waits at most max (ErrStillBusy).
func (s System) RestartWhenIdle(claude string, t Session, max time.Duration) RestartResult {
	idle := 0
	for waited := time.Duration(0); idle < 2; waited += 2 * time.Second {
		if waited > max {
			return RestartResult{Session: t, Outcome: Busy, Err: ErrStillBusy}
		}
		s.sleep(2 * time.Second)
		now, err := s.Sessions(claude)
		if err != nil {
			continue
		}
		quiet, there := true, false
		for _, se := range now {
			mine := (t.ID != "" && se.ID == t.ID) || (t.ID == "" && se.PID == t.PID)
			there = there || mine
			if (mine || (t.Kind == KindRemote && se.Kind == KindRemote && se.Unit == t.Unit)) && se.Status != "idle" {
				quiet = false
			}
		}
		if !there {
			return RestartResult{Session: t, Outcome: Gone}
		}
		if quiet {
			idle++
		} else {
			idle = 0
		}
	}
	return s.restartOne(claude, t)
}

// procParent is the parent of pid: /proc on Linux, ps on macOS, 0 elsewhere or when
// it cannot tell.
func procParent(pid int) int {
	switch runtime.GOOS {
	case "linux":
		data, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
		if err != nil {
			return 0
		}
		// pid (comm) state ppid …: comm may hold spaces and parentheses.
		s := string(data)
		i := strings.LastIndexByte(s, ')')
		if i < 0 {
			return 0
		}
		f := strings.Fields(s[i+1:])
		if len(f) < 2 {
			return 0
		}
		n, _ := strconv.Atoi(f[1])
		return n
	case "darwin":
		out, err := exec.Command("ps", "-o", "ppid=", "-p", strconv.Itoa(pid)).Output()
		if err != nil {
			return 0
		}
		n, _ := strconv.Atoi(strings.TrimSpace(string(out)))
		return n
	}
	return 0
}

// Scoped is the command that runs name with args in a scope of its own, when the
// system has one (scope.go): what restarts a remote service from inside it must not
// die with it.
func (s System) Scoped(name string, args ...string) (string, []string) {
	return s.scoped(name, args...)
}

// procMCPServers are the augmented-brain mcp processes running: /proc on Linux, one ps
// on macOS, none elsewhere.
func procMCPServers() []int {
	var out []int
	switch runtime.GOOS {
	case "linux":
		procs, _ := os.ReadDir("/proc")
		for _, p := range procs {
			pid, err := strconv.Atoi(p.Name())
			if err == nil && isMCPServer(pid) {
				out = append(out, pid)
			}
		}
	case "darwin":
		data, err := exec.Command("ps", "-axo", "pid=,command=").Output()
		if err != nil {
			return nil
		}
		for _, line := range strings.Split(string(data), "\n") {
			f := strings.Fields(line)
			if len(f) >= 3 && filepath.Base(f[1]) == "augmented-brain" && f[2] == "mcp" {
				if pid, err := strconv.Atoi(f[0]); err == nil {
					out = append(out, pid)
				}
			}
		}
	}
	return out
}

// isMCPServer says whether pid runs augmented-brain mcp now.
func isMCPServer(pid int) bool {
	var args []string
	switch runtime.GOOS {
	case "linux":
		data, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/cmdline")
		if err != nil {
			return false
		}
		args = strings.Split(strings.TrimRight(string(data), "\x00"), "\x00")
	case "darwin":
		data, err := exec.Command("ps", "-o", "command=", "-p", strconv.Itoa(pid)).Output()
		if err != nil {
			return false
		}
		args = strings.Fields(string(data))
	}
	return len(args) >= 2 && filepath.Base(args[0]) == "augmented-brain" && args[1] == "mcp"
}

// stopMCPServer asks an augmented brain server to end (SIGTERM), after checking the
// process is still one: its number may have gone to another since it was listed.
func stopMCPServer(pid int) error {
	if !isMCPServer(pid) {
		return nil // ended already
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	if err := p.Signal(syscall.SIGTERM); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return fmt.Errorf("stopping augmented-brain mcp (process %d): %w", pid, err)
	}
	return nil
}
