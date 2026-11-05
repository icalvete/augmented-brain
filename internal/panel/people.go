package panel

import (
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/icalvete/augmented-brain/internal/brain"
	"github.com/icalvete/augmented-brain/internal/memory"
)

// personData is what a person's page shows besides the note: the form to change them,
// and the tasks that wait on them or name them.
type personData struct {
	Role, Aliases string
	Card          *cardForm // the one form for the person across spaces; nil when it cannot be read
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
		Role: field(r, "role"), Notes: field(r, "notes"), ReplaceNotes: r.PostForm.Has("notes") && r.PostForm.Get("replace") == "1"})
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

// ---- the card: one form for a person across the spaces ----

type cardSpaceRow struct {
	Space, Role, Aliases string
	Exists               bool
}

type cardCheck struct {
	Space string
	On    bool
}

// cardRow is one detail: its text, one box per space, and what is wrong with it.
type cardRow struct {
	N      int
	Text   string
	Checks []cardCheck
	Err    string
}

// Lines is how tall the detail's box is: enough for its text (about 70 characters a
// line), from 2 to 10 lines, so a long detail can be read and edited without scrolling.
func (r cardRow) Lines() int {
	n := 0
	for _, l := range strings.Split(r.Text, "\n") {
		n += len(l)/70 + 1
	}
	return min(max(n, 2), 10)
}

// cardForm is what the person's form shows: a role row per space, then the details,
// each with a box per space (the spaces come from the configuration).
type cardForm struct {
	ID     string
	Spaces []cardSpaceRow
	Rows   []cardRow
	Extra  int // the address that adds one more empty row
}

const maxCardRows = 200

// cardFor builds the form for the person with that id. extra is how many empty rows
// to end with besides the one that is always there. posted, when not nil, is a form
// the user sent that was not saved: its rows and values are shown again, with the
// error next to the row that has it.
func (v *view) cardFor(id string, extra int, posted *http.Request, bad *brain.CardError) (*cardForm, error) {
	card, err := brain.PersonCard(v.c, id)
	if err != nil {
		return nil, err
	}
	f := &cardForm{ID: id}
	for _, sp := range card.Spaces {
		row := cardSpaceRow{Space: sp.Space, Exists: sp.Exists, Role: sp.Role, Aliases: strings.Join(sp.Aliases, ", ")}
		if posted != nil {
			row.Role, row.Aliases = posted.PostForm.Get("role."+sp.Space), posted.PostForm.Get("aliases."+sp.Space)
		}
		f.Spaces = append(f.Spaces, row)
	}
	checks := func(on func(space string) bool) []cardCheck {
		var cs []cardCheck
		for _, sp := range card.Spaces {
			cs = append(cs, cardCheck{sp.Space, on(sp.Space)})
		}
		return cs
	}
	if posted != nil {
		n, _ := strconv.Atoi(posted.PostForm.Get("n"))
		for i := 0; i < n && i < maxCardRows; i++ {
			ticked := posted.PostForm["v"+strconv.Itoa(i)]
			row := cardRow{N: i, Text: posted.PostForm.Get("t" + strconv.Itoa(i)),
				Checks: checks(func(sp string) bool { return containsID(ticked, sp) })}
			if bad != nil && bad.Row == i {
				row.Err = v.t("person.nospace")
			}
			f.Rows = append(f.Rows, row)
		}
		f.Extra = 1
		return f, nil
	}
	for i, d := range card.Details {
		f.Rows = append(f.Rows, cardRow{N: i, Text: d.Text, Checks: checks(func(sp string) bool { return containsID(d.Spaces, sp) })})
	}
	// The empty rows to add one: new details start visible where the user is looking.
	for i := 0; i <= extra; i++ {
		f.Rows = append(f.Rows, cardRow{N: len(f.Rows), Checks: checks(func(sp string) bool { return sp == v.b.Space.Name })})
	}
	f.Extra = extra + 1
	return f, nil
}

// extraRows reads how many more empty rows the address asks for.
func extraRows(r *http.Request) int {
	n, _ := strconv.Atoi(r.URL.Query().Get("extra"))
	return max(0, min(n, 20))
}

// personCard saves the whole form. A detail with text and no box ticked is not saved:
// the page comes back with what was typed and the error next to its row.
func (s *Server) personCard(w http.ResponseWriter, r *http.Request) {
	v, release, err := s.open(r)
	if err != nil {
		s.fail(w, r, v, http.StatusBadRequest, err)
		return
	}
	var redirect, name string
	var p page
	var bad *brain.CardError
	func() {
		defer release()
		if redirect, err = v.saveCard(r); errors.As(err, &bad) {
			v.cardErr = bad
			name, p, err = s.note(v, r) // the same page, with what was typed
		}
	}()
	switch {
	case err != nil:
		s.fail(w, r, v, http.StatusBadRequest, err)
	case bad != nil:
		s.render(w, http.StatusBadRequest, name, p)
	default:
		http.Redirect(w, r, s.base()+redirect, http.StatusSeeOther)
	}
}

// saveCard reads the form and writes it through brain, which checks every detail and
// every space before the first write. The server checks what the page also shows.
func (v *view) saveCard(r *http.Request) (string, error) {
	id := field(r, "id")
	card, err := brain.PersonCard(v.c, id)
	if err != nil {
		return "", err
	}
	in := brain.CardInput{ID: id, Spaces: map[string]brain.CardSpaceInput{}}
	for _, sp := range card.Spaces {
		if !r.PostForm.Has("role." + sp.Space) {
			continue
		}
		var aliases []string
		for _, a := range strings.Split(r.PostForm.Get("aliases."+sp.Space), ",") {
			if a = strings.TrimSpace(a); a != "" {
				aliases = append(aliases, a)
			}
		}
		in.Spaces[sp.Space] = brain.CardSpaceInput{Role: field(r, "role."+sp.Space), Aliases: aliases}
	}
	n, _ := strconv.Atoi(field(r, "n"))
	for i := 0; i < n && i < maxCardRows; i++ {
		k := strconv.Itoa(i)
		in.Details = append(in.Details, brain.CardDetail{Text: r.PostForm.Get("t" + k), Spaces: r.PostForm["v"+k]})
	}
	if _, err := brain.SavePersonCard(v.c, in); err != nil {
		return "", err
	}
	// The space open here was read before the save: read it again, or the index written
	// after every change would put the old people back.
	if b, err := brain.Open(v.c, v.b.Space); err == nil {
		b.Now = v.b.Now
		v.b = b
	}
	return "/note?s=" + url.QueryEscape(v.b.Space.Name) + "&ok=1&id=" + url.QueryEscape(id), nil
}
