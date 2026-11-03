package panel

import (
	"errors"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"

	"github.com/icalvete/augmented-brain/internal/memory"
)

type noteRow struct {
	ID, Title, Summary, Kind string
	Shared                   bool
}

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

func (s *Server) areaUpdate(v *view, r *http.Request) (string, error) {
	id := field(r, "id")
	if _, err := v.b.UpdateArea(id, field(r, "title"), field(r, "summary")); err != nil {
		return "", err
	}
	return "/note?id=" + url.QueryEscape(id), nil
}

// stamp says when a note last changed on disk, so an open page can tell it is behind.
// It is what the page's script asks for now and then; asking does not count as using
// the panel (see ServeHTTP), so an open tab never keeps it running.
func (s *Server) stamp(w http.ResponseWriter, r *http.Request) {
	v, release, err := s.open(r)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	defer release()
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(v.stampOf(r.FormValue("id"))))
}

// stampOf is the time a note of the space was last written, in nanoseconds; "" when
// there is no such note.
func (v *view) stampOf(id string) string {
	n, ok := v.b.Mem.Notes[id]
	if !ok {
		return ""
	}
	info, err := os.Stat(filepath.Join(v.b.Mem.Dir, filepath.FromSlash(n.Path)))
	if err != nil {
		return ""
	}
	return strconv.FormatInt(info.ModTime().UnixNano(), 10)
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

// list returns the space's notes of those types, then the shared ones (only rules and
// knowledge can be shared).
func (v *view) list(types ...string) []noteRow {
	var rows []noteRow
	for _, m := range []*memory.Memory{v.b.Mem, v.shared} {
		for _, t := range types {
			for _, n := range m.OfType(t) {
				kind := ""
				if len(types) > 1 {
					kind = v.t("kind." + t)
				}
				rows = append(rows, noteRow{n.Header.ID, n.Header.Title, n.Header.Summary, kind, m == v.shared})
			}
		}
	}
	return rows
}

type notesData struct {
	Heading, Hint, Empty string
	Rows                 []noteRow
	NewPerson            bool // the people page: a form to add one
}

func (s *Server) knowledge(v *view, _ *http.Request) (string, page, error) {
	p := v.frame("knowledge", v.t("knowledge"))
	p.D = notesData{v.t("knowledge"), v.t("knowledge.hint"), v.t("knowledge.none"), v.list("reference", "lesson"), false}
	return "notes", p, nil
}

func (s *Server) rules(v *view, _ *http.Request) (string, page, error) {
	p := v.frame("rules", v.t("rules"))
	p.D = notesData{v.t("rules"), v.t("rules.hint"), v.t("rules.none"), v.list("rule"), false}
	return "notes", p, nil
}

func (s *Server) people(v *view, _ *http.Request) (string, page, error) {
	p := v.frame("people", v.t("people"))
	rows := v.list("person")
	// Each person says how many live tasks wait on them.
	for i := range rows {
		if n, ok := v.b.Mem.Notes[rows[i].ID]; ok {
			if w := len(v.person(n).Waiting); w > 0 {
				rows[i].Kind = v.t("people.waiting", w)
			}
		}
	}
	p.D = notesData{v.t("people"), v.t("people.hint"), v.t("people.none"), rows, true}
	return "notes", p, nil
}

// note shows any note but a task (which has its own page), found by id among the
// space's notes, then the shared ones: the id never becomes a path.
func (s *Server) note(v *view, r *http.Request) (string, page, error) {
	id := r.FormValue("id")
	n, ok := v.b.Mem.Notes[id]
	shared := false
	if !ok {
		n, shared = v.shared.Notes[id]
	}
	if n == nil {
		return "", page{}, errors.New(v.t("err.note", id))
	}
	if n.Header.Type == "task" {
		return s.task(v, r)
	}
	section := map[string]string{"area": "areas", "reference": "knowledge", "lesson": "knowledge", "rule": "rules", "person": "people"}[n.Header.Type]
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
	var person *personData
	if n.Header.Type == "person" && !shared {
		d := v.person(n)
		person = &d
	}
	p := v.frame(section, n.Header.Title)
	p.D = struct {
		H                     memory.Header
		Body, Kind, AreaTitle string
		Back                  string
		Tasks                 []taskRow
		Shared                bool
		Sections              []bodySection
		Person                *personData
		Stamp                 string
	}{n.Header, n.Body, v.t("kind." + n.Header.Type), v.areaTitle(n.Header.Area), v.t(section), tasks, shared,
		v.sections(n.Body, false), person, v.stampOf(id)}
	return "note", p, nil
}
