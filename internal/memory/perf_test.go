package memory

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// bigSpace writes a space of n tasks, a quarter of them live and one in ten of the live
// ones a subtask, with a journal each, plus areas, knowledge and people, the shape of a
// space used for a couple of years. It returns the folder.
func bigSpace(tb testing.TB, n int) string {
	tb.Helper()
	dir := tb.TempDir()
	write := func(rel, body string) {
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			tb.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			tb.Fatal(err)
		}
	}
	for a := range 8 {
		id := fmt.Sprintf("area-%d", a)
		write("areas/"+id+".md", "---\nid: "+id+"\ntype: area\ntitle: Area "+id+"\nsummary: s\n---\n")
	}
	journal := strings.Repeat("- 2026-11-01 — Did a thing and wrote it down, with its reason.\n", 20)
	parent := ""
	for i := range n {
		id := fmt.Sprintf("task-%05d", i)
		status, extra := "closed", ""
		if i%4 == 0 {
			status = []string{"plan", "execution", "parked", "definition"}[i/4%4]
			if i%40 == 0 {
				parent = id
			} else if i%10 == 0 && parent != "" {
				extra = "parent: " + parent + "\n"
			}
		}
		write("tasks/"+id+".md", fmt.Sprintf("---\nid: %s\ntype: task\ntitle: Task %d\nsummary: s\narea: area-%d\n%sstatus: %s\nwaiting_on: me\nnext: do it\n---\n## Plan\n- [x] one\n- [ ] two\n\n## Diario\n%s",
			id, i, i%8, extra, status, journal))
	}
	for i := range n / 4 {
		id := fmt.Sprintf("lesson-%05d", i)
		write("knowledge/"+id+".md", "---\nid: "+id+"\ntype: lesson\ntitle: Lesson "+id+"\nsummary: s\n---\nBody.\n")
	}
	for i := range n / 20 {
		id := fmt.Sprintf("person-%04d", i)
		write("people/"+id+".md", "---\nid: "+id+"\ntype: person\ntitle: Person "+id+"\nsummary: s\n---\n")
	}
	return dir
}

// naiveChildren is Children as it was written first: every task, in id order.
func naiveChildren(m *Memory, id string) []*Note {
	ids := make([]string, 0, len(m.Notes))
	for k := range m.Notes {
		ids = append(ids, k)
	}
	sort.Strings(ids)
	var out []*Note
	for _, k := range ids {
		n := m.Notes[k]
		if n.Header.Type == "task" && n.Header.Parent == id && m.ParentOf(n) != nil {
			out = append(out, n)
		}
	}
	return out
}

// The index of a large space is built in N·log N (N-13): it was quadratic, 150 ms with
// 2,000 tasks and 2.8 s with 8,000. Children and OfType keep their order.
func TestIndexOfALargeSpace(t *testing.T) {
	m, _, err := Load(bigSpace(t, 2000))
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"task-00000", "task-00040", "task-00004", "task-00001", "nothing"} {
		got, want := m.Children(id), naiveChildren(m, id)
		if len(got) != len(want) {
			t.Fatalf("Children(%s): %d, want %d", id, len(got), len(want))
		}
		for i := range got {
			if got[i] != want[i] {
				t.Fatalf("Children(%s)[%d] = %s, want %s", id, i, got[i].Header.ID, want[i].Header.ID)
			}
		}
	}
	if len(m.Children("task-00000")) == 0 {
		t.Fatal("the generated space has no subtasks")
	}
	tasks := m.OfType("task")
	if len(tasks) != 2000 || !sort.SliceIsSorted(tasks, func(i, j int) bool { return tasks[i].Header.ID < tasks[j].Header.ID }) {
		t.Fatalf("OfType: %d tasks, sorted: %v", len(tasks), sort.SliceIsSorted(tasks, func(i, j int) bool { return tasks[i].Header.ID < tasks[j].Header.ID }))
	}
	idx := m.Index("en")
	if !strings.Contains(idx, "- [Task 40](tasks/task-00040.md)") || !strings.Contains(idx, "\n  - [Task 60](tasks/task-00060.md)") {
		t.Fatal("the index lost a task or a subtask")
	}
	// Byte for byte the index the quadratic code wrote.
	for lang, want := range map[string]string{"en": indexSum["en"], "es": indexSum["es"]} {
		if got := fmt.Sprintf("%x", sha256.Sum256([]byte(m.Index(lang)))); got != want {
			t.Errorf("Index(%s) changed: sha256 %s, want %s", lang, got, want)
		}
		if got := fmt.Sprintf("%x", sha256.Sum256([]byte(m.SessionIndex(lang)))); got != sessionSum[lang] {
			t.Errorf("SessionIndex(%s) changed: sha256 %s, want %s", lang, got, sessionSum[lang])
		}
	}
}

// The sums of the indexes of bigSpace(2000) as the code before N-13 wrote them.
var (
	indexSum = map[string]string{
		"en": "8b59bb6ab94362d94ba05bbf35afd5305fa22c860571f89f004d5ae21b41b0e0",
		"es": "b379c02774cec88888e1c28abb59c7ad7332b5a0d977fc146e6568245924e46a",
	}
	sessionSum = map[string]string{
		"en": "71faa66acaec653241ab15fe2ff2c0bdc892606045617fb42abb45c7124bf84a",
		"es": "7cdbbe78eb230a5660bd79b4f8804092a8cc0f5af036613be4213a6796d1d311",
	}
)

func benchSpace(b *testing.B, n int) *Memory {
	m, _, err := Load(bigSpace(b, n))
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	return m
}

func BenchmarkIndex2000(b *testing.B) {
	m := benchSpace(b, 2000)
	for b.Loop() {
		_ = m.Index("en")
	}
}

func BenchmarkIndex8000(b *testing.B) {
	m := benchSpace(b, 8000)
	for b.Loop() {
		_ = m.Index("en")
	}
}

func BenchmarkLoad2000(b *testing.B) {
	dir := bigSpace(b, 2000)
	b.ReportAllocs()
	for b.Loop() {
		if _, _, err := Load(dir); err != nil {
			b.Fatal(err)
		}
	}
}
