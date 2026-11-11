package brain

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/icalvete/augmented-brain/internal/state"
)

// Recover is the one way the MCP tool and the panel read a dead session: by id, from
// where the notes stop, the last max characters, and never a running or ended session.
func TestRecover(t *testing.T) {
	b := open(t, "en")
	now := time.Now()
	dir := t.TempDir()
	msg := func(at time.Time, text string) string {
		return `{"type":"user","timestamp":"` + at.UTC().Format(time.RFC3339) + `","message":{"content":"` + text + `"}}` + "\n"
	}
	write := func(id, body string, mod time.Time) string {
		t.Helper()
		p := filepath.Join(dir, id+".jsonl")
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(p, mod, mod); err != nil {
			t.Fatal(err)
		}
		return p
	}
	long := strings.Repeat("x", RecoverForAssistant)
	dead := write("dead", msg(now.Add(-50*time.Minute), "already journaled")+msg(now.Add(-30*time.Minute), "not yet "+long), now.Add(-20*time.Minute))
	live := write("live", msg(now.Add(-time.Minute), "running"), now)
	if err := state.Update(b.ConfigDir, b.Space.Name, now, func(s *state.Space) {
		s.Sessions = []state.Session{
			{ID: "dead", Transcript: dead, Started: now.Add(-time.Hour), Task: "task-x", Since: now.Add(-40 * time.Minute), Reported: true},
			{ID: "live", Transcript: live, Started: now.Add(-time.Hour)},
		}
	}); err != nil {
		t.Fatal(err)
	}

	if id := b.Offered(now); id != "dead" {
		t.Errorf("offered %q, want dead", id)
	}
	for _, id := range []string{"", "live", "missing"} {
		if _, err := b.Recover(id, RecoverForArchive, now); !errors.Is(err, ErrNoDeadSession) {
			t.Errorf("%q: %v, want ErrNoDeadSession", id, err)
		}
	}
	whole, err := b.Recover("dead", RecoverForArchive, now)
	if err != nil {
		t.Fatal(err)
	}
	if !whole.From.Equal(now.Add(-40*time.Minute)) || strings.Contains(whole.Text, "already journaled") ||
		!strings.HasPrefix(whole.Text, "user: not yet") || whole.Session.ID != "dead" {
		t.Errorf("recover: from %v, %.40q", whole.From, whole.Text)
	}
	short, err := b.Recover("dead", RecoverForAssistant, now)
	if err != nil {
		t.Fatal(err)
	}
	if n := len([]rune(short.Text)); n != RecoverForAssistant+1 || !strings.HasPrefix(short.Text, "…") {
		t.Errorf("the assistant's copy: %d characters, %.10q", n, short.Text)
	}
	// Reading never marks the session as dealt with: the caller does.
	if b.Offered(now) != "dead" {
		t.Error("Recover dismissed the session")
	}
}
