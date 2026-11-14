package brain

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInboxEdit(t *testing.T) {
	p, openWork := captureSpaces(t)
	if _, err := p.CaptureTo("work", "Purge php8.0", "2026-10-20"); err != nil {
		t.Fatal(err)
	}
	w := openWork()
	inbox := filepath.Join(w.Mem.Dir, "inbox.md")
	head, arrived, _ := strings.Cut(mustRead(t, inbox), "- 2026-10-15")
	if err := os.WriteFile(inbox, []byte(head+"- 2026-10-01 — Call Ana\n- 2026-10-01 — Call Ana\n- 2026-10-15"+arrived), 0o600); err != nil {
		t.Fatal(err)
	}

	// Only the first of two equal captures changes.
	if ok, err := w.InboxEdit("2026-10-01", "Call Ana", "Call Ana about the roof", "2026-10-09"); !ok || err != nil {
		t.Fatalf("edit: %v %v", ok, err)
	}
	// One sent from another space keeps where it came from; an empty reminder removes it.
	if ok, err := w.InboxEdit("2026-10-15", "Purge php8.0", "Purge php8.0 and php8.1", ""); !ok || err != nil {
		t.Fatalf("edit from another space: %v %v", ok, err)
	}
	got := mustRead(t, inbox)
	want := head + "- 2026-10-01 — Call Ana about the roof (remind: 2026-10-09)\n- 2026-10-01 — Call Ana\n- 2026-10-15 — Purge php8.0 and php8.1 (from personal)\n"
	if got != want {
		t.Errorf("inbox:\n%s\nwant\n%s", got, want)
	}

	// A mark typed as text stays text.
	if _, err := w.InboxEdit("2026-10-01", "Call Ana", "Call Ana (from personal)", ""); err != nil {
		t.Fatal(err)
	}
	if e, _ := w.Inbox(); e[1].From != "" || !strings.Contains(e[1].Text, "[from personal]") {
		t.Errorf("a typed origin passed for one: %+v", e[1])
	}

	// Refused: not there, empty, a secret, a bad date. Nothing changes.
	before := mustRead(t, inbox)
	if ok, _ := w.InboxEdit("2026-10-02", "Call Ana", "x", ""); ok {
		t.Error("found a capture that is not there")
	}
	for _, c := range [][2]string{{" ", ""}, {"token ghp_" + strings.Repeat("a1B2c3D4e5", 4)[:36], ""}, {"ok", "tomorrow"}} {
		if _, err := w.InboxEdit("2026-10-01", "Call Ana about the roof", c[0], c[1]); err == nil {
			t.Errorf("accepted %q %q", c[0], c[1])
		}
	}
	if mustRead(t, inbox) != before {
		t.Error("a refused edit changed the inbox")
	}
}

func mustRead(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
