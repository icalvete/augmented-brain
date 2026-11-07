package panel

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"embed"
	"encoding/hex"
	"fmt"
	"html/template"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/icalvete/augmented-brain/internal/backup"
	"github.com/icalvete/augmented-brain/internal/doctor"
	"github.com/icalvete/augmented-brain/internal/filelock"
	"github.com/icalvete/augmented-brain/internal/i18n"
)

//go:embed web
var webFS embed.FS

// maxBody is the largest request body the panel reads: forms of a few lines.
const maxBody = 64 << 10

// Server is the panel of one configuration folder, for one run.
type Server struct {
	cfgDir  string
	port    int
	token   string // in the info file: the command's key (Authorization header, ping proof)
	session string // in every address of the browser: /s/<session>/...
	csrf    string // in every form, checked on every POST from the browser
	writer  *filelock.Writer
	pages   map[string]map[string]*template.Template // language → page → template
	static  http.Handler
	mux     *http.ServeMux
	last    atomic.Int64 // when the last authorized request came, in Unix nanoseconds
	stopped chan struct{}
	stop    sync.Once

	diagnose func(lang string) []doctor.Check // the Help page's health check; nil: doctor.Run

	// remote is the panel published to the user's Tailscale network; nil: local only.
	// It can change while the panel runs (POST /remote, POST /local), and every request
	// reads it: hence atomic.
	remote atomic.Pointer[Remote]
	// publish finds what the panel needs to answer through the network (tailnet.Status),
	// for POST /remote; nil: this panel cannot go remote while running.
	publish func() (*Remote, error)
	pid     int
	started time.Time
}

// Remote is the panel published to the user's own Tailscale network (`panel remote`).
// Requests that come through it carry Host and the login of whoever sent them, which
// Tailscale itself sets (and strips when a request brings its own): only Owner gets in.
type Remote struct {
	Host  string `json:"host"`  // the Host header of the network address, "box.example.ts.net:8443"
	Owner string `json:"owner"` // the login of the machine's owner in the network
	// Unpublish takes the address back when the panel stops; nil: nothing to take back.
	Unpublish func() error `json:"-"`
}

// OwnerHeader is the header Tailscale puts the sender's login in.
const OwnerHeader = "Tailscale-User-Login"

// viaRemote reports whether r came through the network address.
func (s *Server) viaRemote(r *http.Request) bool {
	rem := s.remote.Load()
	return rem != nil && r.Host == rem.Host
}

// origin is the scheme and host the browser sees for r: https through the network,
// http on 127.0.0.1.
func (s *Server) origin(r *http.Request) string {
	if s.viaRemote(r) {
		return "https://" + r.Host
	}
	return "http://" + r.Host
}

func random() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// New builds the panel of cfgDir, answering on port. Writes take the notes lock of
// cfgDir, the same one the MCP server takes.
func New(cfgDir string, port int) (*Server, error) {
	s := &Server{cfgDir: cfgDir, port: port, writer: filelock.NewWriter(filepath.Join(cfgDir, backup.NotesLock)),
		stopped: make(chan struct{})}
	var err error
	for _, v := range []*string{&s.token, &s.csrf} {
		if *v, err = random(); err != nil {
			return nil, err
		}
	}
	s.session = sessionOf(s.token)
	if s.pages, err = parsePages(); err != nil {
		return nil, err
	}
	static, err := fs.Sub(webFS, "web/static")
	if err != nil {
		return nil, err
	}
	s.static = http.StripPrefix("/static/", http.FileServer(http.FS(static)))
	s.mux = s.routes()
	s.last.Store(time.Now().UnixNano())
	return s, nil
}

// Token is this run's token.
func (s *Server) Token() string { return s.token }

// base is the start of every address of the panel's pages.
func (s *Server) base() string { return "/s/" + s.session }

func parsePages() (map[string]map[string]*template.Template, error) {
	names, err := fs.Glob(webFS, "web/*.html")
	if err != nil {
		return nil, err
	}
	out := map[string]map[string]*template.Template{}
	for _, l := range []string{"en", "es"} {
		funcs := template.FuncMap{
			"t": func(key string) string { return i18n.T(l, "panel."+key) },
			// inline renders one line of a note (a next step, a summary) with its marks.
			"inline": func(text string) template.HTML { return inlineHTML(text) },
			"inc":    func(n int) int { return n + 1 },
			"tf":     func(key string, args ...any) string { return i18n.T(l, "panel."+key, args...) },
			// tc says a count with its noun: key.1 for one, key.n for the rest.
			"tc": func(key string, n int) string { return i18n.Count(l, "panel."+key, n) },
			"days": func(d int) string {
				switch {
				case d == 0:
					return i18n.T(l, "panel.days.today")
				case d == 1:
					return i18n.T(l, "panel.days.tomorrow")
				case d == -1:
					return i18n.T(l, "panel.days.yesterday")
				case d < 0:
					return i18n.T(l, "panel.days.late", -d)
				default:
					return i18n.T(l, "panel.days.in", d)
				}
			},
			"fields": fields,
		}
		out[l] = map[string]*template.Template{}
		for _, n := range names {
			base := strings.TrimSuffix(filepath.Base(n), ".html")
			if base == "layout" {
				continue
			}
			t, err := template.New(base).Funcs(funcs).ParseFS(webFS, "web/layout.html", n)
			if err != nil {
				return nil, err
			}
			out[l][base] = t
		}
	}
	return out, nil
}

// Serve runs the panel of cfgDir until it is stopped, ctx ends or nobody has used it
// for idle. It listens on 127.0.0.1 only, on a free port, and writes InfoFile so the
// command can find it; the file goes away when it stops.
func Serve(ctx context.Context, cfgDir string, idle time.Duration) error {
	return ServeRemote(ctx, cfgDir, idle, nil)
}

// ServeRemote is Serve that also answers through the user's Tailscale network when
// remote is not nil, and takes that address back (remote.Unpublish) when it stops.
func ServeRemote(ctx context.Context, cfgDir string, idle time.Duration, remote *Remote) error {
	return ServeWith(ctx, cfgDir, idle, remote, nil)
}

// ServeWith is ServeRemote with publish, which lets the running panel go remote when
// `panel remote` asks it to (POST /remote), keeping its key and the pages already open.
func ServeWith(ctx context.Context, cfgDir string, idle time.Duration, remote *Remote, publish func() (*Remote, error)) error {
	if remote != nil && remote.Unpublish != nil {
		// Taken back however the panel stops, even before it listens.
		defer func() { _ = remote.Unpublish() }()
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	port := ln.Addr().(*net.TCPAddr).Port
	s, err := New(cfgDir, port)
	if err != nil {
		ln.Close()
		return err
	}
	s.remote.Store(remote)
	s.publish, s.pid, s.started = publish, os.Getpid(), time.Now()
	if err := s.writeInfo(); err != nil {
		ln.Close()
		return err
	}
	defer removeInfo(cfgDir, s.pid)
	defer func() {
		// A panel that went remote while running takes its address back too.
		if r := s.remote.Load(); r != nil && r != remote && r.Unpublish != nil {
			_ = r.Unpublish()
		}
	}()
	srv := &http.Server{
		Handler:           s,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       time.Minute,
		MaxHeaderBytes:    32 << 10,
	}
	errc := make(chan error, 1)
	go func() { errc <- srv.Serve(ln) }()
	tick := time.NewTicker(min(idle/10+time.Millisecond, time.Minute))
	defer tick.Stop()
	for running := true; running; {
		select {
		case <-ctx.Done():
			running = false
		case <-s.stopped:
			running = false
		case err := <-errc:
			return err
		case <-tick.C:
			if time.Since(time.Unix(0, s.last.Load())) > idle {
				running = false
			}
		}
	}
	shut, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return srv.Shutdown(shut)
}

func equal(a, b string) bool { return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1 }

// bearer reports whether the request carries this run's token in an Authorization
// header: the command itself, which no web page can send without a CORS preflight
// this server never answers.
func (s *Server) bearer(r *http.Request) bool {
	a, found := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	return found && equal(a, s.token)
}

// inSession reports whether the request is for a page of this run, under
// /s/<session>/, and returns it with that prefix taken off the path; bare says the
// address was the panel's root without its final slash. The session is in the
// address, not in a cookie: cookies of 127.0.0.1 are shared by every port, so any
// other local server would receive it.
func (s *Server) inSession(r *http.Request) (out *http.Request, ok, bare bool) {
	rest, found := strings.CutPrefix(r.URL.Path, "/s/")
	if !found {
		return r, false, false
	}
	seg, after, slash := strings.Cut(rest, "/")
	if !equal(seg, s.session) {
		return r, false, false
	}
	if !slash {
		return r, true, true
	}
	r2 := new(http.Request)
	*r2 = *r
	u := *r.URL
	u.Path, u.RawPath = "/"+after, ""
	r2.URL = &u
	return r2, true, false
}

// ping answers the command's challenge (?c=, 64 hex digits) with the proof that this
// server knows the token. It needs no token: it gives nothing away.
func (s *Server) ping(w http.ResponseWriter, r *http.Request) {
	c := r.URL.Query().Get("c")
	if _, err := hex.DecodeString(c); err != nil || len(c) != 64 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	w.Header().Set("X-Augmented-Brain", "panel")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(proof(s.token, c) + "\n"))
}

// sameOrigin reports whether a POST comes from one of the panel's own pages: its
// Origin header, or its Referer when there is no Origin.
func (s *Server) sameOrigin(r *http.Request) bool {
	own := s.origin(r)
	if o := r.Header.Get("Origin"); o != "" {
		return o == own
	}
	ref := r.Header.Get("Referer")
	return ref != "" && strings.HasPrefix(ref, own+"/")
}

func forbidden(w http.ResponseWriter) { http.Error(w, "forbidden", http.StatusForbidden) }

// ServeHTTP checks every request before it reaches a page: the host, the session in
// the address or the token, the method, and for writes where they come from.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h := w.Header()
	h.Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self'; connect-src 'self'; form-action 'self'; base-uri 'none'; frame-ancestors 'none'")
	h.Set("X-Content-Type-Options", "nosniff")
	// same-origin, not no-referrer: with no-referrer, browsers send "Origin: null" on
	// form posts, and the panel could not tell its own forms from anyone else's. Nothing
	// leaves for another site either way: no page loads anything outside, and the Help
	// page's links out (rel="noreferrer" too) never send the address, which holds the key.
	h.Set("Referrer-Policy", "same-origin")
	h.Set("X-Frame-Options", "DENY")
	h.Set("Cross-Origin-Opener-Policy", "same-origin")
	h.Set("Cross-Origin-Resource-Policy", "same-origin")
	h.Set("Cache-Control", "no-store")

	// DNS rebinding: a page of another site whose name now points to 127.0.0.1 sends
	// its own name as Host. Through the network, the name is the machine's own, and the
	// sender must be its owner: someone else in the network (a shared machine, a guest)
	// does not get in, even with the address.
	remote := s.viaRemote(r)
	if rem := s.remote.Load(); remote && (rem == nil || !strings.EqualFold(r.Header.Get(OwnerHeader), rem.Owner)) {
		forbidden(w)
		return
	}
	if !remote && !sameHost(r.Host, s.port) {
		forbidden(w)
		return
	}
	if !remote && r.URL.Path == "/ping" && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
		s.ping(w, r)
		return
	}
	r, browser, bare := s.inSession(r)
	if bare {
		http.Redirect(w, r, s.base()+"/", http.StatusSeeOther)
		return
	}
	bearer := !browser && !remote && s.bearer(r) // the command talks to 127.0.0.1 only
	if !browser && !bearer {
		forbidden(w)
		return
	}
	// The open pages' check for changes is not a person using the panel: it must
	// not keep it running.
	if r.URL.Path != "/stamp" {
		s.last.Store(time.Now().UnixNano())
	}
	switch r.Method {
	case http.MethodGet, http.MethodHead:
	case http.MethodPost:
		if !s.sameOrigin(r) {
			forbidden(w)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxBody)
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if !bearer && !equal(r.PostForm.Get("csrf"), s.csrf) {
			forbidden(w)
			return
		}
	default:
		w.Header().Set("Allow", "GET, HEAD, POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.mux.ServeHTTP(w, r)
}

// writeInfo writes InfoFile with what the panel is now.
func (s *Server) writeInfo() error {
	return writeInfo(s.cfgDir, Info{PID: s.pid, Port: s.port, Token: s.token, Started: s.started, Remote: s.remote.Load()})
}

// commandOnly reports whether r is the command itself: the token in its header, on
// 127.0.0.1. Through the network it never is, even with the token: `tailscale serve`
// reaches the panel from 127.0.0.1 too, so the address alone proves nothing.
func (s *Server) commandOnly(r *http.Request) bool { return s.bearer(r) && !s.viaRemote(r) }

// handleRemote makes the running panel answer through the user's Tailscale network as
// well (`panel remote` with a panel already open): same process, same key, so the
// pages open in the browser go on working. What the network needs (the machine's name
// and its owner) the server finds itself; nothing in the request is trusted for it.
// The command publishes the address afterwards, and asks POST /local if that fails.
func (s *Server) handleRemote(w http.ResponseWriter, r *http.Request) {
	if !s.commandOnly(r) {
		forbidden(w)
		return
	}
	if s.remote.Load() != nil {
		w.WriteHeader(http.StatusOK)
		return
	}
	if s.publish == nil {
		http.Error(w, "this panel cannot go remote while running", http.StatusNotImplemented)
		return
	}
	rem, err := s.publish()
	if err != nil || rem == nil {
		http.Error(w, fmt.Sprint("tailscale: ", err), http.StatusServiceUnavailable)
		return
	}
	s.remote.Store(rem)
	if err := s.writeInfo(); err != nil {
		s.remote.Store(nil)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// handleLocal takes the network back from the panel: the undo of handleRemote when
// publishing the address failed. The address itself is the command's to take back.
func (s *Server) handleLocal(w http.ResponseWriter, r *http.Request) {
	if !s.commandOnly(r) {
		forbidden(w)
		return
	}
	s.remote.Store(nil)
	if err := s.writeInfo(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleStop(w http.ResponseWriter, r *http.Request) {
	lang := s.lang()
	s.render(w, http.StatusOK, "message", page{Lang: lang, Title: i18n.T(lang, "panel.stopped"), Section: "tasks",
		D: message{Heading: i18n.T(lang, "panel.stopped"), Text: i18n.T(lang, "panel.stopped.text")}})
	s.stop.Do(func() { close(s.stopped) })
}
