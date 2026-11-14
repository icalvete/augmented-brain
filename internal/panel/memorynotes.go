package panel

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strconv"

	"net/http"

	"github.com/icalvete/augmented-brain/internal/brain"
	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/memory"
)

// Rules and knowledge are written here through the same brain functions as rule_new,
// knowledge_new and note_retire: a change keeps the old version in archive/, and a
// retired note moves there. A rule reaches every session, so when the user chose to be
// asked (memory_trust = "ask"), the form asks them to say yes first.

// askTrust reports whether memory_trust is "ask" (or a value not known, which counts as
// ask): a change of a rule then needs the form's "confirm".
func (v *view) askTrust() bool {
	ss, err := config.LoadSession(v.c.Dir)
	return err != nil || ss.Trust() == config.TrustAsk
}

// confirmed refuses a change of a rule the user has not said yes to, when they chose to
// be asked.
func confirmed(v *view, r *http.Request, typ string) error {
	if typ != "rule" || !v.askTrust() || r.PostForm.Get("confirm") == "1" {
		return nil
	}
	return onField("confirm", errors.New(v.t("rule.confirm.missing")))
}

// target is the brain a form writes to: the space's, or the shared one (field "shared").
func target(v *view, r *http.Request) (*brain.Brain, bool, error) {
	if r.PostForm.Get("shared") != "1" {
		return v.b, false, nil
	}
	b, err := v.b.Shared()
	return b, true, err
}

// stampIn is the time a note of m was last written, in nanoseconds; "" when there is
// no such note.
func stampIn(m *memory.Memory, id string) string {
	n, ok := m.Notes[id]
	if !ok {
		return ""
	}
	info, err := os.Stat(filepath.Join(m.Dir, filepath.FromSlash(n.Path)))
	if err != nil {
		return ""
	}
	return strconv.FormatInt(info.ModTime().UnixNano(), 10)
}

// notePage is the address of a note's page, shared or not.
func notePage(id string) string { return "/note?id=" + url.QueryEscape(id) }

func (s *Server) ruleNew(v *view, r *http.Request) (string, error) {
	if err := required(v, r, "title", "body"); err != nil {
		return "", err
	}
	if err := confirmed(v, r, "rule"); err != nil {
		return "", err
	}
	b, _, err := target(v, r)
	if err != nil {
		return "", err
	}
	n, err := b.NewRule(field(r, "title"), field(r, "summary"), r.PostForm.Get("body"))
	if err != nil {
		return "", err
	}
	return notePage(n.Header.ID), nil
}

func (s *Server) knowledgeNew(v *view, r *http.Request) (string, error) {
	if err := required(v, r, "title"); err != nil {
		return "", err
	}
	b, shared, err := target(v, r)
	if err != nil {
		return "", err
	}
	area := field(r, "area")
	if shared {
		area = "" // shared knowledge belongs to no space's area
	}
	n, err := b.NewKnowledge(field(r, "kind"), field(r, "title"), field(r, "summary"), area, r.PostForm.Get("body"))
	if err != nil {
		return "", err
	}
	return notePage(n.Header.ID), nil
}

// noteUpdate changes a rule, reference or lesson: title, summary, body and, for
// knowledge of the space, its area. "stamp" is when the note was written as the page
// showed it: a note changed since is not overwritten.
func (s *Server) noteUpdate(v *view, r *http.Request) (string, error) {
	id := field(r, "id")
	b, _, err := target(v, r)
	if err != nil {
		return "", err
	}
	n, ok := b.Mem.Notes[id]
	if !ok {
		return "", errors.New(v.t("err.note", id))
	}
	if st := field(r, "stamp"); st != "" && st != stampIn(b.Mem, id) {
		return "", errors.New(v.t("note.stale"))
	}
	if err := required(v, r, "title"); err != nil {
		return "", err
	}
	if n.Header.Type == "rule" && field(r, "body") == "" {
		return "", onField("body", errors.New(v.t("err.required", v.t("f.body"))))
	}
	if err := confirmed(v, r, n.Header.Type); err != nil {
		return "", err
	}
	area := ""
	if n.Header.Type != "rule" && b == v.b {
		area = field(r, "area")
	}
	// An emptied summary stays as it was: UpdateNote keeps what it is not given.
	if _, err := b.UpdateNote(id, n.Header.Type, field(r, "title"), field(r, "summary"), area, r.PostForm.Get("body")); err != nil {
		return "", err
	}
	return notePage(id), nil
}

// noteRetire takes a rule, reference or lesson out of the memory, to archive/, with
// why ("reason") and, optionally, the id of the note that replaces it.
func (s *Server) noteRetire(v *view, r *http.Request) (string, error) {
	id := field(r, "id")
	b, _, err := target(v, r)
	if err != nil {
		return "", err
	}
	n, ok := b.Mem.Notes[id]
	if !ok {
		return "", errors.New(v.t("err.note", id))
	}
	if err := required(v, r, "reason"); err != nil {
		return "", err
	}
	if err := confirmed(v, r, n.Header.Type); err != nil {
		return "", err
	}
	if _, err := b.RetireNote(id, field(r, "reason"), field(r, "replaced_by")); err != nil {
		return "", err
	}
	if n.Header.Type == "rule" {
		return "/rules", nil
	}
	return "/knowledge", nil
}
