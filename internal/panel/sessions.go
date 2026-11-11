package panel

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/icalvete/augmented-brain/internal/brain"
	"github.com/icalvete/augmented-brain/internal/memory"
	"github.com/icalvete/augmented-brain/internal/state"
)

// previewMax is how much of a dead session's conversation its card shows. A recovery
// keeps more: brain.RecoverForArchive.
const previewMax = 3000

type sessionRow struct {
	ID, Short, Started, Written, Size, TaskID, TaskTitle string
	From, Said                                           string // dead sessions: what may be missing from the notes, and since when
}

const stamp = "2006-01-02 15:04"

// size says how big a file is, in words a person reads at a glance.
func size(n int64) string {
	switch {
	case n >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(n)/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	default:
		return fmt.Sprintf("%d KB", (n+1023)/1024)
	}
}

func (v *view) sessionRow(l state.Listed) sessionRow { return sessionRowOf(l, v.b.Mem) }

// sessionRowOf is the row of a session of the space whose notes are mem.
func sessionRowOf(l state.Listed, mem *memory.Memory) sessionRow {
	r := sessionRow{ID: l.ID, Short: l.ID, Started: l.Started.Local().Format(stamp), TaskID: l.Task}
	if len(r.Short) > 8 {
		r.Short = r.Short[:8]
	}
	if l.Size > 0 {
		r.Size = size(l.Size)
	}
	if !l.Modified.IsZero() {
		r.Written = l.Modified.Local().Format(stamp)
	}
	if n, ok := mem.Notes[l.Task]; ok {
		r.TaskTitle = n.Header.Title
	} else {
		r.TaskTitle = l.Task
	}
	return r
}

func (s *Server) sessions(v *view, _ *http.Request) (string, page, error) {
	var d struct {
		Live, Dead, Closed []sessionRow
		ClosedSize         string
	}
	now := time.Now()
	st := state.Read(v.c.Dir, v.b.Space.Name)
	var closed int64
	for _, l := range st.List(now) {
		r := v.sessionRow(l)
		switch l.Kind {
		case state.Live:
			d.Live = append(d.Live, r)
		case state.Dead:
			from := l.From(v.b.LastEntry(l.Task))
			r.From = from.Local().Format(stamp)
			r.Said, _ = state.Conversation(l.Transcript, from, previewMax)
			d.Dead = append(d.Dead, r)
		case state.Closed:
			if l.Size == 0 {
				continue // the transcript is gone: nothing to show or to free
			}
			closed += l.Size
			d.Closed = append(d.Closed, r)
		}
	}
	if closed > 0 {
		d.ClosedSize = size(closed)
	}
	p := v.frame("sessions", v.t("sessions"))
	p.D = d
	return "sessions", p, nil
}

// sessionRecover keeps what was said in a dead session after the last journal entry of
// its task: in archive/sessions/, with a line in the task's journal pointing at it.
// There is no assistant here to sum it up, so the conversation is kept as it was.
func (s *Server) sessionRecover(v *view, r *http.Request) (string, error) {
	now := time.Now()
	rec, err := v.b.Recover(field(r, "id"), brain.RecoverForArchive, now)
	if errors.Is(err, brain.ErrNoDeadSession) {
		return "", errors.New(v.t("err.session"))
	}
	if err != nil {
		return "", err
	}
	se := rec.Session
	if rec.Text != "" {
		started := se.Started.Local().Format(stamp)
		rel, err := v.b.SaveSession("recovered", se.ID, v.t("sessions.archive", started, rec.From.Local().Format(stamp)), rec.Text)
		if err != nil {
			return "", err
		}
		if _, ok := v.b.Mem.Notes[se.Task]; ok {
			if _, err := v.b.JournalText("recovered", se.Task, started, rel); err != nil {
				return "", err
			}
			v.b.Touched(se.Task, true)
		}
	}
	return sessionsBack(r), state.Dismissed(v.c.Dir, v.b.Space.Name, se.ID, now)
}

// sessionsBack is where a change to a session that did not end goes back to: Today,
// when it was made there, or the Sessions page.
func sessionsBack(r *http.Request) string {
	if field(r, "back") == "today" {
		return "/today"
	}
	return "/sessions"
}

// sessionReviewed takes a dead session off the list without keeping anything.
func (s *Server) sessionReviewed(v *view, r *http.Request) (string, error) {
	id, now := field(r, "id"), time.Now()
	for _, se := range state.Read(v.c.Dir, v.b.Space.Name).Unclosed("", now) {
		if se.ID == id {
			return sessionsBack(r), state.Dismissed(v.c.Dir, v.b.Space.Name, id, now)
		}
	}
	return "", errors.New(v.t("err.session"))
}

// sessionDelete deletes the transcript of a closed session, once the person has ticked
// that they understand it can no longer be resumed.
func (s *Server) sessionDelete(v *view, r *http.Request) (string, error) {
	if field(r, "confirm") != "yes" {
		return "", errors.New(v.t("err.confirm"))
	}
	if err := state.DeleteTranscript(v.c.Dir, v.b.Space.Name, field(r, "id"), time.Now()); err != nil {
		if errors.Is(err, state.ErrNotDeletable) {
			return "", errors.New(v.t("err.delete"))
		}
		return "", err
	}
	return "/sessions", nil
}
