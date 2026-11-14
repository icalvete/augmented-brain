package hook

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/icalvete/augmented-brain/internal/paths"
)

// The session start runs nine hooks at once, one per part of the session text. Part 1
// builds all of them (session.AllParts) and leaves them here, one file per session;
// the other parts read theirs instead of reading and composing the whole space again
// (finding F-03). A part that finds no file in time composes its own, as before.

// PartsDir, in the configuration folder, holds the parts part 1 built.
const PartsDir = "state/session-parts"

// partsSlack is how long before a part started the parts it takes may have been
// written: those of an earlier start of the same session (a second compaction) are
// older than that.
const partsSlack = 5 * time.Second

// partsKeep is how long a file of parts is kept when its session never ended.
const partsKeep = 24 * time.Hour

// storedParts is what part 1 leaves for the others.
type storedParts struct {
	Session string    `json:"session"`
	Space   string    `json:"space"`
	Source  string    `json:"source"`
	Written time.Time `json:"written"`
	Parts   []string  `json:"parts"`
}

// partsFile is the file of a session's parts; "" when the session id cannot name a
// file (empty, or with other characters than letters, digits, '-' and '_').
func partsFile(cfgDir, session string) string {
	if session == "" || len(session) > 128 || strings.IndexFunc(session, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_')
	}) >= 0 {
		return ""
	}
	return filepath.Join(cfgDir, filepath.FromSlash(PartsDir), session+".json")
}

// saveParts leaves the parts of this start for the other parts' hooks. It reports
// whether they were saved: if not, the others compose their own.
func saveParts(cfgDir string, i input, space string, parts []string) bool {
	file := partsFile(cfgDir, i.SessionID)
	if file == "" {
		return false
	}
	data, err := json.Marshal(storedParts{Session: i.SessionID, Space: space, Source: i.Source, Written: Now(), Parts: parts})
	if err != nil || os.MkdirAll(filepath.Dir(file), 0o700) != nil {
		return false
	}
	return paths.WriteAtomicQuick(file, data, 0o600) == nil
}

// waitParts waits, at most wait, for the parts part 1 builds for this start, and
// returns part n. ok is false when they did not come in time, or are of another start
// (another session, space or source, or written before this one began).
func waitParts(cfgDir string, i input, space string, n int, started time.Time, wait time.Duration) (string, bool) {
	file := partsFile(cfgDir, i.SessionID)
	if file == "" {
		return "", false
	}
	deadline := time.Now().Add(wait)
	pause := time.Millisecond
	for {
		if data, err := os.ReadFile(file); err == nil {
			var p storedParts
			if json.Unmarshal(data, &p) == nil && p.Session == i.SessionID && p.Space == space &&
				p.Source == i.Source && !p.Written.Before(started.Add(-partsSlack)) && n >= 1 && n <= len(p.Parts) {
				return p.Parts[n-1], true
			}
		}
		left := time.Until(deadline)
		if left <= 0 {
			return "", false
		}
		time.Sleep(min(pause, left))
		pause = min(2*pause, 20*time.Millisecond)
	}
}

// removeParts deletes the parts of a session that ends, and those of sessions that
// ended without saying so, a day on.
func removeParts(cfgDir, session string, now time.Time) {
	if file := partsFile(cfgDir, session); file != "" {
		_ = os.Remove(file)
	}
	dir := filepath.Join(cfgDir, filepath.FromSlash(PartsDir))
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if fi, err := e.Info(); err == nil && now.Sub(fi.ModTime()) > partsKeep {
			_ = os.Remove(filepath.Join(dir, e.Name()))
		}
	}
}
