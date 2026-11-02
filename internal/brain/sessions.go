package brain

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/icalvete/augmented-brain/internal/guard"
	"github.com/icalvete/augmented-brain/internal/paths"
)

var journalDate = regexp.MustCompile(`(?m)^- (\d{4}-\d{2}-\d{2}) — `)

// LastEntry is the date of the last journal entry of a note, at midnight local time;
// zero when it has none or does not exist.
func (b *Brain) LastEntry(id string) time.Time {
	n, ok := b.Mem.Notes[id]
	if !ok {
		return time.Time{}
	}
	dates := journalDate.FindAllStringSubmatch(n.Body, -1)
	if len(dates) == 0 {
		return time.Time{}
	}
	d, err := time.ParseInLocation("2006-01-02", dates[len(dates)-1][1], time.Local)
	if err != nil {
		return time.Time{}
	}
	return d
}

// SaveSession keeps text from a conversation in archive/sessions/ of the space (never
// loaded at the start, backed up like any note), under heading, with secrets taken
// out. It returns the path, relative to the notes folder. kind goes in the file name
// ("compacted", "recovered"), after the date and the start of the session's id.
// The caller holds the notes lock.
func (b *Brain) SaveSession(kind, sessionID, heading, text string) (string, error) {
	now := b.Now()
	id := sessionID
	if len(id) > 8 {
		id = id[:8]
	}
	if id == "" {
		id = "session"
	}
	name := fmt.Sprintf("%s-%s-%s", now.Format("2006-01-02"), now.Format("150405"), id)
	if kind != "compacted" {
		name += "-" + kind
	}
	rel := "archive/sessions/" + name + ".md"
	path := filepath.Join(b.Space.Memory, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	body := strings.TrimRight(heading, "\n") + "\n\n" + guard.RedactSecrets(strings.TrimSpace(text)) + "\n"
	if err := paths.WriteAtomic(path, []byte(body), 0o644); err != nil {
		return "", err
	}
	return rel, nil
}
