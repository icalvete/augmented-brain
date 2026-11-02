package memory

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

func TestPathFor(t *testing.T) {
	for _, c := range []struct{ typ, id, want string }{
		{"task", "task-x", "tasks/task-x.md"},
		{"area", "area-x", "areas/area-x.md"},
		{"reference", "ref-x", "knowledge/ref-x.md"},
		{"lesson", "lesson-x", "knowledge/lesson-x.md"},
		{"rule", "rule-x", "rules/rule-x.md"},
		{"review", "review-2026-11-02", "reviews/review-2026-11-02.md"},
		{"person", "person-x", "people/person-x.md"},
		{"profile", "profile", "profile.md"},
	} {
		if got := PathFor(c.typ, c.id); got != c.want {
			t.Errorf("PathFor(%q, %q) = %q, want %q", c.typ, c.id, got, c.want)
		}
		// Every path PathFor builds is one check accepts for that type.
		n := &Note{Path: c.want, Header: Header{ID: c.id, Type: c.typ, Title: "x", Status: "plan", WaitingOn: "me", Next: "x"}}
		if c.typ != "task" {
			n.Header.Status, n.Header.WaitingOn, n.Header.Next = "", "", ""
		}
		if err := n.check(); err != nil {
			t.Errorf("%s: %v", c.want, err)
		}
	}
}

func TestTypeOfID(t *testing.T) {
	for id, want := range map[string]string{
		"task-x": "task", "area-x": "area", "ref-x": "reference", "lesson-x": "lesson",
		"rule-x": "rule", "review-x": "review", "person-x": "person",
		"profile": "", "reference-x": "", "x": "",
	} {
		if got := TypeOfID(id); got != want {
			t.Errorf("TypeOfID(%q) = %q, want %q", id, got, want)
		}
	}
}

func TestSplitHeader(t *testing.T) {
	for _, c := range []struct{ in, head, body string }{
		{"---\na: 1\n---\nbody\n", "a: 1", "body\n"},
		{"\xef\xbb\xbf---\r\na: 1\r\n---\r\nbody\r\n", "a: 1", "body\n"},
		{"---\na: 1\n---\n", "a: 1", ""},
		{"---\na: 1\n---", "a: 1", ""},
	} {
		head, body, err := SplitHeader([]byte(c.in))
		if err != nil || string(head) != c.head || string(body) != c.body {
			t.Errorf("SplitHeader(%q) = %q, %q, %v", c.in, head, body, err)
		}
	}
	if _, _, err := SplitHeader([]byte("# no header\n")); !errors.Is(err, ErrNoHeader) {
		t.Errorf("no header: %v", err)
	}
	if _, _, err := SplitHeader([]byte("---\na: 1\n")); !errors.Is(err, ErrNotClosed) {
		t.Errorf("not closed: %v", err)
	}
}

func TestDaysBetween(t *testing.T) {
	for _, c := range []struct {
		from, to string
		days     int
		ok       bool
	}{
		{"2026-11-02", "2026-11-02", 0, true},
		{"2026-11-02", "2026-11-09", 7, true},
		{"2026-11-02", "2026-10-31", -2, true},
		{"2026-03-28", "2026-03-30", 2, true}, // across a change of clock in Europe
		{"2026-11-02", "soon", 0, false},
		{"", "2026-11-02", 0, false},
	} {
		if days, ok := DaysBetween(c.from, c.to); days != c.days || ok != c.ok {
			t.Errorf("DaysBetween(%q, %q) = %d, %v", c.from, c.to, days, ok)
		}
	}
}

func TestDatedLine(t *testing.T) {
	line := DatedLine("2026-11-02", "Called the bank")
	if line != "- 2026-11-02 — Called the bank\n" {
		t.Fatalf("DatedLine: %q", line)
	}
	date, text, ok := ParseDatedLine(strings.TrimSuffix(line, "\n"))
	if !ok || date != "2026-11-02" || text != "Called the bank" {
		t.Errorf("ParseDatedLine: %q %q %v", date, text, ok)
	}
	for _, l := range []string{"## Journal", "- 2026-11-02 - hyphen, not a dash", "- 26-11-02 — short year", "Some prose"} {
		if _, _, ok := ParseDatedLine(l); ok {
			t.Errorf("ParseDatedLine(%q) took it for a dated line", l)
		}
	}
}

func TestHeadings(t *testing.T) {
	body := "intro\n## One\ntext\n```\n## not a heading\n```\n### three\n## Two\n"
	got := Headings(strings.SplitAfter(body, "\n"))
	if want := []int{1, 7}; !slices.Equal(got, want) {
		t.Errorf("Headings = %v, want %v", got, want)
	}
	if got := Headings(strings.SplitAfter("no headings\n", "\n")); got != nil {
		t.Errorf("Headings with none = %v", got)
	}
}
