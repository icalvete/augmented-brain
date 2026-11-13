// Package state keeps what this machine knows about the sessions of each space: which
// sessions started and ended and, for each, which task is active, when the last note
// was written to it and when the assistant was last reminded to write. It lives in the
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
	// Key names the Claude process the session runs in (package fresh: pid-<n>), the one
	// thing its MCP server knows of it. Empty when Claude Code did not say.
	Key string `json:"key,omitempty"`
	// Task is the session's active task: the last one it opened, read or written, until
	// it is parked or closed. Empty when none is. Each session has its own, so two
	// sessions of a space working on different tasks never get each other's reminders.
	Task string `json:"task,omitempty"`
	// Since is when the active task was last written (or became active).
	Since time.Time `json:"since,omitzero"`
	// Reminded is when the assistant was last reminded to write in the journal.
	Reminded time.Time `json:"reminded,omitzero"`
	Closed   bool      `json:"closed,omitempty"`
	Reported bool      `json:"reported,omitempty"` // a later session already offered to recover it
}

// Space is the state of one space on this machine. (Earlier versions kept a single
// active task for the whole space, in fields of their own: they are no longer read, and
// go at the next write.)
type Space struct {
	Sessions []Session `json:"sessions,omitempty"`
	// Outside is when a session of another space last wrote in this one (a task's
	// journal, its next step) and no backup of it has started since: the next session
	// end, of any space, starts one (TakeOutside). Zero when there is nothing waiting.
	Outside time.Time `json:"outside,omitzero"`
}

// Who names one session: by its id, which the hooks get, or else by the key of its
// Claude process, which is all its MCP server knows. Both empty name none.
type Who struct{ ID, Key string }

// Session finds the session w names: by id; by key, the newest one not ended (after
// /clear the process goes on under a new session id, and the old one is ended). Nil
// when there is none.
func (s *Space) Session(w Who) *Session {
	if w.ID != "" {
		return s.find(w.ID)
	}
	if w.Key == "" {
		return nil
	}
	for i := len(s.Sessions) - 1; i >= 0; i-- {
		if se := &s.Sessions[i]; se.Key == w.Key && !se.Closed {
			return se
		}
	}
	return nil
}

// Active returns the session w names, with its active task; ok is false when there is
// no such session.
func Active(cfgDir, space string, w Who) (Session, bool) {
	s := Read(cfgDir, space)
	if se := s.Session(w); se != nil {
		return *se, true
	}
	return Session{}, false
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
	return paths.WriteAtomicQuick(file(cfgDir, space), data, 0o600)
}

// Touched records that the session w opened, read or wrote a task: it becomes that
// session's active one. A write moves Since; a read only does when the task was not
// the active one already, so reading a task never puts off the reminder to write in it.
// Nothing happens when w names no session this machine knows (the panel, a session
// outside the space's work folders).
func Touched(cfgDir, space string, w Who, task string, now time.Time, write bool) error {
	if w == (Who{}) {
		return nil
	}
	return Update(cfgDir, space, now, func(s *Space) {
		se := s.Session(w)
		if se == nil {
			return
		}
		if write || se.Task != task || se.Since.IsZero() {
			se.Since = now
		}
		se.Task = task
	})
}

// Released records that a task was parked or closed: no session has it active any more,
// whichever parked or closed it.
func Released(cfgDir, space, task string, now time.Time) error {
	return Update(cfgDir, space, now, func(s *Space) {
		for i := range s.Sessions {
			if se := &s.Sessions[i]; se.Task == task {
				se.Task, se.Since = "", now
			}
		}
	})
}

// Reminded records that the session w was just reminded to write in its task's journal.
func Reminded(cfgDir, space string, w Who, now time.Time) error {
	return Update(cfgDir, space, now, func(s *Space) {
		if se := s.Session(w); se != nil {
			se.Reminded = now
		}
	})
}

// Started records a session starting (or going on, after a resume or a compaction),
// with the key of the Claude process it runs in: a resumed session may run in another.
// A new session starts with no active task.
func Started(cfgDir, space, id, transcript, key string, now time.Time) error {
	if id == "" {
		return nil
	}
	return Update(cfgDir, space, now, func(s *Space) {
		if se := s.find(id); se != nil {
			se.Closed = false
			if transcript != "" {
				se.Transcript = transcript
			}
			if key != "" {
				se.Key = key
			}
			return
		}
		s.Sessions = append(s.Sessions, Session{ID: id, Transcript: transcript, Started: now, Key: key})
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

// MarkOutside records that a session of another space wrote in this space's notes, so
// the next session end backs it up instead of waiting for a session of its own.
func MarkOutside(cfgDir, space string, now time.Time) error {
	return Update(cfgDir, space, now, func(s *Space) { s.Outside = now })
}

// TakeOutside says whether the space was written from another space since its last
// backup, and clears the mark: whoever gets true starts the backup.
func TakeOutside(cfgDir, space string, now time.Time) bool {
	if Read(cfgDir, space).Outside.IsZero() {
		return false // nothing to do: no lock, no write (SessionEnd has 1.5 seconds)
	}
	var was bool
	_ = Update(cfgDir, space, now, func(s *Space) {
		was = !s.Outside.IsZero()
		s.Outside = time.Time{}
	})
	return was
}
