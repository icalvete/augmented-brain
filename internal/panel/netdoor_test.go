package panel

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Through the network's socket, no Host makes a request local (P-02): a guest of the
// network who forges Host: 127.0.0.1 gets 403, the ping included, and the owner's
// local-only actions are refused whatever Host they carry.
func TestNetSocketIgnoresHost(t *testing.T) {
	s, _ := fixture(t, "en")
	const rhost, owner, guest = "box.example.ts.net:8443", "lucia@example.com", "guest@example.com"
	s.remote.Store(&Remote{Host: rhost, Owner: owner})
	tasks := s.netBase() + "/tasks"
	ping := "/ping?c=" + strings.Repeat("ab", 32)
	for i, r := range []req{
		{net: true, path: tasks, host: host, owner: guest},
		{net: true, path: tasks, host: host},
		{net: true, path: tasks, host: "localhost:45678", owner: guest},
		{net: true, path: ping, host: host},
		{net: true, path: ping, host: host, owner: owner},
		{net: true, path: ping, host: rhost, owner: owner},
		{net: true, path: "/tasks", host: host, bearer: s.token},
		{net: true, method: "POST", path: "/remote", host: host, bearer: s.token, origin: "http://" + host},
		{net: true, method: "POST", path: "/ticket", host: host, bearer: s.token, origin: "http://" + host, form: url.Values{"page": {""}}},
		{net: true, method: "POST", path: "/stop", host: host, bearer: s.token, origin: "http://" + host},
	} {
		if w := s.do(t, r); w.Code != http.StatusForbidden {
			t.Errorf("case %d (%s, Host %s): %d", i, r.path, r.host, w.Code)
		}
	}
	if w := s.do(t, req{net: true, path: tasks, host: rhost, owner: owner}); w.Code != http.StatusOK {
		t.Fatalf("the owner through the network: %d", w.Code)
	}
	var started []string
	s.backupNow = func(space string) error { started = append(started, space); return nil }
	// With the network's own Host they reach the page, which refuses them (400); with
	// 127.0.0.1's, the door itself does (403). Never a copy, never a restore.
	for h, want := range map[string]int{rhost: http.StatusBadRequest, host: http.StatusForbidden} {
		for _, p := range []string{"/backup/setup", "/backup/back", "/backup/run", "/versions/restore"} {
			form := url.Values{"s": {"work"}, "csrf": {s.netCSRF}, "url": {"https://github.com/someone/else.git"}, "confirm": {"yes"}, "id": {"x"}}
			if w := s.do(t, req{net: true, method: "POST", path: s.netBase() + p, host: h, owner: owner, origin: "https://" + h, form: form}); w.Code != want {
				t.Errorf("%s through the network, Host %s: %d, want %d", p, h, w.Code, want)
			}
		}
	}
	if len(started) != 0 {
		t.Errorf("a copy started from the network: %v", started)
	}
}

// On the local socket, a request a proxy passed on is refused: it did not come straight
// from this machine's browser.
func TestLocalSocketRefusesProxied(t *testing.T) {
	s, _ := fixture(t, "en")
	s.remote.Store(&Remote{Host: "box.example.ts.net:8443", Owner: "lucia@example.com"})
	for _, hdr := range []string{"X-Forwarded-For", "X-Forwarded-Host", "X-Forwarded-Proto", "Forwarded", OwnerHeader, "Tailscale-User-Name"} {
		q := newReq(s, "/tasks")
		q.Header.Set(hdr, "x")
		w := record(s.ServeHTTP, q)
		if w.Code != http.StatusForbidden {
			t.Errorf("%s on the local socket: %d", hdr, w.Code)
		}
	}
	if w := s.get(t, "/tasks"); w.Code != http.StatusOK {
		t.Errorf("plain local: %d", w.Code)
	}
}

// A real panel: the network's socket is its own port, open only while remote, and
// written to the info file; with Host: 127.0.0.1 it is still 403.
func TestNetSocketIsItsOwn(t *testing.T) {
	_, cfg := fixture(t, "en")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	const rhost, owner = "box.example.ts.net:8443", "lucia@example.com"
	publish := func() (*Remote, error) { return &Remote{Host: rhost, Owner: owner}, nil }
	errc := make(chan error, 1)
	go func() { errc <- ServeWith(ctx, cfg, time.Hour, nil, publish) }()
	defer func() { cancel(); <-errc }()
	first := waitInfo(t, cfg)
	if first.RemotePort != 0 {
		t.Fatalf("local panel with a network socket: %+v", first)
	}
	info, how, err := GoRemote(cfg, "never-started")
	if err != nil || how != Switched || info.RemotePort == 0 || info.RemotePort == info.Port {
		t.Fatalf("GoRemote: %q %v %+v", how, err, info)
	}
	// Each socket has its session: the one of the other socket is refused there.
	getAs := func(session string, port int, h, login string) int {
		q, _ := http.NewRequest(http.MethodGet, "http://127.0.0.1:"+strconv.Itoa(port)+"/s/"+session+"/tasks", nil)
		q.Host = h
		if login != "" {
			q.Header.Set(OwnerHeader, login)
		}
		resp, err := client.Do(q)
		if err != nil {
			return -1
		}
		resp.Body.Close()
		return resp.StatusCode
	}
	get := func(port int, h, login string) int {
		if port == info.RemotePort {
			return getAs(netSessionOf(info.Token), port, h, login)
		}
		return getAs(sessionOf(info.Token), port, h, login)
	}
	local := "127.0.0.1:" + strconv.Itoa(info.Port)
	if c := getAs(sessionOf(info.Token), info.RemotePort, rhost, owner); c != http.StatusForbidden {
		t.Errorf("network socket, the local session: %d", c)
	}
	if c := getAs(netSessionOf(info.Token), info.Port, local, ""); c != http.StatusForbidden {
		t.Errorf("local socket, the network's session: %d", c)
	}
	if c := get(info.RemotePort, local, "guest@example.com"); c != http.StatusForbidden {
		t.Errorf("network socket, Host 127.0.0.1: %d", c)
	}
	if c := get(info.RemotePort, rhost, owner); c != http.StatusOK {
		t.Errorf("network socket, the owner: %d", c)
	}
	if c := get(info.Port, local, ""); c != http.StatusOK {
		t.Errorf("local socket: %d", c)
	}
	if c := get(info.Port, rhost, owner); c != http.StatusForbidden {
		t.Errorf("local socket, the network's Host: %d", c)
	}
	if err := GoLocal(cfg); err != nil {
		t.Fatal(err)
	}
	if c := get(info.RemotePort, rhost, owner); c != -1 && c != http.StatusForbidden {
		t.Errorf("network socket after going local: %d", c)
	}
	if i, _ := ReadInfo(cfg); i.RemotePort != 0 {
		t.Errorf("info after going local: %+v", i)
	}
}

func newReq(s *Server, path string) *http.Request {
	q := httptest.NewRequest(http.MethodGet, s.localBase()+path, nil)
	q.Host = host
	return q
}

func record(h http.HandlerFunc, q *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	h(w, q)
	return w
}
