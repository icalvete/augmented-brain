package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for rel, body := range files {
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func taskNote(id, area, status string) string {
	return "---\nid: " + id + "\ntype: task\ntitle: " + id + "\narea: " + area +
		"\nstatus: " + status + "\nwaiting_on: me\nnext: do it\n---\n"
}

func TestIndex(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"areas/area-web.md":     "---\nid: area-web\ntype: area\ntitle: Website\nsummary: the public site\n---\n",
		"rules/rule-no-root.md": "---\nid: rule-no-root\ntype: rule\ntitle: Never root\n---\n",
		"tasks/task-a.md":       taskNote("task-a", "area-web", "plan"),
		"tasks/task-b.md":       taskNote("task-b", "area-web", "closed"),
		"tasks/task-c.md":       taskNote("task-c", "area-web", "someday"),
		"tasks/task-d.md":       "---\nid: task-d\ntype: task\ntitle: task-d\nstatus: parked\nwaiting_on: Ana\nnext: call her\n---\n",
	})
	m, problems, err := Load(dir)
	if err != nil || len(problems) > 0 {
		t.Fatal(err, problems)
	}
	idx := m.Index("en")
	for _, want := range []string{
		"- [Never root](rules/rule-no-root.md)",
		"- [Website](areas/area-web.md) — the public site",
		"### Website\n- [task-a](tasks/task-a.md) · plan · waits on me · next: do it",
		"### No area\n- [task-d](tasks/task-d.md) · parked · waits on Ana · next: call her",
		"Someday: 1 tasks",
	} {
		if !strings.Contains(idx, want) {
			t.Errorf("index lacks %q:\n%s", want, idx)
		}
	}
	if strings.Contains(idx, "task-b") {
		t.Errorf("closed task in the index:\n%s", idx)
	}
	if es := m.Index("es"); !strings.Contains(es, "## Tareas vivas") || !strings.Contains(es, "espera a me") {
		t.Errorf("spanish index:\n%s", es)
	}

	changed, err := m.WriteIndex("en")
	if err != nil || !changed {
		t.Fatalf("first write: changed=%v err=%v", changed, err)
	}
	if changed, _ := m.WriteIndex("en"); changed {
		t.Error("second write with no changes reported a change")
	}
	if _, problems, _ := Load(dir); len(problems) > 0 {
		t.Errorf("the generated MEMORY.md must not be read as a note: %v", problems)
	}
}

// Subtasks are listed under their parent, indented, with the steps of their Plan counted.
func TestIndexSubtasksAndSteps(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"tasks/task-p.md": taskNote("task-p", "", "plan") + "## Plan\n- [x] one\n- [ ] two\n\n## Journal\n",
		"tasks/task-k.md": strings.Replace(taskNote("task-k", "", "execution"), "status:", "parent: task-p\nstatus:", 1),
		"tasks/task-o.md": strings.Replace(taskNote("task-o", "", "plan"), "status:", "parent: task-x\nstatus:", 1),
	})
	m, problems, err := Load(dir)
	if err != nil || len(problems) != 1 || !strings.Contains(problems[0].Error(), `parent "task-x" does not exist`) {
		t.Fatal(err, problems)
	}
	idx := m.Index("en")
	if !strings.Contains(idx, "- [task-p](tasks/task-p.md) · plan · 1/2 steps · waits on me · next: do it\n"+
		"  - [task-k](tasks/task-k.md) · execution · waits on me") || !strings.Contains(idx, "\n- [task-o](tasks/task-o.md)") {
		t.Errorf("index:\n%s", idx)
	}
	if !strings.Contains(m.Index("es"), "1/2 pasos") {
		t.Error("spanish step count")
	}
}

// A parent that cannot be one (a loop, the task itself, a note that is not a task)
// never nests anything: every task is listed once, on its own. A parent on a note that
// is not a task is a soft problem: the note stays in the index.
func TestIndexBadParents(t *testing.T) {
	dir := t.TempDir()
	withParent := func(note, parent string) string {
		return strings.Replace(note, "\ntitle:", "\nparent: "+parent+"\ntitle:", 1)
	}
	writeFiles(t, dir, map[string]string{
		"tasks/task-a.md":   withParent(taskNote("task-a", "", "plan"), "task-b"),
		"tasks/task-b.md":   withParent(taskNote("task-b", "", "plan"), "task-a"),
		"tasks/task-s.md":   withParent(taskNote("task-s", "", "plan"), "task-s"),
		"tasks/task-r.md":   withParent(taskNote("task-r", "", "plan"), "area-x"),
		"areas/area-x.md":   withParent("---\nid: area-x\ntype: area\ntitle: X\n---\n", "task-a"),
		"tasks/task-top.md": taskNote("task-top", "", "plan"),
	})
	m, problems, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := m.Notes["area-x"]; !ok {
		t.Fatalf("the area with a parent was dropped: %v", problems)
	}
	if len(problems) != 5 {
		t.Errorf("problems: %v", problems)
	}
	idx := m.Index("en")
	for _, id := range []string{"task-a", "task-b", "task-s", "task-r", "task-top"} {
		if c := strings.Count(idx, "["+id+"]"); c != 1 {
			t.Errorf("%s listed %d times:\n%s", id, c, idx)
		}
		if strings.Contains(idx, "  - ["+id+"]") {
			t.Errorf("%s nested under a parent that cannot be one:\n%s", id, idx)
		}
		if len(m.Children(id)) != 0 {
			t.Errorf("children of %s: %d", id, len(m.Children(id)))
		}
	}
	if !strings.Contains(idx, "[X](areas/area-x.md)") {
		t.Errorf("the area is not in the index:\n%s", idx)
	}
}

func TestSessionIndex(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"areas/area-web.md": "---\nid: area-web\ntype: area\ntitle: Website\nsummary: the public site\n---\n",
		"tasks/task-a.md":   "---\nid: task-a\ntype: task\ntitle: task-a\nsummary: what a is\narea: area-web\nstatus: plan\nwaiting_on: me\nnext: do it\n---\n",
		"tasks/task-d.md":   "---\nid: task-d\ntype: task\ntitle: task-d\nsummary: what d is\nstatus: parked\nwaiting_on: Ana\nnext: call her\n---\n",
	})
	m, problems, err := Load(dir)
	if err != nil || len(problems) > 0 {
		t.Fatal(err, problems)
	}
	idx := m.SessionIndex("en")
	for _, want := range []string{
		"- [Website](areas/area-web.md) — the public site", // only tasks lose their summary
		"- [task-a](tasks/task-a.md) · plan · waits on me · next: do it\n",
		"- [task-d](tasks/task-d.md) · parked · waits on Ana\n",
	} {
		if !strings.Contains(idx, want) {
			t.Errorf("session index lacks %q:\n%s", want, idx)
		}
	}
	for _, not := range []string{"what a is", "what d is", "call her"} {
		if strings.Contains(idx, not) {
			t.Errorf("session index has %q:\n%s", not, idx)
		}
	}
	if got := Clip(strings.Repeat("a", 130), maxSessionNext); len([]rune(got)) != maxSessionNext || !strings.HasSuffix(got, "…") {
		t.Errorf("clip: %q", got)
	}
	if full := m.Index("en"); !strings.Contains(full, "what d is") || !strings.Contains(full, "call her") {
		t.Errorf("MEMORY.md lost detail:\n%s", full)
	}
}
