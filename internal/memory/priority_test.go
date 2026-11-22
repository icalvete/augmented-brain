package memory

import (
	"strings"
	"testing"
)

func TestCompareTasks(t *testing.T) {
	h := func(id, waits, importance, due, updated string) *Note {
		return &Note{Header: Header{ID: id, Type: "task", WaitingOn: waits, Importance: importance, Due: due, Updated: updated}}
	}
	tasks := []*Note{
		h("task-rest-new", "nobody", "", "", "2026-11-20"),
		h("task-rest-old", "nobody", "", "", "2026-11-01"),
		h("task-me", "me", "", "", "2026-11-10"),
		h("task-high", "nobody", "high", "", "2026-11-15"),
		h("task-due-late", "nobody", "", "2026-12-11", "2026-11-01"),
		h("task-due-soon", "Ana", "", "2026-11-27", "2026-11-20"),
		h("task-odd", "nobody", "urgent", "next week", "2026-10-01"),
	}
	SortTasks(tasks)
	var got []string
	for _, n := range tasks {
		got = append(got, n.Header.ID)
	}
	// A deadline first, nearest first; then the important; then waiting on the user;
	// then the rest, untouched longest first. A value this program does not know
	// (importance "urgent", a due that is not a date) counts as nothing.
	want := "task-due-soon task-due-late task-high task-me task-odd task-rest-old task-rest-new"
	if strings.Join(got, " ") != want {
		t.Errorf("order:\n got %s\nwant %s", strings.Join(got, " "), want)
	}
}

// A task with importance and due is read, and a value this program does not know
// never leaves the note out: an earlier program reads what a later one writes.
func TestPriorityFieldsAreRead(t *testing.T) {
	dir := t.TempDir()
	head := "---\nid: %s\ntype: task\ntitle: %s\nstatus: plan\nwaiting_on: me\nnext: do it\n"
	writeFiles(t, dir, map[string]string{
		"tasks/task-a.md": strings.ReplaceAll(head, "%s", "task-a") + "importance: high\ndue: 2026-11-27\n---\n",
		"tasks/task-b.md": strings.ReplaceAll(head, "%s", "task-b") + "importance: someday-maybe\ndue: soon\n---\n",
	})
	m, problems, err := Load(dir)
	if err != nil || len(problems) > 0 {
		t.Fatal(err, problems)
	}
	a := m.Notes["task-a"].Header
	if d, ok := a.DueDate(); !a.High() || !ok || d != "2026-11-27" {
		t.Errorf("task-a: high %v, due %q %v", a.High(), d, ok)
	}
	b := m.Notes["task-b"].Header
	if _, ok := b.DueDate(); b.High() || ok {
		t.Errorf("task-b: unknown values should count as nothing: high %v, due %v", b.High(), ok)
	}
	idx := m.Index("en")
	for _, want := range []string{
		"- [task-a](tasks/task-a.md) · plan · due 2026-11-27 · important · waits on me",
		"- [task-b](tasks/task-b.md) · plan · waits on me",
	} {
		if !strings.Contains(idx, want) {
			t.Errorf("index lacks %q:\n%s", want, idx)
		}
	}
	if es := m.Index("es"); !strings.Contains(es, "· plazo 2026-11-27 · importante ·") {
		t.Errorf("spanish index:\n%s", es)
	}
	if i, j := strings.Index(idx, "task-a.md"), strings.Index(idx, "task-b.md"); i > j {
		t.Errorf("task-a, with a deadline, should come first:\n%s", idx)
	}
}
