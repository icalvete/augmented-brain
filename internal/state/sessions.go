package state

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Kinds of session, as the panel shows them.
const (
	Live   = "live"   // running now: not ended, transcript written in the last Busy minutes
	Dead   = "dead"   // never ended and no longer running: something may be missing from the notes
	Closed = "closed" // ended normally, recovered or marked as reviewed
)

// Listed is a session with what this machine can tell about its transcript.
type Listed struct {
	Session
	Kind     string
	Size     int64     // bytes of the transcript; 0 when it is gone
	Modified time.Time // when the transcript was last written
}

// List returns the sessions of the space, newest first, each with its kind. A session
// whose transcript is gone from this machine is left out unless it is closed (it shows
// that the transcript was deleted).
func (s Space) List(now time.Time) []Listed {
	var out []Listed
	for i := len(s.Sessions) - 1; i >= 0; i-- {
		se := s.Sessions[i]
		l := Listed{Session: se, Kind: Closed}
		if se.Transcript != "" {
			if info, err := os.Stat(se.Transcript); err == nil {
				l.Size, l.Modified = info.Size(), info.ModTime()
			} else if !se.Closed {
				continue
			}
		}
		if !se.Closed {
			if se.Transcript == "" {
				continue
			}
			l.Kind = Dead
			if now.Sub(l.Modified) < Busy {
				l.Kind = Live
			}
		}
		out = append(out, l)
	}
	return out
}

// Busy is how recently a transcript must have been written for its session to count as
// still running (in another window, or the one asking).
const Busy = 10 * time.Minute

// Unclosed returns the dead sessions of the space (see List) other than current,
// newest first: they started but never ended, are not running anywhere, and still have
// a transcript on this machine.
func (s Space) Unclosed(current string, now time.Time) []Session {
	var out []Session
	for _, l := range s.List(now) {
		if l.Kind == Dead && l.ID != current {
			out = append(out, l.Session)
		}
	}
	return out
}

// From is where what may be missing from the notes starts, in a session that did not
// end: the last write to its task while that session was still alive, or lastEntry
// (the date of the task's last journal entry), whichever is later; its start when
// there is neither.
func (s Space) From(se Session, lastEntry time.Time) time.Time {
	since := se.Started
	if s.Task == se.Task && s.Since.After(since) {
		if info, err := os.Stat(se.Transcript); err == nil && s.Since.Before(info.ModTime()) {
			since = s.Since
		}
	}
	if lastEntry.After(since) {
		since = lastEntry
	}
	return since
}

// Dismissed marks a session as dealt with: recovered or reviewed. It is then listed as
// closed, and no later session offers to recover it: to this machine it is as if it
// had ended normally.
func Dismissed(cfgDir, space, id string, now time.Time) error { return Ended(cfgDir, space, id, now) }

// Reported records that a later session offered to recover this one, so it is offered
// once only.
func Reported(cfgDir, space, id string, now time.Time) error {
	return changeSession(cfgDir, space, id, now, func(se *Session) { se.Reported = true })
}

// ErrNotDeletable is returned for a transcript that must not be deleted.
var ErrNotDeletable = errors.New("only the transcript of a closed session, not written in the last minutes, can be deleted")

// DeleteTranscript deletes Claude Code's transcript of a closed session, and the folder
// next to it with the same name, where Claude Code keeps that session's large tool
// output. That session can no longer be resumed with claude --resume. To be sure only
// a transcript goes, the file must be named after the session (<id>.jsonl), as Claude
// Code names them, and must not have been written in the last Busy minutes.
func DeleteTranscript(cfgDir, space, id string, now time.Time) error {
	st := Read(cfgDir, space)
	se := st.find(id)
	if se == nil || !se.Closed || se.Transcript == "" || id == "" ||
		filepath.Base(se.Transcript) != id+".jsonl" || strings.ContainsAny(id, `/\`) {
		return ErrNotDeletable
	}
	info, err := os.Lstat(se.Transcript)
	switch {
	case err == nil && (!info.Mode().IsRegular() || now.Sub(info.ModTime()) < Busy):
		return ErrNotDeletable
	case err == nil:
		if err := os.Remove(se.Transcript); err != nil {
			return err
		}
	case !errors.Is(err, os.ErrNotExist):
		return err
	}
	side := filepath.Join(filepath.Dir(se.Transcript), id)
	if info, err := os.Lstat(side); err == nil && info.IsDir() {
		if err := os.RemoveAll(side); err != nil {
			return err
		}
	}
	return changeSession(cfgDir, space, id, now, func(se *Session) { se.Transcript = "" })
}
