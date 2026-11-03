package brain

import (
	"errors"
	"slices"
	"strings"

	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/memory"
)

// PersonInput is someone the user mentions: who they are, and the names the user uses
// for them, so "Marcos" leads to the right person.
type PersonInput struct {
	Name    string   // full name, or how the user knows them
	Aliases []string // other names the user uses: "Marcos", "Marquitos"
	Role    string   // who they are to the user: "my manager", "client, runs the shop"
	Notes   string   // anything else worth remembering
	// ReplaceNotes makes Notes the person's notes even when empty (the panel's form,
	// which shows them whole); otherwise empty Notes keep what they say.
	ReplaceNotes bool
}

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
		also := i18n.T(b.Language, "person.also")
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
		if body != "" || in.ReplaceNotes {
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

// PersonParts reads a person's role and other names back from their summary, as
// SavePerson wrote them: what an edit form starts from.
func PersonParts(n *memory.Note) (role string, aliases []string) {
	role = n.Header.Summary
	for _, also := range i18n.All("person.also") {
		if i := strings.LastIndex(role, also+": "); i >= 0 {
			role = strings.TrimSuffix(strings.TrimSpace(role[:i]), "·")
			break
		}
	}
	return strings.TrimSpace(role), aliasesOf(n.Header.Summary)
}

// PersonNamed returns the person a name points at (their name or another of theirs,
// in any case), when only one does.
func (b *Brain) PersonNamed(name string) *memory.Note {
	n, _ := b.findPerson(strings.TrimSpace(name))
	return n
}

// aliasesOf reads the other names back from a person's summary ("… · also: A, B").
func aliasesOf(summary string) []string {
	for _, also := range i18n.All("person.also") {
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
