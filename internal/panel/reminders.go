package panel

import (
	"errors"
	"net/http"

	"github.com/icalvete/augmented-brain/internal/brain"
)

// reminderDays is how far ahead the reminders page looks.
const reminderDays = 30

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
	if err := brain.CheckDate(field(r, "date")); err != nil {
		return "", onField("date", err)
	}
	if _, err := v.b.AddWatch(id, field(r, "date"), field(r, "what")); err != nil {
		return "", err
	}
	return back(r, id), nil
}

// watchDrop takes out one watch date of a task: the one of that day ("date") that says
// "what", done or not.
func (s *Server) watchDrop(v *view, r *http.Request) (string, error) {
	id := field(r, "id")
	if _, err := v.b.Update(brain.UpdateInput{ID: id, WatchDrop: field(r, "date"), WatchDropWhat: r.PostForm.Get("what")}); err != nil {
		return "", err
	}
	return back(r, id), nil
}

// watchMove moves one watch date of a task ("date", "what") to another day ("to").
func (s *Server) watchMove(v *view, r *http.Request) (string, error) {
	if field(r, "to") == "" {
		return "", onField("to", errors.New(v.t("err.required", v.t("f.date"))))
	}
	if err := brain.CheckDate(field(r, "to")); err != nil {
		return "", onField("to", err)
	}
	id, what := field(r, "id"), r.PostForm.Get("what")
	if field(r, "to") == field(r, "date") {
		return back(r, id), nil
	}
	if _, err := v.b.Update(brain.UpdateInput{ID: id, WatchDrop: field(r, "date"), WatchDropWhat: what,
		WatchDate: field(r, "to"), WatchWhat: what}); err != nil {
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
