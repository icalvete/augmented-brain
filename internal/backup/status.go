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
}

// RecordStatus remembers how the backup of space went. It never fails: a status that
// cannot be written only means no warning next time.
func RecordStatus(cfgDir, space string, err error, now time.Time) {
	all := readStatus(cfgDir)
	st := Status{Time: now, OK: err == nil}
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
