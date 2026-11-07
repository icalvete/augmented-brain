package brain

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/format"
	"github.com/icalvete/augmented-brain/internal/setup"
)

// captureSpaces sets up "personal" and "work" and opens personal.
func captureSpaces(t *testing.T) (personal *Brain, openWork func() *Brain) {
	t.Helper()
	b := open(t, "en") // space "work"
	home := os.Getenv("HOME")
	other := filepath.Join(home, "me")
	if err := os.MkdirAll(other, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := setup.AddSpace(b.ConfigDir, setup.Options{Space: "personal", WorkFolder: other, Memory: "~/brain-personal"}); err != nil {
		t.Fatal(err)
	}
	openSpace := func(name string) *Brain {
		c, err := config.Load(b.ConfigDir)
		if err != nil {
			t.Fatal(err)
		}
		x, err := Open(c, c.Spaces[name])
		if err != nil {
			t.Fatal(err)
		}
		x.Now = b.Now
		return x
	}
	return openSpace("personal"), func() *Brain { return openSpace("work") }
}

func TestCaptureToAnotherSpace(t *testing.T) {
	p, openWork := captureSpaces(t)
	where, err := p.CaptureTo("work", "Purge php8.0 on the old server", "")
	if err != nil || where != "work" {
		t.Fatalf("CaptureTo = %q, %v", where, err)
	}
	if mine, _ := p.Inbox(); len(mine) != 0 {
		t.Fatalf("it also landed here: %v", mine)
	}
	w := openWork()
	got := w.Arrived()
	if len(got) != 1 || got[0].Text != "Purge php8.0 on the old server" || got[0].From != "personal" || got[0].Date != "2026-10-15" {
		t.Fatalf("Arrived = %+v", got)
	}
	// with a reminder: the reminder still reads, also for programs that know no origin
	if _, err := p.CaptureTo("work", "Check Jenkins", "2026-10-20"); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(filepath.Join(w.Mem.Dir, "inbox.md"))
	if !strings.Contains(string(raw), "- 2026-10-15 — Check Jenkins (from personal) (remind: 2026-10-20)\n") {
		t.Fatalf("inbox:\n%s", raw)
	}
	if r := openWork().Reminders(7); len(r) != 1 || r[0].Date != "2026-10-20" {
		t.Fatalf("Reminders = %+v", r)
	}
}

func TestCaptureToRefuses(t *testing.T) {
	p, _ := captureSpaces(t)
	if _, err := p.CaptureTo("nope", "x", ""); err == nil || !strings.Contains(err.Error(), "personal, work") {
		t.Fatalf("unknown space: %v", err)
	}
	if _, err := p.CaptureTo("work", "   ", ""); err == nil {
		t.Fatal("an empty capture went through")
	}
	if _, err := p.CaptureTo("work", "bad date", "tomorrow"); err == nil {
		t.Fatal("a bad date went through")
	}
	key := "AK" + "IA" + "QWERTYUIOPASDFGH" // built here, so this file holds no key-shaped text
	if _, err := p.CaptureTo("work", "the key is "+key, ""); err == nil {
		t.Fatal("a secret went to another space")
	}
}

// The same space, or none: a plain capture here.
func TestCaptureToSameSpace(t *testing.T) {
	p, _ := captureSpaces(t)
	for _, sp := range []string{"", "personal", " personal "} {
		if where, err := p.CaptureTo(sp, "here "+sp, ""); err != nil || where != "personal" {
			t.Fatalf("CaptureTo(%q) = %q, %v", sp, where, err)
		}
	}
	got, _ := p.Inbox()
	for _, e := range got {
		if e.From != "" {
			t.Fatalf("a capture here got an origin: %+v", e)
		}
	}
	if len(got) != 3 {
		t.Fatalf("inbox: %+v", got)
	}
}

// Notes written by a newer program are never written into.
func TestCaptureToNewerSpace(t *testing.T) {
	p, openWork := captureSpaces(t)
	w := openWork()
	if err := format.Write(w.Mem.Dir, format.Notes()+1); err != nil {
		t.Fatal(err)
	}
	_, err := p.CaptureTo("work", "x", "")
	var newer *format.NewerError
	if !errors.As(err, &newer) {
		t.Fatalf("err = %v", err)
	}
	if raw, _ := os.ReadFile(filepath.Join(w.Mem.Dir, "inbox.md")); strings.Contains(string(raw), "— x") {
		t.Fatal("it wrote into newer notes")
	}
}

func TestCaptureDone(t *testing.T) {
	p, openWork := captureSpaces(t)
	_, _ = p.CaptureTo("work", "Same words", "")
	w := openWork()
	if err := w.Capture("Same words", ""); err != nil { // one made here with the same text
		t.Fatal(err)
	}
	// capture_done with the text as listed (no origin): takes exactly one
	if found, err := w.InboxDoneEntry("2026-10-15", "Same words"); err != nil || !found {
		t.Fatal(found, err)
	}
	if left, _ := w.Inbox(); len(left) != 1 {
		t.Fatalf("left: %+v", left)
	}
	// with the origin in the text, as an earlier program reads it
	_, _ = p.CaptureTo("work", "Older reader", "")
	if found, _ := w.InboxDoneEntry("2026-10-15", "Older reader (from personal)"); !found {
		t.Fatal("the text with its origin was not found")
	}
	for _, e := range w.Arrived() {
		if e.Text == "Older reader" {
			t.Fatalf("still there: %+v", e)
		}
	}
	if found, _ := w.InboxDoneEntry("2026-10-15", "never captured"); found {
		t.Fatal("found what was never captured")
	}
}

// review_done takes processed captures by text. Asking for "X" never takes both "X"
// and "X (from personal)"; an earlier program asks with the origin in the text.
func TestInboxDoneWithOrigins(t *testing.T) {
	p, openWork := captureSpaces(t)
	w := openWork()
	_ = w.Capture("X", "")
	_, _ = p.CaptureTo("work", "X", "")
	_, _ = p.CaptureTo("work", "Only sent", "")
	_, _ = p.CaptureTo("work", "Old way", "")
	w = openWork()
	missing, err := w.InboxDone([]string{"X", "Only sent", "Old way (from personal)", "Not there"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(missing, ",") != "Not there" {
		t.Fatalf("missing = %v", missing)
	}
	left, _ := w.Inbox()
	if len(left) != 1 || left[0].Text != "X" || left[0].From != "personal" {
		t.Fatalf("left = %+v", left)
	}
}

// Taking captures out leaves the rest of the inbox file as it was.
func TestCaptureToKeepsTheInboxFile(t *testing.T) {
	p, openWork := captureSpaces(t)
	w := openWork()
	before, _ := os.ReadFile(filepath.Join(w.Mem.Dir, "inbox.md"))
	_, _ = p.CaptureTo("work", "One", "")
	if _, err := w.InboxDone([]string{"One"}); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(filepath.Join(w.Mem.Dir, "inbox.md"))
	if strings.TrimRight(string(after), "\n") != strings.TrimRight(string(before), "\n") {
		t.Fatalf("before:\n%s\nafter:\n%s", before, after)
	}
}
