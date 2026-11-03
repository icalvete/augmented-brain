package panel

import (
	"errors"
	"net/http"
	"net/url"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/icalvete/augmented-brain/internal/brain"
	"github.com/icalvete/augmented-brain/internal/memory"
)

// phases a task can be moved to from the panel; parking and closing have their own
// forms, because they need the next step or the result.
var phases = []string{"definition", "analysis", "plan", "execution", "validation"}

type taskRow struct {
	ID, Title, Status, StatusLabel, WaitingOn, Next, AreaTitle string
	PartOf, Steps                                              string // the parent's title, when not listed under it; "2/5 steps"
	Mine, Sub                                                  bool
	Stale                                                      int    // days untouched, when the weekly review would bring it up
	PersonID                                                   string // the person "waits on" names, when one does
	Updated                                                    string // the date of the last change, where it is shown
}

func (v *view) steps(done, total int) string {
	if total == 0 {
		return ""
	}
	return v.t("steps.count", done, total)
}

func (v *view) status(st string) string { return v.t("status." + st) }

// areaTitle is the title of an area, its id when the area is missing (so the page still
// says which), and empty for no area.
func (v *view) areaTitle(id string) string {
	if id == "" {
		return ""
	}
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

// taskGroup is the live tasks of one area, in the order of the pending report.
type taskGroup struct {
	ID, Title string
	Tasks     []taskRow
}

// filter is one of the ways to narrow the task list, with its address.
type filter struct {
	Key, Label string
	On         bool
}

func (s *Server) tasks(v *view, r *http.Request) (string, page, error) {
	items, _ := v.b.Pending(0)
	type data struct {
		Groups          []taskGroup
		Filters, People []filter
		Shown, Total    int
		Someday, Closed []taskRow
		Areas           []option
	}
	var d data
	f := r.FormValue("f")     // me, active, parked, stale
	who := r.FormValue("who") // a waiting_on value
	today := time.Now().Format(memory.DateLayout)
	groups := map[string]*taskGroup{}
	var order []string
	waits := map[string]bool{}
	group := ""
	for _, i := range items {
		row := taskRow{ID: i.ID, Title: i.Title, Status: i.Status, StatusLabel: v.status(i.Status),
			WaitingOn: i.WaitingOn, Next: i.Next, AreaTitle: v.areaTitle(i.Area), Mine: i.WaitingOn == "me",
			Sub: i.Sub, Steps: v.steps(i.StepsDone, i.StepsTotal), PersonID: v.personLink(i.WaitingOn)}
		if !i.Sub {
			row.PartOf = i.ParentTitle
			group = i.Area
		}
		if n, ok := v.b.Mem.Notes[i.ID]; ok {
			if days, ok := memory.DaysBetween(n.Header.Updated, today); ok && days >= brain.StaleDays {
				row.Stale = days
			}
		}
		if w := i.WaitingOn; w != "" && w != "me" && w != "nobody" && !strings.HasPrefix(w, "task-") {
			waits[w] = true
		}
		d.Total++
		keep := true
		switch f {
		case "me":
			keep = row.Mine
		case "active":
			keep = i.Status != "parked"
		case "parked":
			keep = i.Status == "parked"
		case "stale":
			keep = row.Stale > 0
		}
		if who != "" {
			keep = keep && i.WaitingOn == who
		}
		if !keep {
			continue
		}
		d.Shown++
		g := groups[group]
		if g == nil {
			title := v.areaTitle(group)
			if title == "" {
				title = v.t("area.none")
			}
			g = &taskGroup{ID: group, Title: title}
			groups[group] = g
			order = append(order, group)
		}
		g.Tasks = append(g.Tasks, row)
	}
	// Areas in the order of the index (their titles), tasks with no area last.
	sort.SliceStable(order, func(a, b int) bool {
		if (order[a] == "") != (order[b] == "") {
			return order[b] == ""
		}
		return groups[order[a]].Title < groups[order[b]].Title
	})
	for _, id := range order {
		d.Groups = append(d.Groups, *groups[id])
	}
	for _, k := range []string{"", "me", "active", "parked", "stale"} {
		d.Filters = append(d.Filters, filter{k, v.t("filter." + map[string]string{"": "all"}[k] + k), f == k && who == ""})
	}
	var names []string
	for w := range waits {
		names = append(names, w)
	}
	sort.Strings(names)
	for _, w := range names {
		d.People = append(d.People, filter{w, w, who == w})
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
	// The first section (the essentials) goes under the facts; the rest, below the steps.
	// Text before the first heading, if any, goes with them.
	essentials, rest := v.sections(n.Body, true), []bodySection(nil)
	cut := 1
	if len(essentials) > 1 && essentials[0].Title == "" {
		cut = 2
	}
	if len(essentials) > cut {
		essentials, rest = essentials[:cut], essentials[cut:]
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
		MissingRepos         []string
		Essentials, Sections []bodySection
		PersonID, Stamp      string
	}{h, n.Body, v.status(h.Status), v.areaTitle(h.Area), v.steps(done, len(steps)), v.areaOptions(h.Area), ph, h.Watch,
		steps, kids, parent, live, live && h.Parent == "", v.b.MissingRepos(n), essentials, rest, v.personLink(h.WaitingOn), v.stampOf(h.ID)}
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

// changed is the value of a field of the task form, and whether the user changed it.
// Each field comes with its value as the page showed it (orig_<name>): only what the
// user changed is saved, so the form never undoes a change made elsewhere since the
// page was shown.
func changed(r *http.Request, name string) (string, bool) {
	val := field(r, name)
	if !r.PostForm.Has(name) || r.PostForm.Has("orig_"+name) && val == field(r, "orig_"+name) {
		return "", false
	}
	return val, true
}

// textField is a text field of the task form, besides the status.
type textField struct {
	name string
	// label names the field when a live task cannot leave it empty; "" when it can.
	label string
	set   func(*brain.UpdateInput, string)
}

// textFields are the task form's text fields, in the order they are saved.
var textFields = []textField{
	{"waiting_on", "waits", func(in *brain.UpdateInput, v string) { in.WaitingOn = v }},
	{"next", "next", func(in *brain.UpdateInput, v string) { in.Next = v }},
	{"summary", "", func(in *brain.UpdateInput, v string) { in.Summary = v }},
	{"area", "", func(in *brain.UpdateInput, v string) { in.Area = v }},
}

func (s *Server) taskUpdate(v *view, r *http.Request) (string, error) {
	id := field(r, "id")
	n, err := v.b.Task(id)
	if err != nil {
		return "", err
	}
	in := brain.UpdateInput{ID: id}
	// The status is never emptied, and only moves to one of the phases.
	if val, ok := changed(r, "status"); ok && val != "" && val != n.Header.Status {
		if !slices.Contains(phases, val) {
			return "", errors.New(v.t("err.status", val))
		}
		in.Status = val
	}
	for _, f := range textFields {
		val, ok := changed(r, f.name)
		switch {
		case !ok:
		case val != "":
			f.set(&in, val)
		case f.label != "" && n.Header.Status != "closed":
			// The user emptied it. A live task always says who it waits on and what
			// comes next: "nobody" is the word for no one.
			return "", errors.New(v.t("err.empty", v.t(f.label)))
		default:
			in.Clear = append(in.Clear, f.name)
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

func (s *Server) taskPark(v *view, r *http.Request) (string, error) {
	if err := required(v, r, "next"); err != nil {
		return "", err
	}
	id := field(r, "id")
	if _, err := v.b.Park(brain.ParkInput{ID: id, Next: field(r, "next"), WaitingOn: field(r, "waiting_on"),
		Essentials: field(r, "essentials"), Someday: r.PostForm.Get("someday") == "1"}); err != nil {
		return "", err
	}
	v.b.Released(id)
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
	v.b.Released(id)
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
