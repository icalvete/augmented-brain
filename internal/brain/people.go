package brain

import (
	"errors"
	"strings"

	"github.com/icalvete/augmented-brain/internal/memory"
)

// PersonInput is someone the user mentions: who they are, and the names the user uses
// for them, so "Marcos" leads to the right person.
type PersonInput struct {
	Name    string   // full name, or how the user knows them
	Aliases []string // other names the user uses: "Marcos", "Marquitos"
	Role    string   // who they are to the user: "my manager", "client, runs the shop"
	Notes   string   // anything else worth remembering
}

var alsoWord = map[string]string{"en": "also", "es": "también"}

// SavePerson creates a person, or updates them when one with the same name exists. The
// summary carries the role and the other names, so the index alone resolves a name.
func (b *Brain) SavePerson(in PersonInput) (*memory.Note, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, errors.New("a person needs a name")
	}
	summary := strings.TrimSpace(in.Role)
	var aliases []string
	for _, a := range in.Aliases {
		if a = strings.TrimSpace(a); a != "" && !strings.EqualFold(a, name) {
			aliases = append(aliases, a)
		}
	}
	if len(aliases) > 0 {
		also := alsoWord[b.Language]
		if also == "" {
			also = alsoWord["en"]
		}
		if summary != "" {
			summary += " · "
		}
		summary += also + ": " + strings.Join(aliases, ", ")
	}
	id := "person-" + Slug(name)
	body := strings.TrimSpace(in.Notes)
	if body != "" {
		body += "\n"
	}
	if old, ok := b.Mem.Notes[id]; ok && old.Header.Type == "person" {
		old.Header.Title = name
		old.Header.Summary = summary
		if body != "" {
			old.Body = body
		}
		return old, b.save(old)
	}
	n := &memory.Note{
		Path:   "people/" + id + ".md",
		Header: memory.Header{ID: id, Type: "person", Title: name, Summary: summary, Created: b.today()},
		Body:   body,
	}
	return n, b.save(n)
}
