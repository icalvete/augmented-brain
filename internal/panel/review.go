package panel

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/icalvete/augmented-brain/internal/brain"
	"github.com/icalvete/augmented-brain/internal/paths"
)

// The weekly review, on a page: each capture of the inbox becomes a task or knowledge,
// or is dropped; each task untouched for two weeks goes on, waits for some day, or is
// opened to close it. What is decided is noted, one line each, in a draft on this
// computer, and becomes the review's record when it is finished.

// draftPath is where this machine keeps the lines of a review under way: in the
// configuration folder, never in the notes, so a review left half-done adds nothing.
func (v *view) draftPath() string {
	return filepath.Join(v.c.Dir, "review-"+v.b.Space.Name+".draft")
}

func (v *view) draft() string {
	b, _ := os.ReadFile(v.draftPath())
	return string(b)
}

func (v *view) note(line string) error {
	return paths.WriteAtomic(v.draftPath(), []byte(v.draft()+"- "+line+"\n"), 0o600)
}

func (s *Server) review(v *view, _ *http.Request) (string, page, error) {
	r := v.b.Review()
	due, days := v.b.ReviewDue()
	var stale []taskRow
	for _, it := range r.Stale {
		row := taskRow{ID: it.ID, Title: it.Title, Status: it.Status, StatusLabel: v.status(it.Status),
			WaitingOn: it.WaitingOn, Next: it.Next, PersonID: v.personLink(it.WaitingOn)}
		if n, ok := v.b.Mem.Notes[it.ID]; ok {
			row.Updated = n.Header.Updated
		}
		stale = append(stale, row)
	}
	var rem []todayReminder
	for _, x := range r.Reminders {
		title := ""
		if n, ok := v.b.Mem.Notes[x.TaskID]; ok {
			title = n.Header.Title
		}
		rem = append(rem, todayReminder{x, title})
	}
	p := v.frame("review", v.t("review"))
	p.D = struct {
		Last, Draft string
		Due         bool
		Days        int
		Inbox       []brain.InboxEntry
		Stale       []taskRow
		Reminders   []todayReminder
		Areas       []option
	}{r.Last, v.draft(), due, days, r.Inbox, stale, rem, v.areaOptions("")}
	return "review", p, nil
}

// reviewCapture deals with one capture: it becomes a task, a reference or a lesson, or
// is dropped; then it leaves the inbox. back=inbox comes back to the inbox page.
func (s *Server) reviewCapture(v *view, r *http.Request) (string, error) {
	date, text := field(r, "date"), r.PostForm.Get("text")
	title := field(r, "title")
	if title == "" {
		title = strings.TrimSpace(text)
	}
	to := "/review"
	if r.PostForm.Get("back") == "inbox" {
		to = "/inbox"
	}
	// The capture must still be there before anything is made of it: a second click,
	// or another tab, must not make a second task.
	entries, err := v.b.Inbox()
	if err != nil {
		return "", err
	}
	if !slices.ContainsFunc(entries, func(e brain.InboxEntry) bool { return e.Date == date && e.Text == text }) {
		return "", errors.New(v.t("err.inbox"))
	}
	var line string
	switch field(r, "as") {
	case "task":
		n, err := v.b.NewTask(brain.NewTaskInput{Title: title, Summary: strings.TrimSpace(text), Area: field(r, "area")})
		if err != nil {
			return "", err
		}
		line = v.t("review.line.task", title, n.Header.ID)
		if to == "/inbox" {
			to = taskPage(n.Header.ID)
		}
	case "reference", "lesson":
		n, err := v.b.NewKnowledge(field(r, "as"), title, "", field(r, "area"), strings.TrimSpace(text))
		if err != nil {
			return "", err
		}
		line = v.t("review.line.knowledge", title, n.Header.ID)
	case "drop":
		line = v.t("review.line.drop", strings.Join(strings.Fields(text), " "))
	default:
		return "", errors.New(v.t("err.required", v.t("f.as")))
	}
	found, err := v.b.InboxDoneEntry(date, text)
	if err != nil {
		return "", err
	}
	if !found {
		return "", errors.New(v.t("err.inbox"))
	}
	if to == "/review" {
		return to, v.note(line)
	}
	return to, nil
}

// reviewTask deals with one task untouched for long: it goes on as it is (a line in
// its journal says it was looked at, which also counts as touching it), or waits for
// some day.
func (s *Server) reviewTask(v *view, r *http.Request) (string, error) {
	id := field(r, "id")
	n, err := v.b.Task(id)
	if err != nil {
		return "", err
	}
	var line string
	switch field(r, "as") {
	case "keep":
		if _, err := v.b.Journal(id, v.t("review.kept")); err != nil {
			return "", err
		}
		line = v.t("review.line.keep", n.Header.Title)
	case "someday":
		next := n.Header.Next
		if next == "" {
			next = v.t("review.someday.next")
		}
		if _, err := v.b.Park(brain.ParkInput{ID: id, Next: next, WaitingOn: n.Header.WaitingOn, Someday: true}); err != nil {
			return "", err
		}
		line = v.t("review.line.someday", n.Header.Title)
	default:
		return "", errors.New(v.t("err.required", v.t("f.as")))
	}
	return "/review", v.note(line)
}

// reviewDone keeps the review in reviews/: what was decided on the page, and what the
// user adds. The draft goes away.
func (s *Server) reviewDone(v *view, r *http.Request) (string, error) {
	summary := strings.TrimSpace(field(r, "summary"))
	if d := strings.TrimSpace(v.draft()); d != "" {
		if summary == "" {
			summary = v.t("review.summary.default")
		}
		summary += "\n\n" + d
	}
	if summary == "" {
		return "", errors.New(v.t("err.required", v.t("f.summary")))
	}
	if _, err := v.b.SaveReview(summary); err != nil {
		return "", err
	}
	_ = os.Remove(v.draftPath())
	return "/today", nil
}
