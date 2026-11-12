package memory

import (
	"strings"
	"testing"
)

func TestSplitJournal(t *testing.T) {
	big := strings.Repeat("w", 3000)
	body := "## Plan\n- 2026-01-01 — not the journal: another section\n## Journal\nSome words first.\n" +
		"- 2026-10-01 — one " + big + "\n  detail\n" +
		"- 2026-10-02 — two " + big + "\n```\n- 2026-10-09 — inside a code block\n```\n" +
		"- 2026-10-03 — three " + big + "\n" +
		"- 2026-10-04 — four\n"
	s, ok := SplitJournal(body, 3500)
	if !ok {
		t.Fatal("nothing to move")
	}
	if s.Before+s.Moved+s.After != body {
		t.Fatal("the parts do not make the body")
	}
	if !strings.HasSuffix(s.Before, "Some words first.\n") || s.First != "2026-10-01" || s.Last != "2026-10-02" || s.Entries != 2 {
		t.Errorf("split: before %q, %s to %s, %d entries", s.Before[len(s.Before)-20:], s.First, s.Last, s.Entries)
	}
	if !strings.Contains(s.Moved, "inside a code block") || !strings.HasPrefix(s.After, "- 2026-10-03") {
		t.Errorf("moved %q… after %q…", s.Moved[:20], s.After[:20])
	}
	// Too little to move, or a single entry: nothing.
	if _, ok := SplitJournal("## Journal\n- 2026-10-01 — a\n- 2026-10-02 — b\n", 1); ok {
		t.Error("moved less than JournalMinMove")
	}
	if _, ok := SplitJournal("## Journal\n- 2026-10-01 — "+big+big+"\n", 1); ok {
		t.Error("moved the only entry")
	}
}

func TestExpandCollapseJournal(t *testing.T) {
	rel := "archive/journals/task-x-2026-10-01-2026-10-02.md"
	file := "# X · journal\n\nMoved here.\n\n- 2026-10-01 — a\n  detail\n- 2026-10-02 — b\n"
	link := "- 2026-10-02 — 2 older entries moved to " + rel + ".\n"
	note := "## Journal\n" + link + "- 2026-10-03 — c\n"
	read := func(r string) (string, bool) { return file, r == rel }
	whole, links := ExpandJournal(note, read)
	if whole != "## Journal\n- 2026-10-01 — a\n  detail\n- 2026-10-02 — b\n- 2026-10-03 — c\n" || len(links) != 1 {
		t.Fatalf("expanded:\n%s", whole)
	}
	back, ok := CollapseJournal(whole+"- 2026-10-04 — d\n", links)
	if !ok || back != note+"- 2026-10-04 — d\n" {
		t.Errorf("collapsed (%v):\n%s", ok, back)
	}
	if _, ok := CollapseJournal("## Journal\n- 2026-10-03 — c\n", links); ok {
		t.Error("collapsed entries that are not there")
	}
	// A line whose file is missing stays as it is.
	if got, l := ExpandJournal(note, func(string) (string, bool) { return "", false }); got != note || len(l) != 0 {
		t.Errorf("expanded without the file:\n%s", got)
	}
	if _, ok := ArchiveLink("see archive/journals/x.md"); ok {
		t.Error("an undated line is not a link line")
	}
}
