package panel

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/brain"
)

// Search looks in every space, the shared notes, each inbox and the journals archived
// from tasks; each result says where it is and leads there.
func TestSearchEverywhere(t *testing.T) {
	s, cfg := fixture(t, "en")
	home := openSpace(t, cfg, "home")
	if _, err := home.NewTask(brain.NewTaskInput{Title: "Fix the boiler"}); err != nil {
		t.Fatal(err)
	}
	if err := home.Capture("Ask the neighbour about the boiler", ""); err != nil {
		t.Fatal(err)
	}
	shared, err := openSpace(t, cfg, "work").Shared()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := shared.NewKnowledge("reference", "Boiler manual", "", "", "Reset with the red button."); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(home.Mem.Dir, "archive", "journals")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	journal := "# Fix the boiler · entries from 2026-09-01 to 2026-09-30\n\n- 2026-09-03 — The boiler <b>made</b> a noise\n"
	if err := os.WriteFile(filepath.Join(dir, "task-fix-the-boiler-2026-09-01-2026-09-30.md"), []byte(journal), 0o600); err != nil {
		t.Fatal(err)
	}

	page := s.get(t, "/search?s=work&q=boiler").Body.String()
	for _, want := range []string{
		"Fix the <mark>boiler</mark>",                             // a task of another space
		`<span class="badge">home</span>`,                         // says which
		"Ask the neighbour about the <mark>boiler</mark>",         // a capture
		"/inbox?s=home",                                           // leads to its inbox
		"<mark>Boiler</mark> manual",                              // a shared note
		"archived journal",                                        // a journal moved out of its task
		"/task?id=task-fix-the-boiler&amp;s=home",                 // leads to its task
		"The <mark>boiler</mark> &lt;b&gt;made&lt;/b&gt; a noise", // its text, never its HTML
	} {
		if !strings.Contains(page, want) {
			t.Errorf("search lacks %s", want)
		}
	}
	if strings.Contains(page, "<b>made</b>") {
		t.Error("a note's HTML reached the page")
	}
}
