package panel

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/brain"
	"github.com/icalvete/augmented-brain/internal/config"
)

// twoFiles writes Lucia Prado the way today's two files look: the same id, a paragraph
// of body each, in two spaces.
func twoFiles(t *testing.T, s *Server, cfg string) string {
	t.Helper()
	loc := saved(t, s.post(t, "/person", url.Values{"s": {"work"}, "name": {"Lucia Prado"}, "role": {"client"}, "notes": {"Runs the shop. Full card is in home."}}), "work file")
	saved(t, s.post(t, "/person", url.Values{"s": {"home"}, "name": {"Lucia Prado"}, "role": {"friend"}, "notes": {"Plays chess on Sundays."}}), "home file")
	return idIn(loc)
}

func personPath(cfg, space, id string) string {
	return filepath.Join(filepath.Dir(cfg), "brain-"+space, "people", id+".md")
}

func TestPeopleOneCardAcrossSpaces(t *testing.T) {
	s, cfg := fixture(t, "en")
	id := twoFiles(t, s, cfg)
	saved(t, s.post(t, "/person", url.Values{"s": {"home"}, "name": {"Only Home"}}), "home only")

	page := s.get(t, "/people?s=work").Body.String()
	if n := strings.Count(page, ">Lucia Prado</a>"); n != 1 {
		t.Errorf("Lucia is listed %d times:\n%s", n, page)
	}
	if !strings.Contains(page, `<span class="badge">home</span><span class="badge">work</span>`) {
		t.Errorf("no badge for each space:\n%s", page)
	}
	// A person of the other space opens in the space that has them.
	if !strings.Contains(page, `note?s=home&amp;id=person-only-home"`) {
		t.Errorf("link of a person who is only in home:\n%s", page)
	}

	// Each paragraph is a detail ticked in its own file's space, and only there.
	card := s.get(t, "/note?s=work&id="+id).Body.String()
	for _, want := range []string{`name="role.work" value="client"`, `name="role.home" value="friend"`,
		`>Runs the shop. Full card is in home.</textarea>`, `>Plays chess on Sundays.</textarea>`, "Another detail"} {
		if !strings.Contains(card, want) {
			t.Errorf("card lacks %q:\n%s", want, card)
		}
	}
	if strings.Count(card, `value="work" checked`) != 2 { // the work paragraph, and the new empty row
		t.Errorf("ticks in work: %d", strings.Count(card, `value="work" checked`))
	}
	if strings.Count(card, `value="home" checked`) != 1 {
		t.Errorf("ticks in home: %d", strings.Count(card, `value="home" checked`))
	}
	if strings.Contains(card, `<form method="post" action="/s/`+s.session+`/person"`) {
		t.Error("the old form is still there")
	}
	// A person with no file in the space still has their card there.
	if w := s.get(t, "/note?s=home&id=person-lucia-prado"); w.Code != 200 {
		t.Errorf("card from home: %d", w.Code)
	}
	// Another detail adds a row.
	more := s.get(t, "/note?s=work&id="+id+"&extra=1").Body.String()
	if strings.Count(more, `<textarea name="t`) != strings.Count(card, `<textarea name="t`)+1 {
		t.Error("extra=1 did not add a row")
	}
}

func cardPost(id string, rows ...[]string) url.Values {
	f := url.Values{"s": {"work"}, "id": {id}, "n": {itoa(len(rows))},
		"role.work": {"client"}, "aliases.work": {"Lu"}, "role.home": {"friend"}, "aliases.home": {""}}
	for i, r := range rows {
		f.Set("t"+itoa(i), r[0])
		for _, sp := range r[1:] {
			f.Add("v"+itoa(i), sp)
		}
	}
	return f
}

func itoa(n int) string { return strconv.Itoa(n) }

func TestPersonCardSave(t *testing.T) {
	s, cfg := fixture(t, "en")
	id := twoFiles(t, s, cfg)
	secret := "Is expecting a baby, tell nobody"
	saved(t, s.post(t, "/person/card", cardPost(id,
		[]string{"Runs the shop. Full card is in home.", "work"},
		[]string{"Plays chess on Sundays.", "home"},
		[]string{secret, "home"},
		[]string{"Prefers email", "home", "work"},
		[]string{""}, // the empty row to add one
	)), "save card")
	work, _ := os.ReadFile(personPath(cfg, "work", id))
	home, _ := os.ReadFile(personPath(cfg, "home", id))
	if strings.Contains(string(work), "baby") || !strings.Contains(string(work), "- Prefers email") || !strings.Contains(string(work), "also: Lu") {
		t.Errorf("work:\n%s", work)
	}
	if !strings.Contains(string(home), secret) || !strings.Contains(string(home), "- Prefers email") {
		t.Errorf("home:\n%s", home)
	}
	// The index of the open space was not written back stale.
	idx, _ := os.ReadFile(filepath.Join(filepath.Dir(cfg), "brain-work", "MEMORY.md"))
	if !strings.Contains(string(idx), "client · also: Lu") {
		t.Errorf("work index:\n%s", idx)
	}
	// Emptying a text removes the detail from both files.
	saved(t, s.post(t, "/person/card", cardPost(id, []string{"", "work"}, []string{"Plays chess on Sundays.", "home"})), "remove")
	c, err := config.Load(cfg)
	if err != nil {
		t.Fatal(err)
	}
	card, err := brain.PersonCard(c, id)
	if err != nil {
		t.Fatal(err)
	}
	if len(card.Details) != 1 || card.Details[0].Text != "Plays chess on Sundays." || strings.Join(card.Details[0].Spaces, ",") != "home" {
		t.Errorf("after removing: %+v", card.Details)
	}
	if f, err := os.Stat(personPath(cfg, "work", id)); err != nil || f.Size() == 0 {
		t.Error("the work file must stay")
	}
}

// A detail with text and no space ticked is not saved: the page comes back with the
// error next to its row and what was typed, and nothing is written.
func TestPersonCardDetailWithoutSpace(t *testing.T) {
	for lang, msg := range map[string]string{"en": "tick at least one space", "es": "marca al menos un espacio"} {
		s, cfg := fixture(t, lang)
		id := twoFiles(t, s, cfg)
		before, _ := os.ReadFile(personPath(cfg, "work", id))
		w := s.post(t, "/person/card", cardPost(id, []string{"Fine", "work"}, []string{"Typed but unticked"}))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("%s: %d", lang, w.Code)
		}
		body := w.Body.String()
		i := strings.Index(body, "Typed but unticked")
		if i < 0 || !strings.Contains(body, msg) || !strings.Contains(body[i:], msg) || strings.Contains(body[:i], msg) {
			t.Errorf("%s: the error is not next to the row:\n%s", lang, body)
		}
		if !strings.Contains(body, ">Fine</textarea>") {
			t.Errorf("%s: what was typed is lost", lang)
		}
		after, _ := os.ReadFile(personPath(cfg, "work", id))
		if string(before) != string(after) {
			t.Errorf("%s: written despite the error", lang)
		}
	}
}

func TestPersonCardChecks(t *testing.T) {
	s, cfg := fixture(t, "en")
	id := twoFiles(t, s, cfg)
	form := cardPost(id, []string{"x", "work"})
	form.Set("csrf", "wrong")
	if w := s.do(t, req{method: http.MethodPost, path: s.base() + "/person/card", origin: "http://" + host, form: form}); w.Code != http.StatusForbidden {
		t.Errorf("bad csrf: %d", w.Code)
	}
	form.Set("csrf", s.csrf)
	if w := s.do(t, req{method: http.MethodPost, path: s.base() + "/person/card", origin: "http://evil.example", form: form}); w.Code != http.StatusForbidden {
		t.Errorf("bad origin: %d", w.Code)
	}
	if w := s.do(t, req{method: http.MethodPost, path: s.base() + "/person/card", host: "evil.example", origin: "http://" + host, form: form}); w.Code != http.StatusForbidden {
		t.Errorf("bad host: %d", w.Code)
	}
	// An unknown space in a box is refused by the server, whatever the page offered.
	if w := s.post(t, "/person/card", cardPost(id, []string{"x", "mars"})); w.Code != http.StatusBadRequest {
		t.Errorf("unknown space: %d", w.Code)
	}
	// An id that is nobody's.
	if w := s.post(t, "/person/card", cardPost("person-nobody", []string{"x", "work"})); w.Code != http.StatusBadRequest {
		t.Errorf("unknown person: %d", w.Code)
	}
	if body, _ := os.ReadFile(personPath(cfg, "work", id)); strings.Contains(string(body), "- x") {
		t.Error("a refused save wrote")
	}
}

// The Spanish page has its own words, and no English left in the form.
func TestPersonCardSpanish(t *testing.T) {
	s, cfg := fixture(t, "es")
	id := twoFiles(t, s, cfg)
	page := s.get(t, "/note?s=work&id="+id).Body.String()
	for _, want := range []string{"Su ficha", "Se ve en", "Otro detalle", "En work", "Detalle 1"} {
		if !strings.Contains(page, want) {
			t.Errorf("falta %q", want)
		}
	}
}

// A space that cannot be opened (its notes written by a newer program) never takes the
// People page of the others down: they are listed, and the page names that space.
func TestPeopleWithASpaceThatDoesNotOpen(t *testing.T) {
	s, cfg := fixture(t, "en")
	saved(t, s.post(t, "/person", url.Values{"s": {"work"}, "name": {"Lucia Prado"}}), "work file")
	home := filepath.Join(filepath.Dir(cfg), "brain-home", ".augmented-brain")
	if err := os.MkdirAll(home, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "format"), []byte("999\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	r := s.get(t, "/people?s=work")
	if r.Code != http.StatusOK {
		t.Fatalf("status %d:\n%s", r.Code, r.Body)
	}
	page := r.Body.String()
	if !strings.Contains(page, ">Lucia Prado</a>") || !strings.Contains(page, "The people of home are not here") {
		t.Errorf("page:\n%s", page)
	}
}
