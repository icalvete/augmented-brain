package hook

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/icalvete/augmented-brain/internal/backup"
	"github.com/icalvete/augmented-brain/internal/brain"
	"github.com/icalvete/augmented-brain/internal/filelock"
)

// With the notes lock held by another process (a backup, an import, a tool), part 1 of
// the session start waits for it once, StartWait at most, and still brings the
// header, the profile and the rules; parts 2 and on wait PartWait at most (N-09: part
// 1 used to wait three times three seconds, 9.3 s of the hook's ten).
func TestSessionStartDoesNotWaitLongForTheLock(t *testing.T) {
	cfgDir, work := setupSpace(t)
	withClaudePID(t, "5560")
	otherSession(t, cfgDir, func(b *brain.Brain) {
		if _, err := b.SaveProfile(brain.ProfileInput{Name: "Lucía"}); err != nil {
			t.Fatal(err)
		}
	})
	unlock, err := filelock.Lock(filepath.Join(cfgDir, backup.NotesLock))
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()

	began := time.Now()
	o := run(t, func(in *bytes.Reader, out *bytes.Buffer) error { return SessionStartPart(in, out, cfgDir, 1) },
		map[string]any{"cwd": work, "session_id": "s1", "source": "startup"})
	if took := time.Since(began); took > 3*time.Second || took < StartWait {
		t.Errorf("part 1 took %v with the lock held; StartWait is %v", took, StartWait)
	}
	text, _ := o["additionalContext"].(string)
	if !strings.Contains(text, "space work") || !strings.Contains(text, "Lucía") {
		t.Fatalf("part 1 without its header or profile:\n%s", text)
	}

	began = time.Now()
	run(t, func(in *bytes.Reader, out *bytes.Buffer) error { return SessionStartPart(in, out, cfgDir, 2) },
		map[string]any{"cwd": work, "session_id": "s1", "source": "startup"})
	if took := time.Since(began); took > PartWait+time.Second {
		t.Errorf("part 2 took %v with the lock held; PartWait is %v", took, PartWait)
	}
}
