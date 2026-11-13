package panel

import (
	"html"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/icalvete/augmented-brain/internal/i18n"
)

// A ticket is how a browser gets into the panel: `augmented-brain panel` asks the
// running panel for one (POST /ticket, with the token only the command can read) and
// opens /t/<ticket>. The first visit spends it and is sent on to the pages, under
// /s/<session>/; the session itself is never printed. So the address the command shows,
// which the assistant sees when it runs /brain:panel, works once: if the assistant used
// it first, the user's browser says so instead of opening, and the user knows.

// TicketTTL is how long a ticket waits for its visit: the browser opens at once. Through
// the user's network (panel remote) it waits longer, for the phone to scan the QR code.
const (
	TicketTTL       = 2 * time.Minute
	RemoteTicketTTL = 10 * time.Minute
	// usedKept is how long a spent ticket is remembered, to tell "already used" apart
	// from an address that never was.
	usedKept = 30 * time.Minute
)

type ticket struct {
	page    string // the page to open, under /s/<session>/
	expires time.Time
	used    bool
	// remote: the ticket opens the panel through the network's socket only; any other
	// opens it on 127.0.0.1 only. So a program of this machine cannot spend on
	// 127.0.0.1 the ticket meant for the phone, nor the network one for this browser.
	remote bool
}

type tickets struct {
	mu sync.Mutex
	m  map[string]*ticket
}

// add makes a ticket for page, valid for ttl, for the network's socket when remote.
func (t *tickets) add(page string, ttl time.Duration, remote bool, now time.Time) (string, error) {
	r, err := random()
	if err != nil {
		return "", err
	}
	id := r[:32]
	t.mu.Lock()
	defer t.mu.Unlock()
	t.prune(now)
	if t.m == nil {
		t.m = map[string]*ticket{}
	}
	t.m[id] = &ticket{page: page, expires: now.Add(ttl), remote: remote}
	return id, nil
}

// spend uses ticket id, which came through the network's socket when remote: its page
// and "" when it is good, or why not ("used", "expired", "unknown"). A good ticket is
// good once, and only through its own socket: through the other one it is unknown.
func (t *tickets) spend(id string, remote bool, now time.Time) (page, why string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.prune(now)
	k, ok := t.m[id]
	switch {
	case !ok || len(id) != 32 || k.remote != remote:
		return "", "unknown"
	case k.used:
		return "", "used"
	case now.After(k.expires):
		return "", "expired"
	}
	k.used = true
	return k.page, ""
}

// prune forgets tickets long past.
func (t *tickets) prune(now time.Time) {
	for id, k := range t.m {
		if now.After(k.expires.Add(usedKept)) {
			delete(t.m, id)
		}
	}
}

// validPage reports whether page is a page of the panel to open after the ticket: a
// path and query of its own, never another site or another session.
func validPage(page string) bool {
	if len(page) > 300 || strings.HasPrefix(page, "/") || strings.Contains(page, "\\") || strings.Contains(page, "..") {
		return false
	}
	u, err := url.Parse(page)
	return err == nil && u.Scheme == "" && u.Host == "" && u.User == nil && u.Fragment == ""
}

// handleTicket gives the command a ticket (POST /ticket, form page= and remote=1).
func (s *Server) handleTicket(w http.ResponseWriter, r *http.Request) {
	if !s.commandOnly(r) {
		forbidden(w)
		return
	}
	page := r.PostForm.Get("page")
	if !validPage(page) {
		http.Error(w, "bad page", http.StatusBadRequest)
		return
	}
	ttl, remote := TicketTTL, r.PostForm.Get("remote") == "1"
	if remote {
		ttl = RemoteTicketTTL
	}
	id, err := s.tickets.add(page, ttl, remote, time.Now())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(id + "\n"))
}

// redeem answers GET /t/<ticket>: the first time, 303 to the page under the session;
// after that, 410 with a page that says the link was used or ran out. That page is
// plain, with no link into the panel: it must not give the session away.
func (s *Server) redeem(w http.ResponseWriter, r *http.Request, remote bool) {
	id := strings.TrimPrefix(r.URL.Path, "/t/")
	page, why := s.tickets.spend(id, remote, time.Now())
	if why == "" {
		s.last.Store(time.Now().UnixNano())
		http.Redirect(w, r, s.base(r)+"/"+page, http.StatusSeeOther)
		return
	}
	if why == "unknown" {
		forbidden(w)
		return
	}
	lang := s.lang()
	head, text := i18n.T(lang, "panel.ticket."+why), i18n.T(lang, "panel.ticket."+why+".text")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusGone)
	_, _ = w.Write([]byte("<!doctype html><html lang=\"" + html.EscapeString(lang) + "\"><meta charset=\"utf-8\">" +
		"<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\"><title>augmented brain</title>" +
		"<h1>" + html.EscapeString(head) + "</h1><p>" + html.EscapeString(text) + "</p></html>"))
}
