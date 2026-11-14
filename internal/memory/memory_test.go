package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const task = `---
id: task-backup
type: task
title: Backup and restore
summary: Copy each space to its own private repository
area: area-platform
related: [lesson-filters]
status: plan
waiting_on: me
next: Approve the plan
watch:
  - date: 2026-11-16
    what: Restore tested?
---
## Essentials
Notas en cualquier idioma.
`

func TestParseTask(t *testing.T) {
	n, err := Parse("tasks/task-backup.md", []byte(task))
	if err != nil {
		t.Fatal(err)
	}
	h := n.Header
	if h.Status != "plan" || h.WaitingOn != "me" || h.Next != "Approve the plan" {
		t.Errorf("task fields: %+v", h)
	}
	if len(h.Watch) != 1 || h.Watch[0].Date != "2026-11-16" || h.Watch[0].Done {
		t.Errorf("watch: %+v", h.Watch)
	}
	if !strings.HasPrefix(n.Body, "## Essentials") {
		t.Errorf("body: %q", n.Body)
	}
}

func TestParseWindowsLineEnds(t *testing.T) {
	data := strings.ReplaceAll(task, "\n", "\r\n")
	if _, err := Parse("tasks/task-backup.md", []byte(data)); err != nil {
		t.Fatal(err)
	}
}

func TestParseErrors(t *testing.T) {
	ok := "---\nid: task-x\ntype: task\ntitle: X\nstatus: plan\nwaiting_on: me\nnext: Do it\n---\n"
	cases := []struct{ path, data, want string }{
		{"tasks/task-x.md", "no header", "missing header"},
		{"tasks/task-x.md", "---\nid: task-x\n", "not closed"},
		{"tasks/task-x.md", strings.Replace(ok, "title: X", "titel: X", 1), "titel"},
		{"tasks/task-y.md", ok, "must match the file name"},
		{"areas/task-x.md", ok, "goes in tasks/"},
		{"tasks/task-x.md", strings.Replace(ok, "status: plan", "status: doing", 1), "status"},
		{"tasks/task-x.md", strings.Replace(ok, "next: Do it\n", "", 1), "next is missing"},
		{"tasks/task-x.md", strings.Replace(ok, "type: task", "type: note", 1), "is not one of"},
		{"areas/area-x.md", "---\nid: area-x\ntype: area\ntitle: X\nstatus: plan\n---\n", "only for tasks"},
	}
	for _, c := range cases {
		_, err := Parse(c.path, []byte(c.data))
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s %q: error %v, want it to mention %q", c.path, c.data, err, c.want)
		}
	}
	closed := strings.Replace(strings.Replace(ok, "status: plan", "status: closed", 1), "next: Do it\n", "", 1)
	if _, err := Parse("tasks/task-x.md", []byte(closed)); err != nil {
		t.Errorf("a closed task needs no next: %v", err)
	}
}

// A field this program does not know is said as one a newer version may have written,
// in the user's language, top level or inside a watch date; a header that is not YAML
// still gets the decoder's words.
func TestParseUnknownField(t *testing.T) {
	ok := "---\nid: task-x\ntype: task\ntitle: X\nstatus: plan\nwaiting_on: me\nnext: Do it\n"
	cases := []struct{ data, field string }{
		{ok + "priority: high\n---\n", "priority"},
		{ok + "watch:\n  - date: 2026-11-20\n    what: x\n    snooze: 2\n---\n", "watch.snooze"},
	}
	for _, c := range cases {
		_, err := Parse("tasks/task-x.md", []byte(c.data))
		p := Problem{Path: "tasks/task-x.md", Err: err}
		field, newer := p.Newer()
		if !newer || field != c.field {
			t.Fatalf("%q: error %v, want the unknown field %q", c.data, err, c.field)
		}
		if en := p.Text("en"); !strings.Contains(en, c.field) || !strings.Contains(en, "augmented-brain update") {
			t.Errorf("in English: %s", en)
		}
		if es := p.Text("es"); !strings.Contains(es, "versión más nueva") {
			t.Errorf("in Spanish: %s", es)
		}
	}
	_, err := Parse("tasks/task-x.md", []byte(ok+"title: [\n---\n"))
	if p := (Problem{Path: "tasks/task-x.md", Err: err}); err == nil || p.Text("en") != p.Error() {
		t.Errorf("broken YAML must keep the decoder's error: %v", err)
	}
	if _, newer := (Problem{Err: ErrNoHeader}).Newer(); newer {
		t.Error("a missing header is not a newer note")
	}
}

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"MEMORY.md":                   "# generated, no header",
		"inbox.md":                    "- 2026-10-15 — an idea",
		"profile.md":                  "---\nid: profile\ntype: profile\ntitle: Profile\n---\n",
		"areas/area-platform.md":      "---\nid: area-platform\ntype: area\ntitle: Platform\n---\n",
		"tasks/task-backup.md":        task,
		"knowledge/lesson-filters.md": "---\nid: lesson-filters\ntype: lesson\ntitle: Filters\narea: area-platform\n---\n",
		"tasks/task-broken.md":        "---\nid: task-broken\ntype: task\ntitle: B\nstatus: plan\nwaiting_on: me\nnext: x\narea: area-nope\n---\n",
		"archive/task-old.md":         "anything goes in the archive",
	}
	for rel, body := range files {
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m, problems, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(m.Notes); got != 5 {
		t.Errorf("notes = %d (%v), want 5", got, m.IDs())
	}
	if len(problems) != 1 || !strings.Contains(problems[0].Error(), `area "area-nope" does not exist`) {
		t.Errorf("problems = %v", problems)
	}
	if tasks := m.OfType("task"); len(tasks) != 2 || tasks[0].Header.ID != "task-backup" {
		t.Errorf("OfType(task) = %v", tasks)
	}
}

// Some Windows editors start a file with a byte order mark: the note is still valid.
func TestParseBOM(t *testing.T) {
	data := "\xef\xbb\xbf---\nid: area-x\ntype: area\ntitle: X\n---\nbody\n"
	n, err := Parse("areas/area-x.md", []byte(data))
	if err != nil || n.Header.Title != "X" || n.Body != "body\n" {
		t.Errorf("BOM: %v %+v", err, n)
	}
}
