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
	"sort"
	"strings"
	"time"

	"github.com/icalvete/augmented-brain/internal/config"
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
}

// Open loads the space's memory. It refuses to work on a memory with invalid notes:
// writing on top of it could make things worse.
func Open(c *config.Config, s *config.Space) (*Brain, error) {
	m, problems, err := memory.Load(s.Memory)
	if err != nil {
		return nil, err
	}
	if len(problems) > 0 {
		return nil, fmt.Errorf("space %s has %d notes that are not valid (first: %v); run `augmented-brain check`",
			s.Name, len(problems), problems[0])
	}
	return &Brain{Space: s, Language: c.Language, Mem: m, ConfigDir: c.Dir, Now: time.Now}, nil
}

func (b *Brain) today() string { return b.Now().Format("2006-01-02") }

// save writes a note, keeps it in memory and rebuilds the index.
func (b *Brain) save(n *memory.Note) error {
	n.Header.Updated = b.today()
	if err := memory.Save(b.Mem.Dir, n); err != nil {
		return err
	}
	b.Mem.Notes[n.Header.ID] = n
	_, err := b.Mem.WriteIndex(b.Language)
	return err
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

// texts the program writes into notes, per language.
var texts = map[string]map[string]string{
	"en": {"next": "Agree on the definition", "status": "Status: %s → %s.", "closed": "Closed: %s",
		"parked": "Parked. Next: %s", "resumed": "Resumed."},
	"es": {"next": "Acordar la definición", "status": "Estado: %s → %s.", "closed": "Cerrada: %s",
		"parked": "Aparcada. Siguiente: %s", "resumed": "Retomada."},
}

func (b *Brain) text(key string) string {
	if t, ok := texts[b.Language]; ok {
		return t[key]
	}
	return texts["en"][key]
}

// ---- tasks ----

// NewTaskInput describes a task to open.
type NewTaskInput struct {
	Title   string
	Summary string
	Area    string // area id; empty means area-general when it exists
}

// NewTask opens a task in the definition phase, waiting on the user.
func (b *Brain) NewTask(in NewTaskInput) (*memory.Note, error) {
	if strings.TrimSpace(in.Title) == "" {
		return nil, errors.New("a task needs a title")
	}
	area := in.Area
	if area == "" {
		if _, ok := b.Mem.Notes["area-general"]; ok {
			area = "area-general"
		}
	} else if _, err := b.get(area, "area"); err != nil {
		return nil, err
	}
	id := b.uniqueID("task-" + Slug(in.Title))
	tmpl, err := templates.FS.ReadFile(b.lang() + "/task.md")
	if err != nil {
		return nil, err
	}
	next := b.text("next")
	body := strings.NewReplacer("{{SUMMARY}}", in.Summary, "{{NEXT}}", next, "{{DATE}}", b.today()).Replace(string(tmpl))
	n := &memory.Note{
		Path: "tasks/" + id + ".md",
		Header: memory.Header{
			ID: id, Type: "task", Title: in.Title, Summary: in.Summary, Area: area,
			Created: b.today(), Status: "definition", WaitingOn: "me", Next: next,
		},
		Body: body,
	}
	return n, b.save(n)
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
}

// Update changes a task's header. A change of status is written in the journal.
func (b *Brain) Update(in UpdateInput) (*memory.Note, error) {
	n, err := b.Task(in.ID)
	if err != nil {
		return nil, err
	}
	cp := *n
	h := &cp.Header
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

func (b *Brain) journalLine(body, entry string) string {
	if body != "" && !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	return body + "- " + b.today() + " — " + strings.TrimSpace(entry) + "\n"
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
	if in.Essentials != "" {
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
	cp := *n
	cp.Header.Status = "closed"
	cp.Header.Next = ""
	cp.Header.WaitingOn = ""
	cp.Body = b.journalLine(cp.Body, fmt.Sprintf(b.text("closed"), result))
	return &cp, b.save(&cp)
}

// replaceFirstSection swaps the text under the first level-2 heading ("## " at the start
// of a line, outside code blocks) for text. It only does so when another level-2
// heading follows, so the rest of the task, the journal above all, is never touched;
// otherwise it reports false and changes nothing.
func replaceFirstSection(body, text string) (string, bool) {
	lines := strings.SplitAfter(body, "\n")
	first, next := -1, -1
	inCode := false
	for i, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), "```") {
			inCode = !inCode
			continue
		}
		if inCode || !strings.HasPrefix(l, "## ") {
			continue
		}
		if first < 0 {
			first = i
		} else {
			next = i
			break
		}
	}
	if first < 0 || next < 0 {
		return body, false
	}
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
	today, _ := time.Parse("2006-01-02", b.today())
	for _, n := range b.Mem.OfType("task") {
		h := n.Header
		if h.Status == "closed" {
			continue
		}
		if h.Status != "someday" {
			items = append(items, Item{h.ID, h.Title, h.Area, h.Status, h.WaitingOn, h.Next})
		}
		for _, w := range h.Watch {
			d, err := time.Parse("2006-01-02", w.Date)
			if err != nil || w.Done {
				continue
			}
			if days := int(d.Sub(today).Hours() / 24); days <= within {
				due = append(due, Due{h.ID, w.Date, w.What, days})
			}
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].WaitingOn == "me" && items[j].WaitingOn != "me"
	})
	sort.Slice(due, func(i, j int) bool { return due[i].Date < due[j].Date })
	return items, due
}

// ---- inbox, areas, knowledge ----

// Capture adds a dated line to the inbox, to be processed in the weekly review.
func (b *Brain) Capture(text string) error {
	text = strings.TrimSpace(strings.ReplaceAll(text, "\n", " "))
	if text == "" {
		return errors.New("nothing to capture")
	}
	path := filepath.Join(b.Mem.Dir, "inbox.md")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintf(f, "- %s — %s\n", b.today(), text)
	return err
}

// NewArea creates an area: a responsibility with no end date.
func (b *Brain) NewArea(title, summary string) (*memory.Note, error) {
	if strings.TrimSpace(title) == "" {
		return nil, errors.New("an area needs a title")
	}
	id := b.uniqueID("area-" + Slug(title))
	n := &memory.Note{
		Path:   "areas/" + id + ".md",
		Header: memory.Header{ID: id, Type: "area", Title: title, Summary: summary, Created: b.today()},
		Body:   "",
	}
	return n, b.save(n)
}

// NewKnowledge saves a reference (how something works) or a lesson (what was learned).
func (b *Brain) NewKnowledge(kind, title, summary, area, body string) (*memory.Note, error) {
	prefix := map[string]string{"reference": "ref-", "lesson": "lesson-"}[kind]
	if prefix == "" {
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
	id := b.uniqueID(prefix + Slug(title))
	if body != "" && !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	n := &memory.Note{
		Path:   "knowledge/" + id + ".md",
		Header: memory.Header{ID: id, Type: kind, Title: title, Summary: summary, Area: area, Created: b.today()},
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

func (b *Brain) uniqueID(base string) string {
	id := base
	for i := 2; ; i++ {
		if _, taken := b.Mem.Notes[id]; !taken {
			return id
		}
		id = fmt.Sprintf("%s-%d", base, i)
	}
}

func (b *Brain) lang() string {
	if _, ok := texts[b.Language]; ok {
		return b.Language
	}
	return "en"
}
