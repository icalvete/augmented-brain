package mcpserver

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A task longer than what a tool may answer (written by hand, or by an earlier program)
// is read with its oldest journal entries left out and a line saying where they are; the
// note itself is not changed.
func TestTaskGetShortensALongJournal(t *testing.T) {
	open := space(t)
	cs := connect(t, open)
	if _, isErr := call(t, cs, "task_new", map[string]any{"title": "Long", "summary": "by hand"}); isErr {
		t.Fatal("task_new failed")
	}
	b, err := open()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(b.Mem.Dir, "tasks", "task-long.md")
	data, _ := os.ReadFile(path)
	var extra strings.Builder
	for i := 0; i < 400; i++ {
		fmt.Fprintf(&extra, "- 2026-10-15 — entry %d %s\n", i, strings.Repeat("z", 200))
	}
	data = append(data, extra.String()...)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	out, isErr := call(t, cs, "task_get", map[string]any{"id": "task-long"})
	if isErr || len(out) > TaskGetMax+2000 {
		t.Fatalf("task_get gave %d bytes (error %v)", len(out), isErr)
	}
	if !strings.Contains(out, "entry 399 ") || strings.Contains(out, "entry 0 ") || !strings.Contains(out, "left out here") || !strings.Contains(out, path) {
		t.Errorf("task_get:\n%.600s", out)
	}
	if !strings.Contains(out, "## ") || !strings.Contains(out, "id: task-long") {
		t.Error("the header or the sections were cut")
	}
	if after, _ := os.ReadFile(path); string(after) != string(data) {
		t.Error("reading the task changed it")
	}
}
