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
