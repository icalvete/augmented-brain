package panel

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/icalvete/augmented-brain/internal/confighistory"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/icalvete/augmented-brain/internal/brain"
	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/memory"
)

func (s *Server) routes() *http.ServeMux {
	m := http.NewServeMux()
	m.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, s.base(r)+"/today", http.StatusSeeOther)
	})
	m.Handle("GET /static/", s.static)
	s.page(m, "/today", "today", s.today)
	s.page(m, "/tasks", "tasks", s.tasks)
	s.page(m, "/task", "tasks", s.task)
	s.page(m, "/reminders", "reminders", s.reminders)
	s.page(m, "/inbox", "inbox", s.inbox)
	s.page(m, "/areas", "areas", s.areas)
	s.page(m, "/knowledge", "knowledge", s.knowledge)
	s.page(m, "/rules", "rules", s.rules)
	s.page(m, "/people", "people", s.people)
	s.page(m, "/note", "", s.note)
	m.HandleFunc("GET /stamp", s.stamp)
	s.page(m, "/search", "search", s.search)
	s.page(m, "/review", "review", s.review)
	s.page(m, "/secrets", "secrets", s.secrets)
	s.page(m, "/profile", "profile", s.profile)
	s.page(m, "/space", "space", s.space)
	s.page(m, "/backup", "backup", s.backupPage)
	s.page(m, "/backup/back", "backup", s.backupBackPage)
	s.page(m, "/sessions", "sessions", s.sessions)
	s.page(m, "/repos", "repos", s.repos)
	s.page(m, "/guard", "guard", s.guardPage)
	s.page(m, "/versions", "versions", s.versionsPage)
	s.page(m, "/help", "help", s.help)
	m.HandleFunc("GET /help/report", s.helpReport)

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
	m.HandleFunc("POST /area/update", s.write(s.areaUpdate))
	m.HandleFunc("POST /person", s.write(s.personSave))
	m.HandleFunc("POST /person/card", s.personCard)
	m.HandleFunc("POST /secret", s.write(s.secretSave))
	m.HandleFunc("POST /secret/forget", s.write(s.secretForget))
	m.HandleFunc("POST /review/capture", s.write(s.reviewCapture))
	m.HandleFunc("POST /review/task", s.write(s.reviewTask))
	m.HandleFunc("POST /review/done", s.write(s.reviewDone))
	m.HandleFunc("POST /profile", s.write(s.profileSave))
	m.HandleFunc("POST /session/recover", s.write(s.sessionRecover))
	m.HandleFunc("POST /session/reviewed", s.write(s.sessionReviewed))
	m.HandleFunc("POST /session/delete", s.write(s.sessionDelete))
	m.HandleFunc("POST /repo/describe", s.write(s.repoDescribe))
	m.HandleFunc("POST /guard/list", s.write(s.localGuard(s.guardList)))
	m.HandleFunc("POST /guard/root", s.write(s.localGuard(s.guardRootAsk)))
	m.HandleFunc("POST /guard/accept", s.write(s.localGuard(s.guardAccept)))
	m.HandleFunc("POST /guard/model", s.write(s.localGuard(s.guardModel)))
	m.HandleFunc("POST /guard/trust", s.write(s.localGuard(s.guardTrust)))
	m.HandleFunc("POST /versions/restore", s.write(s.versionsRestore))
	m.HandleFunc("POST /hints", s.hintsChange)
	m.HandleFunc("POST /backup/run", s.write(s.backupRun))
	m.HandleFunc("POST /backup/setup", s.backupSetup)
	m.HandleFunc("POST /backup/back", s.backupBack)
	m.HandleFunc("POST /stop", s.handleStop)
	m.HandleFunc("POST /remote", s.handleRemote)
	m.HandleFunc("POST /local", s.handleLocal)
	m.HandleFunc("POST /ticket", s.handleTicket)
	return m
}

// ---- the frame every page shares ----

type link struct{ Path, Label string }

// navGroup is one block of the menu: a heading and its sections.
type navGroup struct {
	Label string
	Links []link
}

// menu is the panel's menu, in blocks: what a day brings, the notes, and looking after
// the panel and its copies. Each name is a page's address and its "panel.<name>" label.
var menu = []struct {
	key      string
	sections []string
}{
	{"nav.day", []string{"today", "tasks", "sessions", "reminders", "inbox", "review"}},
	{"nav.notes", []string{"areas", "knowledge", "rules", "people", "repos", "secrets"}},
	{"nav.care", []string{"backup", "space", "guard", "versions", "help"}},
}

type option struct {
	Value, Label string
	Selected     bool
}

type page struct {
	Lang, Title, Section, Space, CSRF string
	Base                              string // the start of every address: /s/<session>
	Spaces                            []string
	Nav                               []navGroup
	Where                             string     // the menu button's text on a phone: this page's name
	OK                                string     // what the change just made did ("Task updated.")
	Form                              *formState // a form that could not be saved, shown again
	Remote                            bool       // also open through the user's Tailscale network
	Problems                          int
	Hints                             hints  // which introductions the person hid
	Here                              string // this page's address after Base, to come back to
	D                                 any
}

type message struct{ Heading, Text, Back string }

// formState is a form that could not be saved, on the page it was sent from: what was
// typed goes back into it, and the error next to the field it is about.
type formState struct {
	ID     string     // the form's action after Base, then "#" and its "form" input when it has one
	Field  string     // the field the error is about; "" for the form as a whole
	Text   string     // the error, in the panel's language
	Values url.Values // what was sent
	shown  bool       // a template put the error next to its form
	Top    bool       // no template did: it goes at the top of the page
}

// fieldError is an error about one field of a form.
type fieldError struct {
	Field string
	Err   error
}

func (e *fieldError) Error() string { return e.Err.Error() }
func (e *fieldError) Unwrap() error { return e.Err }

// onField says err is about the field name of the form, unless it already names one.
func onField(name string, err error) error {
	var fe *fieldError
	if err == nil || errors.As(err, &fe) {
		return err
	}
	return &fieldError{name, err}
}

func (p page) failed(form string) bool { return p.Form != nil && p.Form.ID == form }

// Val is what a field of the form shows: what was typed, when this form could not be
// saved, and def otherwise.
func (p page) Val(form, name string, def any) any {
	if p.failed(form) {
		return p.Form.Values.Get(name)
	}
	return def
}

// On says whether a choice of the form (a box, a radio button, an option) is on: as it
// was sent, when this form could not be saved, and def otherwise.
func (p page) On(form, name, value string, def bool) bool {
	if p.failed(form) {
		for _, v := range p.Form.Values[name] {
			if v == value {
				return true
			}
		}
		return false
	}
	return def
}

// Err is the error of the form, for name "", or of one of its fields; "" when there is
// none there. The form's own place takes an error about a field no template shows.
func (p page) Err(form, name string) string {
	if !p.failed(form) || p.Form.Field != name {
		return ""
	}
	p.Form.shown = true
	return p.Form.Text
}

// FormErr is the error of the form as a whole, written at its end: its own, or one
// about a field no Err of the form has shown.
func (p page) FormErr(form string) string {
	if !p.failed(form) || p.Form.shown {
		return ""
	}
	p.Form.shown = true
	return p.Form.Text
}

// Open says whether a form that may be folded away (in a <details>) shows open: when
// it could not be saved, so its error is in sight.
func (p page) Open(form string) bool { return p.failed(form) }

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

func (s *Server) render(w http.ResponseWriter, r *http.Request, status int, name string, p page) {
	if _, ok := s.pages[p.Lang]; !ok {
		p.Lang = "en"
	}
	p.CSRF, p.Base, p.Remote = s.csrfFor(r), s.base(r), s.remote.Load() != nil
	p.Hints = loadHints(s.cfgDir)
	p.Nav, p.Where = nil, ""
	for _, g := range menu {
		group := navGroup{Label: i18n.T(p.Lang, "panel."+g.key)}
		for _, sec := range g.sections {
			group.Links = append(group.Links, link{sec, i18n.T(p.Lang, "panel."+sec)})
		}
		p.Nav = append(p.Nav, group)
	}
	p.Where = whereLabel(p.Lang, p.Section)
	var buf bytes.Buffer
	if err := s.pages[p.Lang][name].ExecuteTemplate(&buf, "layout", p); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if p.Form != nil && !p.Form.shown {
		// The page has no place for this form's error: it goes at the top.
		p.Form.Top = true
		buf.Reset()
		if err := s.pages[p.Lang][name].ExecuteTemplate(&buf, "layout", p); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}

// whereLabel is the name of the page a section belongs to, as the menu button shows it
// on a phone, or "" for a page outside the menu (the button then just says Menu).
func whereLabel(lang, section string) string {
	switch section {
	case "profile", "search":
		return i18n.T(lang, "panel."+section)
	}
	for _, g := range menu {
		for _, sec := range g.sections {
			if sec == section {
				return i18n.T(lang, "panel."+sec)
			}
		}
	}
	return ""
}

// view is one space, open under the notes lock for the length of a request.
type view struct {
	c      *config.Config
	b      *brain.Brain
	shared *memory.Memory // the notes every space sees: shown, never written here
	lang   string
	// unlock lets the notes go before the page is done: a page that then runs slow
	// commands (git, the keychain) no longer makes Claude wait. Calling it again, or
	// the handler's own release after it, does nothing.
	unlock func()
	// cardErr is the detail of a person's form that could not be saved, to show next to it.
	cardErr *brain.CardError
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
	once := sync.OnceFunc(release)
	return &view{c: c, b: b, shared: shared, lang: c.Language, unlock: once}, once, nil
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
	text := i18n.Message(lang, err)
	if errors.Is(err, errNoSpace) {
		text = i18n.T(lang, "panel.err.space", r.FormValue("s"))
	}
	p.Title = i18n.T(lang, "panel.err")
	back := ""
	if ref, own := r.Header.Get("Referer"), s.origin(r); strings.HasPrefix(ref, own+"/") {
		back = strings.TrimPrefix(ref, own)
	}
	p.D = message{Heading: p.Title, Text: text, Back: back}
	s.render(w, r, status, "message", p)
}

// readRoute is a page: its section and the handler that fills it in.
type readRoute struct {
	section string
	h       func(*view, *http.Request) (string, page, error)
}

// page serves a page at path, and keeps it so a form sent from it that cannot be
// saved can show it again (again).
func (s *Server) page(m *http.ServeMux, path, section string, h func(*view, *http.Request) (string, page, error)) {
	if s.reads == nil {
		s.reads = map[string]readRoute{}
	}
	s.reads[path] = readRoute{section, h}
	m.HandleFunc("GET "+path, s.read(section, h))
}

// read serves a page: the handler fills it in from the open space.
func (s *Server) read(section string, h func(*view, *http.Request) (string, page, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) { s.servePage(w, r, readRoute{section, h}, nil) }
}

// servePage fills in a page and sends it. form, when not nil, is a form of the page that
// could not be saved: the page goes back with what was typed and the error.
func (s *Server) servePage(w http.ResponseWriter, r *http.Request, rt readRoute, form *formState) {
	v, release, err := s.open(r)
	if err != nil {
		s.fail(w, r, v, http.StatusBadRequest, err)
		return
	}
	var name string
	var p page
	func() {
		defer release() // a handler that fails half way never keeps the notes locked
		name, p, err = rt.h(v, r)
	}()
	if err != nil {
		s.fail(w, r, v, http.StatusNotFound, err)
		return
	}
	if p.Section == "" {
		p.Section = rt.section
	}
	status := http.StatusOK
	if form != nil {
		p.Form, status = form, http.StatusBadRequest
	}
	if ok := r.URL.Query().Get("ok"); ok != "" && form == nil {
		p.OK = savedLine(v.lang, ok)
	}
	q := r.URL.Query()
	q.Del("ok")
	p.Here = (&url.URL{Path: r.URL.Path, RawQuery: q.Encode()}).String()
	s.render(w, r, status, name, p)
}

// failForm answers a form that could not be saved: its page again when there is one
// (again), a page of its own otherwise (fail). v may be nil.
func (s *Server) failForm(w http.ResponseWriter, r *http.Request, v *view, err error) {
	lang := s.lang()
	if v != nil {
		lang = v.lang
	}
	if !s.again(w, r, lang, err) {
		s.fail(w, r, v, http.StatusBadRequest, err)
	}
}

// again answers a form that could not be saved with the page it was sent from, again:
// what was typed stays in the form, and the error, in the panel's language, goes next
// to the field it is about. False when the page is not one of this panel's (no Referer,
// or another address): the error then gets a page of its own (fail).
func (s *Server) again(w http.ResponseWriter, r *http.Request, lang string, err error) bool {
	if errors.Is(err, errNoSpace) {
		return false
	}
	ref, own := r.Header.Get("Referer"), s.origin(r)+s.base(r)
	if !strings.HasPrefix(ref, own+"/") {
		return false
	}
	u, perr := url.Parse(strings.TrimPrefix(ref, own))
	if perr != nil {
		return false
	}
	rt, ok := s.reads[u.Path]
	if !ok {
		return false
	}
	q := u.Query()
	q.Del("ok")
	g := r.Clone(r.Context())
	g.Method, g.Body, g.ContentLength = http.MethodGet, http.NoBody, 0
	g.URL = &url.URL{Path: u.Path, RawQuery: q.Encode()}
	g.Form, g.PostForm = nil, nil
	id := r.URL.Path
	if k := r.PostForm.Get("form"); k != "" {
		id += "#" + k
	}
	f := &formState{ID: id, Text: i18n.Message(lang, err), Values: r.PostForm}
	var fe *fieldError
	if errors.As(err, &fe) {
		f.Field = fe.Field
	}
	s.servePage(w, g, rt, f)
	return true
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
		var to string
		func() {
			defer release()
			_, _ = confighistory.Record(s.cfgDir, confighistory.ByOutside, "")
			to, err = h(v, r)
			_, _ = confighistory.Record(s.cfgDir, confighistory.ByPanel, r.URL.Path)
			if err == nil {
				_, err = v.b.Mem.WriteIndex(v.lang)
			}
		}()
		if err != nil {
			s.failForm(w, r, v, err)
			return
		}
		u, _ := url.Parse(to)
		q := u.Query()
		q.Set("s", v.b.Space.Name)
		q.Set("ok", doneKey(r))
		u.RawQuery = q.Encode()
		http.Redirect(w, r, s.base(r)+u.String(), http.StatusSeeOther)
	}
}

// doneKey names what a change did, for the page it goes to: its address, and add or
// remove when the form says which ("task.update", "guard.root.add").
func doneKey(r *http.Request) string {
	k := strings.ReplaceAll(strings.Trim(r.URL.Path, "/"), "/", ".")
	if op := r.PostForm.Get("op"); op == "add" || op == "remove" {
		k += "." + op
	}
	return k
}

// savedLine is the line that says what a change did. An address with ok=1, or one the
// catalogue does not know, gets the plain "Saved.".
func savedLine(lang, key string) string {
	if k := "panel.ok." + key; i18n.Has(k) {
		return i18n.T(lang, k)
	}
	return i18n.T(lang, "panel.saved")
}

func field(r *http.Request, name string) string { return strings.TrimSpace(r.PostForm.Get(name)) }

// required checks that the fields are filled in, and says which one is not.
func required(v *view, r *http.Request, names ...string) error {
	for _, n := range names {
		if field(r, n) == "" {
			return &fieldError{n, errors.New(v.t("err.required", v.t("f."+n)))}
		}
	}
	return nil
}
