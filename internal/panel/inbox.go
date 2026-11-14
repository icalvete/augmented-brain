package panel

import (
	"errors"
	"net/http"

	"github.com/icalvete/augmented-brain/internal/brain"
)

func (s *Server) inbox(v *view, _ *http.Request) (string, page, error) {
	entries, err := v.b.Inbox()
	if err != nil {
		return "", page{}, err
	}
	p := v.frame("inbox", v.t("inbox"))
	p.D = struct{ Entries []brain.InboxEntry }{entries}
	return "inbox", p, nil
}

func (s *Server) capture(v *view, r *http.Request) (string, error) {
	if err := required(v, r, "text"); err != nil {
		return "", err
	}
	if d := field(r, "remind_on"); d != "" {
		if err := brain.CheckDate(d); err != nil {
			return "", onField("remind_on", err)
		}
	}
	return "/inbox", v.b.Capture(field(r, "text"), field(r, "remind_on"))
}

// inboxDone takes one capture out of the inbox: the one with this date and text (as
// sent, spaces included), even when another says the same.
func (s *Server) inboxDone(v *view, r *http.Request) (string, error) {
	found, err := v.b.InboxDoneEntry(field(r, "date"), r.PostForm.Get("text"))
	if err != nil {
		return "", err
	}
	if !found {
		return "", errors.New(v.t("err.inbox"))
	}
	return "/inbox", nil
}
