// Package state keeps what this machine knows about the sessions of each space: which
// task is active, when the last note was written to it, when the assistant was last
// reminded to write, and which sessions started and ended. It lives in the
// configuration folder, never in the notes: it is about this machine, it is not backed
// up, and it holds no text of the conversation.
//
// Every call is quick (a small JSON file and a lock held for a moment), so the hooks can
// use it within their budget: SessionEnd has only 1.5 seconds.
package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/icalvete/augmented-brain/internal/filelock"
	"github.com/icalvete/augmented-brain/internal/paths"
)

// Dir is the folder of the state files, inside the configuration folder.
const Dir = "state"

// keep is how long a session is remembered: Claude Code deletes transcripts after 30
// days by default, so there is nothing to recover after that.
const keep = 30 * 24 * time.Hour

// Session is one Claude Code session that worked in the space.
type Session struct {
	ID         string    `json:"id"`
	Transcript string    `json:"transcript,omitempty"` // Claude Code's local transcript (JSONL)
	Started    time.Time `json:"started"`
	Task       string    `json:"task,omitempty"` // the active task when it started
	Closed     bool      `json:"closed,omitempty"`
	Reported   bool      `json:"reported,omitempty"` // a later session already offered to recover it
}

// Space is the state of one space on this machine.
type Space struct {
	// Task is the active task: the last one opened, read or written, until it is parked
	// or closed. Empty when none is.
	Task string `json:"task,omitempty"`
	// Since is when the active task was last written (or became active).
	Since time.Time `json:"since,omitzero"`
	// Reminded is when the assistant was last reminded to write in the journal.
	Reminded time.Time `json:"reminded,omitzero"`
	Sessions []Session `json:"sessions,omitempty"`
}

func file(cfgDir, space string) string { return filepath.Join(cfgDir, Dir, space+".json") }

// Read returns the state of a space; a missing or unreadable file is an empty state.
func Read(cfgDir, space string) Space {
	var s Space
	if data, err := os.ReadFile(file(cfgDir, space)); err == nil {
		_ = json.Unmarshal(data, &s)
	}
	return s
}

// Update changes the state of a space under its lock, and writes it in one step.
func Update(cfgDir, space string, now time.Time, change func(*Space)) error {
	unlock, err := filelock.Lock(filepath.Join(cfgDir, Dir, ".lock"))
	if err != nil {
		return err
	}
	defer unlock()
	s := Read(cfgDir, space)
	change(&s)
	var kept []Session
	for _, se := range s.Sessions {
		if now.Sub(se.Started) < keep {
			kept = append(kept, se)
		}
	}
	sort.SliceStable(kept, func(i, j int) bool { return kept[i].Started.Before(kept[j].Started) })
	s.Sessions = kept
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return paths.WriteAtomic(file(cfgDir, space), data, 0o600)
}

// Touched records that a task was opened, read or written: it becomes the active one.
// A write moves Since; a read only does when the task was not the active one already,
// so reading a task never puts off the reminder to write in it.
func Touched(cfgDir, space, task string, now time.Time, write bool) error {
	return Update(cfgDir, space, now, func(s *Space) {
		if write || s.Task != task || s.Since.IsZero() {
			s.Since = now
		}
		s.Task = task
	})
}

// Released records that a task was parked or closed: if it was the active one, no task
// is active any more.
func Released(cfgDir, space, task string, now time.Time) error {
	return Update(cfgDir, space, now, func(s *Space) {
		if s.Task == task {
			s.Task, s.Since = "", now
		}
	})
}

// Started records a session starting (or going on, after a resume or a compaction).
func Started(cfgDir, space, id, transcript string, now time.Time) error {
	if id == "" {
		return nil
	}
	return Update(cfgDir, space, now, func(s *Space) {
		if se := s.find(id); se != nil {
			se.Closed = false
			if transcript != "" {
				se.Transcript = transcript
			}
			return
		}
		s.Sessions = append(s.Sessions, Session{ID: id, Transcript: transcript, Started: now, Task: s.Task})
	})
}

// Ended records a session ending normally.
func Ended(cfgDir, space, id string, now time.Time) error {
	return changeSession(cfgDir, space, id, now, func(se *Session) { se.Closed = true })
}

// find is the one way to look a session up by its id; nil when there is none.
func (s *Space) find(id string) *Session {
	for i := range s.Sessions {
		if s.Sessions[i].ID == id {
			return &s.Sessions[i]
		}
	}
	return nil
}

// changeSession changes one session under the lock; nothing happens when it is not
// known (or id is empty, as when Claude Code gives none).
func changeSession(cfgDir, space, id string, now time.Time, change func(*Session)) error {
	if id == "" {
		return nil
	}
	return Update(cfgDir, space, now, func(s *Space) {
		if se := s.find(id); se != nil {
			change(se)
		}
	})
}
