package brain

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
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
	From   string // the space it was sent from (capture with space); "" when captured here
}

// full is the text as the line holds it, origin included: what an earlier program,
// which does not know origins, reads as the text.
func (e InboxEntry) full() string {
	if e.From == "" {
		return e.Text
	}
	return e.Text + fromMark(e.From)
}

// fromMark is how a capture sent from another space ends, before any reminder.
func fromMark(space string) string { return " (from " + space + ")" }

// splitFrom splits the origin off a capture's text: "text (from personal)" gives
// "text" and "personal"; ok is false when the text does not end so. The origin is a
// word of letters, digits, '_', '.' and '-'. Only the name of a space of the
// configuration counts, which the caller checks: "(from Amazon)" in the user's own
// words stays text. A text of more than one line never has one. (It was the regular
// expression `^(.*) \(from ([A-Za-z0-9_.-]+)\)$`, which costs far more on long lines.)
func splitFrom(text string) (rest, space string, ok bool) {
	if !strings.HasSuffix(text, ")") || strings.Contains(text, "\n") {
		return "", "", false
	}
	i := strings.LastIndex(text, " (from ")
	if i < 0 {
		return "", "", false
	}
	name := text[i+len(" (from ") : len(text)-1]
	if name == "" || strings.IndexFunc(name, func(r rune) bool { return !spaceNameRune(r) }) >= 0 {
		return "", "", false
	}
	return text[:i], name, true
}

func spaceNameRune(r rune) bool {
	return r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '.' || r == '-'
}

// inlineSpace are the blanks a line may end in that splitRemind drops.
const inlineSpace = " \t\n\f\r"

// splitRemind splits the text of an inbox line ("- 2026-10-17 — text", see
// memory.DatedLine) from the reminder it may end in: " (remind: 2026-10-22)". Blanks at
// the end go; blanks before the reminder stay in the text. ok is false for a text that
// goes on after a line break. (It was the regular expression
// `^(.*?)(?: \(remind: (\d{4}-\d{2}-\d{2})\))?\s*$`, which backtracks on long lines.)
func splitRemind(text string) (rest, date string, ok bool) {
	t := strings.TrimRight(text, inlineSpace)
	if strings.Contains(t, "\n") {
		return "", "", false
	}
	const open, mark = " (remind: ", len(" (remind: 2026-10-22)")
	if len(t) >= mark && strings.HasSuffix(t, ")") && t[len(t)-mark:len(t)-mark+len(open)] == open {
		if d := t[len(t)-mark+len(open) : len(t)-1]; isDate(d) {
			return t[:len(t)-mark], d, true
		}
	}
	return t, "", true
}

// isDate says s has the shape 2026-10-22 (digits, not a check of the calendar).
func isDate(s string) bool {
	if len(s) != 10 {
		return false
	}
	for i := range len(s) {
		if i == 4 || i == 7 {
			if s[i] != '-' {
				return false
			}
		} else if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// parseInboxLine reads one line of the inbox; ok is false for a line that is not a
// capture (the heading, a paragraph).
func parseInboxLine(l string, spaces map[string]bool) (e InboxEntry, ok bool) {
	date, rest, ok := memory.ParseDatedLine(l)
	if !ok {
		return InboxEntry{}, false
	}
	text, remind, _ := splitRemind(rest) // a line of the file has no line break
	e = InboxEntry{Date: date, Text: memory.OneLine(text), Remind: remind}
	if t, from, ok := splitFrom(e.Text); ok && spaces[from] {
		e.Text, e.From = t, from
	}
	return e, true
}

// ownWords keeps text written by the user or the assistant from passing for the marks
// the program adds at the end of a line: " (from <space>)", which says another space
// sent it, and, in a capture, " (remind: <date>)". Such an ending, typed as text, keeps
// its words between square brackets instead, which nothing reads as a mark.
func (b *Brain) ownWords(text string, capture bool) string {
	spaces := b.spaceSet()
	for range 4 { // each pass undoes one ending; more would be text made on purpose
		if t, date, ok := splitRemind(text); capture && ok && date != "" {
			text = t + " [remind: " + date + "]"
			continue
		}
		if t, from, ok := splitFrom(text); ok && spaces[from] {
			text = t + " [from " + from + "]"
			continue
		}
		break
	}
	return text
}

func (b *Brain) inboxPath() string { return filepath.Join(b.Mem.Dir, "inbox.md") }

// Inbox returns the captures waiting to be processed, oldest first. The session start
// asks for it several times (captures arrived, reminders, the weekly review): it reads
// the file each time, and reads its lines again only when the file changed.
func (b *Brain) Inbox() ([]InboxEntry, error) {
	data, err := os.ReadFile(b.inboxPath())
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	b.inboxMu.Lock()
	defer b.inboxMu.Unlock()
	if b.inboxRead && b.inboxData == string(data) {
		return slices.Clone(b.inbox), nil
	}
	spaces := b.spaceSet()
	var out []InboxEntry
	for _, l := range strings.Split(string(data), "\n") {
		if e, ok := parseInboxLine(l, spaces); ok {
			out = append(out, e)
		}
	}
	b.inboxData, b.inbox, b.inboxRead = string(data), out, true
	return slices.Clone(out), nil
}

// Arrived are the captures other spaces sent to this one (CaptureTo), oldest first,
// still in the inbox.
func (b *Brain) Arrived() []InboxEntry {
	all, _ := b.Inbox()
	var out []InboxEntry
	for _, e := range all {
		if e.From != "" {
			out = append(out, e)
		}
	}
	return out
}

// InboxDone takes processed captures out of the inbox. Each item is the text of a
// capture, as Inbox returns it (spaces around it do not count); every capture with
// that text goes. It returns the items it could not find.
func (b *Brain) InboxDone(items []string) ([]string, error) {
	p, err := b.planInboxDone(items)
	if err != nil {
		return nil, err
	}
	return p.missing, p.write()
}

// inboxDone is what InboxDone would do, worked out before anything is written.
type inboxDone struct {
	path    string
	keep    []string     // the lines that stay
	gone    []InboxEntry // the captures that go, in the inbox's order
	missing []string     // the items no capture matched
}

// write takes the captures out of the inbox, when there are any to take.
func (p inboxDone) write() error {
	if len(p.gone) == 0 {
		return nil
	}
	return paths.WriteAtomicNoLink(p.path, []byte(strings.Join(p.keep, "\n")), 0o600)
}

func (b *Brain) planInboxDone(items []string) (inboxDone, error) {
	p := inboxDone{path: b.inboxPath()}
	want := map[string]bool{}
	for _, it := range items {
		if it = memory.OneLine(it); it != "" {
			want[it] = true
		}
	}
	data, err := os.ReadFile(p.path)
	if errors.Is(err, os.ErrNotExist) {
		for it := range want {
			p.missing = append(p.missing, it) // no inbox: nothing to take out
		}
		sort.Strings(p.missing)
		return p, nil
	}
	if err != nil {
		return p, err
	}
	lines := strings.Split(string(data), "\n")
	// A capture sent from another space is found by its text with or without the
	// origin (an earlier program sends it with). Without it, it is only taken when no
	// capture made here says exactly that, so asking for "X" never takes "X" and
	// "X (from personal)" at once.
	plain := map[string]bool{}
	for _, l := range lines {
		if e, ok := parseInboxLine(l, b.spaceSet()); ok && e.From == "" {
			plain[strings.TrimSpace(e.Text)] = true
		}
	}
	found := map[string]bool{}
	for _, l := range lines {
		if e, ok := parseInboxLine(l, b.spaceSet()); ok {
			text, full := strings.TrimSpace(e.Text), strings.TrimSpace(e.full())
			key := ""
			switch {
			case e.From == "" && want[text]:
				key = text
			case e.From != "" && want[full]:
				key = full
			case e.From != "" && want[text] && !plain[text]:
				key = text
			}
			if key != "" {
				found[key] = true
				p.gone = append(p.gone, e)
				continue
			}
		}
		p.keep = append(p.keep, l)
	}
	for it := range want {
		if !found[it] {
			p.missing = append(p.missing, it)
		}
	}
	sort.Strings(p.missing)
	return p, nil
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
	text = memory.OneLine(text)
	lines := strings.Split(string(data), "\n")
	for i, l := range lines {
		if e, ok := parseInboxLine(l, b.spaceSet()); ok && e.Date == date && (e.Text == text || e.full() == text) {
			lines = append(lines[:i], lines[i+1:]...)
			return true, paths.WriteAtomicNoLink(b.inboxPath(), []byte(strings.Join(lines, "\n")), 0o600)
		}
	}
	// The session start shortens a long capture and marks the cut with "…": that text
	// takes it out too, when it fits only one capture of that date.
	start, short := strings.CutSuffix(text, "…")
	if !short || strings.TrimSpace(start) == "" {
		return false, nil
	}
	found := -1
	for i, l := range lines {
		if e, ok := parseInboxLine(l, b.spaceSet()); ok && e.Date == date && strings.HasPrefix(e.Text, start) {
			if found >= 0 {
				return false, nil // more than one: the whole text says which
			}
			found = i
		}
	}
	if found < 0 {
		return false, nil
	}
	lines = append(lines[:found], lines[found+1:]...)
	return true, paths.WriteAtomicNoLink(b.inboxPath(), []byte(strings.Join(lines, "\n")), 0o600)
}

// InboxEdit corrects one capture: the first with that date and exactly that text, as
// Inbox returns them. It gets newText and the reminder remind ("" for none); its date
// and the space it came from stay. It reports whether it found the capture.
func (b *Brain) InboxEdit(date, text, newText, remind string) (bool, error) {
	data, err := os.ReadFile(b.inboxPath())
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	text = memory.OneLine(text)
	lines := strings.Split(string(data), "\n")
	for i, l := range lines {
		e, ok := parseInboxLine(l, b.spaceSet())
		if !ok || e.Date != date || (e.Text != text && e.full() != text) {
			continue
		}
		line, err := b.captureText(newText, remind, e.From)
		if err != nil {
			return true, err
		}
		lines[i] = strings.TrimSuffix(memory.DatedLine(e.Date, line), "\n")
		return true, paths.WriteAtomicNoLink(b.inboxPath(), []byte(strings.Join(lines, "\n")), 0o600)
	}
	return false, nil
}

// ---- watch dates ----

// CheckDate says whether d is a date as the notes write it (YYYY-MM-DD): a form can
// then put the error next to the field before anything is written.
func CheckDate(d string) error { return checkDate(d) }

func checkDate(d string) error {
	if _, err := memory.ParseDate(d); err != nil {
		return i18n.Errorf("err.date", d)
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
	n, _, err := b.FinishReview(summary, nil)
	return n, err
}

// FinishReview ends the weekly review, all of it or nothing that is lost: it checks the
// review, saves it with the captures it takes out of the inbox (each with its date, so
// none goes without a trace), and only then takes them out. A review that cannot be
// saved leaves the inbox as it was. It returns the review and the processed items no
// capture matched, which stay where they were.
func (b *Brain) FinishReview(summary string, processed []string) (*memory.Note, []string, error) {
	summary = strings.TrimSpace(summary)
	if summary == "" {
		return nil, nil, i18n.Errorf("err.review.summary")
	}
	p, err := b.planInboxDone(processed)
	if err != nil {
		return nil, nil, err
	}
	title := i18n.T(b.Language, "review.title")
	title = fmt.Sprintf(title, b.today())
	first := strings.SplitN(summary, "\n", 2)[0]
	if len([]rune(first)) > 120 {
		first = string([]rune(first)[:117]) + "…"
	}
	body := summary + "\n"
	if len(p.gone) > 0 {
		var out strings.Builder
		fmt.Fprintf(&out, "%s\n\n## %s\n\n", summary, i18n.T(b.Language, "review.processed"))
		for _, e := range p.gone {
			out.WriteString(memory.DatedLine(e.Date, e.full()))
		}
		body = out.String()
	}
	id := b.newID("review", b.today())
	n := &memory.Note{
		Path:   memory.PathFor("review", id),
		Header: memory.Header{ID: id, Type: "review", Title: title, Summary: first, Created: b.today()},
		Body:   body,
	}
	if err := b.save(n); err != nil {
		return nil, nil, i18n.Errorf("err.review.inbox", err)
	}
	if err := p.write(); err != nil {
		return n, nil, fmt.Errorf("the review was saved in %s, but the captures could not be taken out of the inbox, so they are still there (take them out again with capture_done): %w", n.Path, err)
	}
	return n, p.missing, nil
}
