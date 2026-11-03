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

// A piece of knowledge that lists people, one section each, is named in the report so
// each can become a person; other notes with sections are not.
func TestPeopleIn(t *testing.T) {
	n := func(id, title, body string) *memory.Note {
		return &memory.Note{Header: memory.Header{ID: id, Type: "reference", Title: title}, Body: body}
	}
	got := peopleIn(n("ref-team-people", "People I work with", "intro\n\n## Ana Ruiz\n- baker\n\n## Leo\n- driver\n"))
	if len(got) != 2 || got[0] != "Ana Ruiz" || got[1] != "Leo" {
		t.Errorf("people: %q", got)
	}
	if got := peopleIn(n("ref-oven", "How the oven works", "## Heat\n\n## Steam\n")); got != nil {
		t.Errorf("not about people: %q", got)
	}
	if got := peopleIn(n("ref-personal-setup", "My personal setup", "## Laptop\n\n## Phone\n")); got != nil {
		t.Errorf("personal is not people: %q", got)
	}
	if got := peopleIn(n("ref-equipo", "Equipo", "## Ana\n")); got != nil {
		t.Errorf("a single section is no list: %q", got)
	}
}
