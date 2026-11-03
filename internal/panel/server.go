package panel

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"embed"
	"encoding/hex"
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
			"tf":     func(key string, args ...any) string { return i18n.T(l, "panel."+key, args...) },
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
	pid := os.Getpid()
	if err := writeInfo(cfgDir, Info{PID: pid, Port: port, Token: s.token, Started: time.Now()}); err != nil {
		ln.Close()
		return err
	}
	defer removeInfo(cfgDir, pid)
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
	own := "http://" + r.Host
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
	// its own name as Host.
	if !sameHost(r.Host, s.port) {
		forbidden(w)
		return
	}
	if r.URL.Path == "/ping" && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
		s.ping(w, r)
		return
	}
	r, browser, bare := s.inSession(r)
	if bare {
		http.Redirect(w, r, s.base()+"/", http.StatusSeeOther)
		return
	}
	bearer := !browser && s.bearer(r)
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

func (s *Server) handleStop(w http.ResponseWriter, r *http.Request) {
	lang := s.lang()
	s.render(w, http.StatusOK, "message", page{Lang: lang, Title: i18n.T(lang, "panel.stopped"), Section: "tasks",
		D: message{Heading: i18n.T(lang, "panel.stopped"), Text: i18n.T(lang, "panel.stopped.text")}})
	s.stop.Do(func() { close(s.stopped) })
}
