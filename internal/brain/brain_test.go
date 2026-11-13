package brain

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/memory"
	"github.com/icalvete/augmented-brain/internal/setup"
)

// open sets up a fresh space in a temporary home and opens it on a fixed date.
func open(t *testing.T, lang string) *Brain {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	work := filepath.Join(home, "work")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	cfgDir := filepath.Join(home, "cfg")
	if _, err := setup.Init(cfgDir, setup.Options{Language: lang, Space: "work", WorkFolder: work,
		Memory: "~/brain", Today: "2026-10-15"}); err != nil {
		t.Fatal(err)
	}
	c, err := config.Load(cfgDir)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Open(c, c.Spaces["work"])
	if err != nil {
		t.Fatal(err)
	}
	b.Now = func() time.Time { return time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC) }
	return b
}

func read(t *testing.T, b *Brain, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(b.Mem.Dir, rel))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestTaskLifecycle(t *testing.T) {
	b := open(t, "en")

	n, err := b.NewTask(NewTaskInput{Title: "Backup & Restore", Summary: "Copy each space"})
	if err != nil {
		t.Fatal(err)
	}
	if n.Header.ID != "task-backup-restore" || n.Header.Area != "area-general" || n.Header.Status != "definition" {
		t.Fatalf("new task header: %+v", n.Header)
	}
	if !strings.Contains(read(t, b, "MEMORY.md"), "[Backup & Restore](tasks/task-backup-restore.md) — Copy each space · definition") {
		t.Errorf("index not updated:\n%s", read(t, b, "MEMORY.md"))
	}
	if again, _ := b.NewTask(NewTaskInput{Title: "Backup & restore"}); again.Header.ID != "task-backup-restore-2" {
		t.Errorf("duplicate id: %s", again.Header.ID)
	}

	if _, err := b.Update(UpdateInput{ID: n.Header.ID, Status: "analysis", Next: "Measure the folder sizes"}); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Journal(n.Header.ID, "Measured: 2 MB per space."); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Update(UpdateInput{ID: n.Header.ID, Status: "doing"}); err == nil {
		t.Error("an invalid status was accepted")
	}

	if _, err := b.Park(ParkInput{ID: n.Header.ID, Next: "Write the plan", Essentials: "- Backup per space, to a private repo."}); err != nil {
		t.Fatal(err)
	}
	text := read(t, b, "tasks/task-backup-restore.md")
	for _, want := range []string{
		"status: parked", "next: Write the plan",
		"## Essentials\n- Backup per space, to a private repo.\n\n## 1. Definition",
		"- 2026-10-15 — Status: definition → analysis.",
		"- 2026-10-15 — Measured: 2 MB per space.",
		"- 2026-10-15 — Parked. Next: Write the plan\n",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("task file lacks %q:\n%s", want, text)
		}
	}
	if strings.Index(text, "Measured") > strings.Index(text, "Parked.") {
		t.Error("journal entries are not in order")
	}

	if _, err := b.Close(n.Header.ID, "Backups run every night."); err != nil {
		t.Fatal(err)
	}
	if idx := read(t, b, "MEMORY.md"); strings.Contains(idx, "task-backup-restore.md)") {
		t.Errorf("a closed task is still in the index:\n%s", idx)
	}
	if _, problems, _ := memory.Load(b.Mem.Dir); len(problems) > 0 {
		t.Errorf("the memory is no longer valid: %v", problems)
	}
}

func TestSpanishTask(t *testing.T) {
	b := open(t, "es")
	n, err := b.NewTask(NewTaskInput{Title: "Migración de la base de datos"})
	if err != nil {
		t.Fatal(err)
	}
	if n.Header.ID != "task-migracion-de-la-base-de-datos" {
		t.Errorf("id = %s", n.Header.ID)
	}
	if text := read(t, b, n.Path); !strings.Contains(text, "## Lo esencial") || !strings.Contains(text, "Tarea abierta.") {
		t.Errorf("spanish template:\n%s", text)
	}
}

func TestPendingAndCapture(t *testing.T) {
	b := open(t, "en")
	a, _ := b.NewTask(NewTaskInput{Title: "Waits on Ana"})
	if _, err := b.Update(UpdateInput{ID: a.Header.ID, WaitingOn: "Ana"}); err != nil {
		t.Fatal(err)
	}
	me, _ := b.NewTask(NewTaskInput{Title: "Mine"})
	later, _ := b.NewTask(NewTaskInput{Title: "Later"})
	if _, err := b.Park(ParkInput{ID: later.Header.ID, Next: "someday", Someday: true}); err != nil {
		t.Fatal(err)
	}
	me.Header.Watch = []memory.Watch{{Date: "2026-10-18", What: "certificate renewed?"}, {Date: "2026-12-01", What: "far away"}}
	if err := b.save(me); err != nil {
		t.Fatal(err)
	}

	items, due := b.Pending(7)
	if len(items) != 2 || items[0].ID != me.Header.ID {
		t.Errorf("pending = %+v (waiting on me must come first, someday left out)", items)
	}
	if len(due) != 1 || due[0].Days != 3 {
		t.Errorf("due = %+v", due)
	}

	if err := b.Capture("look at\nthe new runner", ""); err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(read(t, b, "inbox.md"), "- 2026-10-15 — look at the new runner\n") {
		t.Errorf("inbox:\n%s", read(t, b, "inbox.md"))
	}
}

func TestAreasAndKnowledge(t *testing.T) {
	b := open(t, "en")
	area, err := b.NewArea("Platform", "The servers and the cloud")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.NewKnowledge("lesson", "Filters need smudge", "or files vanish", area.Header.ID, "Body."); err != nil {
		t.Fatal(err)
	}
	if _, err := b.NewKnowledge("lesson", "x", "", "area-nope", ""); err == nil {
		t.Error("a missing area was accepted")
	}
	if _, err := b.NewKnowledge("note", "x", "", "", ""); err == nil {
		t.Error("an unknown kind was accepted")
	}
	if _, err := b.NewTask(NewTaskInput{Title: "Upgrade", Area: area.Header.ID}); err != nil {
		t.Fatal(err)
	}
	if idx := read(t, b, "MEMORY.md"); !strings.Contains(idx, "### Platform\n- [Upgrade]") {
		t.Errorf("index:\n%s", idx)
	}
}

func TestSlug(t *testing.T) {
	cases := map[string]string{
		"Backup & Restore":      "backup-restore",
		"  Ñandú, árbol  ":      "nandu-arbol",
		"!!!":                   "untitled",
		strings.Repeat("a", 60): strings.Repeat("a", 40),
	}
	for in, want := range cases {
		if got := Slug(in); got != want {
			t.Errorf("Slug(%q) = %q, want %q", in, got, want)
		}
	}
}

// Found in the review before 0.1: parking with new essentials must never lose the rest
// of a task written or edited by hand.
func TestReplaceFirstSection(t *testing.T) {
	cases := []struct {
		name, body, want string
		ok               bool
	}{
		{"template", "## Essentials\nold\n\n## 1. Definition\nkeep\n", "## Essentials\nnew\n\n## 1. Definition\nkeep\n", true},
		{"only one section", "Notes\n\n## Journal\n- one\n- two\n", "Notes\n\n## Journal\n- one\n- two\n", false},
		{"level 3 is not a section", "### Context\nx\n## Essentials\nold\n## Journal\n- one\n", "### Context\nx\n## Essentials\nnew\n\n## Journal\n- one\n", true},
		{"heading in a code block", "## Essentials\nold\n```\n## not a heading\n```\n## Journal\n- one\n", "## Essentials\nnew\n\n## Journal\n- one\n", true},
		{"no heading", "just text\n", "just text\n", false},
	}
	for _, c := range cases {
		got, ok := replaceFirstSection(c.body, "new")
		if got != c.want || ok != c.ok {
			t.Errorf("%s: got %q (%v), want %q (%v)", c.name, got, ok, c.want, c.ok)
		}
	}
}

func TestParkKeepsHandWrittenJournal(t *testing.T) {
	b := open(t, "en")
	n, _ := b.NewTask(NewTaskInput{Title: "Hand written"})
	n.Body = "Notes by hand.\n\n## Journal\n- 2026-10-01 — first entry\n"
	if err := b.save(n); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Park(ParkInput{ID: n.Header.ID, Next: "go on", Essentials: "new essentials"}); err != nil {
		t.Fatal(err)
	}
	text := read(t, b, n.Path)
	if !strings.Contains(text, "first entry") || !strings.Contains(text, "Parked. Next: go on new essentials") {
		t.Errorf("journal lost or essentials dropped:\n%s", text)
	}
}

// One file that is not a valid note does not stop the rest: the brain opens, works with
// the valid notes, and never writes over the broken file.
func TestOpenWithInvalidNotes(t *testing.T) {
	b := open(t, "en")
	broken := filepath.Join(b.Mem.Dir, "tasks", "task-backup.md")
	if err := os.WriteFile(broken, []byte("---\nid: task-backup\ntype: task\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := config.Load(b.ConfigDir)
	if err != nil {
		t.Fatal(err)
	}
	b2, err := Open(c, c.Spaces["work"])
	if err != nil {
		t.Fatalf("one broken note locked the space: %v", err)
	}
	b2.Now = b.Now
	if len(b2.Problems) != 1 || b2.Problems[0].Path != "tasks/task-backup.md" {
		t.Fatalf("problems: %v", b2.Problems)
	}
	n, err := b2.NewTask(NewTaskInput{Title: "Backup"})
	if err != nil {
		t.Fatal(err)
	}
	if n.Header.ID != "task-backup-2" {
		t.Errorf("the new task took the broken file's name: %s", n.Header.ID)
	}
	if data, _ := os.ReadFile(broken); !strings.HasPrefix(string(data), "---\nid: task-backup\ntype: task\n") || strings.Contains(string(data), "title") {
		t.Errorf("the broken file was written over:\n%s", data)
	}
	if err := b2.save(&memory.Note{Path: "tasks/task-backup.md", Header: memory.Header{ID: "task-backup", Type: "task",
		Title: "x", Status: "plan", Next: "x", WaitingOn: "me"}}); err == nil {
		t.Error("a write over the broken file was let through")
	}
	if _, err := b2.AddWatch(n.Header.ID, "2026-10-15", "check"); err != nil {
		t.Fatal(err)
	}
	if r := b2.Reminders(1); len(r) != 1 {
		t.Errorf("reminders with a broken note around: %v", r)
	}
	if note := ProblemsNote(b2.Problems, 3); !strings.Contains(note, "tasks/task-backup.md") || !strings.Contains(note, "augmented-brain check") {
		t.Errorf("problems note: %s", note)
	}
	if note := ProblemsNote(b2.Problems, 3); strings.Contains(note, "newer augmented brain") {
		t.Errorf("a broken file is not from a newer version: %s", note)
	}
	newer := []memory.Problem{{Path: "areas/area-x.md", Err: &memory.UnknownFieldError{Field: "colour"}}}
	if note := ProblemsNote(newer, 3); !strings.Contains(note, "colour") || !strings.Contains(note, "augmented-brain update") || !strings.Contains(note, "do not remove it") {
		t.Errorf("a newer note must keep the assistant from rewriting it: %s", note)
	}
}

// Names with no Latin letters make no readable id: each person still gets their own
// note, and saving the same name again updates it.
func TestPeopleWithNonLatinNames(t *testing.T) {
	b := open(t, "en")
	m, err := b.SavePerson(PersonInput{Name: "Мария", Role: "client"})
	if err != nil {
		t.Fatal(err)
	}
	w, err := b.SavePerson(PersonInput{Name: "王伟", Role: "supplier"})
	if err != nil {
		t.Fatal(err)
	}
	if m.Header.ID == w.Header.ID {
		t.Fatalf("two people share %s", m.Header.ID)
	}
	again, err := b.SavePerson(PersonInput{Name: "мария", Role: "client, the shop"})
	if err != nil {
		t.Fatal(err)
	}
	if again.Header.ID != m.Header.ID || len(b.Mem.OfType("person")) != 2 {
		t.Errorf("saving Мария again made %s; people: %d", again.Header.ID, len(b.Mem.OfType("person")))
	}
	if b.Mem.Notes[w.Header.ID].Header.Summary != "supplier" {
		t.Error("王伟 was overwritten")
	}
	// An alias leads to the person too.
	if _, err := b.SavePerson(PersonInput{Name: "Marcos García", Aliases: []string{"Marquitos"}, Role: "manager"}); err != nil {
		t.Fatal(err)
	}
	n, err := b.SavePerson(PersonInput{Name: "Marquitos", Role: "my manager"})
	if err != nil {
		t.Fatal(err)
	}
	if n.Header.ID != "person-marcos-garcia" || n.Header.Title != "Marcos García" || !strings.HasSuffix(n.Header.Summary, "also: Marquitos") {
		t.Errorf("by alias: %+v", n.Header)
	}
}
