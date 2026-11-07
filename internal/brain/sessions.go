package brain

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/icalvete/augmented-brain/internal/guard"
	"github.com/icalvete/augmented-brain/internal/paths"
	"github.com/icalvete/augmented-brain/internal/state"
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

// Touched and Released keep this machine's state of the space (package state): which
// task is active and when it was last written, for the Stop hook's reminder. The MCP
// tools and the panel call them after a change; a failure there never fails the change.
func (b *Brain) Touched(id string, write bool) {
	b.read = append(b.read, id)
	if b.ConfigDir != "" {
		_ = state.Touched(b.ConfigDir, b.Space.Name, id, time.Now(), write)
	}
}

// Released: see Touched.
func (b *Brain) Released(id string) {
	if b.ConfigDir != "" {
		_ = state.Released(b.ConfigDir, b.Space.Name, id, time.Now())
	}
}

// How much of a dead session's conversation a recovery reads: the last part, when
// there is more. The assistant gets less than the archive keeps, because a tool's
// answer must stay well within what Claude Code accepts from an MCP server (about
// 25,000 tokens); the panel has no such limit, so its archive keeps more.
const (
	RecoverForAssistant = 20000  // session_recover, read by the assistant
	RecoverForArchive   = 200000 // the panel, kept in archive/sessions/
)

// ErrNoDeadSession is returned by Recover when the session asked for is not a dead one
// of the space: unknown, ended, running, or without a transcript on this machine.
var ErrNoDeadSession = errors.New("no such session to recover")

// Recovery is what a dead session said after the last journal entry of its task.
type Recovery struct {
	Session state.Session
	From    time.Time // where the text starts: what may be missing from the notes
	Text    string    // the user's and the assistant's messages only; empty when nothing was said
}

// Recover reads a dead session of the space (one that never ended and is not running)
// from the last journal entry of its task on, at most max characters. Both the MCP tool
// and the panel recover through it, so they read the same part of the conversation.
// It does not mark the session as dealt with: the caller does, with state.Dismissed,
// once the text is safe.
func (b *Brain) Recover(id string, max int, now time.Time) (Recovery, error) {
	st := state.Read(b.ConfigDir, b.Space.Name)
	// The session asking is running, so its transcript is fresh and Unclosed leaves it out.
	for _, se := range st.Unclosed("", now) {
		if id != "" && se.ID == id {
			from := st.From(se, b.LastEntry(se.Task))
			text, err := state.Conversation(se.Transcript, from, max)
			return Recovery{Session: se, From: from, Text: text}, err
		}
	}
	return Recovery{}, ErrNoDeadSession
}

// Offered is the dead session a session start offered to recover; empty when none is
// waiting.
func (b *Brain) Offered(now time.Time) string {
	for _, se := range state.Read(b.ConfigDir, b.Space.Name).Unclosed("", now) {
		if se.Reported {
			return se.ID
		}
	}
	return ""
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
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", err
	}
	body := strings.TrimRight(heading, "\n") + "\n\n" + guard.RedactSecrets(strings.TrimSpace(text)) + "\n"
	if err := paths.WriteAtomic(path, []byte(body), 0o600); err != nil {
		return "", err
	}
	return rel, nil
}
