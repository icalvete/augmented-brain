package remote

import (
	"slices"
	"strings"
	"testing"
)

// keepFake is a watch fake whose personal service holds its server (100, with a session
// 101 and that session's tool 102) and what a session left behind: Claude's daemon (500)
// and its background session (501).
func keepFake(t *testing.T) (*fake, map[int][]int) {
	t.Helper()
	f, kids := watchFake(t)
	const unit = "augmented-brain-remote-personal.service"
	f.pids[unit], f.pids["augmented-brain-remote-work.service"] = 100, 900
	kids[100], kids[101], kids[500] = []int{101}, []int{102}, []int{501}
	kids[900] = []int{901}
	f.groups = map[string][]int{
		"/app.slice/" + unit:                             {100, 101, 102, 500, 501},
		"/app.slice/augmented-brain-remote-work.service": {900, 901},
	}
	f.sys.GroupProcs = func(group string) []int { return slices.Clone(f.groups[group]) }
	return f, kids
}

func busctlCalls(calls []string) []string {
	var out []string
	for _, c := range calls {
		if strings.HasPrefix(c, "busctl ") {
			out = append(out, c)
		}
	}
	return out
}

// The watch moves out of a service what does not come from its server, on every look,
// into one scope named after the first process; the server and its children stay.
func TestWatchKeepsTheDaemonOutside(t *testing.T) {
	f, _ := keepFake(t)
	out, acts := f.watch()
	if acts != nil {
		t.Errorf("acted: %v", acts)
	}
	if !strings.Contains(out, "augmented-brain-remote-personal.service") || !strings.Contains(out, " 2 ") {
		t.Errorf("out: %q", out)
	}
	calls := busctlCalls(f.calls)
	if len(calls) != 1 || !strings.Contains(calls[0], " StartTransientUnit ") ||
		!strings.Contains(calls[0], " "+keptPrefix+"500.scope fail 2 ") || !strings.HasSuffix(calls[0], " PIDs au 2 500 501 0") {
		t.Fatalf("busctl: %q", calls)
	}
	if g := f.groups["/app.slice/augmented-brain-remote-personal.service"]; !slices.Equal(g, []int{100, 101, 102}) {
		t.Errorf("left in the service: %v", g)
	}
	// Nothing left: the next look says nothing and moves nothing.
	if out, _ := f.watch(); out != "" || busctlCalls(f.calls) != nil {
		t.Errorf("second look: %q %v", out, f.calls)
	}
}

// When they cannot be moved, the watch says so and does not restart the service, even
// with its session gone; once they move, it goes on as before.
func TestWatchDoesNotRestartOverTheDaemon(t *testing.T) {
	f, kids := keepFake(t)
	f.stuck = true
	kids[100] = nil // the server lost its session
	for range 3 {
		out, acts := f.watch()
		if acts != nil {
			t.Fatalf("restarted with the daemon inside: %v", acts)
		}
		if !strings.Contains(out, "augmented-brain-remote-personal.service") || !strings.Contains(out, "Access denied") {
			t.Errorf("out: %q", out)
		}
	}
	f.stuck = false
	if _, acts := f.watch(); !slices.Equal(acts, []string{"restart augmented-brain-remote-personal.service"}) {
		t.Errorf("acts: %v", acts)
	}
}

// install and uninstall move them out before they restart or stop a service, and refuse
// when they cannot.
func TestInstallAndUninstallKeepTheDaemon(t *testing.T) {
	f, _ := keepFake(t)
	c, set := f.load()
	f.sys.LookPath = func(string) (string, error) { return "/usr/local/bin/claude", nil } // the unit changes
	f.stuck = true
	f.calls = nil
	if _, err := f.sys.Install(c, set); err == nil || !strings.Contains(err.Error(), "augmented-brain-remote-personal.service") {
		t.Fatalf("install over the daemon: %v", err)
	}
	for _, call := range f.calls {
		if strings.Contains(call, " restart ") {
			t.Errorf("restarted: %q", call)
		}
	}
	f.stuck = false
	f.calls = nil
	if _, err := f.sys.Install(c, set); err != nil {
		t.Fatal(err)
	}
	b := slices.IndexFunc(f.calls, func(s string) bool { return strings.HasPrefix(s, "busctl ") })
	r := slices.Index(f.calls, "systemctl --user restart augmented-brain-remote-personal.service")
	if b < 0 || r < b {
		t.Errorf("not moved before the restart: %v", f.calls)
	}

	f.groups["/app.slice/augmented-brain-remote-personal.service"] = []int{100, 101, 700}
	f.stuck = true
	if _, err := f.sys.Uninstall(c); err == nil {
		t.Fatal("uninstall over the daemon")
	}
	if !f.sys.watchInstalled() {
		t.Error("a refused uninstall took the watch away")
	}
	f.stuck = false
	f.calls = nil
	if _, err := f.sys.Uninstall(c); err != nil {
		t.Fatal(err)
	}
	b = slices.IndexFunc(f.calls, func(s string) bool { return strings.Contains(s, keptPrefix+"700.scope") })
	d := slices.Index(f.calls, "systemctl --user disable --now augmented-brain-remote-personal.service")
	if b < 0 || d < b {
		t.Errorf("not moved before stopping: %v", f.calls)
	}
}

// With OwnScope, session start runs claude --bg through systemd-run, so the daemon it
// may start is born outside any service.
func TestStartInItsOwnScope(t *testing.T) {
	s := System{OwnScope: true}
	name, args := s.scoped("/bin/claude", "--bg", "x")
	if name != "systemd-run" || strings.Join(args, " ") != "--user --scope --collect --quiet -- /bin/claude --bg x" {
		t.Errorf("%s %v", name, args)
	}
	if name, args := (System{}).scoped("/bin/claude", "--bg"); name != "/bin/claude" || strings.Join(args, " ") != "--bg" {
		t.Errorf("without OwnScope: %s %v", name, args)
	}
}
