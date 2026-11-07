//go:build unix || windows

package panel

import (
	"context"
	"github.com/icalvete/augmented-brain/internal/testenv"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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
	UnpublishLeftover = func() error { return nil } // never the real Tailscale
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

// A server that does not answer, but whose process is still this program, is ended
// by its process id; a process of another program is left alone.
func TestStopEndsAServerThatHangs(t *testing.T) {
	_, cfg := fixture(t, "en")
	hang := exec.Command(os.Args[0], "-test.run=^$")
	hang.Env = append(os.Environ(), "AB_PANEL_TEST_HANG=1")
	if err := hang.Start(); err != nil {
		t.Fatal(err)
	}
	go func() { _ = hang.Wait() }()
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close() // nothing answers there
	if err := writeInfo(cfg, Info{PID: hang.Process.Pid, Port: port, Token: "1234"}); err != nil {
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
	if err := writeInfo(cfg, Info{PID: other.Process.Pid, Port: port, Token: "1234"}); err != nil {
		t.Fatal(err)
	}
	if stopped, _ := Stop(cfg); stopped || !pidAlive(other.Process.Pid) {
		t.Error("Stop ended a process of another program")
	}
}
