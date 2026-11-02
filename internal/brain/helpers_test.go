package brain

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestParseInboxLine(t *testing.T) {
	for line, want := range map[string]InboxEntry{
		"- 2026-10-17 — Look at the CI":                              {Date: "2026-10-17", Text: "Look at the CI"},
		"- 2026-10-17 — Call the insurance (remind: 2026-10-22)":     {Date: "2026-10-17", Text: "Call the insurance", Remind: "2026-10-22"},
		"- 2026-10-17 — Trailing spaces and a CR   \r":               {Date: "2026-10-17", Text: "Trailing spaces and a CR"},
		"- 2026-10-17 — (remind: soon) is not a date, so it is text": {Date: "2026-10-17", Text: "(remind: soon) is not a date, so it is text"},
	} {
		if got, ok := parseInboxLine(line); !ok || got != want {
			t.Errorf("parseInboxLine(%q) = %+v, %v", line, got, ok)
		}
	}
	if _, ok := parseInboxLine("# Inbox"); ok {
		t.Error("a heading is not a capture")
	}
}

// A person whose save fails stays in the index as they were.
func TestSavePersonFailedKeepsIndex(t *testing.T) {
	if runtime.GOOS == "windows" || os.Getuid() == 0 {
		t.Skip("needs a folder this user cannot write in")
	}
	b := open(t, "en")
	p, err := b.SavePerson(PersonInput{Name: "Ana", Role: "my manager"})
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(b.Mem.Dir, "people")
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o755) })
	if _, err := b.SavePerson(PersonInput{Name: "Ana", Role: "my client", Notes: "new notes"}); err == nil {
		t.Fatal("the save should have failed")
	}
	got := b.Mem.Notes[p.Header.ID]
	if got.Header.Summary != "my manager" || got.Body != "" {
		t.Errorf("the index changed after a failed save: %+v", got)
	}
}
