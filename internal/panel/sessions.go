package panel

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/icalvete/augmented-brain/internal/state"
)

// previewMax is how much of a dead session's conversation its card shows; recoverMax,
// how much a recovery keeps in the archive (the last part, when there is more).
const (
	previewMax = 3000
	recoverMax = 200000
)

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

func (v *view) sessionRow(l state.Listed) sessionRow {
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
	if n, ok := v.b.Mem.Notes[l.Task]; ok {
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
			from := st.From(l.Session, v.b.LastEntry(l.Task))
			r.From = from.Local().Format(stamp)
			r.Said, _ = state.Conversation(l.Transcript, from, previewMax)
			d.Dead = append(d.Dead, r)
		default:
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

// deadSession finds a session of the space that did not end and is not running.
func (v *view) deadSession(id string) (state.Space, state.Listed, error) {
	st := state.Read(v.c.Dir, v.b.Space.Name)
	for _, l := range st.List(time.Now()) {
		if l.ID == id && l.Kind == state.Dead {
			return st, l, nil
		}
	}
	return st, state.Listed{}, errors.New(v.t("err.session"))
}

// sessionRecover keeps what was said in a dead session after the last journal entry of
// its task: in archive/sessions/, with a line in the task's journal pointing at it.
// There is no assistant here to sum it up, so the conversation is kept as it was.
func (s *Server) sessionRecover(v *view, r *http.Request) (string, error) {
	st, l, err := v.deadSession(field(r, "id"))
	if err != nil {
		return "", err
	}
	from := st.From(l.Session, v.b.LastEntry(l.Task))
	text, err := state.Conversation(l.Transcript, from, recoverMax)
	if err != nil {
		return "", err
	}
	now := time.Now()
	if text != "" {
		started := l.Started.Local().Format(stamp)
		rel, err := v.b.SaveSession("recovered", l.ID, v.t("sessions.archive", started, from.Local().Format(stamp)), text)
		if err != nil {
			return "", err
		}
		if _, ok := v.b.Mem.Notes[l.Task]; ok {
			if _, err := v.b.JournalText("recovered", l.Task, started, rel); err != nil {
				return "", err
			}
			_ = state.Touched(v.c.Dir, v.b.Space.Name, l.Task, now, true)
		}
	}
	return "/sessions", state.Dismissed(v.c.Dir, v.b.Space.Name, l.ID, now)
}

// sessionReviewed takes a dead session off the list without keeping anything.
func (s *Server) sessionReviewed(v *view, r *http.Request) (string, error) {
	_, l, err := v.deadSession(field(r, "id"))
	if err != nil {
		return "", err
	}
	return "/sessions", state.Dismissed(v.c.Dir, v.b.Space.Name, l.ID, time.Now())
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
