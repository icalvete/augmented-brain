package brain

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/memory"
)

// A person can live in several spaces. Each space keeps its own file for them
// (people/person-<id>.md, the same id everywhere) and that file holds only what is
// visible in that space: the role and other names in the summary, and the details in
// the body. The card of a person is the union of those files, joined by id (never by
// name: two people can share one). Nothing about this is a new format.

// ErrNoPerson says no space holds a person with that id.
var ErrNoPerson = errors.New("no such person in any space")

// cardNow gives today's date for the notes the card functions write; tests replace it.
var cardNow = time.Now

// CardSpace is one space's part of a person's card.
type CardSpace struct {
	Space   string
	Exists  bool // this space has a file for them
	Role    string
	Aliases []string
}

// CardDetail is one thing worth knowing about a person, and the spaces where it is
// written. A detail visible in two spaces is in both files.
type CardDetail struct {
	Text   string
	Spaces []string
}

// Card is a person across the spaces of the configuration that exist on this machine.
type Card struct {
	ID, Name string
	Spaces   []CardSpace // every space that can be read, in the order of the configuration (sorted)
	Details  []CardDetail
}

// PersonRow is a person in the list of people: once, with the spaces they are in.
type PersonRow struct {
	ID, Name, Role string
	Spaces         []string
}

// ParseDetails reads a person's body as a list of details: each list line ("- …") is
// one, and any other paragraph is ONE detail as a whole (never split), so a body
// written by hand or by an older version reads as it was meant.
func ParseDetails(body string) []string {
	body = strings.ReplaceAll(body, "\r\n", "\n")
	var out []string
	for _, block := range splitBlocks(body) {
		lines := strings.Split(block, "\n")
		list := true
		for _, l := range lines {
			if !strings.HasPrefix(l, "- ") || strings.TrimSpace(l[2:]) == "" {
				list = false
			}
		}
		if !list {
			out = append(out, block)
			continue
		}
		for _, l := range lines {
			out = append(out, strings.TrimSpace(l[2:]))
		}
	}
	return out
}

// splitBlocks cuts text at blank lines, trimming each block.
func splitBlocks(s string) []string {
	var blocks []string
	var cur []string
	flush := func() {
		if len(cur) > 0 {
			blocks = append(blocks, strings.TrimSpace(strings.Join(cur, "\n")))
			cur = nil
		}
	}
	for _, l := range strings.Split(s, "\n") {
		if strings.TrimSpace(l) == "" {
			flush()
			continue
		}
		cur = append(cur, strings.TrimRight(l, " \t"))
	}
	flush()
	return blocks
}

// cleanDetail is the text of a detail as the card compares it: trimmed, with the same
// line ends.
func cleanDetail(s string) string {
	return strings.TrimSpace(strings.ReplaceAll(s, "\r\n", "\n"))
}

// renderDetails writes details as a body: a one-line detail is a list line, a longer
// one a paragraph of its own. Reading it back gives the same details.
func renderDetails(details []string) string {
	var out strings.Builder
	inList := false
	for _, d := range details {
		if strings.Contains(d, "\n") {
			if out.Len() > 0 {
				out.WriteString("\n")
			}
			out.WriteString(d + "\n")
			inList = false
			continue
		}
		if !inList && out.Len() > 0 {
			out.WriteString("\n")
		}
		out.WriteString("- " + d + "\n")
		inList = true
	}
	return out.String()
}

// openSpaces opens every space of the configuration whose notes exist on this
// machine, in order. A space whose folder is missing (a machine with only the work
// space, say) is left out: it is read as empty and never written.
func openSpaces(c *config.Config) ([]*Brain, error) {
	var out []*Brain
	for _, name := range c.SpaceNames() {
		sp := c.Spaces[name]
		if _, err := os.Stat(sp.Memory); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, err
		}
		b, err := Open(c, sp)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		b.Now = cardNow
		out = append(out, b)
	}
	return out, nil
}

// People lists every person once, with the spaces they are in. A space that cannot be
// opened is left out and named in unread, so one broken space never hides the people
// of the others.
func People(c *config.Config) (rows []PersonRow, unread []string, err error) {
	var brains []*Brain
	for _, name := range c.SpaceNames() {
		sp := c.Spaces[name]
		if _, err := os.Stat(sp.Memory); errors.Is(err, os.ErrNotExist) {
			continue
		}
		b, err := Open(c, sp)
		if err != nil {
			unread = append(unread, name)
			continue
		}
		brains = append(brains, b)
	}
	byID := map[string]*PersonRow{}
	var order []string
	for _, b := range brains {
		for _, n := range b.Mem.OfType("person") {
			r := byID[n.Header.ID]
			if r == nil {
				role, _ := PersonParts(n)
				r = &PersonRow{ID: n.Header.ID, Name: n.Header.Title, Role: role}
				byID[n.Header.ID] = r
				order = append(order, n.Header.ID)
			}
			r.Spaces = append(r.Spaces, b.Space.Name)
		}
	}
	rows = make([]PersonRow, 0, len(order))
	for _, id := range order {
		rows = append(rows, *byID[id])
	}
	sort.SliceStable(rows, func(i, j int) bool {
		return strings.ToLower(rows[i].Name) < strings.ToLower(rows[j].Name)
	})
	return rows, unread, nil
}

// PersonCard joins the files of the person with that id in every space.
func PersonCard(c *config.Config, id string) (*Card, error) {
	brains, err := openSpaces(c)
	if err != nil {
		return nil, err
	}
	return cardOf(brains, id)
}

func cardOf(brains []*Brain, id string) (*Card, error) {
	card := &Card{ID: id}
	for _, b := range brains {
		cs := CardSpace{Space: b.Space.Name}
		if n := b.Mem.Notes[id]; n != nil && n.Header.Type == "person" {
			cs.Exists = true
			cs.Role, cs.Aliases = PersonParts(n)
			if card.Name == "" {
				card.Name = n.Header.Title
			}
			for _, d := range ParseDetails(n.Body) {
				i := slices.IndexFunc(card.Details, func(x CardDetail) bool { return x.Text == d })
				if i < 0 {
					card.Details = append(card.Details, CardDetail{Text: d})
					i = len(card.Details) - 1
				}
				if !slices.Contains(card.Details[i].Spaces, cs.Space) {
					card.Details[i].Spaces = append(card.Details[i].Spaces, cs.Space)
				}
			}
		}
		card.Spaces = append(card.Spaces, cs)
	}
	if card.Name == "" {
		return nil, ErrNoPerson
	}
	return card, nil
}

// CardSpaceInput is the role and other names a person has in one space.
type CardSpaceInput struct {
	Role    string
	Aliases []string
}

// CardInput is the whole card as the user left it.
type CardInput struct {
	ID string
	// Spaces holds the role and other names to write per space; a space left out keeps
	// its own as they are.
	Spaces map[string]CardSpaceInput
	// Details is the whole list. A detail with no text is dropped (that is how one is
	// removed); one with text must say at least one space where it is visible.
	Details []CardDetail
}

// CardError is a problem with one detail, so a form can show it next to its row.
type CardError struct {
	Row int // index in CardInput.Details
	Err error
}

func (e *CardError) Error() string { return fmt.Sprintf("detail %d: %v", e.Row+1, e.Err) }
func (e *CardError) Unwrap() error { return e.Err }

// ErrNoSpaceForDetail says a detail with text was left with no space to live in.
var ErrNoSpaceForDetail = errors.New("a detail needs at least one space where it is visible")

// SavePersonCard writes the card back, in one operation: everything is checked first
// (the details, every note that would be written, in every space), then each file that
// changed is written once. A space where a detail is not ticked loses it; a space
// that has no file for the person gets one, with the same id, when there is something
// to put in it; a file is never deleted, it is left with an empty body. It returns the
// spaces written. The caller holds the notes lock.
func SavePersonCard(c *config.Config, in CardInput) ([]string, error) {
	brains, err := openSpaces(c)
	if err != nil {
		return nil, err
	}
	card, err := cardOf(brains, in.ID)
	if err != nil {
		return nil, err
	}
	have := map[string]bool{}
	for _, b := range brains {
		have[b.Space.Name] = true
	}
	// 1. The details.
	for i, d := range in.Details {
		if cleanDetail(d.Text) == "" {
			continue
		}
		if len(d.Spaces) == 0 {
			return nil, &CardError{i, ErrNoSpaceForDetail}
		}
		for _, sp := range d.Spaces {
			if !have[sp] {
				return nil, &CardError{i, fmt.Errorf("space %q is not one of the spaces of this machine", sp)}
			}
		}
	}
	for sp := range in.Spaces {
		if !have[sp] {
			return nil, fmt.Errorf("space %q is not one of the spaces of this machine", sp)
		}
	}
	// 2. The note each space would hold.
	type plan struct {
		b        *Brain
		note     *memory.Note
		previous *memory.Note
	}
	var plans []plan
	for _, b := range brains {
		sp := b.Space.Name
		var want []string
		for _, d := range in.Details {
			t := cleanDetail(d.Text)
			if t != "" && slices.Contains(d.Spaces, sp) && !slices.Contains(want, t) {
				want = append(want, t)
			}
		}
		roleIn, hasRole := in.Spaces[sp]
		old := b.Mem.Notes[in.ID]
		if old != nil && old.Header.Type != "person" {
			return nil, fmt.Errorf("%s in %s is not a person", in.ID, sp)
		}
		if old == nil {
			if len(want) == 0 && (!hasRole || (strings.TrimSpace(roleIn.Role) == "" && len(roleIn.Aliases) == 0)) {
				continue // nothing of theirs is visible here: no file
			}
			n := &memory.Note{
				Path: memory.PathFor("person", in.ID),
				Header: memory.Header{ID: in.ID, Type: "person", Title: card.Name, Created: b.today(),
					Summary: personSummary(b.Language, card.Name, roleIn.Role, roleIn.Aliases)},
				Body: renderDetails(want),
			}
			plans = append(plans, plan{b, n, nil})
			continue
		}
		cp := *old
		if hasRole {
			cp.Header.Summary = personSummary(b.Language, old.Header.Title, roleIn.Role, roleIn.Aliases)
		}
		if !slices.Equal(ParseDetails(old.Body), want) {
			cp.Body = renderDetails(want)
		}
		if cp.Header.Summary != old.Header.Summary || cp.Body != old.Body {
			plans = append(plans, plan{b, &cp, old})
		}
	}
	// 3. Everything is checked before anything is written.
	for _, p := range plans {
		if p.b.broken(p.note.Path) {
			return nil, fmt.Errorf("%s (%s) is not a valid note, so it is not written over; fix it first (`augmented-brain check` says what is wrong)", p.note.Path, p.b.Space.Name)
		}
		if err := noNewSecret(p.note, p.previous); err != nil {
			return nil, fmt.Errorf("%s: %w", p.b.Space.Name, err)
		}
	}
	// 4. Write.
	var written []string
	for _, p := range plans {
		if err := p.b.save(p.note); err != nil {
			return written, fmt.Errorf("%s: %w", p.b.Space.Name, err)
		}
		written = append(written, p.b.Space.Name)
	}
	return written, nil
}

// SavePersonIn is person_save across spaces: the person is saved, with this role,
// other names and notes, in the spaces given and no others. One id serves every space:
// the person's if some space of the configuration already knows them by that name,
// otherwise a new one free in all the spaces given. It returns the person's id and
// the spaces written. The caller holds the notes lock.
func SavePersonIn(c *config.Config, spaces []string, in PersonInput) (string, []string, error) {
	if strings.TrimSpace(in.Name) == "" {
		return "", nil, errors.New("a person needs a name")
	}
	brains, err := openSpaces(c)
	if err != nil {
		return "", nil, err
	}
	var chosen []*Brain
	for _, name := range spaces {
		i := slices.IndexFunc(brains, func(b *Brain) bool { return b.Space.Name == name })
		if i < 0 {
			return "", nil, fmt.Errorf("%q is not one of the spaces: %s", name, strings.Join(c.SpaceNames(), ", "))
		}
		if !slices.Contains(chosen, brains[i]) {
			chosen = append(chosen, brains[i])
		}
	}
	if len(chosen) == 0 {
		return "", nil, errors.New("no space to save the person in")
	}
	id, title := "", ""
	for _, b := range brains {
		if n, _ := b.findPerson(strings.TrimSpace(in.Name)); n != nil {
			id, title = n.Header.ID, n.Header.Title
			break
		}
	}
	if id == "" {
		base := "person-" + Slug(in.Name)
		id = base
		for i := 2; slices.ContainsFunc(chosen, func(b *Brain) bool { _, taken := b.Mem.Notes[id]; return taken || b.brokenID(id) }); i++ {
			id = fmt.Sprintf("%s-%d", base, i)
		}
	}
	var notes []*memory.Note
	for _, b := range chosen {
		n, err := b.buildPerson(in, id, title)
		if err != nil {
			return "", nil, err
		}
		if b.broken(n.Path) {
			return "", nil, fmt.Errorf("%s (%s) is not a valid note, so it is not written over; fix it first", n.Path, b.Space.Name)
		}
		if err := noNewSecret(n, b.Mem.Notes[id]); err != nil {
			return "", nil, fmt.Errorf("%s: %w", b.Space.Name, err)
		}
		notes = append(notes, n)
	}
	var written []string
	for i, b := range chosen {
		if err := b.save(notes[i]); err != nil {
			return id, written, fmt.Errorf("%s: %w", b.Space.Name, err)
		}
		written = append(written, b.Space.Name)
	}
	return id, written, nil
}
