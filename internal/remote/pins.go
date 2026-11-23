package remote

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/icalvete/augmented-brain/internal/paths"
)

// A task's session is pinned in Claude's list of background sessions, so Claude does
// not retire it. Claude retires a background session left idle for 8 hours ("bg retire
// <id>: settled, idle 8h" in its daemon.log), and a retired one cannot come back as it
// was: `claude rm` leaves its entry in claude.ai "disconnected", and `claude --bg
// --resume` opens a new entry beside it, with no message written to it meanwhile
// (measured on 22 and 23-nov-2026, 2.1.330). One pinned is never retired: Claude's
// retireIfSettled answers "pinned" for an id in jobs/pins.json, the file the pin
// toggle of `claude agents` writes. It has no command of its own and no documented
// format: a JSON array of short session ids, behind a proper-lockfile lock, the folder
// pins.json.lock, which counts as stale after 10 seconds. So pinning is help, never a
// condition: a file that is not that array is left as it is, and nothing fails when
// the pin cannot be written. Unpinning takes out only the ids of the task's sessions:
// `claude rm` leaves them there (23-nov), and the user's own pins stay.

const (
	pinsFile     = "pins.json"
	pinLockStale = 10 * time.Second // proper-lockfile's default
	pinLockWait  = 5 * time.Second
)

// errPinsFormat: pins.json holds something other than an array of ids.
var errPinsFormat = errors.New("pins.json is not a list of session ids")

func (s System) pinsPath() string {
	if s.ClaudeDir == "" {
		return ""
	}
	return filepath.Join(s.ClaudeDir, "jobs", pinsFile)
}

// pin adds a session to Claude's pinned ones.
func (s System) pin(id string) error {
	return s.editPins(func(ids []string) []string {
		if slices.Contains(ids, id) {
			return nil
		}
		return append(ids, id)
	})
}

// unpin takes a session out of Claude's pinned ones.
func (s System) unpin(id string) error {
	return s.editPins(func(ids []string) []string {
		if !slices.Contains(ids, id) {
			return nil
		}
		return slices.DeleteFunc(ids, func(x string) bool { return x == id })
	})
}

// editPins changes pins.json under Claude's own lock; change returns nil to leave it
// as it is. Without Claude's jobs folder (a claude with no background sessions yet)
// there is nothing to pin in.
func (s System) editPins(change func([]string) []string) error {
	path := s.pinsPath()
	if path == "" {
		return nil
	}
	if fi, err := os.Stat(filepath.Dir(path)); err != nil || !fi.IsDir() {
		return nil
	}
	unlock, err := s.lockPins(path + ".lock")
	if err != nil {
		return err
	}
	defer unlock()
	ids := []string{}
	data, err := os.ReadFile(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
	case err != nil:
		return err
	default:
		if json.Unmarshal(data, &ids) != nil || ids == nil {
			return errPinsFormat
		}
	}
	changed := change(ids)
	if changed == nil {
		return nil
	}
	out, err := json.Marshal(changed)
	if err != nil {
		return err
	}
	mode := os.FileMode(0o644)
	if fi, err := os.Stat(path); err == nil {
		mode = fi.Mode().Perm()
	}
	return paths.WriteAtomic(path, out, mode)
}

// pinSpacing: Claude holds the lock for a read and a write, a moment.
func pinSpacing(time.Duration) time.Duration { return 100 * time.Millisecond }

// lockPins takes the lock proper-lockfile takes: the folder lock, made with mkdir. One
// left behind longer than pinLockStale belongs to no one and is taken over.
func (s System) lockPins(lock string) (func(), error) {
	for p := s.newPace(pinLockWait, pinSpacing); ; p.sleep(false) {
		err := os.Mkdir(lock, 0o755)
		if err == nil {
			return func() { _ = os.Remove(lock) }, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, err
		}
		if fi, err := os.Stat(lock); err == nil && time.Since(fi.ModTime()) > pinLockStale {
			_ = os.Remove(lock) // taken at the next look
		}
		if p.over() {
			return nil, errors.New("pins.json is locked")
		}
	}
}
