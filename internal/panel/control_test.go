//go:build unix || windows

package panel

import (
	"context"
	"github.com/icalvete/augmented-brain/internal/tailnet"
	"github.com/icalvete/augmented-brain/internal/testenv"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestMain lets the test program stand in for augmented-brain: run as
// `<test> panel serve` with AB_PANEL_TEST_CFG, it is the panel's server; with
// AB_PANEL_TEST_HANG=1, a process of this program that never answers.
func TestMain(m *testing.M) {
	testenv.CleanGit()
	realUnpublish = UnpublishLeftover
	UnpublishLeftover = func(string) error { return nil } // never the real Tailscale
	if len(os.Args) == 3 && os.Args[1] == "panel" && os.Args[2] == "serve" && os.Getenv("AB_PANEL_TEST_CFG") != "" {
		if err := Serve(context.Background(), os.Getenv("AB_PANEL_TEST_CFG"), time.Minute); err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}
	if os.Getenv("AB_PANEL_TEST_HANG") == "1" {
		time.Sleep(time.Minute)
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// realUnpublish is UnpublishLeftover as the program has it, for the test that runs it
// against a pretend tailscale.
var realUnpublish func(string) error

// Starting the panel takes back only what this configuration recorded (P-03): with
// nothing recorded, tailscale is not even run, so the user's own `tailscale serve` on
// port 8443 stays; with a record, only a proxy to exactly that port is taken back.
func TestUnpublishLeftoverOnlyOurs(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("a pretend tailscale written as a shell script")
	}
	bin := t.TempDir()
	logf := filepath.Join(bin, "calls.log")
	served := filepath.Join(bin, "served.json")
	script := "#!/bin/sh\necho \"$*\" >> " + logf + "\ncase \"$*\" in\n  \"serve status --json\") cat " + served + " ;;\nesac\n"
	if err := os.WriteFile(filepath.Join(bin, "tailscale"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH")) // ours first
	user := `{"TCP":{"8443":{"HTTPS":true}},"Web":{"box.example.ts.net:8443":{"Handlers":{"/":{"Proxy":"http://127.0.0.1:3000"}}}}}`
	if err := os.WriteFile(served, []byte(user), 0o600); err != nil {
		t.Fatal(err)
	}
	calls := func() string { data, _ := os.ReadFile(logf); return string(data) }
	cfg := t.TempDir()
	if err := realUnpublish(cfg); err != nil || calls() != "" {
		t.Fatalf("nothing recorded: %v, ran %q", err, calls())
	}
	if err := tailnet.Record(cfg, 4321); err != nil {
		t.Fatal(err)
	}
	if err := realUnpublish(cfg); err != nil || strings.Contains(calls(), "off") {
		t.Fatalf("the user's proxy: %v, ran %q", err, calls())
	}
	ours := strings.Replace(user, "3000", "4321", 1)
	if err := os.WriteFile(served, []byte(ours), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := tailnet.Record(cfg, 4321); err != nil {
		t.Fatal(err)
	}
	if err := realUnpublish(cfg); err != nil || !strings.Contains(calls(), "serve --https=8443 off") {
		t.Fatalf("ours: %v, ran %q", err, calls())
	}
	if tailnet.Published(cfg) != 0 {
		t.Error("the record outlived what it named")
	}
}

// Three `panel` at once start one server, and all three get its address.
func TestConcurrentStarts(t *testing.T) {
	_, cfg := fixture(t, "en")
	t.Setenv("AB_PANEL_TEST_CFG", cfg)
	var wg sync.WaitGroup
	infos := make([]Info, 3)
	started := make([]bool, 3)
	errs := make([]error, 3)
	for k := range 3 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			infos[k], started[k], errs[k] = Start(cfg, os.Args[0])
		}()
	}
	wg.Wait()
	t.Cleanup(func() { _, _ = Stop(cfg) })
	n := 0
	for k := range 3 {
		if errs[k] != nil {
			t.Fatal(errs[k])
		}
		if started[k] {
			n++
		}
		if infos[k] != infos[0] || infos[k].URL() != infos[0].URL() {
			t.Errorf("start %d got another panel: %+v and %+v", k, infos[k], infos[0])
		}
	}
	if n != 1 {
		t.Fatalf("%d servers started", n)
	}
	pid := infos[0].PID
	if !pidAlive(pid) || !Alive(infos[0]) {
		t.Fatal("the server is not running")
	}
	// The server does not run in the configuration folder (uninstall removes it).
	if wd, err := os.Readlink("/proc/" + strconv.Itoa(pid) + "/cwd"); err == nil && wd == cfg {
		t.Errorf("the server runs in %s", wd)
	}
	// The URL opens the panel.
	resp, err := http.Get(infos[0].URL())
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s: %v %v", infos[0].URL(), resp, err)
	}
	resp.Body.Close()
	// Stop waits until the process has ended.
	if stopped, err := Stop(cfg); !stopped || err != nil {
		t.Fatalf("Stop: %v %v", stopped, err)
	}
	if pidAlive(pid) {
		t.Error("Stop returned before the server ended")
	}
	if _, ok := ReadInfo(cfg); ok {
		t.Error("the info file outlived the server")
	}
	if matches, _ := filepath.Glob(filepath.Join(cfg, "panel-*.tmp")); len(matches) > 0 {
		t.Errorf("temporary files left: %v", matches)
	}
}

// Whatever took the port of a panel that ended cannot pass for it: it answers the ping
// without the proof. Then the info file is stale: Stop and Start forget it.
func TestStaleInfoIsNotThePanel(t *testing.T) {
	_, cfg := fixture(t, "en")
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	var sawToken atomic.Bool
	fake := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			sawToken.Store(true)
		}
		w.Header().Set("X-Augmented-Brain", "panel")
		_, _ = w.Write([]byte("ok\n"))
	})}
	go func() { _ = fake.Serve(ln) }()
	defer fake.Close()
	stale := Info{PID: 999999999, Port: ln.Addr().(*net.TCPAddr).Port, Token: "1234", Started: time.Now()}
	if err := writeInfo(cfg, stale); err != nil {
		t.Fatal(err)
	}
	if Alive(stale) {
		t.Fatal("a server without the proof passed for the panel")
	}
	if stopped, err := Stop(cfg); stopped || err != nil {
		t.Errorf("Stop: %v %v", stopped, err)
	}
	if _, ok := ReadInfo(cfg); ok {
		t.Error("the stale info file is still there")
	}
	if sawToken.Load() {
		t.Error("the token was sent to whatever listens on the port")
	}
}

// hangs starts a process of this program that never answers, with args.
func hangs(t *testing.T, args ...string) *exec.Cmd {
	t.Helper()
	hang := exec.Command(os.Args[0], args...)
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "AB_PANEL_TEST_CFG=") {
			hang.Env = append(hang.Env, e)
		}
	}
	hang.Env = append(hang.Env, "AB_PANEL_TEST_HANG=1")
	if err := hang.Start(); err != nil {
		t.Fatal(err)
	}
	go func() { _ = hang.Wait() }()
	t.Cleanup(func() { _ = hang.Process.Kill() })
	return hang
}

// deadPort is a port of 127.0.0.1 where nothing answers.
func deadPort(t *testing.T) int {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

// A stale panel.json whose process id another process of this program now has (an MCP
// server, a backup, a hook: P-05) does not get that process ended: it is not `panel
// serve`, or it started after the panel did.
func TestStopLeavesAReusedPID(t *testing.T) {
	_, cfg := fixture(t, "en")
	port := deadPort(t)
	cases := []struct {
		name string
		args []string
	}{
		{"a panel server that started later", []string{"panel", "serve"}},
	}
	if runtime.GOOS != "windows" { // Windows does not tell the command line
		cases = append(cases, struct {
			name string
			args []string
		}{"another command of this program", []string{"mcp", "-test.run=^$"}})
	}
	for _, c := range cases {
		before := time.Now().Add(-time.Hour)
		started := before
		if c.args[0] == "mcp" {
			started = time.Now().Add(time.Hour) // the time would let it through
		}
		hang := hangs(t, c.args...)
		time.Sleep(50 * time.Millisecond)
		if err := writeInfo(cfg, Info{PID: hang.Process.Pid, Port: port, Token: "1234", Started: started}); err != nil {
			t.Fatal(err)
		}
		if stopped, err := Stop(cfg); stopped || err != nil {
			t.Errorf("%s: Stop said %v %v", c.name, stopped, err)
		}
		if !pidAlive(hang.Process.Pid) {
			t.Errorf("%s: ended", c.name)
		}
		if _, ok := ReadInfo(cfg); ok {
			t.Errorf("%s: the stale info file is still there", c.name)
		}
	}
}

// A server that does not answer, but whose process is still this program's `panel
// serve`, started before the panel's file said, is ended by its process id; a process
// of another program is left alone.
func TestStopEndsAServerThatHangs(t *testing.T) {
	_, cfg := fixture(t, "en")
	hang := hangs(t, "panel", "serve")
	time.Sleep(50 * time.Millisecond)
	port := deadPort(t)
	if err := writeInfo(cfg, Info{PID: hang.Process.Pid, Port: port, Token: "1234", Started: time.Now()}); err != nil {
		t.Fatal(err)
	}
	if stopped, err := Stop(cfg); !stopped || err != nil {
		t.Fatalf("Stop: %v %v", stopped, err)
	}
	if pidAlive(hang.Process.Pid) {
		_ = hang.Process.Kill()
		t.Fatal("the hung server is still running")
	}

	// Another program: not touched.
	other := exec.Command("sleep", "30")
	if err := other.Start(); err != nil {
		t.Skip("no sleep command")
	}
	go func() { _ = other.Wait() }()
	defer other.Process.Kill()
	if err := writeInfo(cfg, Info{PID: other.Process.Pid, Port: port, Token: "1234", Started: time.Now()}); err != nil {
		t.Fatal(err)
	}
	if stopped, _ := Stop(cfg); stopped || !pidAlive(other.Process.Pid) {
		t.Error("Stop ended a process of another program")
	}
}
