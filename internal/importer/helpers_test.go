package importer

import (
	"errors"
	"testing"

	"github.com/icalvete/augmented-brain/internal/memory"
)

func TestKindOfID(t *testing.T) {
	for id, want := range map[string]string{
		"task-x": "task", "area-x": "area", "ref-x": "reference", "reference-x": "reference",
		"lesson-x": "lesson", "rule-x": "rule", "person-x": "person",
		"review-x": "", "profile": "", "notes": "",
	} {
		if got := kindOfID(id); got != want {
			t.Errorf("kindOfID(%q) = %q, want %q", id, got, want)
		}
	}
}

func TestRead(t *testing.T) {
	s, err := read("a.md", []byte("---\nname: x\nmetadata:\n  type: feedback\n---\n\nbody\n"))
	if err != nil || s.str("name") != "x" || s.claudeType() != "feedback" || s.body != "\nbody\n" {
		t.Fatalf("read: %+v, %v", s, err)
	}
	if _, err := read("a.md", []byte("# Index\n")); !errors.Is(err, memory.ErrNoHeader) {
		t.Errorf("no header: %v", err)
	}
	if _, err := read("a.md", []byte("---\nname: x\n")); err == nil || err.Error() != "header not closed" {
		t.Errorf("not closed: %v", err)
	}
}
