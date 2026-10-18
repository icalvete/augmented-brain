package brain

import (
	"strings"
	"testing"
	"time"

	"github.com/icalvete/augmented-brain/internal/memory"
)

func TestInboxRemindAndDone(t *testing.T) {
	b := open(t, "en")
	if err := b.Capture("call the insurance", "2026-10-16"); err != nil {
		t.Fatal(err)
	}
	if err := b.Capture("look at the new runner", ""); err != nil {
		t.Fatal(err)
	}
	if err := b.Capture("bad date", "next thursday"); err == nil {
		t.Error("a remind date that is not YYYY-MM-DD must be refused")
	}
	entries, err := b.Inbox()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[0].Text != "call the insurance" || entries[0].Remind != "2026-10-16" || entries[1].Remind != "" {
		t.Fatalf("inbox = %+v", entries)
	}

	r := b.Reminders(1)
	if len(r) != 1 || r[0].What != "call the insurance" || r[0].Days != 1 || r[0].TaskID != "" {
		t.Errorf("reminders = %+v", r)
	}

	missing, err := b.InboxDone([]string{"call the insurance", "never captured"})
	if err != nil {
		t.Fatal(err)
	}
	if len(missing) != 1 || missing[0] != "never captured" {
		t.Errorf("missing = %v", missing)
	}
	inbox := read(t, b, "inbox.md")
	if strings.Contains(inbox, "insurance") || !strings.Contains(inbox, "new runner") {
		t.Errorf("inbox after review:\n%s", inbox)
	}
	if !strings.Contains(inbox, "## ") {
		t.Errorf("the inbox lost its heading:\n%s", inbox)
	}
}

func TestWatchDates(t *testing.T) {
	b := open(t, "en")
	task, _ := b.NewTask(NewTaskInput{Title: "Certificates"})
	id := task.Header.ID
	if _, err := b.AddWatch(id, "2026-11-16", "renewed?"); err != nil {
		t.Fatal(err)
	}
	if _, err := b.AddWatch(id, "2026-10-14", "ordered?"); err != nil {
		t.Fatal(err)
	}
	if _, err := b.AddWatch(id, "16/11", "x"); err == nil {
		t.Error("a bad date must be refused")
	}
	n, _ := b.Task(id)
	if len(n.Header.Watch) != 2 || n.Header.Watch[0].Date != "2026-10-14" {
		t.Errorf("watch = %+v (kept in date order)", n.Header.Watch)
	}

	r := b.Reminders(1)
	if len(r) != 1 || r[0].Days != -1 || r[0].TaskID != id {
		t.Errorf("reminders = %+v (the overdue one only)", r)
	}
	if _, err := b.WatchDone(id, "2026-10-14"); err != nil {
		t.Fatal(err)
	}
	if r := b.Reminders(1); len(r) != 0 {
		t.Errorf("a done watch date still comes up: %+v", r)
	}
	if _, err := b.WatchDone(id, "2026-10-14"); err == nil {
		t.Error("closing it twice must say there is nothing open")
	}
}

func TestWeeklyReview(t *testing.T) {
	b := open(t, "es")
	if due, _ := b.ReviewDue(); due {
		t.Error("nothing to go through: the review is not due")
	}
	old, _ := b.NewTask(NewTaskInput{Title: "Vieja"})
	old.Header.Updated = "2026-09-20"
	if err := b.writeWithoutTouching(old); err != nil {
		t.Fatal(err)
	}
	if _, err := b.NewTask(NewTaskInput{Title: "Nueva"}); err != nil {
		t.Fatal(err)
	}
	if err := b.Capture("idea suelta", ""); err != nil {
		t.Fatal(err)
	}

	r := b.Review()
	if r.Last != "" || len(r.Inbox) != 1 || len(r.Stale) != 1 || r.Stale[0].ID != old.Header.ID {
		t.Errorf("review = %+v", r)
	}
	if due, days := b.ReviewDue(); !due || days != 25 {
		t.Errorf("due = %v, %d days (since the oldest untouched task)", due, days)
	}

	n, err := b.SaveReview("La idea suelta pasa a tarea.\nLa vieja, a algún día.")
	if err != nil {
		t.Fatal(err)
	}
	if n.Path != "reviews/review-2026-10-15.md" || n.Header.Title != "Revisión semanal 2026-10-15" || n.Header.Summary != "La idea suelta pasa a tarea." {
		t.Errorf("review note = %s %+v", n.Path, n.Header)
	}
	if b.LastReview() != "2026-10-15" {
		t.Errorf("last review = %q", b.LastReview())
	}
	if due, _ := b.ReviewDue(); due {
		t.Error("just reviewed: not due")
	}
	b.Now = func() time.Time { return time.Date(2026, 10, 22, 9, 0, 0, 0, time.UTC) }
	if due, days := b.ReviewDue(); !due || days != 7 {
		t.Errorf("a week later: due = %v, %d days", due, days)
	}
	if _, err := b.SaveReview("  "); err == nil {
		t.Error("a review without a summary must be refused")
	}
}

// writeWithoutTouching saves a note keeping its updated date, to fake an old one.
func (b *Brain) writeWithoutTouching(n *memory.Note) error {
	b.Mem.Notes[n.Header.ID] = n
	return memory.Save(b.Mem.Dir, n)
}
