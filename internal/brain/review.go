package brain

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/memory"
	"github.com/icalvete/augmented-brain/internal/paths"
)

// ---- inbox ----

// InboxEntry is one capture waiting in the inbox.
type InboxEntry struct {
	Date   string // when it was captured
	Text   string
	Remind string // optional: the day to bring it up again
}

// remindSuffix splits the text of an inbox line ("- 2026-10-17 — text", see
// memory.DatedLine) from the reminder it may end in: " (remind: 2026-10-22)".
var remindSuffix = regexp.MustCompile(`^(.*?)(?: \(remind: (\d{4}-\d{2}-\d{2})\))?\s*$`)

// parseInboxLine reads one line of the inbox; ok is false for a line that is not a
// capture (the heading, a paragraph).
func parseInboxLine(l string) (e InboxEntry, ok bool) {
	date, rest, ok := memory.ParseDatedLine(l)
	if !ok {
		return InboxEntry{}, false
	}
	m := remindSuffix.FindStringSubmatch(rest)
	return InboxEntry{Date: date, Text: m[1], Remind: m[2]}, true
}

func (b *Brain) inboxPath() string { return filepath.Join(b.Mem.Dir, "inbox.md") }

// Inbox returns the captures waiting to be processed, oldest first.
func (b *Brain) Inbox() ([]InboxEntry, error) {
	data, err := os.ReadFile(b.inboxPath())
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []InboxEntry
	for _, l := range strings.Split(string(data), "\n") {
		if e, ok := parseInboxLine(l); ok {
			out = append(out, e)
		}
	}
	return out, nil
}

// InboxDone takes processed captures out of the inbox. Each item is the text of a
// capture, as Inbox returns it (spaces around it do not count); every capture with
// that text goes. It returns the items it could not find.
func (b *Brain) InboxDone(items []string) ([]string, error) {
	data, err := os.ReadFile(b.inboxPath())
	if err != nil {
		return nil, err
	}
	want := map[string]bool{}
	for _, it := range items {
		if it = strings.TrimSpace(it); it != "" {
			want[it] = true
		}
	}
	found := map[string]bool{}
	var keep []string
	for _, l := range strings.Split(string(data), "\n") {
		if e, ok := parseInboxLine(l); ok && want[strings.TrimSpace(e.Text)] {
			found[strings.TrimSpace(e.Text)] = true
			continue
		}
		keep = append(keep, l)
	}
	var missing []string
	for it := range want {
		if !found[it] {
			missing = append(missing, it)
		}
	}
	sort.Strings(missing)
	return missing, paths.WriteAtomic(b.inboxPath(), []byte(strings.Join(keep, "\n")), 0o600)
}

// InboxDoneEntry takes one capture out of the inbox: the first with that date and
// exactly that text, as Inbox returns them. Another capture with the same text stays.
// It reports whether it found one.
func (b *Brain) InboxDoneEntry(date, text string) (bool, error) {
	data, err := os.ReadFile(b.inboxPath())
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	lines := strings.Split(string(data), "\n")
	for i, l := range lines {
		if e, ok := parseInboxLine(l); ok && e.Date == date && e.Text == text {
			lines = append(lines[:i], lines[i+1:]...)
			return true, paths.WriteAtomic(b.inboxPath(), []byte(strings.Join(lines, "\n")), 0o600)
		}
	}
	return false, nil
}

// ---- watch dates ----

func checkDate(d string) error {
	if _, err := memory.ParseDate(d); err != nil {
		return fmt.Errorf("date %q: use YYYY-MM-DD", d)
	}
	return nil
}

// AddWatch adds a date to a task to look at something again.
func (b *Brain) AddWatch(id, date, what string) (*memory.Note, error) {
	return b.Update(UpdateInput{ID: id, WatchDate: date, WatchWhat: what})
}

// WatchDone marks the task's watch dates on that day as done. They are kept.
func (b *Brain) WatchDone(id, date string) (*memory.Note, error) {
	return b.Update(UpdateInput{ID: id, WatchDone: date})
}

// ---- what is due ----

// Reminder is something to bring up today: a task's watch date or an inbox capture
// with a reminder, due within the days asked or already past.
type Reminder struct {
	Date   string
	Days   int // negative when overdue
	What   string
	TaskID string // empty for an inbox capture
}

// Reminders returns what is due within the next `within` days, overdue first.
func (b *Brain) Reminders(within int) []Reminder {
	_, due := b.Pending(within)
	var out []Reminder
	for _, d := range due {
		out = append(out, Reminder{Date: d.Date, Days: d.Days, What: d.What, TaskID: d.TaskID})
	}
	today := b.today()
	entries, _ := b.Inbox()
	for _, e := range entries {
		if e.Remind == "" {
			continue
		}
		if days, ok := memory.DaysBetween(today, e.Remind); ok && days <= within {
			out = append(out, Reminder{Date: e.Remind, Days: days, What: e.Text})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Date < out[j].Date })
	return out
}

// ---- weekly review ----

// StaleDays is how long a live task may go untouched before the review brings it up.
const StaleDays = 14

// ReviewEvery is how many days apart the weekly reviews should be.
const ReviewEvery = 7

// Review is what the weekly review goes through.
type Review struct {
	Last      string // date of the last review, empty if there was none
	Inbox     []InboxEntry
	Stale     []Item // live tasks untouched for StaleDays or more
	Reminders []Reminder
}

// LastReview returns the date of the most recent review, or "".
func (b *Brain) LastReview() string {
	last := ""
	for _, r := range b.Mem.OfType("review") {
		if r.Header.Created > last {
			last = r.Header.Created
		}
	}
	return last
}

// ReviewDue says whether the weekly review is due, and how many days it has been since
// the last one (or since the oldest capture, when there was none). It is not due when
// there is nothing to go through.
func (b *Brain) ReviewDue() (bool, int) {
	r := b.Review()
	if len(r.Inbox) == 0 && len(r.Stale) == 0 {
		return false, 0
	}
	since := r.Last
	if since == "" {
		since = b.today()
		for _, e := range r.Inbox {
			if e.Date < since {
				since = e.Date
			}
		}
		for _, t := range r.Stale {
			if n, ok := b.Mem.Notes[t.ID]; ok && n.Header.Updated < since {
				since = n.Header.Updated
			}
		}
	}
	days, ok := memory.DaysBetween(since, b.today())
	if !ok {
		return false, 0
	}
	return days >= ReviewEvery, days
}

// Review gathers what the weekly review goes through.
func (b *Brain) Review() Review {
	r := Review{Last: b.LastReview(), Reminders: b.Reminders(ReviewEvery)}
	r.Inbox, _ = b.Inbox()
	today := b.today()
	items, _ := b.Pending(0)
	for _, it := range items {
		n := b.Mem.Notes[it.ID]
		if days, ok := memory.DaysBetween(n.Header.Updated, today); ok && days >= StaleDays {
			r.Stale = append(r.Stale, it)
		}
	}
	return r
}

// SaveReview keeps a record of a finished review in reviews/.
func (b *Brain) SaveReview(summary string) (*memory.Note, error) {
	summary = strings.TrimSpace(summary)
	if summary == "" {
		return nil, errors.New("a review needs a summary: what was decided")
	}
	title := i18n.T(b.Language, "review.title")
	title = fmt.Sprintf(title, b.today())
	first := strings.SplitN(summary, "\n", 2)[0]
	if len([]rune(first)) > 120 {
		first = string([]rune(first)[:117]) + "…"
	}
	id := b.newID("review", b.today())
	n := &memory.Note{
		Path:   memory.PathFor("review", id),
		Header: memory.Header{ID: id, Type: "review", Title: title, Summary: first, Created: b.today()},
		Body:   summary + "\n",
	}
	return n, b.save(n)
}
