package brain

import (
	"errors"
	"slices"
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

// SavePerson creates a person, or updates them when one with the same name exists (see
// findPerson). The summary carries the role and the other names, so the index alone
// resolves a name.
func (b *Brain) SavePerson(in PersonInput) (*memory.Note, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, errors.New("a person needs a name")
	}
	old, byAlias := b.findPerson(name)
	title, extra := name, in.Aliases
	if byAlias {
		// Found by another of their names: keep the name they have, and this one as an alias.
		title = old.Header.Title
		extra = append(append(aliasesOf(old.Header.Summary), name), in.Aliases...)
	}
	summary := strings.TrimSpace(in.Role)
	var aliases []string
	for _, a := range extra {
		if a = strings.TrimSpace(a); a != "" && !strings.EqualFold(a, title) && !slices.ContainsFunc(aliases, func(x string) bool { return strings.EqualFold(x, a) }) {
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
	body := strings.TrimSpace(in.Notes)
	if body != "" {
		body += "\n"
	}
	if old != nil {
		cp := *old // the note in the index stays as it was until saved
		cp.Header.Title = title
		cp.Header.Summary = summary
		if body != "" {
			cp.Body = body
		}
		return &cp, b.save(&cp)
	}
	id := b.newID("person", Slug(name))
	n := &memory.Note{
		Path:   memory.PathFor("person", id),
		Header: memory.Header{ID: id, Type: "person", Title: name, Summary: summary, Created: b.today()},
		Body:   body,
	}
	return n, b.save(n)
}

// findPerson returns the person called name: the one with that name (in any case), or
// else the one whose name makes the same id (José and Jose), or else the only one who
// has it among their other names. Names with no Latin letters (Мария, 王伟) make no id
// of their own, so they are never matched by id.
// byAlias says the match was by one of their other names.
func (b *Brain) findPerson(name string) (n *memory.Note, byAlias bool) {
	people := b.Mem.OfType("person")
	for _, p := range people {
		if strings.EqualFold(p.Header.Title, name) {
			return p, false
		}
	}
	if slug := Slug(name); slug != "untitled" {
		for _, p := range people {
			if Slug(p.Header.Title) == slug {
				return p, false
			}
		}
	}
	var found *memory.Note
	for _, p := range people {
		for _, a := range aliasesOf(p.Header.Summary) {
			if strings.EqualFold(a, name) {
				if found != nil && found != p {
					return nil, false // two people go by that name: make a new one, do not guess
				}
				found = p
			}
		}
	}
	return found, found != nil
}

// aliasesOf reads the other names back from a person's summary ("… · also: A, B").
func aliasesOf(summary string) []string {
	for _, also := range alsoWord {
		if i := strings.LastIndex(summary, also+": "); i >= 0 {
			var out []string
			for _, a := range strings.Split(summary[i+len(also)+2:], ",") {
				if a = strings.TrimSpace(a); a != "" {
					out = append(out, a)
				}
			}
			return out
		}
	}
	return nil
}
