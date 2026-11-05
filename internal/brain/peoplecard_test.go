package brain

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/memory"
	"github.com/icalvete/augmented-brain/internal/setup"
)

// twoSpaces makes a configuration with the spaces "home" and "work" and returns it.
func twoSpaces(t *testing.T) *config.Config {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg := filepath.Join(home, "cfg")
	for i, sp := range []string{"work", "home"} {
		folder := filepath.Join(home, "f-"+sp)
		if err := os.MkdirAll(folder, 0o755); err != nil {
			t.Fatal(err)
		}
		o := setup.Options{Language: "en", Space: sp, WorkFolder: folder, Memory: "~/brain-" + sp, Today: "2026-10-15"}
		var err error
		if i == 0 {
			_, err = setup.Init(cfg, o)
		} else {
			_, err = setup.AddSpace(cfg, o)
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	cardNow = func() time.Time { return time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC) }
	t.Cleanup(func() { cardNow = time.Now })
	c, err := config.Load(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func personFile(t *testing.T, c *config.Config, space, id string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(c.Spaces[space].Memory, "people", id+".md"))
	if err != nil {
		return ""
	}
	return string(data)
}

func writePerson(t *testing.T, c *config.Config, space string, in PersonInput) string {
	t.Helper()
	b, err := Open(c, c.Spaces[space])
	if err != nil {
		t.Fatal(err)
	}
	n, err := b.SavePerson(in)
	if err != nil {
		t.Fatal(err)
	}
	return n.Header.ID
}

func TestParseDetails(t *testing.T) {
	body := "- Likes coffee\n- Birthday in May\n\nA paragraph that\nspans two lines.\n\n- one\nnot a list line\n"
	want := []string{"Likes coffee", "Birthday in May", "A paragraph that\nspans two lines.", "- one\nnot a list line"}
	if got := ParseDetails(body); !reflect.DeepEqual(got, want) {
		t.Errorf("got %q", got)
	}
	if got := ParseDetails(renderDetails(want)); !reflect.DeepEqual(got, want) {
		t.Errorf("render and parse again: %q", got)
	}
	if got := ParseDetails(""); len(got) != 0 {
		t.Errorf("empty body: %q", got)
	}
}

// Two files with the same id, each with a paragraph, read as one card: the paragraphs
// are details tagged with the space of their own file (what the Marcos case needs).
func TestPersonCardJoinsByIDNotName(t *testing.T) {
	c := twoSpaces(t)
	id := writePerson(t, c, "work", PersonInput{Name: "Lucia Prado", Role: "client", Aliases: []string{"Lu"}, Notes: "Runs the shop. Full card is in home."})
	if id2 := writePerson(t, c, "home", PersonInput{Name: "Lucia Prado", Role: "friend", Notes: "Plays chess on Sundays."}); id2 != id {
		t.Fatalf("ids differ: %s %s", id, id2)
	}
	// Another person with the same name, other id: never joined.
	hb, _ := Open(c, c.Spaces["home"])
	if err := hb.save(&memory.Note{Path: memory.PathFor("person", "person-lucia-prado-2"),
		Header: memory.Header{ID: "person-lucia-prado-2", Type: "person", Title: "Lucia Prado", Summary: "neighbour", Created: "2026-10-15"}}); err != nil {
		t.Fatal(err)
	}

	card, err := PersonCard(c, id)
	if err != nil {
		t.Fatal(err)
	}
	if card.Name != "Lucia Prado" {
		t.Errorf("name %q", card.Name)
	}
	want := []CardDetail{
		{"Plays chess on Sundays.", []string{"home"}},
		{"Runs the shop. Full card is in home.", []string{"work"}},
	}
	if !reflect.DeepEqual(card.Details, want) {
		t.Errorf("details: %+v", card.Details)
	}
	byName := map[string]CardSpace{}
	for _, s := range card.Spaces {
		byName[s.Space] = s
	}
	if byName["work"].Role != "client" || !reflect.DeepEqual(byName["work"].Aliases, []string{"Lu"}) || byName["home"].Role != "friend" {
		t.Errorf("per space: %+v", card.Spaces)
	}
	rows, err := People(c)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].ID == rows[1].ID {
		t.Fatalf("people: %+v", rows)
	}
	if _, err := PersonCard(c, "person-nobody"); !errors.Is(err, ErrNoPerson) {
		t.Errorf("unknown id: %v", err)
	}
}

// A detail that is only in personal never reaches a work file; one in both is in both.
func TestSavePersonCardVisibility(t *testing.T) {
	c := twoSpaces(t)
	id := writePerson(t, c, "work", PersonInput{Name: "Lucia Prado", Role: "client"})
	secret := "Is expecting a baby, tell nobody at work"
	written, err := SavePersonCard(c, CardInput{ID: id,
		Spaces: map[string]CardSpaceInput{"work": {Role: "client", Aliases: []string{"Lu"}}, "home": {Role: "friend"}},
		Details: []CardDetail{
			{Text: secret, Spaces: []string{"home"}},
			{Text: "Prefers email", Spaces: []string{"home", "work"}},
			{Text: "   ", Spaces: nil}, // an empty row: ignored
		}})
	if err != nil {
		t.Fatal(err)
	}
	if len(written) != 2 {
		t.Errorf("written %v", written)
	}
	if work := personFile(t, c, "work", id); strings.Contains(work, "baby") || !strings.Contains(work, "- Prefers email") {
		t.Errorf("work file:\n%s", work)
	}
	home := personFile(t, c, "home", id)
	if !strings.Contains(home, secret) || !strings.Contains(home, "title: Lucia Prado") || !strings.Contains(home, "id: "+id) {
		t.Errorf("home file (created with the same id):\n%s", home)
	}
	// The index of work does not hold it either.
	idx, _ := os.ReadFile(filepath.Join(c.Spaces["work"].Memory, "MEMORY.md"))
	if strings.Contains(string(idx), "baby") || !strings.Contains(string(idx), "client · also: Lu") {
		t.Errorf("work index:\n%s", idx)
	}
}

func TestSavePersonCardRemovesAndNeverDeletes(t *testing.T) {
	c := twoSpaces(t)
	id := writePerson(t, c, "work", PersonInput{Name: "Lucia Prado", Notes: "- one\n- two"})
	if _, err := SavePersonCard(c, CardInput{ID: id, Details: []CardDetail{{"one", []string{"work", "home"}}, {"two", []string{"work"}}}}); err != nil {
		t.Fatal(err)
	}
	// Untick two in work, empty one: nothing is left, the files stay.
	if _, err := SavePersonCard(c, CardInput{ID: id, Details: []CardDetail{{"one", []string{"home"}}, {"", nil}}}); err != nil {
		t.Fatal(err)
	}
	work := personFile(t, c, "work", id)
	if work == "" || strings.Contains(work, "one") || strings.Contains(work, "two") {
		t.Errorf("work file should stay, empty of details:\n%s", work)
	}
	if !strings.Contains(personFile(t, c, "home", id), "- one") {
		t.Error("home lost its detail")
	}
	card, _ := PersonCard(c, id)
	if len(card.Details) != 1 || !reflect.DeepEqual(card.Details[0].Spaces, []string{"home"}) {
		t.Errorf("card: %+v", card.Details)
	}
	// Nothing at all: both files stay, empty.
	if _, err := SavePersonCard(c, CardInput{ID: id}); err != nil {
		t.Fatal(err)
	}
	for _, sp := range []string{"work", "home"} {
		if f := personFile(t, c, sp, id); f == "" || strings.Contains(f, "- ") {
			t.Errorf("%s: %q", sp, f)
		}
	}
}

// A detail with no space is refused with its row, and nothing is written, not even
// the other changes of the same save.
func TestSavePersonCardRejectsDetailWithoutSpace(t *testing.T) {
	c := twoSpaces(t)
	id := writePerson(t, c, "work", PersonInput{Name: "Lucia Prado", Role: "client"})
	before := personFile(t, c, "work", id)
	_, err := SavePersonCard(c, CardInput{ID: id,
		Spaces:  map[string]CardSpaceInput{"work": {Role: "boss"}},
		Details: []CardDetail{{"fine", []string{"work"}}, {"lost", nil}}})
	var ce *CardError
	if !errors.As(err, &ce) || ce.Row != 1 || !errors.Is(err, ErrNoSpaceForDetail) {
		t.Fatalf("err = %v", err)
	}
	if personFile(t, c, "work", id) != before || personFile(t, c, "home", id) != "" {
		t.Error("something was written")
	}
	if _, err := SavePersonCard(c, CardInput{ID: id, Details: []CardDetail{{"x", []string{"nowhere"}}}}); err == nil {
		t.Error("an unknown space was accepted")
	}
}

// Validation covers every space before the first write: a note that cannot be
// written in one space stops the save in all of them.
func TestSavePersonCardChecksAllBeforeWriting(t *testing.T) {
	c := twoSpaces(t)
	id := writePerson(t, c, "work", PersonInput{Name: "Lucia Prado"})
	before := personFile(t, c, "work", id)
	_, err := SavePersonCard(c, CardInput{ID: id, Details: []CardDetail{
		{"fine", []string{"work"}},
		{"token: ghp_" + strings.Repeat("a1B2", 9), []string{"home"}},
	}})
	if err == nil {
		t.Skip("the secret detector does not flag this text")
	}
	if personFile(t, c, "work", id) != before {
		t.Error("work was written although home was refused")
	}
}

func TestSavePersonCardKeepsBodyWhenDetailsSame(t *testing.T) {
	c := twoSpaces(t)
	id := writePerson(t, c, "work", PersonInput{Name: "Lucia Prado", Notes: "A free paragraph\nof two lines."})
	before := personFile(t, c, "work", id)
	card, _ := PersonCard(c, id)
	if _, err := SavePersonCard(c, CardInput{ID: id, Details: card.Details}); err != nil {
		t.Fatal(err)
	}
	if personFile(t, c, "work", id) != before {
		t.Error("an unchanged card rewrote the file")
	}
}

func TestSavePersonIn(t *testing.T) {
	c := twoSpaces(t)
	id, written, err := SavePersonIn(c, []string{"home"}, PersonInput{Name: "Lucia Prado", Role: "friend", Notes: "Plays chess."})
	if err != nil || len(written) != 1 || written[0] != "home" {
		t.Fatalf("%v %v", written, err)
	}
	if personFile(t, c, "work", id) != "" {
		t.Error("written in work, which was not asked for")
	}
	// Saved again for both: the same id is used in both.
	id2, written, err := SavePersonIn(c, []string{"home", "work"}, PersonInput{Name: "Lucia Prado", Role: "client"})
	if err != nil || id2 != id || len(written) != 2 {
		t.Fatalf("%q %v %v", id2, written, err)
	}
	if !strings.Contains(personFile(t, c, "work", id), "summary: client") {
		t.Errorf("work:\n%s", personFile(t, c, "work", id))
	}
	if _, _, err := SavePersonIn(c, []string{"mars"}, PersonInput{Name: "X"}); err == nil || !strings.Contains(err.Error(), "home, work") {
		t.Errorf("unknown space: %v", err)
	}
	if _, _, err := SavePersonIn(c, nil, PersonInput{Name: "X"}); err == nil {
		t.Error("no spaces accepted")
	}
}

// A space whose folder is missing on this machine is left out and never created.
func TestPersonCardMissingSpace(t *testing.T) {
	c := twoSpaces(t)
	id := writePerson(t, c, "work", PersonInput{Name: "Lucia Prado"})
	if err := os.RemoveAll(c.Spaces["home"].Memory); err != nil {
		t.Fatal(err)
	}
	card, err := PersonCard(c, id)
	if err != nil || len(card.Spaces) != 1 {
		t.Fatalf("%+v %v", card, err)
	}
	if _, err := SavePersonCard(c, CardInput{ID: id, Details: []CardDetail{{"x", []string{"home"}}}}); err == nil {
		t.Error("wrote to a space that is not here")
	}
	if _, err := os.Stat(c.Spaces["home"].Memory); err == nil {
		t.Error("the missing space was created")
	}
}
