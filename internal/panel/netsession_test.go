package panel

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// A program of this machine can reach the network's socket (it listens on 127.0.0.1)
// and set the headers Tailscale would set. What it gets from spending a ticket meant for
// the phone there is the network's session: on the local socket it opens nothing, and
// its form code changes nothing, so the guard's lists stay as they were. Before each
// socket had its own session, this test's steps added a repository to the list of
// commits allowed without asking.
func TestRemoteTicketGivesNoLocalSession(t *testing.T) {
	s, cfg := fixture(t, "en")
	const rhost, owner = "box.example.ts.net:8443", "lucia@example.com"
	s.remote.Store(&Remote{Host: rhost, Owner: owner})
	before, err := os.ReadFile(filepath.Join(cfg, "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	id, err := s.tickets.add("", RemoteTicketTTL, true, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	w := s.do(t, req{net: true, path: "/t/" + id, host: rhost, owner: owner})
	loc := w.Header().Get("Location")
	if w.Code != http.StatusSeeOther || loc != s.netBase()+"/" {
		t.Fatalf("spending the remote ticket: %d %q", w.Code, loc)
	}
	if s.netBase() == s.localBase() {
		t.Fatal("both sockets share a session")
	}

	// That session, on the local socket: nothing, not even the guard's page.
	for _, p := range []string{"/", "/today", "/guard?s=work", "/versions?s=work", "/backup?s=work"} {
		if g := s.do(t, req{path: loc + strings.TrimPrefix(p, "/")}); g.Code != http.StatusForbidden {
			t.Errorf("network session on 127.0.0.1, %s: %d", p, g.Code)
		}
	}

	// Through the network it works, and the guard's page only shows: no form code there
	// changes the lists, on either socket.
	g := s.do(t, req{net: true, path: loc + "guard?s=work", host: rhost, owner: owner})
	if g.Code != http.StatusOK {
		t.Fatalf("guard page through the network: %d", g.Code)
	}
	codes := map[string]bool{s.netCSRF: true}
	for _, m := range regexp.MustCompile(`name="csrf" value="([0-9a-f]+)"`).FindAllStringSubmatch(g.Body.String(), -1) {
		codes[m[1]] = true
	}
	if codes[s.csrf] {
		t.Fatal("a page of the network socket carries the local form code")
	}
	form := func(code string) url.Values {
		return url.Values{"csrf": {code}, "s": {"work"}, "list": {"allowed"}, "op": {"add"}, "space": {"work"},
			"value": {"/tmp/anything"}, "confirm": {"yes"}}
	}
	for code := range codes {
		// The local socket, with the network's session or the local path.
		for _, base := range []string{s.netBase(), s.localBase()} {
			p := s.do(t, req{method: "POST", path: base + "/guard/list", origin: "http://" + host, form: form(code)})
			if p.Code == http.StatusSeeOther {
				t.Errorf("guard list changed on 127.0.0.1 with the network's form code (%s)", base)
			}
		}
		// The network socket: read only.
		p := s.do(t, req{net: true, method: "POST", path: s.netBase() + "/guard/list", host: rhost, owner: owner,
			origin: "https://" + rhost, form: form(code)})
		if p.Code == http.StatusSeeOther {
			t.Error("guard list changed through the network")
		}
	}
	after, err := os.ReadFile(filepath.Join(cfg, "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatalf("the configuration changed:\n%s", after)
	}

	// And the other way: the local session is refused through the network.
	if w := s.do(t, req{net: true, path: s.localBase() + "/today", host: rhost, owner: owner}); w.Code != http.StatusForbidden {
		t.Errorf("local session through the network: %d", w.Code)
	}
	// The local socket still works as before, with its own form code.
	if w := s.do(t, req{method: "POST", path: s.localBase() + "/inbox/capture", origin: "http://" + host,
		form: url.Values{"s": {"work"}, "text": {"x"}, "csrf": {s.netCSRF}}}); w.Code != http.StatusForbidden {
		t.Errorf("local socket with the network's form code: %d", w.Code)
	}
	if w := s.do(t, req{method: "POST", path: s.localBase() + "/inbox/capture", origin: "http://" + host,
		form: url.Values{"s": {"work"}, "text": {"x"}, "csrf": {s.csrf}}}); w.Code != http.StatusSeeOther {
		t.Errorf("local socket with its own form code: %d", w.Code)
	}
}
