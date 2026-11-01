package panel

import (
	"bytes"
	"errors"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/icalvete/augmented-brain/internal/backup"
	"github.com/icalvete/augmented-brain/internal/brain"
	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/memory"
	"github.com/icalvete/augmented-brain/internal/state"
	"github.com/icalvete/augmented-brain/internal/version"
)

// reminderDays is how far ahead the reminders page looks.
const reminderDays = 30

// phases a task can be moved to from the panel; parking and closing have their own
// forms, because they need the next step or the result.
var phases = []string{"definition", "analysis", "plan", "execution", "validation"}

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
	m.HandleFunc("GET /people", s.read("people", s.people))
	m.HandleFunc("GET /note", s.read("", s.note))
	m.HandleFunc("GET /profile", s.read("profile", s.profile))
	m.HandleFunc("GET /space", s.read("space", s.space))

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
	Lang, Title, Section, Space, CSRF, Error, Version string
	Base                                              string // the start of every address: /s/<session>
	Spaces                                            []string
	Nav                                               []link
	OK                                                bool
	Problems                                          int
	D                                                 any
}

type message struct{ Heading, Text, Back string }

// lang is the language of the configuration, or of the environment when it cannot be read.
func (s *Server) lang() string {
	if c, err := config.Load(s.cfgDir); err == nil {
		return c.Language
	}
	return i18n.FromEnv()
}

func (s *Server) render(w http.ResponseWriter, _ *http.Request, status int, name string, p page) {
	if _, ok := s.pages[p.Lang]; !ok {
		p.Lang = "en"
	}
	p.CSRF, p.Version, p.Base = s.csrf, version.Version, s.base()
	for _, sec := range []string{"tasks", "reminders", "inbox", "areas", "knowledge", "people", "space"} {
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
	c    *config.Config
	b    *brain.Brain
	lang string
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
	return &view{c: c, b: b, lang: c.Language}, release, nil
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
	s.render(w, r, status, "message", p)
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
		s.render(w, r, http.StatusOK, name, p)
	}
}

// write runs a change and sends the browser to the page it returns, which then shows
// "saved". The index MEMORY.md is rebuilt after every change, as the MCP tools do.
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

// ---- tasks ----

type taskRow struct {
	ID, Title, Status, StatusLabel, WaitingOn, Next, AreaTitle string
	PartOf, Steps                                              string // the parent's title, when not listed under it; "2/5 steps"
	Mine, Sub                                                  bool
}

func (v *view) steps(done, total int) string {
	if total == 0 {
		return ""
	}
	return v.t("steps.count", done, total)
}

func (v *view) status(st string) string { return v.t("status." + st) }

func (v *view) areaTitle(id string) string {
	if n, ok := v.b.Mem.Notes[id]; ok {
		return n.Header.Title
	}
	return id
}

func (v *view) areaOptions(selected string) []option {
	out := []option{{Value: "", Label: v.t("area.none"), Selected: selected == ""}}
	known := selected == ""
	for _, a := range v.b.Mem.OfType("area") {
		out = append(out, option{a.Header.ID, a.Header.Title, a.Header.ID == selected})
		known = known || a.Header.ID == selected
	}
	if !known {
		// An area that is missing still shows, so saving the form does not drop it.
		out = append(out, option{selected, selected, true})
	}
	return out
}

func (s *Server) tasks(v *view, _ *http.Request) (string, page, error) {
	items, _ := v.b.Pending(0)
	type data struct {
		Live, Someday, Closed []taskRow
		Areas                 []option
	}
	var d data
	for _, i := range items {
		row := taskRow{ID: i.ID, Title: i.Title, Status: i.Status, StatusLabel: v.status(i.Status),
			WaitingOn: i.WaitingOn, Next: i.Next, AreaTitle: v.areaTitleOrEmpty(i.Area), Mine: i.WaitingOn == "me",
			Sub: i.Sub, Steps: v.steps(i.StepsDone, i.StepsTotal)}
		if !i.Sub {
			row.PartOf = i.ParentTitle
		}
		d.Live = append(d.Live, row)
	}
	for _, n := range v.b.Mem.OfType("task") {
		row := taskRow{ID: n.Header.ID, Title: n.Header.Title}
		switch n.Header.Status {
		case "someday":
			d.Someday = append(d.Someday, row)
		case "closed":
			d.Closed = append(d.Closed, row)
		}
	}
	def := ""
	if _, ok := v.b.Mem.Notes["area-general"]; ok {
		def = "area-general"
	}
	d.Areas = v.areaOptions(def)
	p := v.frame("tasks", v.t("tasks"))
	p.D = d
	return "tasks", p, nil
}

func (v *view) areaTitleOrEmpty(id string) string {
	if id == "" {
		return ""
	}
	return v.areaTitle(id)
}

func (s *Server) task(v *view, r *http.Request) (string, page, error) {
	n, err := v.b.Task(r.FormValue("id"))
	if err != nil {
		return "", page{}, err
	}
	h := n.Header
	live := h.Status != "closed"
	var ph []option
	if !slices.Contains(phases, h.Status) {
		ph = append(ph, option{h.Status, v.status(h.Status), true})
	}
	for _, p := range phases {
		ph = append(ph, option{p, v.status(p), p == h.Status})
	}
	type step struct {
		memory.Step
		Next string // "1" to tick it, "0" to untick it
	}
	var steps []step
	done := 0
	for _, st := range memory.Steps(n.Body) {
		next := "1"
		if st.Done {
			next = "0"
			done++
		}
		steps = append(steps, step{st, next})
	}
	var kids []taskRow
	for _, c := range v.b.Mem.Children(h.ID) {
		d, t := memory.StepCount(c.Body)
		kids = append(kids, taskRow{ID: c.Header.ID, Title: c.Header.Title, Status: c.Header.Status,
			StatusLabel: v.status(c.Header.Status), WaitingOn: c.Header.WaitingOn, Next: c.Header.Next,
			Mine: c.Header.WaitingOn == "me" && c.Header.Status != "closed", Steps: v.steps(d, t)})
	}
	var parent *taskRow
	if pn := v.b.Mem.ParentOf(n); pn != nil {
		parent = &taskRow{ID: pn.Header.ID, Title: pn.Header.Title}
	}
	p := v.frame("tasks", h.Title)
	p.D = struct {
		H                    memory.Header
		Body, StatusLabel    string
		AreaTitle, StepCount string
		Areas, Phases        []option
		Watch                []memory.Watch
		Steps                []step
		Children             []taskRow
		Parent               *taskRow
		Live, CanHaveSubs    bool
	}{h, n.Body, v.status(h.Status), v.areaTitleOrEmpty(h.Area), v.steps(done, len(steps)), v.areaOptions(h.Area), ph, h.Watch,
		steps, kids, parent, live, live && h.Parent == ""}
	return "task", p, nil
}

func taskPage(id string) string { return "/task?id=" + url.QueryEscape(id) }

func (s *Server) taskNew(v *view, r *http.Request) (string, error) {
	if err := required(v, r, "title"); err != nil {
		return "", err
	}
	n, err := v.b.NewTask(brain.NewTaskInput{Title: field(r, "title"), Summary: field(r, "summary"),
		Area: field(r, "area"), Parent: field(r, "parent")})
	if err != nil {
		return "", err
	}
	return taskPage(n.Header.ID), nil
}

// editable are the fields of the task form. Each comes with its value as the page
// showed it (orig_<name>): only what the user changed is saved, so the form never
// undoes a change made elsewhere since the page was shown.
var editable = []string{"status", "waiting_on", "next", "summary", "area"}

func (s *Server) taskUpdate(v *view, r *http.Request) (string, error) {
	id := field(r, "id")
	n, err := v.b.Task(id)
	if err != nil {
		return "", err
	}
	in := brain.UpdateInput{ID: id}
	for _, f := range editable {
		val := field(r, f)
		if !r.PostForm.Has(f) || r.PostForm.Has("orig_"+f) && val == field(r, "orig_"+f) {
			continue // not changed on the page
		}
		if val == "" {
			if f == "status" {
				continue
			}
			// The user emptied it. A live task always says who it waits on and what
			// comes next: "nobody" is the word for no one.
			if (f == "waiting_on" || f == "next") && n.Header.Status != "closed" {
				return "", errors.New(v.t("err.empty", v.t(map[string]string{"waiting_on": "waits", "next": "next"}[f])))
			}
			in.Clear = append(in.Clear, f)
			continue
		}
		switch f {
		case "status":
			if val == n.Header.Status {
				continue
			}
			if !slices.Contains(phases, val) {
				return "", errors.New(v.t("err.status", val))
			}
			in.Status = val
		case "waiting_on":
			in.WaitingOn = val
		case "next":
			in.Next = val
		case "summary":
			in.Summary = val
		case "area":
			in.Area = val
		}
	}
	if _, err := v.b.Update(in); err != nil {
		return "", err
	}
	return taskPage(id), nil
}

func (s *Server) taskJournal(v *view, r *http.Request) (string, error) {
	if err := required(v, r, "entry"); err != nil {
		return "", err
	}
	id := field(r, "id")
	if _, err := v.b.Journal(id, field(r, "entry")); err != nil {
		return "", err
	}
	return taskPage(id), nil
}

// released tells the machine's state that the task is no longer the active one, as the
// MCP tools do when they park or close it; it never fails the change.
func released(v *view, id string) {
	_ = state.Released(v.b.ConfigDir, v.b.Space.Name, id, time.Now())
}

func (s *Server) taskPark(v *view, r *http.Request) (string, error) {
	if err := required(v, r, "next"); err != nil {
		return "", err
	}
	id := field(r, "id")
	if _, err := v.b.Park(brain.ParkInput{ID: id, Next: field(r, "next"), WaitingOn: field(r, "waiting_on"),
		Essentials: field(r, "essentials"), Someday: r.PostForm.Get("someday") == "1"}); err != nil {
		return "", err
	}
	released(v, id)
	return taskPage(id), nil
}

func (s *Server) taskClose(v *view, r *http.Request) (string, error) {
	if err := required(v, r, "result"); err != nil {
		return "", err
	}
	id := field(r, "id")
	if _, err := v.b.Close(id, field(r, "result")); err != nil {
		return "", err
	}
	released(v, id)
	return taskPage(id), nil
}

// taskStep adds a step to the task's Plan (field "step", its text), or ticks or unticks
// one (fields "n", its number, "text", its text as the page showed it, and "done", 1 or
// 0). The text must still be that of step n: a page shown before the plan changed
// never ticks another step.
func (s *Server) taskStep(v *view, r *http.Request) (string, error) {
	id := field(r, "id")
	if r.PostForm.Has("step") {
		if err := required(v, r, "step"); err != nil {
			return "", err
		}
		_, err := v.b.AddStep(id, field(r, "step"))
		return taskPage(id) + "#steps", err
	}
	n, err := strconv.Atoi(field(r, "n"))
	if err != nil || n < 1 {
		return "", errors.New(v.t("err.required", v.t("f.step")))
	}
	if field(r, "text") == "" {
		return "", errors.New(v.t("err.required", v.t("f.step")))
	}
	_, err = v.b.Step(brain.StepInput{ID: id, Number: n, Expect: field(r, "text"), Done: field(r, "done") == "1"})
	return taskPage(id) + "#steps", err
}

// ---- reminders ----

func (s *Server) reminders(v *view, _ *http.Request) (string, page, error) {
	type due struct {
		brain.Reminder
		TaskTitle string
	}
	var d struct {
		Within int
		Due    []due
		Tasks  []option
	}
	d.Within = reminderDays
	for _, x := range v.b.Reminders(reminderDays) {
		title := ""
		if n, ok := v.b.Mem.Notes[x.TaskID]; ok {
			title = n.Header.Title
		}
		d.Due = append(d.Due, due{x, title})
	}
	for _, n := range v.b.Mem.OfType("task") {
		if n.Header.Status != "closed" {
			d.Tasks = append(d.Tasks, option{Value: n.Header.ID, Label: n.Header.Title})
		}
	}
	p := v.frame("reminders", v.t("reminders"))
	p.D = d
	return "reminders", p, nil
}

func back(r *http.Request, id string) string {
	if r.PostForm.Get("back") == "reminders" {
		return "/reminders"
	}
	return taskPage(id)
}

func (s *Server) watchAdd(v *view, r *http.Request) (string, error) {
	if err := required(v, r, "date", "what"); err != nil {
		return "", err
	}
	id := field(r, "id")
	if _, err := v.b.AddWatch(id, field(r, "date"), field(r, "what")); err != nil {
		return "", err
	}
	return back(r, id), nil
}

func (s *Server) watchDone(v *view, r *http.Request) (string, error) {
	id := field(r, "id")
	if _, err := v.b.WatchDone(id, field(r, "date")); err != nil {
		return "", err
	}
	return back(r, id), nil
}

// ---- inbox ----

func (s *Server) inbox(v *view, _ *http.Request) (string, page, error) {
	entries, err := v.b.Inbox()
	if err != nil {
		return "", page{}, err
	}
	p := v.frame("inbox", v.t("inbox"))
	p.D = struct{ Entries []brain.InboxEntry }{entries}
	return "inbox", p, nil
}

func (s *Server) capture(v *view, r *http.Request) (string, error) {
	if err := required(v, r, "text"); err != nil {
		return "", err
	}
	return "/inbox", v.b.Capture(field(r, "text"), field(r, "remind_on"))
}

// inboxDone takes one capture out of the inbox: the one with this date and text (as
// sent, spaces included), even when another says the same.
func (s *Server) inboxDone(v *view, r *http.Request) (string, error) {
	found, err := v.b.InboxDoneEntry(field(r, "date"), r.PostForm.Get("text"))
	if err != nil {
		return "", err
	}
	if !found {
		return "", errors.New(v.t("err.inbox"))
	}
	return "/inbox", nil
}

// ---- areas, knowledge, people ----

type noteRow struct{ ID, Title, Summary, Kind string }

func (s *Server) areas(v *view, _ *http.Request) (string, page, error) {
	type row struct {
		ID, Title, Summary string
		Count              int
	}
	count := map[string]int{}
	for _, t := range v.b.Mem.OfType("task") {
		if t.Header.Status != "closed" {
			count[t.Header.Area]++
		}
	}
	var rows []row
	for _, a := range v.b.Mem.OfType("area") {
		rows = append(rows, row{a.Header.ID, a.Header.Title, a.Header.Summary, count[a.Header.ID]})
	}
	p := v.frame("areas", v.t("areas"))
	p.D = struct{ Rows []row }{rows}
	return "areas", p, nil
}

func (s *Server) areaNew(v *view, r *http.Request) (string, error) {
	if err := required(v, r, "title"); err != nil {
		return "", err
	}
	n, err := v.b.NewArea(field(r, "title"), field(r, "summary"))
	if err != nil {
		return "", err
	}
	return "/note?id=" + url.QueryEscape(n.Header.ID), nil
}

func (v *view) list(types ...string) []noteRow {
	var rows []noteRow
	for _, t := range types {
		for _, n := range v.b.Mem.OfType(t) {
			kind := ""
			if len(types) > 1 {
				kind = v.t("kind." + t)
			}
			rows = append(rows, noteRow{n.Header.ID, n.Header.Title, n.Header.Summary, kind})
		}
	}
	return rows
}

type notesData struct {
	Heading, Hint, Empty string
	Rows                 []noteRow
}

func (s *Server) knowledge(v *view, _ *http.Request) (string, page, error) {
	p := v.frame("knowledge", v.t("knowledge"))
	p.D = notesData{v.t("knowledge"), v.t("knowledge.hint"), v.t("knowledge.none"), v.list("reference", "lesson")}
	return "notes", p, nil
}

func (s *Server) people(v *view, _ *http.Request) (string, page, error) {
	p := v.frame("people", v.t("people"))
	p.D = notesData{v.t("people"), v.t("people.hint"), v.t("people.none"), v.list("person")}
	return "notes", p, nil
}

// note shows any note but a task (which has its own page), found by id among the
// space's notes only: the id never becomes a path.
func (s *Server) note(v *view, r *http.Request) (string, page, error) {
	id := r.FormValue("id")
	n, ok := v.b.Mem.Notes[id]
	if !ok {
		return "", page{}, errors.New(v.t("err.note", id))
	}
	if n.Header.Type == "task" {
		return s.task(v, r)
	}
	section := map[string]string{"area": "areas", "reference": "knowledge", "lesson": "knowledge", "person": "people"}[n.Header.Type]
	if section == "" {
		section = "space"
	}
	var tasks []taskRow
	if n.Header.Type == "area" {
		for _, t := range v.b.Mem.OfType("task") {
			if t.Header.Area == id {
				tasks = append(tasks, taskRow{ID: t.Header.ID, Title: t.Header.Title, StatusLabel: v.status(t.Header.Status)})
			}
		}
	}
	p := v.frame(section, n.Header.Title)
	p.D = struct {
		H                     memory.Header
		Body, Kind, AreaTitle string
		Back                  string
		Tasks                 []taskRow
	}{n.Header, n.Body, v.t("kind." + n.Header.Type), v.areaTitleOrEmpty(n.Header.Area), v.t(section), tasks}
	return "note", p, nil
}

// ---- profile ----

func (s *Server) profile(v *view, _ *http.Request) (string, page, error) {
	in, err := brain.LoadProfile(v.b.ConfigDir)
	if err != nil {
		return "", page{}, err
	}
	p := v.frame("profile", v.t("profile"))
	p.D = struct {
		P    brain.ProfileInput
		Path string
	}{in, filepath.Join(v.b.ConfigDir, brain.ProfileFile)}
	return "profile", p, nil
}

func (s *Server) profileSave(v *view, r *http.Request) (string, error) {
	if err := required(v, r, "name"); err != nil {
		return "", err
	}
	// Browsers send \r\n in text areas; notes use \n.
	nl := strings.NewReplacer("\r\n", "\n")
	_, err := v.b.SaveProfile(brain.ProfileInput{Name: field(r, "name"),
		About: nl.Replace(field(r, "about")), WorkStyle: nl.Replace(field(r, "work_style"))})
	return "/profile", err
}

// ---- the space's health ----

func (s *Server) space(v *view, _ *http.Request) (string, page, error) {
	sp := v.b.Space
	var problems []string
	for _, pr := range v.b.Problems {
		problems = append(problems, pr.Error())
	}
	current, _ := os.ReadFile(filepath.Join(sp.Memory, memory.IndexFile))
	due, _ := v.b.ReviewDue()
	d := struct {
		Memory, Backup, BackupTime, BackupError, BackupWarning, LastReview string
		WorkFolders, Problems                                              []string
		Notes                                                              int
		IndexStale, BackupKnown, BackupOK, ReviewDue, IsDefault            bool
	}{Memory: sp.Memory, Backup: sp.Backup, LastReview: v.b.LastReview(), WorkFolders: sp.WorkFolders,
		Problems: problems, Notes: len(v.b.Mem.Notes), IndexStale: string(current) != v.b.Mem.Index(v.lang),
		ReviewDue: due, IsDefault: v.c.DefaultSpace == sp.Name}
	if st, ok := backup.LastStatus(v.c.Dir, sp.Name); ok {
		d.BackupKnown, d.BackupOK, d.BackupError, d.BackupWarning = true, st.OK, st.Error, st.Warning
		d.BackupTime = st.Time.Local().Format("2006-01-02 15:04")
	}
	p := v.frame("space", v.t("space.title", sp.Name))
	p.D = d
	return "space", p, nil
}
