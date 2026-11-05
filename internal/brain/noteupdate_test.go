package brain

import (
	"github.com/icalvete/augmented-brain/internal/config"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A title that exists creates nothing: no silent "-2", and the error says the id to
// change it with. The same for areas.
func TestNoSilentDuplicate(t *testing.T) {
	b := open(t, "es")
	if _, err := b.NewRule("Respuestas cortas", "", "Una idea por frase."); err != nil {
		t.Fatal(err)
	}
	if _, err := b.NewRule("Respuestas cortas", "", "Otra cosa."); err == nil || !strings.Contains(err.Error(), "rule-respuestas-cortas") {
		t.Errorf("second rule with the same title: %v", err)
	}
	if _, err := b.NewKnowledge("lesson", "El pan necesita tiempo", "", "", "x"); err != nil {
		t.Fatal(err)
	}
	if _, err := b.NewKnowledge("lesson", "El pan necesita tiempo", "", "", "y"); err == nil {
		t.Error("second lesson with the same title was created")
	}
	if _, err := b.NewArea("General", ""); err == nil || !strings.Contains(err.Error(), "panel") {
		t.Errorf("an area that exists (init makes General): %v", err)
	}
	for _, id := range []string{"rule-respuestas-cortas-2", "lesson-el-pan-necesita-tiempo-2", "area-general-2"} {
		if _, ok := b.Mem.Notes[id]; ok {
			t.Errorf("%s was created", id)
		}
	}
}

// With an id, a rule or knowledge note is changed in place; what is not sent stays, and
// the version it replaces is kept in archive/.
func TestUpdateNote(t *testing.T) {
	b := open(t, "es")
	n, err := b.NewRule("Respuestas cortas", "una idea", "Una idea por frase.")
	if err != nil {
		t.Fatal(err)
	}
	created := n.Header.Created
	up, err := b.UpdateNote(n.Header.ID, "rule", "", "", "", "Una idea por frase, sin relleno.")
	if err != nil {
		t.Fatal(err)
	}
	if up.Header.Title != "Respuestas cortas" || up.Header.Summary != "una idea" || up.Header.Created != created ||
		!strings.Contains(up.Body, "sin relleno") {
		t.Errorf("updated rule = %+v %q", up.Header, up.Body)
	}
	old, err := filepath.Glob(filepath.Join(b.Mem.Dir, "archive", "rules", n.Header.ID+"-*.md"))
	if err != nil || len(old) != 1 {
		t.Fatalf("archived versions: %v %v", old, err)
	}
	if data, _ := os.ReadFile(old[0]); !strings.Contains(string(data), "Una idea por frase.\n") || strings.Contains(string(data), "sin relleno") {
		t.Errorf("the archived copy is not the old version:\n%s", data)
	}
	// A second change the same day keeps both old versions.
	if _, err := b.UpdateNote(n.Header.ID, "rule", "", "", "", "Tercera."); err != nil {
		t.Fatal(err)
	}
	if old, _ := filepath.Glob(filepath.Join(b.Mem.Dir, "archive", "rules", n.Header.ID+"-*.md")); len(old) != 2 {
		t.Errorf("two changes, archived versions: %v", old)
	}
	if _, err := b.UpdateNote(n.Header.ID, "lesson", "", "", "", "x"); err == nil {
		t.Error("a rule was changed as a lesson")
	}
	if _, err := b.UpdateNote("area-general", "area", "x", "", "", "x"); err == nil {
		t.Error("an area was changed as knowledge")
	}
}

// Retired, a rule leaves the memory for archive/ with the reason and what replaces it,
// and no longer loads.
func TestRetireNote(t *testing.T) {
	b := open(t, "es")
	oldRule, _ := b.NewRule("Nunca tocar core", "", "No se toca.")
	newRule, _ := b.NewRule("Core es de Marcos", "", "No sin su permiso.")
	if _, err := b.RetireNote(oldRule.Header.ID, "", ""); err == nil {
		t.Error("retired without a reason")
	}
	if _, err := b.RetireNote(oldRule.Header.ID, "duplicada", "rule-no-existe"); err == nil {
		t.Error("replaced by a note that does not exist")
	}
	dest, err := b.RetireNote(oldRule.Header.ID, "duplicada.", newRule.Header.ID)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(b.Mem.Dir, "archive", "rules", oldRule.Header.ID+".md"); dest != want {
		t.Errorf("retired to %s, want %s", dest, want)
	}
	data, _ := os.ReadFile(dest)
	if !strings.Contains(string(data), "Retirada el 2026-10-15: duplicada. Sustituida por [[rule-core-es-de-marcos]].") {
		t.Errorf("retired note:\n%s", data)
	}
	if _, err := os.Stat(filepath.Join(b.Mem.Dir, filepath.FromSlash(oldRule.Path))); !os.IsNotExist(err) {
		t.Errorf("the rule is still in rules/: %v", err)
	}
	again := open2(t, b)
	if _, ok := again.Mem.Notes[oldRule.Header.ID]; ok {
		t.Error("a retired rule still loads")
	}
	if _, ok := again.Mem.Notes[newRule.Header.ID]; !ok {
		t.Error("the rule that replaces it is gone")
	}
}

// open2 opens the space of b again from disk, as the next session would.
func open2(t *testing.T, b *Brain) *Brain {
	t.Helper()
	c, err := config.Load(b.ConfigDir)
	if err != nil {
		t.Fatal(err)
	}
	again, err := Open(c, c.Spaces["work"])
	if err != nil {
		t.Fatal(err)
	}
	return again
}
