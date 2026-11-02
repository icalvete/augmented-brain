package panel

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/icalvete/augmented-brain/internal/brain"
	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/memory"
)

func (s *Server) routes() *http.ServeMux {
	m := http.NewServeMux()
	m.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, s.base()+"/tasks", http.StatusSeeOther)
	})
	m.Handle("GET /static/", s.static)
	m.HandleFunc("GET /tasks", s.read("tasks", s.tasks))
	m.HandleFunc("GET /task", s.read("tasks", s.task))
	m.HandleFunc("GET /reminders", s.read("reminders", s.reminders))
	m.HandleFunc("GET /inbox", s.read("inbox", s.inbox))
	m.HandleFunc("GET /areas", s.read("areas", s.areas))
	m.HandleFunc("GET /knowledge", s.read("knowledge", s.knowledge))
	m.HandleFunc("GET /rules", s.read("rules", s.rules))
	m.HandleFunc("GET /people", s.read("people", s.people))
	m.HandleFunc("GET /note", s.read("", s.note))
	m.HandleFunc("GET /profile", s.read("profile", s.profile))
	m.HandleFunc("GET /space", s.read("space", s.space))
	m.HandleFunc("GET /sessions", s.read("sessions", s.sessions))

	m.HandleFunc("POST /task/new", s.write(s.taskNew))
	m.HandleFunc("POST /task/update", s.write(s.taskUpdate))
	m.HandleFunc("POST /task/journal", s.write(s.taskJournal))
	m.HandleFunc("POST /task/park", s.write(s.taskPark))
	m.HandleFunc("POST /task/close", s.write(s.taskClose))
	m.HandleFunc("POST /task/step", s.write(s.taskStep))
	m.HandleFunc("POST /watch/add", s.write(s.watchAdd))
	m.HandleFunc("POST /watch/done", s.write(s.watchDone))
	m.HandleFunc("POST /inbox/capture", s.write(s.capture))
	m.HandleFunc("POST /inbox/done", s.write(s.inboxDone))
	m.HandleFunc("POST /area/new", s.write(s.areaNew))
	m.HandleFunc("POST /profile", s.write(s.profileSave))
	m.HandleFunc("POST /session/recover", s.write(s.sessionRecover))
	m.HandleFunc("POST /session/reviewed", s.write(s.sessionReviewed))
	m.HandleFunc("POST /session/delete", s.write(s.sessionDelete))
	m.HandleFunc("POST /stop", s.handleStop)
	return m
}

// ---- the frame every page shares ----

type link struct{ Path, Label string }

type option struct {
	Value, Label string
	Selected     bool
}

type page struct {
	Lang, Title, Section, Space, CSRF string
	Base                              string // the start of every address: /s/<session>
	Spaces                            []string
	Nav                               []link
	OK                                bool
	Problems                          int
	D                                 any
}

type message struct{ Heading, Text, Back string }

// hidden is one hidden input of a form.
type hidden struct {
	Name  string
	Value any
}

// formFields are the hidden inputs of one form: the CSRF token and the space, which
// every form sends, then the form's own (template "fields" in layout.html writes them).
type formFields struct {
	CSRF, Space string
	Own         []hidden
}

// fields is the template function behind {{template "fields" (fields $ "id" .ID …)}}:
// the page, then the form's own hidden inputs as name and value pairs, in order. The
// template still escapes every value, as it would one written in the form.
func fields(p page, pairs ...any) (formFields, error) {
	if len(pairs)%2 != 0 {
		return formFields{}, errors.New("fields: a name without a value")
	}
	f := formFields{CSRF: p.CSRF, Space: p.Space}
	for i := 0; i < len(pairs); i += 2 {
		name, ok := pairs[i].(string)
		if !ok {
			return formFields{}, fmt.Errorf("fields: the name %v is not a string", pairs[i])
		}
		f.Own = append(f.Own, hidden{name, pairs[i+1]})
	}
	return f, nil
}

// lang is the language of the configuration, or of the environment when it cannot be read.
func (s *Server) lang() string {
	if c, err := config.Load(s.cfgDir); err == nil {
		return c.Language
	}
	return i18n.FromEnv()
}

func (s *Server) render(w http.ResponseWriter, status int, name string, p page) {
	if _, ok := s.pages[p.Lang]; !ok {
		p.Lang = "en"
	}
	p.CSRF, p.Base = s.csrf, s.base()
	for _, sec := range []string{"tasks", "reminders", "inbox", "areas", "knowledge", "rules", "people", "sessions", "space"} {
		p.Nav = append(p.Nav, link{sec, i18n.T(p.Lang, "panel."+sec)})
	}
	var buf bytes.Buffer
	if err := s.pages[p.Lang][name].ExecuteTemplate(&buf, "layout", p); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}

// view is one space, open under the notes lock for the length of a request.
type view struct {
	c      *config.Config
	b      *brain.Brain
	shared *memory.Memory // the notes every space sees: shown, never written here
	lang   string
}

var errNoSpace = errors.New("no such space")

// open takes the notes lock (the same as the MCP server: in this process and among
// every process of the user), reads the configuration and opens the space asked for,
// or the default one.
func (s *Server) open(r *http.Request) (*view, func(), error) {
	release, err := s.writer.Lock()
	if err != nil {
		return nil, nil, err
	}
	c, err := config.Load(s.cfgDir)
	if err != nil {
		release()
		return nil, nil, err
	}
	name := r.FormValue("s")
	if name == "" {
		name = c.DefaultSpace
	}
	if names := c.SpaceNames(); name == "" && len(names) > 0 {
		name = names[0]
	}
	sp, ok := c.Spaces[name]
	if !ok {
		release()
		return &view{c: c, lang: c.Language}, nil, errNoSpace
	}
	b, err := brain.Open(c, sp)
	if err != nil {
		release()
		return &view{c: c, lang: c.Language}, nil, err
	}
	shared, _, err := memory.LoadShared(c.SharedDir())
	if err != nil {
		shared = &memory.Memory{Dir: c.SharedDir(), Notes: map[string]*memory.Note{}}
	}
	return &view{c: c, b: b, shared: shared, lang: c.Language}, release, nil
}

func (v *view) t(key string, args ...any) string { return i18n.T(v.lang, "panel."+key, args...) }

func (v *view) frame(section, title string) page {
	return page{Lang: v.lang, Title: title, Section: section, Space: v.b.Space.Name,
		Spaces: v.c.SpaceNames(), Problems: len(v.b.Problems)}
}

// fail answers with a page that says what went wrong.
func (s *Server) fail(w http.ResponseWriter, r *http.Request, v *view, status int, err error) {
	lang := s.lang()
	p := page{Lang: lang, Section: "tasks"}
	if v != nil {
		lang, p.Lang = v.lang, v.lang
		if v.c != nil {
			p.Spaces = v.c.SpaceNames()
		}
		if v.b != nil {
			p.Space = v.b.Space.Name
		}
	}
	text := err.Error()
	if errors.Is(err, errNoSpace) {
		text = i18n.T(lang, "panel.err.space", r.FormValue("s"))
	}
	p.Title = i18n.T(lang, "panel.err")
	back := ""
	if ref := r.Header.Get("Referer"); strings.HasPrefix(ref, "http://"+r.Host+"/") {
		back = strings.TrimPrefix(ref, "http://"+r.Host)
	}
	p.D = message{Heading: p.Title, Text: text, Back: back}
	s.render(w, status, "message", p)
}

// read serves a page: the handler fills it in from the open space.
func (s *Server) read(section string, h func(*view, *http.Request) (string, page, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		v, release, err := s.open(r)
		if err != nil {
			s.fail(w, r, v, http.StatusBadRequest, err)
			return
		}
		name, p, err := h(v, r)
		release()
		if err != nil {
			s.fail(w, r, v, http.StatusNotFound, err)
			return
		}
		if p.Section == "" {
			p.Section = section
		}
		p.OK = r.URL.Query().Get("ok") == "1"
		s.render(w, http.StatusOK, name, p)
	}
}

// write runs a change and sends the browser to the page it returns, which then shows
// "saved". The index MEMORY.md is rebuilt after every change, not only those that save
// a note (which rebuild it already): the space page promises that an index left stale
// by a hand edit is "rebuilt with the next change". It costs one read when nothing changed.
func (s *Server) write(h func(*view, *http.Request) (string, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		v, release, err := s.open(r)
		if err != nil {
			s.fail(w, r, v, http.StatusBadRequest, err)
			return
		}
		to, err := h(v, r)
		if err == nil {
			_, err = v.b.Mem.WriteIndex(v.lang)
		}
		release()
		if err != nil {
			s.fail(w, r, v, http.StatusBadRequest, err)
			return
		}
		u, _ := url.Parse(to)
		q := u.Query()
		q.Set("s", v.b.Space.Name)
		q.Set("ok", "1")
		u.RawQuery = q.Encode()
		http.Redirect(w, r, s.base()+u.String(), http.StatusSeeOther)
	}
}

func field(r *http.Request, name string) string { return strings.TrimSpace(r.PostForm.Get(name)) }

// required checks that the fields are filled in, and says which one is not.
func required(v *view, r *http.Request, names ...string) error {
	for _, n := range names {
		if field(r, n) == "" {
			return errors.New(v.t("err.required", v.t("f."+n)))
		}
	}
	return nil
}
