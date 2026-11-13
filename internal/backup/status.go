package backup

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/icalvete/augmented-brain/internal/paths"
)

// statusFile, in the configuration folder, keeps how the last backup of each space went,
// so the next session can tell the user when an automatic one failed.
const statusFile = ".backup-status.json"

// Status is how the last backup of a space went.
type Status struct {
	Time  time.Time `json:"time"`
	OK    bool      `json:"ok"`
	Error string    `json:"error,omitempty"`
	// Warning is what a backup that went through still wants the user to hear (files
	// it left out).
	Warning string `json:"warning,omitempty"`
	// Started is when a backup that has not ended yet began; a backup that ends clears
	// it. One still there long after was cut short: the computer went off, the process
	// was killed.
	Started time.Time `json:"started,omitzero"`
}

// StuckAfter is how long after it began a backup that never ended counts as cut short:
// one waits LockWait at most for another, and each of its git calls ends within
// GitTimeout.
var StuckAfter = 30 * time.Minute

// StaleAfter is how old the last good backup may be, with changes in the notes it does
// not hold, before the user hears of it.
var StaleAfter = 3 * 24 * time.Hour

// RecordStarted notes that a backup of space begins now, keeping how the last one went.
// Like RecordStatusWarning, it never fails.
func RecordStarted(cfgDir, space string, now time.Time) {
	all := readStatus(cfgDir)
	st := all[space]
	st.Started = now
	all[space] = st
	writeStatus(cfgDir, all)
}

// CutShort returns when the last backup of space began, if it began over StuckAfter ago
// and never ended.
func CutShort(cfgDir, space string, now time.Time) (time.Time, bool) {
	st := readStatus(cfgDir)[space]
	return st.Started, !st.Started.IsZero() && now.Sub(st.Started) > StuckAfter
}

// Behind reports whether the last backup that went through is over StaleAfter old and
// the notes at mem hold changes it does not: not committed, or committed and not pushed.
// It looks only at the repository on this computer, never at the network. Git runs
// only when the backup is old, so a session start pays nothing most days.
func (st Status) Behind(mem string, now time.Time) bool {
	if !st.OK || now.Sub(st.Time) <= StaleAfter {
		return false
	}
	if out, err := git(mem, "status", "--porcelain"); err != nil || strings.TrimSpace(out) != "" {
		return err == nil
	}
	out, err := git(mem, "rev-list", "--count", "origin/main..HEAD")
	return err == nil && strings.TrimSpace(out) != "0"
}

// RecordStatusWarning remembers how the backup of space went, with the warning of one
// that went through (empty when there is none). It never fails: a status that cannot be
// written only means no warning next time.
func RecordStatusWarning(cfgDir, space string, err error, warning string, now time.Time) {
	all := readStatus(cfgDir)
	st := Status{Time: now, OK: err == nil, Warning: warning}
	if err != nil {
		st.Error = err.Error()
	}
	all[space] = st
	writeStatus(cfgDir, all)
}

func writeStatus(cfgDir string, all map[string]Status) {
	if data, e := json.MarshalIndent(all, "", "  "); e == nil {
		_ = paths.WriteAtomic(filepath.Join(cfgDir, statusFile), data, 0o600)
	}
}

// LastStatus returns how the last backup of space went, if one ended (a first one that
// only began is not one).
func LastStatus(cfgDir, space string) (Status, bool) {
	st, ok := readStatus(cfgDir)[space]
	return st, ok && !st.Time.IsZero()
}

func readStatus(cfgDir string) map[string]Status {
	all := map[string]Status{}
	if data, err := os.ReadFile(filepath.Join(cfgDir, statusFile)); err == nil {
		_ = json.Unmarshal(data, &all)
	}
	return all
}
