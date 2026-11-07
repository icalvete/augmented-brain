package mcpserver

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/icalvete/augmented-brain/internal/brain"
	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/setup"
)

// From a session of one space, capture with space sends it to the other, whose
// session lists it and takes it out with capture_done.
func TestCaptureToAnotherSpaceThroughTools(t *testing.T) {
	openWork := space(t)
	home := os.Getenv("HOME")
	cfgDir := filepath.Join(home, "cfg")
	other := filepath.Join(home, "me")
	if err := os.MkdirAll(other, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := setup.AddSpace(cfgDir, setup.Options{Space: "personal", WorkFolder: other, Memory: "~/brain-personal"}); err != nil {
		t.Fatal(err)
	}
	openPersonal := func() (*brain.Brain, error) {
		c, err := config.Load(cfgDir)
		if err != nil {
			return nil, err
		}
		b, err := brain.Open(c, c.Spaces["personal"])
		if b != nil {
			b.Now = func() time.Time { return time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC) }
		}
		return b, err
	}
	p, w := connect(t, openPersonal), connect(t, openWork)

	out := mustCall(t, p, "capture", map[string]any{"text": "Purge php8.0 on the old server", "space": "work"})
	if !strings.Contains(out, "inbox of space work") {
		t.Fatalf("capture: %s", out)
	}
	if out, isErr := call(t, p, "capture", map[string]any{"text": "x", "space": "nowhere"}); !isErr || !strings.Contains(out, "personal, work") {
		t.Fatalf("unknown space: %v %s", isErr, out)
	}
	review := mustCall(t, w, "weekly_review", nil)
	if !strings.Contains(review, "Purge php8.0 on the old server [from space personal]") {
		t.Fatalf("weekly_review in work:\n%s", review)
	}
	if out, isErr := call(t, w, "capture_done", map[string]any{"date": "2026-10-15", "text": "not this"}); !isErr || !strings.Contains(out, "weekly_review") {
		t.Fatalf("capture_done of nothing: %v %s", isErr, out)
	}
	mustCall(t, w, "capture_done", map[string]any{"date": "2026-10-15", "text": "Purge php8.0 on the old server"})
	if review := mustCall(t, w, "weekly_review", nil); strings.Contains(review, "Purge php8.0") {
		t.Fatalf("still in the inbox:\n%s", review)
	}
	if review := mustCall(t, p, "weekly_review", nil); strings.Contains(review, "Purge php8.0") {
		t.Fatalf("it landed in personal too:\n%s", review)
	}
}
