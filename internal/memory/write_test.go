package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	n, err := Parse("tasks/task-backup.md", []byte(task))
	if err != nil {
		t.Fatal(err)
	}
	n.Header.Status = "execution"
	if err := Save(dir, n); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "tasks", "task-backup.md"))
	if err != nil {
		t.Fatal(err)
	}
	back, err := Parse("tasks/task-backup.md", data)
	if err != nil {
		t.Fatalf("saved note does not parse: %v\n%s", err, data)
	}
	if back.Header.Status != "execution" || back.Body != n.Body || len(back.Header.Watch) != 1 {
		t.Errorf("round trip changed the note:\n%s", data)
	}
	if !strings.HasPrefix(string(data), "---\nid: task-backup\ntype: task\n") {
		t.Errorf("header order:\n%s", data)
	}

	n.Header.Status = "doing"
	if err := Save(dir, n); err == nil {
		t.Error("an invalid note was saved")
	}
}
