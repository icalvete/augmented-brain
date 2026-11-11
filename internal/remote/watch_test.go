package remote

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// watchFake is a fake with both services installed and running, a pid each, and the
// watch's own pieces: who has children, and Claude's folder with a pointer per server.
func watchFake(t *testing.T) (*fake, map[int][]int) {
	t.Helper()
	f := newFake(t)
	f.sys.Self = filepath.Join(f.root, ".local", "bin", "augmented-brain")
	f.sys.ClaudeDir = filepath.Join(f.root, ".claude")
	kids := map[int][]int{}
	f.sys.Children = func(pid int) []int { return kids[pid] }
	c, set := f.load()
	if _, err := f.sys.Install(c, set); err != nil {
		t.Fatal(err)
	}
	f.pids = map[string]int{}
	for _, u := range []string{"augmented-brain-remote-personal.service", "augmented-brain-remote-work.service"} {
		f.active[u] = "active"
	}
	return f, kids
}

func (f *fake) pointer(folder string, pid int) string {
	f.t.Helper()
	dir := filepath.Join(f.sys.ClaudeDir, "projects", folder)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		f.t.Fatal(err)
	}
	body, _ := json.Marshal(map[string]any{"sessionId": "session_x", "environmentId": "env_x", "pid": pid})
	path := filepath.Join(dir, pointerFile)
	if err := os.WriteFile(path, body, 0o600); err != nil {
		f.t.Fatal(err)
	}
	return path
}

func (f *fake) watch() (string, []string) {
	f.t.Helper()
	f.calls = nil
	c, _ := f.load()
	var out bytes.Buffer
	if err := f.sys.Watch(c, &out); err != nil {
		f.t.Fatal(err)
	}
	var acts []string
	for _, call := range f.calls {
		for _, verb := range []string{" restart ", " stop ", " start "} {
			if strings.Contains(call, verb) {
				acts = append(acts, strings.TrimSpace(verb)+" "+call[strings.LastIndex(call, " ")+1:])
			}
		}
	}
	return out.String(), acts
}

// Install writes the watch and starts its timer; uninstall takes them away first; a
// service installed before the watch is out of date.
func TestWatchInstalled(t *testing.T) {
	f, _ := watchFake(t)
	svc, err := os.ReadFile(filepath.Join(f.sys.UnitDir, WatchName+".service"))
	if err != nil {
		t.Fatal(err)
	}
	// Written as systemd reads them: on Windows (where the tests also run) the paths have
	// backslashes, so they go in quotes.
	if !strings.Contains(string(svc), "ExecStart="+unitArg(f.sys.Self, true)+" remote watch\n") || !strings.Contains(string(svc), "Environment="+unitEnv("AB_CONFIG_DIR="+f.cfgDir)+"\n") {
		t.Errorf("watch service:\n%s", svc)
	}
	if !strings.Contains(WatchTimerText(), "OnUnitActiveSec="+WatchEvery+"\n") {
		t.Errorf("timer:\n%s", WatchTimerText())
	}
	c, set := f.load()
	if _, err := f.sys.Install(c, set); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(f.calls, "systemctl --user enable --now "+WatchName+".timer") {
		t.Errorf("timer not started: %v", f.calls)
	}
	st, err := f.sys.Status(c, set)
	if err != nil {
		t.Fatal(err)
	}
	for _, x := range st.Spaces {
		if x.Outdated {
			t.Errorf("%s outdated with the watch installed", x.Space)
		}
	}
	if err := os.Remove(filepath.Join(f.sys.UnitDir, WatchName+".timer")); err != nil {
		t.Fatal(err)
	}
	if st, _ = f.sys.Status(c, set); !st.Spaces[0].Outdated {
		t.Error("a service without the watch is not out of date")
	}
	if _, err := f.sys.Install(c, set); err != nil {
		t.Fatal(err)
	}
	f.calls = nil
	removed, err := f.sys.Uninstall(c)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(removed, WatchName+".timer") {
		t.Errorf("removed %v", removed)
	}
	if f.calls[0] != "systemctl --user disable --now "+WatchName+".timer" {
		t.Errorf("the watch is not stopped first: %v", f.calls)
	}
	for _, n := range []string{".service", ".timer"} {
		if _, err := os.Stat(filepath.Join(f.sys.UnitDir, WatchName+n)); !os.IsNotExist(err) {
			t.Errorf("%s left behind", n)
		}
	}
}

// The two steps: a server seen twice with no child is restarted; if the new one too has
// none twice, its pointer is set aside and it is restarted again. One with a child, or a
// server that changed, is left alone.
func TestWatchTwoSteps(t *testing.T) {
	f, kids := watchFake(t)
	const unit = "augmented-brain-remote-personal.service"
	other := "augmented-brain-remote-work.service"
	f.pids[unit], f.pids[other] = 100, 900
	kids[100], kids[900] = []int{101}, []int{901}
	if out, acts := f.watch(); out != "" || acts != nil {
		t.Fatalf("healthy: %q %v", out, acts)
	}

	// Archived: the child is gone. The first look only notes it.
	kids[100] = nil
	if _, acts := f.watch(); acts != nil {
		t.Fatalf("first look acted: %v", acts)
	}
	out, acts := f.watch()
	if !slices.Equal(acts, []string{"restart " + unit}) || !strings.Contains(out, unit) {
		t.Fatalf("second look: %q %v", out, acts)
	}

	// The new server adopts the archived session again and has no child either.
	f.pids[unit] = 200
	pointer := f.pointer("-p", 200)
	f.pointer("-w", 900) // the other server's, never touched
	if _, acts := f.watch(); acts != nil {
		t.Fatalf("new server, first look: %v", acts)
	}
	out, acts = f.watch()
	if !slices.Equal(acts, []string{"stop " + unit, "start " + unit}) || !strings.Contains(out, PointerAside) {
		t.Fatalf("set aside: %q %v", out, acts)
	}
	if _, err := os.Stat(pointer); !os.IsNotExist(err) {
		t.Error("the pointer is still there")
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(pointer), PointerAside)); err != nil {
		t.Error("the pointer was not set aside:", err)
	}
	if _, err := os.Stat(filepath.Join(f.sys.ClaudeDir, "projects", "-w", pointerFile)); err != nil {
		t.Error("the other server's pointer moved:", err)
	}

	// "Already served": servers that come and go are never seen twice.
	for pid := 300; pid < 303; pid++ {
		f.pids[unit] = pid
		if _, acts := f.watch(); acts != nil {
			t.Fatalf("pid %d: %v", pid, acts)
		}
	}

	// Back with a session: forgotten, so a later loss starts with a plain restart.
	f.pids[unit] = 400
	kids[400] = []int{401}
	f.watch()
	kids[400] = nil
	f.watch()
	if _, acts := f.watch(); !slices.Equal(acts, []string{"restart " + unit}) {
		t.Fatalf("after recovering: %v", acts)
	}
}

// A service that is not running is systemd's business: the watch does nothing.
func TestWatchLeavesStoppedServices(t *testing.T) {
	f, _ := watchFake(t)
	for u := range f.active {
		f.active[u] = "failed"
	}
	f.watch()
	if _, acts := f.watch(); acts != nil {
		t.Fatalf("acted on stopped services: %v", acts)
	}
}

func TestWatchOnlyOnLinux(t *testing.T) {
	f, _ := watchFake(t)
	f.sys.GOOS = "darwin"
	c, _ := f.load()
	if err := f.sys.Watch(c, &bytes.Buffer{}); err != ErrUnsupported {
		t.Fatalf("got %v", err)
	}
}
