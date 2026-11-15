package brain

import (
	"math/rand/v2"
	"regexp"
	"strings"
	"testing"
)

// The regular expressions splitFrom and splitRemind replace: both must split every
// text as these did.
var (
	oldFrom   = regexp.MustCompile(`^(.*) \(from ([A-Za-z0-9_.-]+)\)$`)
	oldRemind = regexp.MustCompile(`^(.*?)(?: \(remind: (\d{4}-\d{2}-\d{2})\))?\s*$`)
)

func checkSplit(t *testing.T, text string) {
	t.Helper()
	m := oldRemind.FindStringSubmatch(text)
	rest, date, ok := splitRemind(text)
	if ok != (m != nil) || ok && (rest != m[1] || date != m[2]) {
		t.Errorf("splitRemind(%q) = %q, %q, %v; the expression gave %q", text, rest, date, ok, m)
	}
	f := oldFrom.FindStringSubmatch(text)
	rest, space, ok := splitFrom(text)
	if ok != (f != nil) || ok && (rest != f[1] || space != f[2]) {
		t.Errorf("splitFrom(%q) = %q, %q, %v; the expression gave %q", text, rest, space, ok, f)
	}
}

func TestSplitLikeTheExpressions(t *testing.T) {
	for _, text := range []string{
		"", " ", "text", "text  ", "text\t\r", "a\nb", "a\n", "a\n\n ",
		"buy milk (remind: 2026-10-22)", "buy milk (remind: 2026-10-22)  ",
		"buy milk  (remind: 2026-10-22)", "x (remind: 2026-10-22) (remind: 2026-11-01)",
		"x (remind: 2026-1-22)", "x (remind: 20261022  )", "x(remind: 2026-10-22)",
		" (remind: 2026-10-22)", "(remind: 2026-10-22)", "x (remind: 2026-10-22)\n",
		"x (remind: 2026-10-22)\nmore", "x (remind: ２０26-10-22)",
		"idea (from personal)", "idea (from personal) ", "idea (from Amazon)",
		"idea (from per sonal)", "idea (from )", " (from a)", "(from a)",
		"a (from b) (from c)", "a (from b) c)", "a (from b-c.d_e)", "a (from b\n)",
		"a\nb (from c)", "idea (from personal) (remind: 2026-10-22)",
		"idea (from \xff)", "\xff (from a)", strings.Repeat("ab ", 500) + "(remind: 2026-10-22)",
	} {
		checkSplit(t, text)
	}
	pieces := []string{"a", "Z", "7", " ", "  ", "\t", "\n", "\r", "(", ")", "-", ".", "_", "é", "\xff",
		" (from ", "from", " (remind: ", "remind: ", "2026-10-22", "2026", "-10", ")", "personal"}
	r := rand.New(rand.NewPCG(1, 2))
	for range 20000 {
		var b strings.Builder
		for range r.IntN(9) {
			b.WriteString(pieces[r.IntN(len(pieces))])
		}
		checkSplit(t, b.String())
	}
}

// Inbox reads the lines again when the file changes, and its answer is the caller's to
// change.
func TestInboxRereadsAChangedFile(t *testing.T) {
	b := open(t, "en")
	if err := b.Capture("first", ""); err != nil {
		t.Fatal(err)
	}
	got, err := b.Inbox()
	if err != nil || len(got) != 1 {
		t.Fatalf("Inbox = %v, %v; want one capture", got, err)
	}
	got[0].Text = "changed by the caller"
	if again, _ := b.Inbox(); len(again) != 1 || again[0].Text != "first" {
		t.Fatalf("Inbox after the caller changed its answer = %v", again)
	}
	other := open2(t, b)
	if err := other.Capture("second", ""); err != nil {
		t.Fatal(err)
	}
	if again, _ := b.Inbox(); len(again) != 2 {
		t.Fatalf("Inbox after another process wrote = %v, want two captures", again)
	}
}
