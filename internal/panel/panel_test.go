package panel

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/icalvete/augmented-brain/internal/backup"
	"github.com/icalvete/augmented-brain/internal/brain"
	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/filelock"
	"github.com/icalvete/augmented-brain/internal/memory"
	"github.com/icalvete/augmented-brain/internal/setup"
)

const port = 45678

var host = "127.0.0.1:45678"

// fixture sets up two spaces, work and home, in a temporary home, and a panel on them.
func fixture(t *testing.T, lang string) (*Server, string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg := filepath.Join(home, "cfg")
	for i, sp := range []string{"work", "home"} {
		work := filepath.Join(home, sp)
		if err := os.MkdirAll(work, 0o755); err != nil {
			t.Fatal(err)
		}
		o := setup.Options{Language: lang, Space: sp, WorkFolder: work, Memory: "~/brain-" + sp, Today: "2026-10-15"}
		var err error
		if i == 0 {
			_, err = setup.Init(cfg, o)
		} else {
			_, err = setup.AddSpace(cfg, o)
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	s, err := New(cfg, port)
	if err != nil {
		t.Fatal(err)
	}
	return s, cfg
}

type req struct {
	method, path, host, origin, referer string
	bearer                              string
	owner                               string // the Tailscale login header
	form                                url.Values
	body                                string
}

func (s *Server) do(t *testing.T, r req) *httptest.ResponseRecorder {
	t.Helper()
	if r.method == "" {
		r.method = http.MethodGet
	}
	var body io.Reader
	if r.form != nil {
		body = strings.NewReader(r.form.Encode())
	} else if r.body != "" {
		body = strings.NewReader(r.body)
	}
	q := httptest.NewRequest(r.method, r.path, body)
	q.Host = host
	if r.host != "" {
		q.Host = r.host
	}
	if body != nil {
		q.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if r.origin != "" {
		q.Header.Set("Origin", r.origin)
	}
	if r.referer != "" {
		q.Header.Set("Referer", r.referer)
	}
	if r.bearer != "" {
		q.Header.Set("Authorization", "Bearer "+r.bearer)
	}
	if r.owner != "" {
		q.Header.Set(OwnerHeader, r.owner)
	}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, q)
	return w
}

// get is a page read by the browser, at the panel's address of this run.
func (s *Server) get(t *testing.T, path string) *httptest.ResponseRecorder {
	return s.do(t, req{path: s.base() + path})
}

// post is a form sent by one of the panel's own pages.
func (s *Server) post(t *testing.T, path string, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	form.Set("csrf", s.csrf)
	return s.do(t, req{method: http.MethodPost, path: s.base() + path, origin: "http://" + host, form: form})
}

func TestAccess(t *testing.T) {
	s, _ := fixture(t, "en")

	// Outside the session's address: nothing, not even the stylesheet.
	for _, p := range []string{"/", "/tasks", "/static/style.css", "/?t=" + s.token, "/s/" + s.token + "/tasks",
		"/s/" + strings.Repeat("0", 32) + "/tasks", "/s//tasks", "/s/" + s.session + "x/tasks"} {
		if w := s.do(t, req{path: p}); w.Code != http.StatusForbidden {
			t.Errorf("%s: %d", p, w.Code)
		}
	}
	// The session's address: its root goes to today, and no cookie is ever set
	// (cookies of 127.0.0.1 would reach every other local port).
	if w := s.do(t, req{path: s.base()}); w.Code != http.StatusSeeOther || w.Header().Get("Location") != s.base()+"/" {
		t.Errorf("bare session: %d %q", w.Code, w.Header().Get("Location"))
	}
	w := s.do(t, req{path: s.base() + "/"})
	if w.Code != http.StatusSeeOther || w.Header().Get("Location") != s.base()+"/today" {
		t.Fatalf("root: %d %q", w.Code, w.Header().Get("Location"))
	}
	w = s.get(t, "/tasks?s=home")
	if w.Code != http.StatusOK || len(w.Result().Cookies()) != 0 {
		t.Fatalf("tasks: %d, cookies %v", w.Code, w.Result().Cookies())
	}
	if strings.Contains(w.Body.String(), s.token) || !strings.Contains(w.Body.String(), `href="`+s.base()+`/static/style.css"`) {
		t.Error("the page leaks the token, or its links are not under the session")
	}
	if s.session == s.token || len(s.session) != 32 {
		t.Error("the session must not be the token")
	}
	if (Info{Port: port, Token: s.token}).URL() != "http://127.0.0.1:45678"+s.base()+"/" {
		t.Errorf("URL: %s", Info{Port: port, Token: s.token}.URL())
	}

	// DNS rebinding: any other Host is refused.
	for _, h := range []string{"evil.example:45678", "127.0.0.1:1", "127.0.0.1", "localhost", "x127.0.0.1:45678", "[::1]:45678"} {
		if w := s.do(t, req{path: s.base() + "/tasks", host: h}); w.Code != http.StatusForbidden {
			t.Errorf("Host %s: %d", h, w.Code)
		}
	}
	if w := s.do(t, req{path: s.base() + "/tasks", host: "localhost:45678"}); w.Code != http.StatusOK {
		t.Errorf("Host localhost: %d", w.Code)
	}

	// The ping proves the server knows the token, without the token being sent.
	c := strings.Repeat("ab", 32)
	if w := s.do(t, req{path: "/ping?c=" + c}); w.Code != http.StatusOK || w.Header().Get("X-Augmented-Brain") != "panel" ||
		strings.TrimSpace(w.Body.String()) != proof(s.token, c) || strings.Contains(w.Body.String(), s.token) {
		t.Errorf("ping: %d %q", w.Code, w.Body)
	}
	if proof(s.token, c) == proof(strings.Repeat("0", 64), c) || proof(s.token, "session") == sessionOf(s.token) {
		t.Error("proofs must depend on the token and stay apart from the session")
	}
	for _, bad := range []string{"/ping", "/ping?c=xyz", "/ping?c=" + c[:62]} {
		if w := s.do(t, req{path: bad}); w.Code != http.StatusBadRequest {
			t.Errorf("%s: %d", bad, w.Code)
		}
	}
	// The command's own requests carry the token in a header.
	if w := s.do(t, req{path: "/tasks", bearer: s.token}); w.Code != http.StatusOK {
		t.Errorf("tasks with the token: %d", w.Code)
	}

	// Headers on every answer, the refusals included.
	for _, w := range []*httptest.ResponseRecorder{s.do(t, req{path: "/tasks"}), s.get(t, "/tasks"), s.do(t, req{path: "/ping?c=" + c})} {
		h := w.Header()
		csp := h.Get("Content-Security-Policy")
		for _, want := range []string{"default-src 'none'", "script-src 'self'", "frame-ancestors 'none'", "form-action 'self'"} {
			if !strings.Contains(csp, want) {
				t.Errorf("CSP %q lacks %q", csp, want)
			}
		}
		if strings.Contains(csp, "unsafe-inline") || strings.Contains(csp, "http") {
			t.Errorf("CSP too loose: %q", csp)
		}
		if h.Get("X-Content-Type-Options") != "nosniff" || h.Get("X-Frame-Options") != "DENY" || h.Get("Referrer-Policy") == "" {
			t.Errorf("headers: %v", h)
		}
		if h.Get("Access-Control-Allow-Origin") != "" {
			t.Error("no CORS")
		}
	}
}

// Through the Tailscale network: the machine's own name, and only its owner.
func TestRemoteAccess(t *testing.T) {
	s, cfg := fixture(t, "en")
	const rhost, owner = "box.example.ts.net:8443", "lucia@example.com"
	tasks := s.base() + "/tasks"
	// Not published: the network's name is refused like any other.
	if w := s.do(t, req{path: tasks, host: rhost, owner: owner}); w.Code != http.StatusForbidden {
		t.Errorf("not published: %d", w.Code)
	}
	s.remote.Store(&Remote{Host: rhost, Owner: owner})
	if w := s.do(t, req{path: tasks, host: rhost, owner: owner}); w.Code != http.StatusOK {
		t.Fatalf("owner: %d", w.Code)
	}
	if w := s.do(t, req{path: tasks, host: rhost, owner: "LUCIA@example.com"}); w.Code != http.StatusOK {
		t.Errorf("logins ignore case: %d", w.Code)
	}
	for i, r := range []req{
		{path: tasks, host: rhost},                                   // no login: not from the network
		{path: tasks, host: rhost, owner: "guest@example.com"},       // someone else in the network
		{path: tasks, host: "box.example.ts.net", owner: owner},      // another port
		{path: tasks, host: "evil.example:8443", owner: owner},       // another name
		{path: "/tasks", host: rhost, owner: owner},                  // outside the session
		{path: "/tasks", host: rhost, owner: owner, bearer: s.token}, // the command talks to 127.0.0.1 only
		{path: "/ping?c=" + strings.Repeat("ab", 32), host: rhost, owner: owner},
	} {
		if w := s.do(t, r); w.Code != http.StatusForbidden {
			t.Errorf("case %d: %d", i, w.Code)
		}
	}
	// 127.0.0.1 works as before.
	if w := s.get(t, "/tasks"); w.Code != http.StatusOK {
		t.Errorf("local: %d", w.Code)
	}
	// Writes through the network need its https origin.
	path := s.base() + "/inbox/capture"
	form := url.Values{"s": {"work"}, "text": {"From the phone"}, "csrf": {s.csrf}}
	if w := s.do(t, req{method: "POST", path: path, host: rhost, owner: owner, form: form, origin: "http://" + rhost}); w.Code != http.StatusForbidden {
		t.Errorf("http origin through the network: %d", w.Code)
	}
	if w := s.do(t, req{method: "POST", path: path, host: rhost, owner: owner, form: form, origin: "https://" + rhost}); w.Code != http.StatusSeeOther {
		t.Fatalf("capture through the network: %d %s", w.Code, w.Body)
	}
	data, _ := os.ReadFile(filepath.Join(filepath.Dir(cfg), "brain-work", "inbox.md"))
	if !strings.Contains(string(data), "From the phone") {
		t.Error("the capture did not land")
	}
	i := Info{Port: port, Token: s.token, Remote: s.remote.Load()}
	if i.RemoteURL() != "https://"+rhost+s.base()+"/" {
		t.Errorf("remote URL: %s", i.RemoteURL())
	}
	if (Info{Port: port, Token: s.token}).RemoteURL() != "" {
		t.Error("no remote, no remote URL")
	}
}

func TestWritesNeedTheirOrigin(t *testing.T) {
	s, cfg := fixture(t, "en")
	inbox := func() string {
		data, _ := os.ReadFile(filepath.Join(filepath.Dir(cfg), "brain-work", "inbox.md"))
		return string(data)
	}
	form := func() url.Values { return url.Values{"s": {"work"}, "text": {"Call Ana"}, "csrf": {s.csrf}} }
	own := "http://" + host
	path := s.base() + "/inbox/capture"
	refused := []req{
		{method: "POST", path: path, form: form()},                                      // no Origin, no Referer
		{method: "POST", path: path, form: form(), origin: "null"},                      // a sandboxed or no-referrer page
		{method: "POST", path: path, form: form(), origin: "http://evil.example"},       // another site
		{method: "POST", path: path, form: form(), origin: "http://127.0.0.1:1"},        // another local server
		{method: "POST", path: path, form: form(), referer: "http://evil.example/x"},    // another site, by Referer
		{method: "POST", path: path, form: url.Values{"text": {"x"}}, origin: own},      // no csrf field
		{method: "POST", path: "/inbox/capture", form: form(), origin: own},             // not under the session
		{method: "POST", path: path, form: form(), origin: own, host: "evil.example:1"}, // rebinding
	}
	for i, r := range refused {
		if w := s.do(t, r); w.Code != http.StatusForbidden {
			t.Errorf("case %d: %d", i, w.Code)
		}
	}
	if strings.Contains(inbox(), "Call Ana") {
		t.Fatal("a refused request wrote")
	}
	// Writes are POST only.
	if w := s.do(t, req{path: path + "?text=x"}); w.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET of a write: %d", w.Code)
	}
	if w := s.do(t, req{method: "PUT", path: path, origin: own}); w.Code != http.StatusMethodNotAllowed {
		t.Errorf("PUT: %d", w.Code)
	}
	// Too big a body.
	big := url.Values{"csrf": {s.csrf}, "text": {strings.Repeat("a", maxBody)}}
	if w := s.do(t, req{method: "POST", path: path, origin: own, form: big}); w.Code != http.StatusBadRequest {
		t.Errorf("big body: %d", w.Code)
	}

	// Accepted: Origin, or Referer when there is no Origin.
	if w := s.do(t, req{method: "POST", path: path, form: form(), origin: own}); w.Code != http.StatusSeeOther {
		t.Fatalf("capture: %d %s", w.Code, w.Body)
	}
	f := form()
	f.Set("text", "Buy flour")
	if w := s.do(t, req{method: "POST", path: path, form: f, referer: own + s.base() + "/inbox"}); w.Code != http.StatusSeeOther {
		t.Fatalf("capture by Referer: %d", w.Code)
	}
	if got := inbox(); !strings.Contains(got, "Call Ana") || !strings.Contains(got, "Buy flour") {
		t.Errorf("inbox: %q", got)
	}
}

func TestPagesAndChanges(t *testing.T) {
	s, cfg := fixture(t, "es")
	mem := filepath.Join(filepath.Dir(cfg), "brain-work")
	ok := func(w *httptest.ResponseRecorder, what string) string {
		t.Helper()
		if w.Code != http.StatusSeeOther {
			t.Fatalf("%s: %d %s", what, w.Code, w.Body)
		}
		loc := w.Header().Get("Location")
		if !strings.Contains(loc, "ok=1") || !strings.Contains(loc, "s=work") {
			t.Errorf("%s: redirect %q", what, loc)
		}
		return loc
	}

	// A task whose title is HTML: it is shown as text.
	loc := ok(s.post(t, "/task/new", url.Values{"s": {"work"}, "title": {"<script>alert(1)</script> Hornear pan"}, "summary": {"para Lucía"}}), "new task")
	id := "task-script-alert-1-script-hornear-pan"
	if !strings.Contains(loc, "id="+id) {
		t.Fatalf("new task went to %q", loc)
	}
	page := s.get(t, "/task?s=work&id="+id).Body.String()
	if strings.Contains(page, "<script>alert") || !strings.Contains(page, "&lt;script&gt;alert(1)&lt;/script&gt; Hornear pan") {
		t.Error("the title was not escaped")
	}
	if !strings.Contains(page, `lang="es"`) || !strings.Contains(page, "Escribir en el diario") {
		t.Error("the page is not in the configuration's language")
	}

	ok(s.post(t, "/task/update", url.Values{"s": {"work"}, "id": {id}, "status": {"analysis"}, "waiting_on": {"Ana"}, "next": {"Pedir harina"}}), "update")
	ok(s.post(t, "/task/journal", url.Values{"s": {"work"}, "id": {id}, "entry": {"Probada la receta"}}), "journal")
	ok(s.post(t, "/watch/add", url.Values{"s": {"work"}, "id": {id}, "date": {"2026-10-20"}, "what": {"¿Llegó la harina?"}}), "watch")
	if w := s.post(t, "/task/update", url.Values{"s": {"work"}, "id": {id}, "status": {"closed"}}); w.Code != http.StatusBadRequest {
		t.Errorf("closing through update: %d", w.Code)
	}
	if w := s.post(t, "/task/journal", url.Values{"s": {"work"}, "id": {id}, "entry": {" "}}); w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "Rellena") {
		t.Errorf("empty entry: %d", w.Code)
	}
	rem := s.get(t, "/reminders?s=work").Body.String()
	if !strings.Contains(rem, "¿Llegó la harina?") {
		t.Error("the watch date is not among the reminders")
	}
	ok(s.post(t, "/watch/done", url.Values{"s": {"work"}, "id": {id}, "date": {"2026-10-20"}, "back": {"reminders"}}), "watch done")
	ok(s.post(t, "/task/park", url.Values{"s": {"work"}, "id": {id}, "next": {"Llamar al molino"}}), "park")
	ok(s.post(t, "/task/close", url.Values{"s": {"work"}, "id": {id}, "result": {"Pan hecho"}}), "close")

	c, _ := config.Load(cfg)
	b, err := brain.Open(c, c.Spaces["work"])
	if err != nil {
		t.Fatal(err)
	}
	n, err := b.Task(id)
	if err != nil {
		t.Fatal(err)
	}
	if n.Header.Status != "closed" || len(n.Header.Watch) != 1 || !n.Header.Watch[0].Done ||
		!strings.Contains(n.Body, "Probada la receta") || !strings.Contains(n.Body, "Pan hecho") {
		t.Errorf("task after the changes: %+v\n%s", n.Header, n.Body)
	}
	// The index is the program's, up to date.
	index, _ := os.ReadFile(filepath.Join(mem, "MEMORY.md"))
	if string(index) != b.Mem.Index("es") {
		t.Error("MEMORY.md is not up to date")
	}

	ok(s.post(t, "/area/new", url.Values{"s": {"work"}, "title": {"Panadería"}}), "area")
	ok(s.post(t, "/inbox/capture", url.Values{"s": {"work"}, "text": {"Idea: pan de centeno"}, "remind_on": {"2026-10-18"}}), "capture")
	today := time.Now().Format("2006-01-02")
	ok(s.post(t, "/inbox/done", url.Values{"s": {"work"}, "date": {today}, "text": {"Idea: pan de centeno"}}), "inbox done")
	if w := s.post(t, "/inbox/done", url.Values{"s": {"work"}, "date": {today}, "text": {"Idea: pan de centeno"}}); w.Code != http.StatusBadRequest {
		t.Errorf("done twice: %d", w.Code)
	}
	ok(s.post(t, "/profile", url.Values{"s": {"work"}, "name": {"Lucía"}, "about": {"- Panadera\r\n- Aprende a programar"}, "work_style": {"- Respuestas cortas"}}), "profile")
	p, _ := brain.LoadProfile(cfg)
	if p.Name != "Lucía" || p.About != "- Panadera\n- Aprende a programar" {
		t.Errorf("profile: %+v", p)
	}

	// Every page answers, in each space.
	for _, sp := range []string{"work", "home"} {
		for _, path := range []string{"/tasks", "/reminders", "/inbox", "/areas", "/knowledge", "/rules", "/people", "/profile", "/space"} {
			if w := s.get(t, path+"?s="+sp); w.Code != http.StatusOK {
				t.Errorf("%s in %s: %d", path, sp, w.Code)
			}
		}
		if w := s.get(t, "/note?s="+sp+"&id=area-general"); w.Code != http.StatusOK {
			t.Errorf("area note in %s: %d", sp, w.Code)
		}
	}
	if !strings.Contains(s.get(t, "/areas?s=work").Body.String(), "Panadería") {
		t.Error("the new area is not listed")
	}
	if !strings.Contains(s.get(t, "/tasks?s=work").Body.String(), "Cerradas (1)") {
		t.Error("the closed task is not listed")
	}
	// The shared notes show in every space, marked as such, next to the space's own.
	for path, text := range map[string]string{
		"rules/rule-brief.md":       "---\nid: rule-brief\ntype: rule\ntitle: Respuestas breves\n---\nUna idea por frase.\n",
		"knowledge/lesson-small.md": "---\nid: lesson-small\ntype: lesson\ntitle: Pasos pequeños\n---\n",
	} {
		full := filepath.Join(cfg, "shared", filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, sp := range []string{"work", "home"} {
		if body := s.get(t, "/rules?s="+sp).Body.String(); !strings.Contains(body, "Respuestas breves") || !strings.Contains(body, ">compartida<") {
			t.Errorf("rules in %s:\n%s", sp, body)
		}
		if body := s.get(t, "/knowledge?s="+sp).Body.String(); !strings.Contains(body, "Pasos pequeños") {
			t.Errorf("knowledge in %s lacks the shared lesson", sp)
		}
		if body := s.get(t, "/note?s="+sp+"&id=rule-brief").Body.String(); !strings.Contains(body, "compartida por todos los espacios") || !strings.Contains(body, "Una idea por frase.") {
			t.Errorf("shared note in %s:\n%s", sp, body)
		}
	}
	// Ids only resolve through the notes: never a path.
	for _, bad := range []string{"../../etc/passwd", "..%2F..%2Fconfig", "MEMORY"} {
		if w := s.get(t, "/note?s=work&id="+bad); w.Code != http.StatusNotFound {
			t.Errorf("note %q: %d", bad, w.Code)
		}
	}
	if w := s.get(t, "/tasks?s=nope"); w.Code != http.StatusBadRequest {
		t.Errorf("unknown space: %d", w.Code)
	}
	if w := s.get(t, "/static/app.js"); w.Code != http.StatusOK {
		t.Errorf("app.js: %d", w.Code)
	}
}

// Writes wait for the notes lock, the same one the MCP server takes.
func TestWritesTakeTheNotesLock(t *testing.T) {
	s, cfg := fixture(t, "en")
	unlock, err := filelock.Lock(filepath.Join(cfg, backup.NotesLock))
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan int)
	go func() {
		done <- s.post(t, "/inbox/capture", url.Values{"s": {"work"}, "text": {"Ana's birthday"}}).Code
	}()
	select {
	case <-done:
		t.Fatal("the write did not wait for the lock")
	case <-time.After(200 * time.Millisecond):
	}
	unlock()
	if code := <-done; code != http.StatusSeeOther {
		t.Errorf("write after the lock: %d", code)
	}
}

func TestSameHost(t *testing.T) {
	for h, want := range map[string]bool{"127.0.0.1:80": true, "localhost:80": true, "127.0.0.1:8080": false,
		"evil:80": false, "127.0.0.1.evil:80": false, "": false, ":80": false} {
		if sameHost(h, 80) != want {
			t.Errorf("sameHost(%q) = %v", h, !want)
		}
	}
}

// The real server: it writes where it listens (only the user can read it), answers its
// token, stops when asked, and stops by itself when nobody uses it.
func TestServeAndStop(t *testing.T) {
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
	if fi, err := os.Stat(filepath.Join(cfg, InfoFile)); err != nil || runtime.GOOS != "windows" && fi.Mode().Perm() != 0o600 {
		t.Errorf("info file: %v %v", fi.Mode(), err)
	}
	if Alive(Info{Port: info.Port, Token: strings.Repeat("0", 64)}) {
		t.Error("alive with a wrong token")
	}
	stopped, err := Stop(cfg)
	if err != nil || !stopped {
		t.Fatalf("Stop: %v %v", stopped, err)
	}
	if err := <-errc; err != nil {
		t.Fatal(err)
	}
	if _, ok := ReadInfo(cfg); ok {
		t.Error("the info file outlived the server")
	}
	if stopped, err := Stop(cfg); stopped || err != nil {
		t.Errorf("Stop with none running: %v %v", stopped, err)
	}

	// Idle.
	go func() { errc <- Serve(ctx, cfg, 100*time.Millisecond) }()
	select {
	case err := <-errc:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the idle server did not stop")
	}
}

// A remote panel records its network address in the info file and takes it back when
// it stops, also when it stops by itself for being idle.
func TestServeRemoteTakesItBack(t *testing.T) {
	_, cfg := fixture(t, "en")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var taken atomic.Int32
	remote := &Remote{Host: "box.example.ts.net:8443", Owner: "lucia@example.com", Unpublish: func() error { taken.Add(1); return nil }}
	errc := make(chan error, 1)
	go func() { errc <- ServeRemote(ctx, cfg, 300*time.Millisecond, remote) }()
	var info Info
	for k := 0; k < 100 && info.Port == 0; k++ {
		if i, ok := ReadInfo(cfg); ok {
			info = i
		}
		time.Sleep(10 * time.Millisecond)
	}
	if info.Remote == nil || info.Remote.Host != remote.Host || info.Remote.Owner != remote.Owner {
		t.Fatalf("info: %+v", info)
	}
	if data, _ := os.ReadFile(filepath.Join(cfg, InfoFile)); strings.Contains(string(data), "Unpublish") {
		t.Errorf("the info file holds the function: %s", data)
	}
	select {
	case err := <-errc:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the idle server did not stop")
	}
	if taken.Load() != 1 {
		t.Errorf("unpublished %d times", taken.Load())
	}
}

// Subtasks and steps from the panel: the forms work, and a step is ticked only by a
// form of the panel's own pages (Origin and csrf token), like every other change.
func TestSubtasksAndSteps(t *testing.T) {
	s, _ := fixture(t, "en")
	if w := s.post(t, "/task/new", url.Values{"s": {"work"}, "title": {"Plant a garden"}}); w.Code != http.StatusSeeOther {
		t.Fatalf("new task: %d %s", w.Code, w.Body)
	}
	parent := "task-plant-a-garden"
	if w := s.post(t, "/task/new", url.Values{"s": {"work"}, "title": {"Buy seeds"}, "parent": {parent}}); w.Code != http.StatusSeeOther {
		t.Fatalf("new subtask: %d %s", w.Code, w.Body)
	}
	if w := s.post(t, "/task/new", url.Values{"s": {"work"}, "title": {"Too deep"}, "parent": {"task-buy-seeds"}}); w.Code != http.StatusBadRequest {
		t.Errorf("grandchild: %d", w.Code)
	}
	for _, st := range []string{"Dig the beds", "Water"} {
		if w := s.post(t, "/task/step", url.Values{"s": {"work"}, "id": {parent}, "step": {st}}); w.Code != http.StatusSeeOther ||
			!strings.HasSuffix(w.Header().Get("Location"), "#steps") {
			t.Fatalf("add step: %d %q", w.Code, w.Header().Get("Location"))
		}
	}
	if w := s.post(t, "/task/step", url.Values{"s": {"work"}, "id": {parent}, "step": {" "}}); w.Code != http.StatusBadRequest {
		t.Errorf("empty step: %d", w.Code)
	}

	page := s.get(t, "/task?s=work&id="+parent).Body.String()
	for _, want := range []string{"0/2 steps", "Dig the beds", `name="n" value="2"><input type="hidden" name="text" value="Water">`, "Subtasks", "Buy seeds", "New subtask", `name="parent" value="` + parent + `"`} {
		if !strings.Contains(page, want) {
			t.Errorf("task page lacks %q", want)
		}
	}
	child := s.get(t, "/task?s=work&id=task-buy-seeds").Body.String()
	if !strings.Contains(child, "Part of: <a") || strings.Contains(child, "New subtask") {
		t.Error("the subtask page: no link to its parent, or it offers a subtask of its own")
	}
	list := s.get(t, "/tasks?s=work").Body.String()
	if !strings.Contains(list, `class="card mine sub"`) || strings.Index(list, "Plant a garden") > strings.Index(list, "Buy seeds") {
		t.Error("the task list does not show the subtask under its parent")
	}

	tick := url.Values{"s": {"work"}, "id": {parent}, "n": {"1"}, "text": {"Dig the beds"}, "done": {"1"}}
	// No csrf token, or another site's Origin: refused, nothing written.
	if w := s.do(t, req{method: http.MethodPost, path: s.base() + "/task/step", origin: "http://" + host, form: tick}); w.Code != http.StatusForbidden {
		t.Errorf("tick without the csrf token: %d", w.Code)
	}
	bad := url.Values{"csrf": {s.csrf}}
	for k, v := range tick {
		bad[k] = v
	}
	if w := s.do(t, req{method: http.MethodPost, path: s.base() + "/task/step", origin: "http://evil.example", form: bad}); w.Code != http.StatusForbidden {
		t.Errorf("tick from another origin: %d", w.Code)
	}
	if strings.Contains(s.get(t, "/task?s=work&id="+parent).Body.String(), "1/2 steps") {
		t.Fatal("a refused tick was written")
	}
	if w := s.post(t, "/task/step", tick); w.Code != http.StatusSeeOther {
		t.Fatalf("tick: %d %s", w.Code, w.Body)
	}
	if !strings.Contains(s.get(t, "/task?s=work&id="+parent).Body.String(), "1/2 steps") ||
		!strings.Contains(s.get(t, "/tasks?s=work").Body.String(), "1/2 steps") {
		t.Error("the tick is not shown")
	}
	if w := s.post(t, "/task/step", url.Values{"s": {"work"}, "id": {parent}, "n": {"9"}, "text": {"Water"}, "done": {"1"}}); w.Code != http.StatusBadRequest {
		t.Errorf("step 9 of 2: %d", w.Code)
	}
	if w := s.post(t, "/task/step", url.Values{"s": {"work"}, "id": {parent}, "n": {"2"}, "done": {"1"}}); w.Code != http.StatusBadRequest {
		t.Errorf("tick without the step's text: %d", w.Code)
	}

	// The parent closes only after its subtask.
	if w := s.post(t, "/task/close", url.Values{"s": {"work"}, "id": {parent}, "result": {"Done"}}); w.Code != http.StatusBadRequest ||
		!strings.Contains(w.Body.String(), "task-buy-seeds") {
		t.Errorf("close with an open subtask: %d", w.Code)
	}
	for _, id := range []string{"task-buy-seeds", parent} {
		if w := s.post(t, "/task/close", url.Values{"s": {"work"}, "id": {id}, "result": {"Done"}}); w.Code != http.StatusSeeOther {
			t.Errorf("close %s: %d %s", id, w.Code, w.Body)
		}
	}
}

// openBrain opens the work space of the fixture, as another writer would (the MCP server).
func openBrain(t *testing.T, cfg string) *brain.Brain {
	t.Helper()
	c, err := config.Load(cfg)
	if err != nil {
		t.Fatal(err)
	}
	b, err := brain.Open(c, c.Spaces["work"])
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// The edit form saves only what the user changed on it: a change made elsewhere after
// the page was shown stays. Emptying the summary or the area removes them; emptying who
// it waits on or the next step of a live task is refused.
func TestEditSavesOnlyWhatChanged(t *testing.T) {
	s, cfg := fixture(t, "en")
	if w := s.post(t, "/task/new", url.Values{"s": {"work"}, "title": {"Fix the fence"}, "summary": {"the north side"}}); w.Code != http.StatusSeeOther {
		t.Fatalf("new: %d", w.Code)
	}
	id := "task-fix-the-fence"
	page := s.get(t, "/task?s=work&id="+id).Body.String()
	for _, want := range []string{`name="orig_next" value="Agree on the definition"`, `name="orig_summary" value="the north side"`,
		`name="orig_area" value="area-general"`, `name="orig_status" value="definition"`, `name="orig_waiting_on" value="me"`} {
		if !strings.Contains(page, want) {
			t.Errorf("the form lacks %s", want)
		}
	}
	// As the page showed it.
	form := func() url.Values {
		return url.Values{"s": {"work"}, "id": {id},
			"status": {"definition"}, "orig_status": {"definition"}, "waiting_on": {"me"}, "orig_waiting_on": {"me"},
			"next": {"Agree on the definition"}, "orig_next": {"Agree on the definition"},
			"summary": {"the north side"}, "orig_summary": {"the north side"}, "area": {"area-general"}, "orig_area": {"area-general"}}
	}
	// Meanwhile, the assistant changes the next step and the status.
	if _, err := openBrain(t, cfg).Update(brain.UpdateInput{ID: id, Next: "Buy posts", Status: "plan"}); err != nil {
		t.Fatal(err)
	}
	f := form()
	f.Set("waiting_on", "Luis")
	if w := s.post(t, "/task/update", f); w.Code != http.StatusSeeOther {
		t.Fatalf("update: %d %s", w.Code, w.Body)
	}
	n, _ := openBrain(t, cfg).Task(id)
	if n.Header.WaitingOn != "Luis" || n.Header.Next != "Buy posts" || n.Header.Status != "plan" || n.Header.Summary != "the north side" {
		t.Errorf("the form undid a change made elsewhere: %+v", n.Header)
	}

	// Emptied: the summary and the area go.
	f = form()
	f.Set("summary", "")
	f.Set("area", "")
	if w := s.post(t, "/task/update", f); w.Code != http.StatusSeeOther {
		t.Fatalf("clear: %d %s", w.Code, w.Body)
	}
	n, _ = openBrain(t, cfg).Task(id)
	if n.Header.Summary != "" || n.Header.Area != "" || n.Header.WaitingOn != "Luis" {
		t.Errorf("after clearing: %+v", n.Header)
	}
	// Who it waits on and the next step cannot be emptied while it is live.
	for _, k := range []string{"waiting_on", "next"} {
		f = form()
		f.Set(k, " ")
		if w := s.post(t, "/task/update", f); w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "cannot be empty") {
			t.Errorf("empty %s: %d", k, w.Code)
		}
	}
	// A form without the originals (an old page) changes what it fills in, as before.
	if w := s.post(t, "/task/update", url.Values{"s": {"work"}, "id": {id}, "next": {"Dig holes"}}); w.Code != http.StatusSeeOther {
		t.Fatalf("plain update: %d", w.Code)
	}
	if n, _ = openBrain(t, cfg).Task(id); n.Header.Next != "Dig holes" || n.Header.WaitingOn != "Luis" {
		t.Errorf("plain update: %+v", n.Header)
	}
}

// A page shown before the plan changed never ticks another step.
func TestStalePageTicksNoOtherStep(t *testing.T) {
	s, cfg := fixture(t, "en")
	s.post(t, "/task/new", url.Values{"s": {"work"}, "title": {"Paint"}})
	id := "task-paint"
	for _, st := range []string{"Sand", "Prime", "Paint"} {
		s.post(t, "/task/step", url.Values{"s": {"work"}, "id": {id}, "step": {st}})
	}
	// The page shows step 2 as "Prime"; meanwhile the plan loses its first step.
	b := openBrain(t, cfg)
	n, _ := b.Task(id)
	cp := *n
	cp.Body = strings.Replace(cp.Body, "- [ ] Sand\n", "", 1)
	if err := memory.Save(b.Mem.Dir, &cp); err != nil {
		t.Fatal(err)
	}
	w := s.post(t, "/task/step", url.Values{"s": {"work"}, "id": {id}, "n": {"2"}, "text": {"Prime"}, "done": {"1"}})
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "no longer says") {
		t.Fatalf("stale tick: %d %s", w.Code, w.Body)
	}
	if n, _ := openBrain(t, cfg).Task(id); strings.Contains(n.Body, "- [x]") {
		t.Fatalf("a stale page ticked a step:\n%s", n.Body)
	}
	// With the right number it works, and done=0 unticks.
	if w := s.post(t, "/task/step", url.Values{"s": {"work"}, "id": {id}, "n": {"1"}, "text": {"Prime"}, "done": {"1"}}); w.Code != http.StatusSeeOther {
		t.Fatalf("tick: %d %s", w.Code, w.Body)
	}
	if n, _ := openBrain(t, cfg).Task(id); !strings.Contains(n.Body, "- [x] Prime") || strings.Contains(n.Body, "- [x] Paint") {
		t.Fatalf("tick:\n%s", n.Body)
	}
}

// Marking a capture as dealt with takes that one out of the inbox only, even when
// another says the same, and finds one whose text starts with a space.
func TestInboxDoneTakesOne(t *testing.T) {
	s, cfg := fixture(t, "en")
	inbox := filepath.Join(filepath.Dir(cfg), "brain-work", "inbox.md")
	start := "- 2026-10-01 — Call Ana\n- 2026-10-02 — Call Ana\n- 2026-10-03 —  spaced out\n"
	if err := os.WriteFile(inbox, []byte(start), 0o644); err != nil {
		t.Fatal(err)
	}
	page := s.get(t, "/inbox?s=work").Body.String()
	if !strings.Contains(page, `name="date" value="2026-10-02"><input type="hidden" name="text" value="Call Ana">`) ||
		!strings.Contains(page, `value=" spaced out"`) {
		t.Fatalf("inbox page:\n%s", page)
	}
	if w := s.post(t, "/inbox/done", url.Values{"s": {"work"}, "date": {"2026-10-02"}, "text": {"Call Ana"}}); w.Code != http.StatusSeeOther {
		t.Fatalf("done: %d %s", w.Code, w.Body)
	}
	if w := s.post(t, "/inbox/done", url.Values{"s": {"work"}, "date": {"2026-10-03"}, "text": {" spaced out"}}); w.Code != http.StatusSeeOther {
		t.Fatalf("done, leading space: %d %s", w.Code, w.Body)
	}
	if data, _ := os.ReadFile(inbox); string(data) != "- 2026-10-01 — Call Ana\n" {
		t.Errorf("inbox:\n%s", data)
	}
	if w := s.post(t, "/inbox/done", url.Values{"s": {"work"}, "date": {"2026-10-02"}, "text": {"Call Ana"}}); w.Code != http.StatusBadRequest {
		t.Errorf("done twice: %d", w.Code)
	}
}

// A capture another space sent shows where it came from, and "done" takes it out.
func TestInboxShowsOrigin(t *testing.T) {
	s, cfg := fixture(t, "en")
	inbox := filepath.Join(filepath.Dir(cfg), "brain-work", "inbox.md")
	start := "- 2026-11-07 — Purge php8.0 (from home)\n- 2026-11-07 — Cable (from Amazon)\n"
	if err := os.WriteFile(inbox, []byte(start), 0o644); err != nil {
		t.Fatal(err)
	}
	page := s.get(t, "/inbox?s=work").Body.String()
	if !strings.Contains(page, "from space home") || strings.Contains(page, "from space Amazon") {
		t.Fatalf("inbox page:\n%s", page)
	}
	if w := s.post(t, "/inbox/done", url.Values{"s": {"work"}, "date": {"2026-11-07"}, "text": {"Purge php8.0"}}); w.Code != http.StatusSeeOther {
		t.Fatalf("done: %d %s", w.Code, w.Body)
	}
	if data, _ := os.ReadFile(inbox); string(data) != "- 2026-11-07 — Cable (from Amazon)\n" {
		t.Errorf("inbox:\n%s", data)
	}
}
