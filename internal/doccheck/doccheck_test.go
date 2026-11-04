package doccheck

import (
	"slices"
	"strings"
	"testing"
)

func TestSpansAndMentions(t *testing.T) {
	if got := Spans("a `--all` and ``x ` y`` and `open"); !slices.Equal(got, []string{"--all", "x ` y"}) {
		t.Errorf("Spans = %q", got)
	}
	code := []string{"backup --all", "task_id", "stopped", "[spaces.work]"}
	for token, want := range map[string]bool{"--all": true, "id": false, "stop": false, "spaces": true, "--al": false} {
		if Mentions(code, token) != want {
			t.Errorf("Mentions(%q) = %v", token, !want)
		}
	}
}

func TestParse(t *testing.T) {
	p := Parse("x.md", strings.Join([]string{
		"`top`",
		"## augmented-brain backup",
		"`--all`",
		"```",
		"## augmented-brain fenced",
		"`--space`",
		"```",
		"### More",
		"`--auto`",
		"## `task_new`",
		"`title`",
	}, "\n"))
	var heads []string
	for _, s := range p.Sections {
		heads = append(heads, s.Heading)
	}
	if !slices.Equal(heads, []string{"augmented-brain backup", "`task_new`"}) {
		t.Fatalf("headings = %q", heads)
	}
	if got := p.Sections[0].Code; !slices.Equal(got, []string{"--all", "--auto"}) {
		t.Errorf("backup's code = %q", got)
	}
	if got := p.Sections[1].Code; !slices.Equal(got, []string{"title"}) {
		t.Errorf("task_new's code = %q", got)
	}
	want := map[string][]string{"augmented-brain backup": {"--all", "--space"}, "augmented-brain gone": {}}
	gaps := Check(p, []string{"augmented-brain backup", "augmented-brain gone"}, want,
		func(h string) bool { return strings.HasPrefix(h, "augmented-brain ") }, "command")
	if !slices.Equal(gaps, []string{`x.md: section "## augmented-brain backup" lacks ` + "`--space`", `x.md: no section "## augmented-brain gone"`}) {
		t.Errorf("gaps = %q", gaps)
	}
}
