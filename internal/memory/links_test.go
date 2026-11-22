package memory

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBrokenLinks(t *testing.T) {
	dir := t.TempDir()
	write := func(rel, text string) {
		t.Helper()
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("rules/rule-core.md", "---\nid: rule-core\ntype: rule\ntitle: Core\n---\nSee [[rule-core-tests-solo]] and [[ref-deploy|how to deploy]], [[lesson-old]], [[id]], [[…]], [[profile]] and [[ref-other-space]].\n")
	write("knowledge/ref-deploy.md", "---\nid: ref-deploy\ntype: reference\ntitle: Deploy\n---\nAgain [[rule-core-tests-solo]] twice: [[rule-core-tests-solo]].\n")
	write("tasks/task-done.md", "---\nid: task-done\ntype: task\ntitle: Done\nstatus: closed\n---\n[[user-profile]] in history.\n")
	write("archive/rules/lesson-old.md", "retired\n")
	m, problems, err := Load(dir)
	if err != nil || len(problems) > 0 {
		t.Fatal(err, problems)
	}
	other := &Memory{Notes: map[string]*Note{"ref-other-space": {}}}
	got := m.BrokenLinks(other)
	want := []BrokenLink{{"knowledge/ref-deploy.md", "rule-core-tests-solo"}, {"rules/rule-core.md", "rule-core-tests-solo"}}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("broken links: %v, want %v", got, want)
	}
}
