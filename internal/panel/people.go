package panel

import (
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/icalvete/augmented-brain/internal/brain"
	"github.com/icalvete/augmented-brain/internal/memory"
)

// personData is what a person's page shows besides the note: the form to change them,
// and the tasks that wait on them or name them.
type personData struct {
	Role, Aliases string
	Waiting       []taskRow // live tasks whose "waits on" names them
	Named         []taskRow // other tasks that link to them
}

// names are every name a person goes by: their name, their other names and their id.
func names(n *memory.Note) []string {
	_, aliases := brain.PersonParts(n)
	return append([]string{n.Header.Title, n.Header.ID}, aliases...)
}

// namer finds any of a person's names in a text, as a whole word, in any case.
func namer(who []string) *regexp.Regexp {
	var alt []string
	for _, w := range who {
		if w != "" {
			alt = append(alt, regexp.QuoteMeta(w))
		}
	}
	return regexp.MustCompile(`(?i)(^|[^\p{L}\p{N}_-])(` + strings.Join(alt, "|") + `)($|[^\p{L}\p{N}_-])`)
}

func (v *view) person(n *memory.Note) personData {
	role, aliases := brain.PersonParts(n)
	d := personData{Role: role, Aliases: strings.Join(aliases, ", ")}
	who := namer(names(n))
	for _, t := range v.b.Mem.OfType("task") {
		h := t.Header
		row := taskRow{ID: h.ID, Title: h.Title, Status: h.Status, StatusLabel: v.status(h.Status), Next: h.Next}
		switch {
		case h.Status != "closed" && who.MatchString(h.WaitingOn):
			d.Waiting = append(d.Waiting, row)
		case strings.Contains(t.Body, "[["+n.Header.ID+"]]") || strings.Contains(t.Body, "[["+n.Header.ID+"|") || containsID(h.Related, n.Header.ID):
			d.Named = append(d.Named, row)
		}
	}
	return d
}

func containsID(ids []string, id string) bool {
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}

// personSave creates a person or changes one. Changing keeps their name (the field is
// fixed on their page): a new name would be a new person.
func (s *Server) personSave(v *view, r *http.Request) (string, error) {
	if err := required(v, r, "name"); err != nil {
		return "", err
	}
	var aliases []string
	for _, a := range strings.Split(field(r, "aliases"), ",") {
		if a = strings.TrimSpace(a); a != "" {
			aliases = append(aliases, a)
		}
	}
	n, err := v.b.SavePerson(brain.PersonInput{Name: field(r, "name"), Aliases: aliases,
		Role: field(r, "role"), Notes: field(r, "notes")})
	if err != nil {
		return "", err
	}
	return "/note?id=" + url.QueryEscape(n.Header.ID), nil
}

// personLink is the id of the person a "waits on" names exactly, or "".
func (v *view) personLink(waiting string) string {
	if waiting == "" || waiting == "me" || waiting == "nobody" {
		return ""
	}
	if p := v.b.PersonNamed(waiting); p != nil {
		return p.Header.ID
	}
	return ""
}
