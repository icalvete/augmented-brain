package backup

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
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
	if data, e := json.MarshalIndent(all, "", "  "); e == nil {
		_ = os.WriteFile(filepath.Join(cfgDir, statusFile), data, 0o600)
	}
}

// LastStatus returns how the last backup of space went, if there was one.
func LastStatus(cfgDir, space string) (Status, bool) {
	st, ok := readStatus(cfgDir)[space]
	return st, ok
}

func readStatus(cfgDir string) map[string]Status {
	all := map[string]Status{}
	if data, err := os.ReadFile(filepath.Join(cfgDir, statusFile)); err == nil {
		_ = json.Unmarshal(data, &all)
	}
	return all
}
