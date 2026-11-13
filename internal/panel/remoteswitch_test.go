package panel

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const (
	switchHost  = "box.example.ts.net:8443"
	switchOwner = "lucia@example.com"
)

// `panel remote` with a panel already open (7-nov: it stopped the user's panel and
// changed its key). The running panel switches in place: the same key, so the pages
// open in the browser go on working, and only the command can ask for it.
func TestRemoteSwitchInPlace(t *testing.T) {
	s, cfg := fixture(t, "en")
	var asked atomic.Int32
	s.publish = func() (*Remote, error) {
		asked.Add(1)
		return &Remote{Host: switchHost, Owner: switchOwner}, nil
	}
	tasks := s.base() + "/tasks"
	before := s.base()
	if w := s.do(t, req{path: tasks, host: switchHost, owner: switchOwner}); w.Code != http.StatusForbidden {
		t.Fatalf("before the switch, through the network: %d", w.Code)
	}
	// A page of the panel cannot ask for it, even with its csrf token.
	if w := s.post(t, "/remote", url.Values{"s": {"work"}}); w.Code != http.StatusForbidden {
		t.Fatalf("from a page: %d", w.Code)
	}
	// Nor anything without the token.
	if w := s.do(t, req{method: "POST", path: "/remote"}); w.Code != http.StatusForbidden {
		t.Fatalf("without the token: %d", w.Code)
	}
	if asked.Load() != 0 || s.remote.Load() != nil {
		t.Fatal("it switched without the command")
	}
	// The command: token, on 127.0.0.1.
	if w := s.do(t, req{method: "POST", path: "/remote", bearer: s.token, origin: "http://" + host}); w.Code != http.StatusOK {
		t.Fatalf("the command: %d %s", w.Code, w.Body)
	}
	if s.base() != before {
		t.Fatal("the key changed")
	}
	if w := s.do(t, req{path: tasks, host: switchHost, owner: switchOwner}); w.Code != http.StatusOK {
		t.Fatalf("through the network after the switch: %d", w.Code)
	}
	if w := s.get(t, "/tasks"); w.Code != http.StatusOK {
		t.Fatalf("the page open on 127.0.0.1: %d", w.Code)
	}
	if i, ok := ReadInfo(cfg); !ok || i.Remote == nil || i.Remote.Host != switchHost || i.Token != s.token {
		t.Fatalf("info file: %+v", i)
	}
	// Asking again changes nothing.
	if w := s.do(t, req{method: "POST", path: "/remote", bearer: s.token, origin: "http://" + host}); w.Code != http.StatusOK || asked.Load() != 1 {
		t.Fatalf("again: %d, asked %d", w.Code, asked.Load())
	}
	// Through the network, the token is no use: the command talks to 127.0.0.1 only.
	for _, p := range []string{"/local", "/remote"} {
		if w := s.do(t, req{method: "POST", path: p, host: switchHost, owner: switchOwner, bearer: s.token}); w.Code != http.StatusForbidden {
			t.Fatalf("%s through the network: %d", p, w.Code)
		}
	}
	// Undo: back to local only.
	if w := s.do(t, req{method: "POST", path: "/local", bearer: s.token, origin: "http://" + host}); w.Code != http.StatusOK {
		t.Fatalf("local: %d", w.Code)
	}
	if w := s.do(t, req{path: tasks, host: switchHost, owner: switchOwner}); w.Code != http.StatusForbidden {
		t.Fatalf("after going local, through the network: %d", w.Code)
	}
	if i, _ := ReadInfo(cfg); i.Remote != nil {
		t.Fatalf("info file still remote: %+v", i)
	}
}

func TestRemoteSwitchRefused(t *testing.T) {
	s, cfg := fixture(t, "en")
	// A panel started without a way to publish (an earlier program's command).
	if w := s.do(t, req{method: "POST", path: "/remote", bearer: s.token, origin: "http://" + host}); w.Code != http.StatusNotImplemented {
		t.Fatalf("no publish: %d", w.Code)
	}
	// Tailscale not ready: it stays local.
	s.publish = func() (*Remote, error) { return nil, errors.New("tailscale is stopped") }
	if w := s.do(t, req{method: "POST", path: "/remote", bearer: s.token, origin: "http://" + host}); w.Code != http.StatusServiceUnavailable {
		t.Fatalf("tailscale stopped: %d", w.Code)
	}
	if s.remote.Load() != nil {
		t.Fatal("it went remote without Tailscale")
	}
	if i, ok := ReadInfo(cfg); ok && i.Remote != nil {
		t.Fatalf("info file: %+v", i)
	}
}

// Requests keep coming while the panel switches: no race (go test -race).
func TestRemoteSwitchUnderLoad(t *testing.T) {
	s, _ := fixture(t, "en")
	s.publish = func() (*Remote, error) { return &Remote{Host: switchHost, Owner: switchOwner}, nil }
	var wg sync.WaitGroup
	stop := make(chan struct{})
	for k := 0; k < 4; k++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					s.get(t, "/tasks")
					s.do(t, req{path: s.base() + "/tasks", host: switchHost, owner: switchOwner})
				}
			}
		}()
	}
	for k := 0; k < 20; k++ {
		s.do(t, req{method: "POST", path: "/remote", bearer: s.token, origin: "http://" + host})
		s.do(t, req{method: "POST", path: "/local", bearer: s.token, origin: "http://" + host})
	}
	close(stop)
	wg.Wait()
}

// The whole way, with a real server: GoRemote switches the running panel (same
// process, same key), GoLocal takes it back, and a panel that went remote while
// running takes its address back when it stops.
func TestGoRemoteKeepsTheRunningPanel(t *testing.T) {
	_, cfg := fixture(t, "en")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var taken atomic.Int32
	publish := func() (*Remote, error) {
		return &Remote{Host: switchHost, Owner: switchOwner, Unpublish: func() error { taken.Add(1); return nil }}, nil
	}
	errc := make(chan error, 1)
	go func() { errc <- ServeWith(ctx, cfg, time.Hour, nil, publish) }()
	first := waitInfo(t, cfg)
	info, how, err := GoRemote(cfg, "never-started")
	if err != nil || how != Switched {
		t.Fatalf("GoRemote: %q, %v", how, err)
	}
	if info.PID != first.PID || info.Token != first.Token || info.Remote == nil {
		t.Fatalf("before %+v, after %+v", first, info)
	}
	if _, how, _ := GoRemote(cfg, "never-started"); how != WasRemote {
		t.Fatalf("second time: %q", how)
	}
	if err := GoLocal(cfg); err != nil {
		t.Fatal(err)
	}
	if i, _ := ReadInfo(cfg); i.Remote != nil || i.Token != first.Token {
		t.Fatalf("after GoLocal: %+v", i)
	}
	if _, how, err := GoRemote(cfg, "never-started"); err != nil || how != Switched {
		t.Fatalf("again: %q %v", how, err)
	}
	cancel()
	select {
	case err := <-errc:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("did not stop")
	}
	if taken.Load() != 1 {
		t.Fatalf("unpublished %d times", taken.Load())
	}
	if _, err := os.Stat(filepath.Join(cfg, InfoFile)); !os.IsNotExist(err) {
		t.Fatal("info file left behind")
	}
}

// An earlier panel cannot switch: GoRemote's ask fails, so it falls back to starting
// again (not run here: it would start a real process).
func TestAskAnEarlierPanel(t *testing.T) {
	_, cfg := fixture(t, "en")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = ServeWith(ctx, cfg, time.Hour, nil, nil) }()
	i := waitInfo(t, cfg)
	if err := ask(i, "/remote"); err == nil || !strings.Contains(err.Error(), "501") {
		t.Fatalf("ask: %v", err)
	}
	bad := i
	bad.Token = strings.Repeat("0", len(i.Token))
	if err := ask(bad, "/local"); err == nil || !strings.Contains(err.Error(), "403") {
		t.Fatalf("wrong token: %v", err)
	}
}

func waitInfo(t *testing.T, cfg string) Info {
	t.Helper()
	for k := 0; k < 200; k++ {
		if i, ok := ReadInfo(cfg); ok && Alive(i) {
			return i
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("the panel did not start")
	return Info{}
}

// Tailscale slow to answer (P-09): `tailscale status` takes 3 seconds, more than the
// command's usual 2. The panel still switches in place, same process and key; and when
// Tailscale refuses, the panel is left as it was, not started again.
func TestGoRemoteWaitsForASlowTailscale(t *testing.T) {
	_, cfg := fixture(t, "en")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var fail atomic.Bool
	publish := func() (*Remote, error) {
		time.Sleep(3 * time.Second)
		if fail.Load() {
			return nil, errors.New("tailscale is stopped")
		}
		return &Remote{Host: switchHost, Owner: switchOwner}, nil
	}
	errc := make(chan error, 1)
	go func() { errc <- ServeWith(ctx, cfg, time.Hour, nil, publish) }()
	defer func() { cancel(); <-errc }()
	first := waitInfo(t, cfg)
	fail.Store(true)
	if _, how, err := GoRemote(cfg, "never-started"); err == nil || how != "" || !strings.Contains(err.Error(), "503") {
		t.Fatalf("Tailscale stopped: %q %v", how, err)
	}
	if i, ok := ReadInfo(cfg); !ok || i.PID != first.PID || i.Token != first.Token || i.Remote != nil {
		t.Fatalf("the panel changed: %+v", i)
	}
	fail.Store(false)
	info, how, err := GoRemote(cfg, "never-started")
	if err != nil || how != Switched || info.PID != first.PID || info.Token != first.Token {
		t.Fatalf("slow Tailscale: %q %v %+v", how, err, info)
	}
}
