package remote

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// LingerAge is how long a background session no task opened must have gone without a
// change of status before doctor names it.
const LingerAge = 12 * time.Hour

// Lingering are the background sessions that keep their process while nobody uses
// them, for doctor to name with `claude stop <id>`: those archived or ended (done,
// stopped: Ended), and those no task's `session start` opened, idle or blocked, whose
// status has not changed for LingerAge. Archiving one in Claude Desktop tells Claude
// nothing (its state stays blocked and its process runs, measured on 12-nov), so age is
// all there is to go by; one whose age cannot be read is left out.
func (s System) Lingering(cfgDir string, now time.Time) ([]Session, error) {
	claude, err := s.FindClaude()
	if err != nil {
		return nil, err
	}
	list, err := s.Sessions(claude)
	if err != nil {
		return nil, err
	}
	MarkUnlaunched(cfgDir, list)
	var out []Session
	for _, se := range list {
		switch {
		case se.Kind != KindBackground || se.ID == "":
		case se.Ended():
			out = append(out, se)
		case se.Unlaunched && (se.Status == "idle" || se.State == "blocked"):
			if since, ok := s.statusSince(se.PID); ok && now.Sub(since) >= LingerAge {
				out = append(out, se)
			}
		}
	}
	return out, nil
}

// statusSince is when the session of process pid last changed status, from the file
// Claude keeps of each running session (sessions/<pid>.json in its folder).
func (s System) statusSince(pid int) (time.Time, bool) {
	if s.ClaudeDir == "" {
		return time.Time{}, false
	}
	data, err := os.ReadFile(filepath.Join(s.ClaudeDir, "sessions", strconv.Itoa(pid)+".json"))
	if err != nil {
		return time.Time{}, false
	}
	var f struct {
		StatusUpdatedAt int64 `json:"statusUpdatedAt"`
		UpdatedAt       int64 `json:"updatedAt"`
	}
	if json.Unmarshal(data, &f) != nil {
		return time.Time{}, false
	}
	ms := f.StatusUpdatedAt
	if ms == 0 {
		ms = f.UpdatedAt
	}
	if ms == 0 {
		return time.Time{}, false
	}
	return time.UnixMilli(ms), true
}
