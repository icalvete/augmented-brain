package remote

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/icalvete/augmented-brain/internal/state"
)

// restartFake is the machine of 12-nov, made up: a background session (pid 500), one
// session of each remote service (600 under personal's server 60, 700 under work's
// server 70), a Claude Desktop session (800), and this process (pid 5) run from a
// terminal of its own. Each session's augmented brain server is its child (pid+1).
type restartFake struct {
	*fake
	status  map[int]string // pid -> status claude agents gives
	parents map[int]int
	// turnBusy: pid -> how many more looks it answers, after the first list
	turnBusy map[int]int
}

func newRestartFake(t *testing.T) *restartFake {
	t.Helper()
	f, _ := watchFake(t)
	// The service of work too, as if remote.toml named both spaces.
	unit, err := os.ReadFile(filepath.Join(f.sys.UnitDir, "augmented-brain-remote-personal.service"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(f.sys.UnitDir, "augmented-brain-remote-work.service"), unit, 0o600); err != nil {
		t.Fatal(err)
	}
	f.pids["augmented-brain-remote-personal.service"] = 60
	f.pids["augmented-brain-remote-work.service"] = 70
	r := &restartFake{fake: f,
		status:   map[int]string{500: "idle", 600: "idle", 700: "idle", 800: "idle"},
		parents:  map[int]int{500: 400, 400: 300, 600: 60, 700: 70, 800: 790, 5: 4, 4: 3},
		turnBusy: map[int]int{},
	}
	for _, p := range []int{500, 600, 700, 800} {
		r.parents[p+1] = p
	}
	f.sys.Parent = func(pid int) int { return r.parents[pid] }
	f.sys.PID = 5
	f.sys.Sleep = func(time.Duration) {}
	f.agentList = func() []map[string]any {
		st := func(pid int) string {
			if r.turnBusy[pid] > 0 {
				r.turnBusy[pid]--
				return "busy"
			}
			return r.status[pid]
		}
		return []map[string]any{
			{"pid": 500, "id": "bg123456", "kind": "background", "name": "box · Una tarea", "sessionId": "bg123456-aaaa", "status": st(500)},
			{"pid": 600, "kind": "interactive", "name": "me-74", "sessionId": "5febc476-bbbb", "status": st(600)},
			{"pid": 700, "kind": "interactive", "name": "work-b7", "sessionId": "949eabbc-cccc", "status": st(700)},
			{"pid": 800, "kind": "interactive", "name": "Augmented Brain", "sessionId": "a7e1ab3d-dddd", "status": st(800)},
			// Retired: no process, nothing to restart.
			{"id": "old00000", "kind": "background", "name": "dormida", "sessionId": "old00000-eeee"},
		}
	}
	return r
}

func (r *restartFake) claude() string {
	c, err := r.sys.FindClaude()
	if err != nil {
		r.t.Fatal(err)
	}
	return c
}

// acts are the restarts it asked for: respawn <id> and restart <unit>.
func (r *restartFake) acts() []string {
	var out []string
	for _, c := range r.calls {
		f := strings.Fields(c)
		switch {
		case len(f) == 3 && f[1] == "respawn":
			out = append(out, "respawn "+f[2])
		case len(f) == 4 && f[0] == "systemctl" && f[2] == "restart":
			out = append(out, "restart "+f[3])
		}
	}
	return out
}

func outcomes(rs []RestartResult) map[string]Outcome {
	m := map[string]Outcome{}
	for _, r := range rs {
		m[r.Session.Label()] = r.Outcome
	}
	return m
}

// Each session gets its kind: background by Claude's list, remote by its parent being
// a service's main process, any other left as KindOther. A retired one is not listed.
func TestSessionsKinds(t *testing.T) {
	r := newRestartFake(t)
	list, err := r.sys.Sessions(r.claude())
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, se := range list {
		got[se.Label()] = []string{"other", "background", "remote"}[se.Kind] + " " + se.Unit
	}
	want := map[string]string{
		"box · Una tarea": "background ",
		"me-74":           "remote augmented-brain-remote-personal.service",
		"work-b7":         "remote augmented-brain-remote-work.service",
		"Augmented Brain": "other ",
	}
	if len(got) != len(want) {
		t.Fatalf("sessions: %v", got)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s: %q, want %q", k, got[k], v)
		}
	}
}

// An old server leads to its session through its parents; one with no session above
// it (a server started by hand) is left out.
func TestOldSessions(t *testing.T) {
	r := newRestartFake(t)
	r.parents[901] = 900
	old := []state.Server{{PID: 601, Version: "0.17.3-beta"}, {PID: 801, Version: "0.17.0-beta"}, {PID: 901}}
	list, unknown, err := r.sys.OldSessions(r.claude(), old)
	if err != nil || unknown != 1 {
		t.Fatal(unknown, err)
	}
	var names []string
	for _, se := range list {
		if !se.Old {
			t.Errorf("%s not marked old", se.Label())
		}
		names = append(names, se.Label()+" "+se.Version)
	}
	if !slices.Equal(names, []string{"me-74 0.17.3-beta", "Augmented Brain 0.17.0-beta"}) {
		t.Errorf("old sessions: %v", names)
	}
}

// Idle ones restart their way, Claude Desktop is left with its reason, one at a time.
func TestRestartEachItsWay(t *testing.T) {
	r := newRestartFake(t)
	list, _ := r.sys.Sessions(r.claude())
	r.calls = nil
	got := outcomes(r.sys.Restart(r.claude(), list))
	want := map[string]Outcome{"box · Una tarea": Restarted, "me-74": Restarted, "work-b7": Restarted, "Augmented Brain": Unsupported}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s: %v, want %v", k, got[k], v)
		}
	}
	acts := r.acts()
	wantActs := []string{"respawn bg123456", "restart augmented-brain-remote-personal.service", "restart augmented-brain-remote-work.service"}
	if !slices.Equal(acts, wantActs) {
		t.Errorf("acts: %v, want %v", acts, wantActs)
	}
}

// A session that started answering after the list was made is looked at again and
// left; so is a remote service with another of its sessions answering.
func TestRestartLooksAgain(t *testing.T) {
	r := newRestartFake(t)
	list, _ := r.sys.Sessions(r.claude())
	r.status[500] = "busy"
	r.status[700] = "waiting" // a permission question: not idle either
	// A second session of personal's service, answering.
	inner := r.agentList
	r.agentList = func() []map[string]any {
		return append(inner(), map[string]any{"pid": 650, "kind": "interactive", "name": "me-75", "sessionId": "77777777-ffff", "status": "busy"})
	}
	r.parents[650] = 60
	r.calls = nil
	got := outcomes(r.sys.Restart(r.claude(), list))
	if got["box · Una tarea"] != Busy || got["work-b7"] != Busy || got["me-74"] != SharedBusy {
		t.Errorf("outcomes: %v", got)
	}
	if acts := r.acts(); len(acts) != 0 {
		t.Errorf("restarted %v", acts)
	}
}

// Two old sessions of one service restart it once.
func TestRestartServiceOnce(t *testing.T) {
	r := newRestartFake(t)
	inner := r.agentList
	r.agentList = func() []map[string]any {
		return append(inner(), map[string]any{"pid": 650, "kind": "interactive", "name": "me-75", "sessionId": "77777777-ffff", "status": "idle"})
	}
	r.parents[650] = 60
	list, _ := r.sys.Sessions(r.claude())
	var mine []Session
	for _, se := range list {
		if se.Unit == "augmented-brain-remote-personal.service" {
			mine = append(mine, se)
		}
	}
	r.calls = nil
	res := r.sys.Restart(r.claude(), mine)
	if len(res) != 2 || res[0].Outcome != Restarted || res[1].Outcome != Restarted {
		t.Errorf("results: %+v", res)
	}
	if acts := r.acts(); !slices.Equal(acts, []string{"restart augmented-brain-remote-personal.service"}) {
		t.Errorf("acts: %v", acts)
	}
}

// Never the session this process runs in, nor the service it runs under.
func TestRestartNotSelf(t *testing.T) {
	r := newRestartFake(t)
	list, _ := r.sys.Sessions(r.claude())
	// This process is a command the background session runs.
	r.parents[4] = 500
	r.calls = nil
	got := outcomes(r.sys.Restart(r.claude(), list))
	if got["box · Una tarea"] != Self || got["me-74"] != Restarted {
		t.Errorf("outcomes: %v", got)
	}
	// Now one the remote service of work runs, below its session.
	r.parents[4] = 701
	r.calls = nil
	got = outcomes(r.sys.Restart(r.claude(), list))
	if got["work-b7"] != Self || got["box · Una tarea"] != Restarted {
		t.Errorf("outcomes: %v", got)
	}
	if slices.Contains(r.acts(), "restart augmented-brain-remote-work.service") {
		t.Error("restarted the service it runs in")
	}
}

// A respawn that fails says why and leaves the others going.
func TestRestartFailureLeavesOthers(t *testing.T) {
	r := newRestartFake(t)
	r.failOn = "restart augmented-brain-remote-personal.service"
	list, _ := r.sys.Sessions(r.claude())
	res := r.sys.Restart(r.claude(), list)
	got := outcomes(res)
	if got["me-74"] != RestartFailed || got["work-b7"] != Restarted || got["box · Una tarea"] != Restarted {
		t.Errorf("outcomes: %v", got)
	}
	for _, x := range res {
		if x.Outcome == RestartFailed && x.Err == nil {
			t.Error("a failure with no reason")
		}
	}
}

// A service whose group holds what it cannot move out is not restarted.
func TestRestartKeepsOutsideFirst(t *testing.T) {
	r := newRestartFake(t)
	r.sys.GroupProcs = func(g string) []int { return r.groups[strings.TrimPrefix(g, "/app.slice/")] }
	r.groups = map[string][]int{"augmented-brain-remote-personal.service": {60, 600, 9999}}
	r.parents[9999] = 1
	r.sys.Children = func(pid int) []int {
		if pid == 60 {
			return []int{600}
		}
		return nil
	}
	r.stuck = true
	se, err := r.sys.FindSession(r.claude(), r.cfgDir, "me-74")
	if err != nil {
		t.Fatal(err)
	}
	r.calls = nil
	res := r.sys.Restart(r.claude(), []Session{se})
	if res[0].Outcome != RestartFailed || len(r.acts()) != 0 {
		t.Errorf("result %+v, acts %v", res[0], r.acts())
	}
}

// A session is found by its name, a background id, or the start of its session id;
// a key two sessions share is ambiguous; one nobody has is ErrNoSession.
func TestFindSession(t *testing.T) {
	r := newRestartFake(t)
	for key, want := range map[string]string{"work-b7": "work-b7", "bg123456": "box · Una tarea", "5febc476": "me-74", "a7e1ab3d-dddd": "Augmented Brain"} {
		se, err := r.sys.FindSession(r.claude(), r.cfgDir, key)
		if err != nil || se.Label() != want {
			t.Errorf("%s: %v %v", key, se.Label(), err)
		}
	}
	if _, err := r.sys.FindSession(r.claude(), r.cfgDir, "nadie"); !errors.Is(err, ErrNoSession) {
		t.Errorf("unknown: %v", err)
	}
	inner := r.agentList
	r.agentList = func() []map[string]any {
		return append(inner(), map[string]any{"pid": 650, "kind": "interactive", "name": "work-b7", "sessionId": "77777777-ffff", "status": "idle"})
	}
	if _, err := r.sys.FindSession(r.claude(), r.cfgDir, "work-b7"); !errors.Is(err, ErrAmbiguous) {
		t.Errorf("shared name: %v", err)
	}
}

// After parking, a session restarts itself once its answer is over: it waits while it
// answers, then restarts; it gives up after max.
func TestRestartWhenIdle(t *testing.T) {
	r := newRestartFake(t)
	se, _ := r.sys.FindSession(r.claude(), r.cfgDir, "bg123456")
	r.turnBusy[500] = 3
	r.calls = nil
	if res := r.sys.RestartWhenIdle(r.claude(), se, time.Minute); res.Outcome != Restarted {
		t.Fatalf("result %+v", res)
	}
	if acts := r.acts(); !slices.Equal(acts, []string{"respawn bg123456"}) {
		t.Errorf("acts: %v", acts)
	}
	r.status[500] = "busy"
	r.calls = nil
	if res := r.sys.RestartWhenIdle(r.claude(), se, 10*time.Second); res.Outcome != Busy || !errors.Is(res.Err, ErrStillBusy) || len(r.acts()) != 0 {
		t.Errorf("still busy: %+v %v", res, r.acts())
	}
}

// withServers lets the fake see each session's augmented brain server (pid+1) and
// records which ones it stops.
func (r *restartFake) withServers() *[]int {
	var stopped []int
	r.sys.MCPServers = func() []int { return []int{501, 601, 701, 801} }
	r.sys.StopServer = func(pid int) error {
		stopped = append(stopped, pid)
		return nil
	}
	return &stopped
}

// A Claude Desktop session keeps its window: only its own augmented brain server is
// stopped, and only while it is idle (measured on 12-nov: Claude Code starts it again).
func TestRestartDesktopServer(t *testing.T) {
	r := newRestartFake(t)
	stopped := r.withServers()
	se, err := r.sys.FindSession(r.claude(), r.cfgDir, "Augmented Brain")
	if err != nil {
		t.Fatal(err)
	}
	if res := r.sys.Restart(r.claude(), []Session{se}); res[0].Outcome != ServerRestarted || !slices.Equal(*stopped, []int{801}) {
		t.Errorf("result %+v, stopped %v", res[0], *stopped)
	}
	*stopped = nil
	r.status[800] = "busy"
	if res := r.sys.Restart(r.claude(), []Session{se}); res[0].Outcome != Busy || len(*stopped) > 0 {
		t.Errorf("busy: %+v, stopped %v", res[0], *stopped)
	}
	// Run from a command of that session itself: never in the middle of its answer.
	r.status[800] = "idle"
	r.parents[4] = 800
	if res := r.sys.Restart(r.claude(), []Session{se}); res[0].Outcome != Self || len(*stopped) > 0 {
		t.Errorf("self: %+v, stopped %v", res[0], *stopped)
	}
	// A server that cannot be stopped says why.
	r.parents[4] = 3
	r.sys.StopServer = func(int) error { return errors.New("denied") }
	if res := r.sys.Restart(r.claude(), []Session{se}); res[0].Outcome != RestartFailed || res[0].Err == nil {
		t.Errorf("failed stop: %+v", res[0])
	}
}

// With --old, only the old servers of a Desktop session are stopped; where this
// program cannot see the servers (Windows), it is left as before.
func TestRestartOldDesktopServer(t *testing.T) {
	r := newRestartFake(t)
	stopped := r.withServers()
	r.parents[802] = 800 // a second server of the session, already of now
	list, _, err := r.sys.OldSessions(r.claude(), []state.Server{{PID: 801, Version: "0.18.0-beta"}})
	if err != nil || len(list) != 1 || !slices.Equal(list[0].Servers, []int{801}) {
		t.Fatalf("old: %+v %v", list, err)
	}
	if res := r.sys.Restart(r.claude(), list); res[0].Outcome != ServerRestarted || !slices.Equal(*stopped, []int{801}) {
		t.Errorf("result %+v, stopped %v", res[0], *stopped)
	}
	r.sys.MCPServers, r.sys.StopServer = nil, nil
	se, _ := r.sys.FindSession(r.claude(), r.cfgDir, "Augmented Brain")
	if res := r.sys.Restart(r.claude(), []Session{se}); res[0].Outcome != Unsupported {
		t.Errorf("no way to see servers: %+v", res[0])
	}
}

// A background session the user archived (or that ended) keeps its process, idle, with
// state done or stopped: restart --old never brings it back with respawn (0.19.1's
// update did, on 12-nov), it only stops its old server. One waiting for the user
// (blocked) is still restarted.
func TestRestartLeavesArchived(t *testing.T) {
	r := newRestartFake(t)
	stopped := r.withServers()
	list := r.agentList
	r.agentList = func() []map[string]any {
		return append(list(),
			map[string]any{"pid": 900, "id": "arch0001", "kind": "background", "name": "archivada", "sessionId": "arch0001-ffff", "status": "idle", "state": "done"},
			map[string]any{"pid": 910, "id": "stop0001", "kind": "background", "name": "parada", "sessionId": "stop0001-gggg", "status": "idle", "state": "stopped"},
			map[string]any{"pid": 920, "id": "wait0001", "kind": "background", "name": "esperando", "sessionId": "wait0001-hhhh", "status": "idle", "state": "blocked"})
	}
	r.parents[901], r.parents[911], r.parents[921] = 900, 910, 920
	old := []state.Server{{PID: 901, Version: "0.19.0-beta"}, {PID: 911, Version: "0.19.0-beta"}, {PID: 921, Version: "0.19.0-beta"}}
	targets, _, err := r.sys.OldSessions(r.claude(), old)
	if err != nil || len(targets) != 3 {
		t.Fatalf("old: %+v %v", targets, err)
	}
	r.calls = nil
	got := outcomes(r.sys.Restart(r.claude(), targets))
	if got["archivada"] != Ended || got["parada"] != Ended || got["esperando"] != Restarted {
		t.Errorf("outcomes: %v", got)
	}
	if acts := r.acts(); !slices.Equal(acts, []string{"respawn wait0001"}) {
		t.Errorf("acts: %v, want only the blocked one respawned", acts)
	}
	if !slices.Equal(*stopped, []int{901, 911}) {
		t.Errorf("stopped servers: %v", *stopped)
	}
}
