package panel

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/state"
)

var ticketShape = regexp.MustCompile(`^[0-9a-f]{32}$`)

// askTicket is the command asking for a ticket, with its token.
func (s *Server) askTicket(t *testing.T, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	return s.do(t, req{method: http.MethodPost, path: "/ticket", origin: "http://" + host, bearer: s.token, form: form})
}

// A ticket opens the panel once: 303 to the page under the session the first time,
// 410 after, with a page that does not give the session away (P-01).
func TestTicketOnce(t *testing.T) {
	s, _ := fixture(t, "en")
	w := s.askTicket(t, url.Values{"page": {"today?s=work"}})
	id := strings.TrimSpace(w.Body.String())
	if w.Code != http.StatusOK || !ticketShape.MatchString(id) {
		t.Fatalf("ticket: %d %q", w.Code, id)
	}
	first := s.do(t, req{path: "/t/" + id})
	if first.Code != http.StatusSeeOther || first.Header().Get("Location") != s.base()+"/today?s=work" {
		t.Fatalf("first visit: %d %q", first.Code, first.Header().Get("Location"))
	}
	again := s.do(t, req{path: "/t/" + id})
	if again.Code != http.StatusGone {
		t.Fatalf("second visit: %d", again.Code)
	}
	if b := again.Body.String(); strings.Contains(b, s.session) || !strings.Contains(b, "already used") {
		t.Errorf("second visit's page: %s", b)
	}
	if w := s.do(t, req{path: "/t/" + strings.Repeat("ab", 16)}); w.Code != http.StatusForbidden {
		t.Errorf("a ticket that never was: %d", w.Code)
	}

	// Run out: 410 too, and it stays spent.
	w = s.askTicket(t, url.Values{"page": {""}})
	id = strings.TrimSpace(w.Body.String())
	if page, why := s.tickets.spend(id, false, time.Now().Add(TicketTTL+time.Second)); page != "" || why != "expired" {
		t.Errorf("late: %q %q", page, why)
	}
	s.tickets.m[id].expires = time.Now().Add(-time.Second)
	if w := s.do(t, req{path: "/t/" + id}); w.Code != http.StatusGone || !strings.Contains(w.Body.String(), "run out") {
		t.Errorf("after running out: %d %s", w.Code, w.Body)
	}
}

// Only the command gets a ticket, and only for a page of the panel.
func TestTicketOnlyForTheCommand(t *testing.T) {
	s, _ := fixture(t, "en")
	form := url.Values{"page": {"tasks"}, "csrf": {s.csrf}}
	// A page of the panel cannot make tickets: they would outlive its session.
	if w := s.do(t, req{method: http.MethodPost, path: s.base() + "/ticket", origin: "http://" + host, form: form}); w.Code != http.StatusForbidden {
		t.Errorf("from a page: %d", w.Code)
	}
	if w := s.do(t, req{method: http.MethodPost, path: "/ticket", origin: "http://" + host, form: form}); w.Code != http.StatusForbidden {
		t.Errorf("without the token: %d", w.Code)
	}
	for _, page := range []string{"//evil.example/x", "https://evil.example/", "/s/x/tasks", "../ping", `\\evil`, "tasks#x"} {
		if w := s.askTicket(t, url.Values{"page": {page}}); w.Code != http.StatusBadRequest {
			t.Errorf("page %q: %d", page, w.Code)
		}
	}
	// Through the network: the command talks to 127.0.0.1 only; the ticket it got
	// opens from the phone, for the owner only.
	const rhost, owner = "box.example.ts.net:8443", "lucia@example.com"
	s.remote.Store(&Remote{Host: rhost, Owner: owner})
	if w := s.do(t, req{method: http.MethodPost, path: "/ticket", host: rhost, owner: owner, origin: "https://" + rhost, bearer: s.token, form: url.Values{"page": {""}}}); w.Code != http.StatusForbidden {
		t.Errorf("ticket asked through the network: %d", w.Code)
	}
	w := s.askTicket(t, url.Values{"page": {"tasks"}, "remote": {"1"}})
	id := strings.TrimSpace(w.Body.String())
	if w := s.do(t, req{path: "/t/" + id, host: rhost, owner: "guest@example.com"}); w.Code != http.StatusForbidden {
		t.Errorf("someone else in the network: %d", w.Code)
	}
	// A program of this machine cannot spend the phone's ticket on 127.0.0.1.
	if w := s.do(t, req{path: "/t/" + id}); w.Code != http.StatusForbidden {
		t.Errorf("the phone's ticket on 127.0.0.1: %d", w.Code)
	}
	if w := s.do(t, req{path: "/t/" + id, host: rhost, owner: owner}); w.Code != http.StatusSeeOther {
		t.Errorf("the owner's phone: %d", w.Code)
	}
	// Nor is a ticket for this machine's browser any use through the network.
	w = s.askTicket(t, url.Values{"page": {""}})
	local := strings.TrimSpace(w.Body.String())
	if w := s.do(t, req{path: "/t/" + local, host: rhost, owner: owner}); w.Code != http.StatusForbidden {
		t.Errorf("a local ticket through the network: %d", w.Code)
	}
	if w := s.do(t, req{path: "/t/" + local}); w.Code != http.StatusSeeOther {
		t.Errorf("the local ticket, still good on 127.0.0.1: %d", w.Code)
	}
	if page, why := s.tickets.spend("x", true, time.Now()); page != "" || why != "unknown" {
		t.Errorf("spend x: %q %q", page, why)
	}
	// A remote ticket waits longer, for the phone.
	w = s.askTicket(t, url.Values{"page": {""}, "remote": {"1"}})
	id = strings.TrimSpace(w.Body.String())
	if page, why := s.tickets.spend(id, true, time.Now().Add(TicketTTL+time.Minute)); why != "" || page != "" {
		t.Errorf("remote ticket after %v: %q", TicketTTL+time.Minute, why)
	}
}

// The command asks a real panel for a ticket, and the ticket opens it.
func TestTicketFromTheCommand(t *testing.T) {
	_, cfg := fixture(t, "en")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errc := make(chan error, 1)
	go func() { errc <- Serve(ctx, cfg, time.Hour) }()
	var info Info
	for k := 0; k < 100; k++ {
		if i, ok := ReadInfo(cfg); ok && Alive(i) {
			info = i
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if info.Port == 0 {
		t.Fatal("the server did not start")
	}
	defer func() { _, _ = Stop(cfg); <-errc }()
	id, err := Ticket(info, "tasks", false)
	if err != nil {
		t.Fatal(err)
	}
	u := info.TicketURL(id)
	if strings.Contains(u, sessionOf(info.Token)) {
		t.Errorf("the ticket's address holds the session: %s", u)
	}
	resp, err := client.Get(u)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/s/"+sessionOf(info.Token)+"/tasks" {
		t.Errorf("visit: %d %q", resp.StatusCode, resp.Header.Get("Location"))
	}
	if _, err := Ticket(Info{Port: info.Port, Token: strings.Repeat("0", 64)}, "", false); err == nil {
		t.Error("a ticket with a wrong token")
	}
}

// A change to the lists made in the panel leaves the user a notice, which the next hook
// shows them; accepting changed lists too.
func TestGuardChangeLeavesNotice(t *testing.T) {
	s, cfg := fixture(t, "en")
	sealNow(t, cfg)
	redirected := func(w *httptest.ResponseRecorder, what string) {
		t.Helper()
		if w.Code != http.StatusSeeOther {
			t.Fatalf("%s: %d %s", what, w.Code, w.Body)
		}
	}
	redirected(s.post(t, "/guard/list", url.Values{"s": {"work"}, "list": {"hosts"}, "op": {"add"}, "value": {"db01"}}), "add host")
	if n := state.TakeGuardNotice(cfg); !strings.Contains(n, "changed in the panel") || !strings.Contains(n, "+db01") {
		t.Errorf("notice after a change: %q", n)
	}
	if err := config.SetTopList(cfg, "untouchable_hosts", []string{"db01", "sneaky"}); err != nil {
		t.Fatal(err)
	}
	redirected(s.post(t, "/guard/accept", url.Values{"s": {"work"}}), "accept")
	if n := state.TakeGuardNotice(cfg); !strings.Contains(n, "accepted in the panel") {
		t.Errorf("notice after accepting: %q", n)
	}
}
