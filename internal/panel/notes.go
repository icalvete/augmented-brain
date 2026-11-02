package panel

import (
	"errors"
	"net/http"
	"net/url"

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
}

func (s *Server) knowledge(v *view, _ *http.Request) (string, page, error) {
	p := v.frame("knowledge", v.t("knowledge"))
	p.D = notesData{v.t("knowledge"), v.t("knowledge.hint"), v.t("knowledge.none"), v.list("reference", "lesson")}
	return "notes", p, nil
}

func (s *Server) rules(v *view, _ *http.Request) (string, page, error) {
	p := v.frame("rules", v.t("rules"))
	p.D = notesData{v.t("rules"), v.t("rules.hint"), v.t("rules.none"), v.list("rule")}
	return "notes", p, nil
}

func (s *Server) people(v *view, _ *http.Request) (string, page, error) {
	p := v.frame("people", v.t("people"))
	p.D = notesData{v.t("people"), v.t("people.hint"), v.t("people.none"), v.list("person")}
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
	p := v.frame(section, n.Header.Title)
	p.D = struct {
		H                     memory.Header
		Body, Kind, AreaTitle string
		Back                  string
		Tasks                 []taskRow
		Shared                bool
	}{n.Header, n.Body, v.t("kind." + n.Header.Type), v.areaTitle(n.Header.Area), v.t(section), tasks, shared}
	return "note", p, nil
}
