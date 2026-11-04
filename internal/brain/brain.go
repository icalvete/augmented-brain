// Package brain is the method: what can be done with a space's memory. Open a task,
// move it through its phases, write in its journal, park it, close it, see what is
// pending, capture an idea, create areas and knowledge.
//
// Every change rewrites the session index. Nothing here knows about MCP or about any
// particular assistant: the MCP server and the CLI are thin layers on top.
package brain

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/format"
	"github.com/icalvete/augmented-brain/internal/guard"
	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/memory"
	"github.com/icalvete/augmented-brain/internal/templates"
)

// Brain is one space, open for reading and writing.
type Brain struct {
	Space     *config.Space
	Language  string
	Mem       *memory.Memory
	ConfigDir string // where the configuration and the profile live
	// Now gives today's date; tests replace it.
	Now func() time.Time
	// Problems are the files of the memory that are not valid notes. The brain works with
	// the valid ones, and never writes over a file that could not be read.
	Problems []memory.Problem
}

// Open loads the space's memory. A file that is not a valid note does not stop the
// rest: it is left out, listed in Problems, and never written over.
func Open(c *config.Config, s *config.Space) (*Brain, error) {
	if err := format.CheckNotes(s.Memory, c.Language); err != nil {
		return nil, err // written by a newer program: writing here could break them
	}
	m, problems, err := memory.LoadSpace(s.Memory, c.SharedDir())
	if err != nil {
		return nil, err
	}
	return &Brain{Space: s, Language: c.Language, Mem: m, ConfigDir: c.Dir, Now: time.Now, Problems: problems}, nil
}

// Shared opens the shared notes, the rules and knowledge that apply in every space, to
// write one there. The folder is made the first time.
func (b *Brain) Shared() (*Brain, error) {
	dir := filepath.Join(b.ConfigDir, config.SharedFolder)
	if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
		if err := format.Write(dir, format.Notes()); err != nil {
			return nil, err
		}
	}
	if err := format.CheckNotes(dir, b.Language); err != nil {
		return nil, err
	}
	m, problems, err := memory.LoadShared(dir)
	if err != nil {
		return nil, err
	}
	s := &config.Space{Name: config.SharedFolder, Memory: dir}
	return &Brain{Space: s, Language: b.Language, Mem: m, ConfigDir: b.ConfigDir, Now: b.Now, Problems: problems}, nil
}

// broken reports whether the file at path (relative to the memory folder) is one that
// could not be read as a note.
func (b *Brain) broken(path string) bool {
	for _, p := range b.Problems {
		if p.Path == path {
			if n, ok := b.Mem.Notes[strings.TrimSuffix(filepath.Base(path), ".md")]; !ok || n.Path != path {
				return true
			}
		}
	}
	return false
}

// ProblemsNote says, for the assistant, which files are not valid notes and how to fix
// them; empty when all are. At most max files are named.
func ProblemsNote(problems []memory.Problem, max int) string {
	if len(problems) == 0 {
		return ""
	}
	var out strings.Builder
	fmt.Fprintf(&out, "%d files of this space are not valid notes; they are left out until fixed, and nothing writes over them:\n", len(problems))
	for i, p := range problems {
		if i == max {
			fmt.Fprintf(&out, "- and %d more\n", len(problems)-max)
			break
		}
		fmt.Fprintf(&out, "- %s\n", p.Error())
	}
	out.WriteString("Tell the user, in their language, and offer to fix each file (its header, between the --- lines at the top); `augmented-brain check` in the space's folder lists them all.\n")
	return out.String()
}

// today is the date of b.Now (local time), as the notes write dates.
func (b *Brain) today() string { return b.Now().Format(memory.DateLayout) }

// save writes a note, keeps it in memory and rebuilds the index.
func (b *Brain) save(n *memory.Note) error {
	if b.broken(n.Path) {
		return fmt.Errorf("%s is not a valid note, so it is not written over; fix it first (`augmented-brain check` says what is wrong)", n.Path)
	}
	if err := noNewSecret(n, b.Mem.Notes[n.Header.ID]); err != nil {
		return err
	}
	n.Header.Updated = b.today()
	if err := memory.Save(b.Mem.Dir, n); err != nil {
		return err
	}
	b.Mem.Notes[n.Header.ID] = n
	_, err := b.Mem.WriteIndex(b.Language)
	return err
}

// noNewSecret refuses a note that would hold more secrets than its old version (nil:
// none): the backup would refuse the whole space for it. A note that already held one,
// as an import may leave, can still be changed, so it can be fixed.
func noNewSecret(n, old *memory.Note) error {
	data, err := n.Render()
	if err != nil {
		return err
	}
	what := guard.FindSecret(string(data))
	if what == "" {
		return nil
	}
	if old != nil {
		if before, err := old.Render(); err == nil && guard.CountSecrets(string(data)) <= guard.CountSecrets(string(before)) {
			return nil // no new one: the note can be fixed
		}
	}
	return secretError(what)
}

// secretError is why a write with a secret in it is refused, worded as the guard's.
func secretError(what string) error {
	return fmt.Errorf("nothing was saved: the text has %s in it. Secrets never go into notes, and the backup would refuse the whole space: keep it in the system keychain or a secrets manager, and write down only where it is", what)
}

func (b *Brain) get(id, kind string) (*memory.Note, error) {
	n, ok := b.Mem.Notes[id]
	if !ok {
		return nil, fmt.Errorf("there is no %s %q", kind, id)
	}
	if kind != "" && n.Header.Type != kind {
		return nil, fmt.Errorf("%q is a %s, not a %s", id, n.Header.Type, kind)
	}
	return n, nil
}

// ---- tasks ----

// NewTaskInput describes a task to open.
type NewTaskInput struct {
	Title   string
	Summary string
	Area    string // area id; empty means the parent's area, or area-general when it exists
	Parent  string // optional: the id of the task this one is a part of (a subtask)
	// Goal and DoneWhen go into the Definition, when given.
	Goal, DoneWhen string
}

// NewTask opens a task in the definition phase, waiting on the user.
func (b *Brain) NewTask(in NewTaskInput) (*memory.Note, error) {
	if strings.TrimSpace(in.Title) == "" {
		return nil, errors.New("a task needs a title")
	}
	var parent *memory.Note
	if in.Parent != "" {
		var err error
		if parent, err = b.parent(in.Parent); err != nil {
			return nil, err
		}
	}
	area := in.Area
	if area == "" && parent != nil {
		area = parent.Header.Area
	}
	if area == "" {
		if _, ok := b.Mem.Notes["area-general"]; ok {
			area = "area-general"
		}
	} else if _, err := b.get(area, "area"); err != nil {
		return nil, err
	}
	id := b.newID("task", Slug(in.Title))
	tmpl, err := templates.FS.ReadFile(b.lang() + "/task.md")
	if err != nil {
		return nil, err
	}
	next := b.text("next")
	body := strings.NewReplacer("{{SUMMARY}}", in.Summary, "{{NEXT}}", next, "{{DATE}}", b.today()).Replace(string(tmpl))
	if g := oneLine(in.Goal); g != "" {
		body = setLine(body, "goal", g)
	}
	if d := oneLine(in.DoneWhen); d != "" {
		body = setLine(body, "done", d)
	}
	n := &memory.Note{
		Path: memory.PathFor("task", id),
		Header: memory.Header{
			ID: id, Type: "task", Title: in.Title, Summary: in.Summary, Area: area,
			Created: b.today(), Parent: in.Parent, Status: "definition", WaitingOn: "me", Next: next,
		},
		Body: body,
	}
	return n, b.save(n)
}

// parent checks that id can take a subtask: a task, not closed, and not a subtask
// itself (subtasks go one level deep only).
func (b *Brain) parent(id string) (*memory.Note, error) {
	p, err := b.Task(id)
	if err != nil {
		return nil, fmt.Errorf("the parent must be an existing task: %w", err)
	}
	if p.Header.Parent != "" {
		return nil, fmt.Errorf("%q is already a subtask of %q, and subtasks go one level deep only: make this one a subtask of %q, or a step in the Plan of %q", id, p.Header.Parent, p.Header.Parent, id)
	}
	if p.Header.Status == "closed" {
		return nil, fmt.Errorf("%q is closed: a closed task takes no new subtasks", id)
	}
	return p, nil
}

// openChildren refuses to close a task while any of its subtasks is not closed.
func (b *Brain) openChildren(id string) error {
	var open []string
	for _, c := range b.Mem.Children(id) {
		if c.Header.Status != "closed" {
			open = append(open, fmt.Sprintf("%s (%s, %s)", c.Header.ID, c.Header.Title, c.Header.Status))
		}
	}
	if len(open) > 0 {
		return fmt.Errorf("%q cannot be closed while its subtasks are open; close them first: %s", id, strings.Join(open, "; "))
	}
	return nil
}

// Task returns a task.
func (b *Brain) Task(id string) (*memory.Note, error) { return b.get(id, "task") }

// UpdateInput changes some fields of a task; empty fields stay as they are.
type UpdateInput struct {
	ID        string
	Status    string
	WaitingOn string
	Next      string
	Summary   string
	Area      string
	// Clear empties fields by name: "waiting_on", "next", "summary", "area". The MCP
	// tool never sets it (a blank there keeps the field); the panel does, when the
	// user empties a field. A live task still needs waiting_on and next.
	Clear []string
	// WatchDate and WatchWhat add a watch date; WatchDone marks those of that day done.
	WatchDate, WatchWhat, WatchDone string
}

// Update changes a task's header, all of it or nothing: the note is written once, after
// every change has been checked. A change of status is written in the journal.
func (b *Brain) Update(in UpdateInput) (*memory.Note, error) {
	n, err := b.Task(in.ID)
	if err != nil {
		return nil, err
	}
	cp := *n
	h := &cp.Header
	h.Watch = slices.Clone(h.Watch) // the note in the index stays as it was until saved
	old := h.Status
	if in.Status != "" {
		h.Status = in.Status
	}
	if in.WaitingOn != "" {
		h.WaitingOn = in.WaitingOn
	}
	if in.Next != "" {
		h.Next = in.Next
	}
	if in.Summary != "" {
		h.Summary = in.Summary
	}
	if in.Area != "" {
		if _, err := b.get(in.Area, "area"); err != nil {
			return nil, err
		}
		h.Area = in.Area
	}
	for _, f := range in.Clear {
		switch f {
		case "waiting_on":
			h.WaitingOn = ""
		case "next":
			h.Next = ""
		case "summary":
			h.Summary = ""
		case "area":
			h.Area = ""
		default:
			return nil, fmt.Errorf("%q cannot be cleared", f)
		}
	}
	if in.WatchDate != "" {
		if err := checkDate(in.WatchDate); err != nil {
			return nil, err
		}
		if strings.TrimSpace(in.WatchWhat) == "" {
			return nil, errors.New("a watch date needs what to look at")
		}
		h.Watch = append(h.Watch, memory.Watch{Date: in.WatchDate, What: strings.TrimSpace(in.WatchWhat)})
		sort.SliceStable(h.Watch, func(i, j int) bool { return h.Watch[i].Date < h.Watch[j].Date })
	}
	if in.WatchDone != "" {
		hit := false
		for i := range h.Watch {
			if h.Watch[i].Date == in.WatchDone && !h.Watch[i].Done {
				h.Watch[i].Done = true
				hit = true
			}
		}
		if !hit {
			return nil, fmt.Errorf("task %s has no open watch date on %s", in.ID, in.WatchDone)
		}
	}
	if h.Status == "closed" && old != "closed" {
		if err := b.openChildren(h.ID); err != nil {
			return nil, err
		}
	}
	if h.Status != old {
		cp.Body = b.journalLine(cp.Body, fmt.Sprintf(b.text("status"), old, h.Status))
	}
	return &cp, b.save(&cp)
}

// Journal adds a dated entry at the end of a task: the journal is its last section.
func (b *Brain) Journal(id, entry string) (*memory.Note, error) {
	n, err := b.Task(id)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(entry) == "" {
		return nil, errors.New("the journal entry is empty")
	}
	cp := *n
	cp.Body = b.journalLine(cp.Body, entry)
	return &cp, b.save(&cp)
}

// text is one of the entries the program writes into notes itself, in the user's
// language (i18n, "note.<key>").
func (b *Brain) text(key string) string { return i18n.T(b.Language, "note."+key) }

// JournalText adds to a task's journal one of the entries the program writes itself,
// in the user's language.
func (b *Brain) JournalText(key, id string, args ...any) (*memory.Note, error) {
	return b.Journal(id, fmt.Sprintf(b.text(key), args...))
}

func (b *Brain) journalLine(body, entry string) string {
	if body != "" && !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	return body + memory.DatedLine(b.today(), strings.TrimSpace(entry))
}

// ParkInput pauses a task without losing anything.
type ParkInput struct {
	ID         string
	Next       string // exactly how to resume it
	WaitingOn  string // who it waits on now; empty keeps it
	Essentials string // new text for the task's first section; empty keeps it
	Someday    bool   // not for weeks: leaves the session index
}

// Park pauses a task: status parked (or someday), the next step, a journal entry, and
// optionally a rewritten essentials section (the first section of the body).
func (b *Brain) Park(in ParkInput) (*memory.Note, error) {
	if strings.TrimSpace(in.Next) == "" {
		return nil, errors.New("parking needs the next step: exactly how to resume")
	}
	n, err := b.Task(in.ID)
	if err != nil {
		return nil, err
	}
	cp := *n
	cp.Header.Status = "parked"
	if in.Someday {
		cp.Header.Status = "someday"
	}
	cp.Header.Next = in.Next
	if in.WaitingOn != "" {
		cp.Header.WaitingOn = in.WaitingOn
	}
	entry := fmt.Sprintf(b.text("parked"), in.Next)
	if in.Essentials == "" {
		// Without new essentials, at least where it is and the next step are current.
		cp.Body = setLine(cp.Body, "where", fmt.Sprintf(b.text("where.parked"), b.today()))
		cp.Body = setLine(cp.Body, "next", oneLine(in.Next))
	} else {
		if body, ok := replaceFirstSection(cp.Body, in.Essentials); ok {
			cp.Body = body
		} else {
			// No safe place to put them: keep them in the journal, never lose text.
			entry += " " + strings.Join(strings.Fields(in.Essentials), " ")
		}
	}
	cp.Body = b.journalLine(cp.Body, entry)
	return &cp, b.save(&cp)
}

// Close marks a task closed with its result. It is never deleted.
func (b *Brain) Close(id, result string) (*memory.Note, error) {
	if strings.TrimSpace(result) == "" {
		return nil, errors.New("closing needs the result")
	}
	n, err := b.Task(id)
	if err != nil {
		return nil, err
	}
	if err := b.openChildren(id); err != nil {
		return nil, err
	}
	cp := *n
	cp.Header.Status = "closed"
	cp.Header.Next = ""
	cp.Header.WaitingOn = ""
	cp.Body = setLine(cp.Body, "where", fmt.Sprintf(b.text("where.closed"), b.today(), oneLine(result)))
	cp.Body = setLine(cp.Body, "next", b.text("next.closed"))
	cp.Body = b.journalLine(cp.Body, fmt.Sprintf(b.text("closed"), result))
	return &cp, b.save(&cp)
}

// setLine writes value after the first "- <label>:" line of body, with the label in
// either language (Essentials and Definition); a body without one is left as it is.
func setLine(body, key, value string) string {
	lines := strings.SplitAfter(body, "\n")
	end := len(lines)
	if heads := memory.Headings(lines); len(heads) > 2 {
		end = heads[2] // the first two sections only
	}
	for i, l := range lines[:end] {
		for _, label := range i18n.All("note.label." + key) {
			if strings.HasPrefix(l, "- "+label+":") {
				end := ""
				if strings.HasSuffix(l, "\n") {
					end = "\n"
				}
				lines[i] = "- " + label + ": " + value + end
				return strings.Join(lines, "")
			}
		}
	}
	return body
}

// oneLine joins text into one line.
func oneLine(text string) string { return strings.Join(strings.Fields(text), " ") }

// replaceFirstSection swaps the text under the first level-2 heading ("## " at the start
// of a line, outside code blocks) for text. It only does so when another level-2
// heading follows, so the rest of the task, the journal above all, is never touched;
// otherwise it reports false and changes nothing.
func replaceFirstSection(body, text string) (string, bool) {
	lines := strings.SplitAfter(body, "\n")
	heads := memory.Headings(lines)
	if len(heads) < 2 {
		return body, false
	}
	first, next := heads[0], heads[1]
	head := strings.Join(lines[:first+1], "")
	if !strings.HasSuffix(head, "\n") {
		head += "\n"
	}
	return head + strings.TrimSpace(text) + "\n\n" + strings.Join(lines[next:], ""), true
}

// ---- pending ----

// Item is one line of the pending report.
type Item struct {
	ID, Title, Area, Status, WaitingOn, Next string
	// Parent is the id of the task this one is part of, and ParentTitle its title;
	// Sub is true when the item is listed right under its parent.
	Parent, ParentTitle   string
	Sub                   bool
	StepsDone, StepsTotal int
}

// Due is a watch date that has come or is near.
type Due struct {
	TaskID, Date, What string
	Days               int // negative when overdue
}

// Pending returns the live tasks, those waiting on the user first, and the watch
// dates due within the next `within` days.
func (b *Brain) Pending(within int) ([]Item, []Due) {
	var items []Item
	var due []Due
	today := b.today()
	for _, n := range b.Mem.OfType("task") {
		h := n.Header
		// A closed task is no longer pending, but its watch dates still are: closing
		// is when the checks for later are written down.
		if h.Status != "someday" && h.Status != "closed" {
			it := Item{ID: h.ID, Title: h.Title, Area: h.Area, Status: h.Status, WaitingOn: h.WaitingOn, Next: h.Next,
				Parent: h.Parent, Sub: b.Mem.Nested(n)}
			if p := b.Mem.ParentOf(n); p != nil {
				it.ParentTitle = p.Header.Title
			}
			it.StepsDone, it.StepsTotal = memory.StepCount(n.Body)
			items = append(items, it)
		}
		for _, w := range h.Watch {
			days, ok := memory.DaysBetween(today, w.Date)
			if !ok || w.Done {
				continue
			}
			if days <= within {
				due = append(due, Due{h.ID, w.Date, w.What, days})
			}
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].WaitingOn == "me" && items[j].WaitingOn != "me"
	})
	items = nest(items)
	sort.Slice(due, func(i, j int) bool { return due[i].Date < due[j].Date })
	return items, due
}

// nest puts every subtask listed under its parent right after it, keeping the order of
// the rest.
func nest(items []Item) []Item {
	subs := map[string][]Item{}
	for _, i := range items {
		if i.Sub {
			subs[i.Parent] = append(subs[i.Parent], i)
		}
	}
	out := make([]Item, 0, len(items))
	for _, i := range items {
		if !i.Sub {
			out = append(out, i)
			out = append(out, subs[i.ID]...)
		}
	}
	return out
}

// ---- inbox, areas, knowledge ----

// Capture adds a dated line to the inbox, to be processed in the weekly review. With a
// remind date (YYYY-MM-DD), the session start brings it up again on that day.
func (b *Brain) Capture(text, remind string) error {
	text = strings.TrimSpace(strings.ReplaceAll(text, "\n", " "))
	if text == "" {
		return errors.New("nothing to capture")
	}
	if remind != "" {
		if err := checkDate(remind); err != nil {
			return err
		}
		text += " (remind: " + remind + ")"
	}
	if what := guard.FindSecret(text); what != "" {
		return secretError(what)
	}
	if fi, err := os.Lstat(b.inboxPath()); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%s is a symbolic link, which augmented brain never writes through; replace it with a plain file", b.inboxPath())
	}
	f, err := os.OpenFile(b.inboxPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(memory.DatedLine(b.today(), text))
	return err
}

// NewArea creates an area: a responsibility with no end date.
func (b *Brain) NewArea(title, summary string) (*memory.Note, error) {
	if strings.TrimSpace(title) == "" {
		return nil, errors.New("an area needs a title")
	}
	id := b.newID("area", Slug(title))
	n := &memory.Note{
		Path:   memory.PathFor("area", id),
		Header: memory.Header{ID: id, Type: "area", Title: title, Summary: summary, Created: b.today()},
		Body:   "",
	}
	return n, b.save(n)
}

// UpdateArea changes an area's title (empty keeps it: an area always has one) and its
// summary (empty clears it). The id stays, so every task in the area stays in it.
func (b *Brain) UpdateArea(id, title, summary string) (*memory.Note, error) {
	old, err := b.get(id, "area")
	if err != nil {
		return nil, err
	}
	cp := *old
	if t := strings.TrimSpace(title); t != "" {
		cp.Header.Title = t
	}
	cp.Header.Summary = strings.TrimSpace(summary)
	return &cp, b.save(&cp)
}

// NewKnowledge saves a reference (how something works) or a lesson (what was learned).
func (b *Brain) NewKnowledge(kind, title, summary, area, body string) (*memory.Note, error) {
	if kind != "reference" && kind != "lesson" {
		return nil, fmt.Errorf("kind %q: use reference or lesson", kind)
	}
	if strings.TrimSpace(title) == "" {
		return nil, errors.New("a note needs a title")
	}
	if area != "" {
		if _, err := b.get(area, "area"); err != nil {
			return nil, err
		}
	}
	id := b.newID(kind, Slug(title))
	if body != "" && !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	n := &memory.Note{
		Path:   memory.PathFor(kind, id),
		Header: memory.Header{ID: id, Type: kind, Title: title, Summary: summary, Area: area, Created: b.today()},
		Body:   body,
	}
	return n, b.save(n)
}

// NewRule saves a rule: how the assistant must work, loaded at the start of every
// session.
func (b *Brain) NewRule(title, summary, body string) (*memory.Note, error) {
	if strings.TrimSpace(title) == "" || strings.TrimSpace(body) == "" {
		return nil, errors.New("a rule needs a title and a body: what to do, and why")
	}
	id := b.newID("rule", Slug(title))
	if !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	n := &memory.Note{
		Path:   memory.PathFor("rule", id),
		Header: memory.Header{ID: id, Type: "rule", Title: title, Summary: summary, Created: b.today()},
		Body:   body,
	}
	return n, b.save(n)
}

// ---- ids ----

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

// accents maps the accented letters of the Latin alphabets to plain ASCII.
var accents = strings.NewReplacer(
	"á", "a", "à", "a", "â", "a", "ä", "a", "ã", "a", "å", "a",
	"é", "e", "è", "e", "ê", "e", "ë", "e",
	"í", "i", "ì", "i", "î", "i", "ï", "i",
	"ó", "o", "ò", "o", "ô", "o", "ö", "o", "õ", "o", "ø", "o",
	"ú", "u", "ù", "u", "û", "u", "ü", "u",
	"ñ", "n", "ç", "c", "ß", "ss", "æ", "ae", "œ", "oe",
)

// Slug turns a title into the readable part of an id: lowercase ASCII, words joined by
// dashes, accents dropped, at most 40 characters.
func Slug(title string) string {
	s := strings.Trim(nonSlug.ReplaceAllString(accents.Replace(strings.ToLower(title)), "-"), "-")
	if len(s) > 40 {
		s = strings.TrimRight(s[:40], "-")
	}
	if s == "" {
		s = "untitled"
	}
	return s
}

// newID returns a free id for a new note of that type: its prefix, then name.
func (b *Brain) newID(typ, name string) string {
	return b.uniqueID(memory.Types[typ].Prefix + name)
}

// uniqueID returns base, or base-2, base-3… when it is taken by a note or by a file
// that is not a valid one.
func (b *Brain) uniqueID(base string) string {
	id := base
	for i := 2; ; i++ {
		if _, taken := b.Mem.Notes[id]; !taken && !b.brokenID(id) {
			return id
		}
		id = fmt.Sprintf("%s-%d", base, i)
	}
}

// brokenID reports whether a file named id.md, anywhere in the memory, is not valid.
func (b *Brain) brokenID(id string) bool {
	for _, p := range b.Problems {
		if strings.TrimSuffix(filepath.Base(p.Path), ".md") == id && b.broken(p.Path) {
			return true
		}
	}
	return false
}

func (b *Brain) lang() string { return i18n.Known(b.Language) }
