package brain

import (
	"strings"
	"testing"
)

func TestSetSection(t *testing.T) {
	for _, lang := range []string{"en", "es"} {
		t.Run(lang, func(t *testing.T) {
			b := open(t, lang)
			n, err := b.NewTask(NewTaskInput{Title: "Paint the kitchen"})
			if err != nil {
				t.Fatal(err)
			}
			id := n.Header.ID
			if _, err := b.AddStep(id, "Buy the paint"); err != nil {
				t.Fatal(err)
			}
			if _, err := b.Step(StepInput{ID: id, Number: 1, Done: true}); err != nil {
				t.Fatal(err)
			}

			// Definition: the text replaces the template's; the next heading follows.
			def := "- Goal: a white kitchen.\n- Done when: two coats, dry."
			if _, err := b.SetSection(id, "Definition", def); err != nil {
				t.Fatal(err)
			}
			body := read(t, b, n.Path)
			if !strings.Contains(body, "\n"+def+"\n\n## 2. ") {
				t.Errorf("definition not in place:\n%s", body)
			}

			// Plan: new prose, and the steps stay after it, ticked as they were.
			if _, err := b.SetSection(id, "plan", "### Order\nCeiling first, then walls."); err != nil {
				t.Fatal(err)
			}
			body = read(t, b, n.Path)
			if !strings.Contains(body, "## 3. Plan\n### Order\nCeiling first, then walls.\n\n- [x] Buy the paint\n\n## 4. ") {
				t.Errorf("plan:\n%s", body)
			}

			// Closing is the last section before the journal, which stays whole.
			if _, err := b.SetSection(id, "closing", "Nothing to clean up."); err != nil {
				t.Fatal(err)
			}
			body = read(t, b, n.Path)
			journal := body[strings.LastIndex(body, "\n## "):]
			if !strings.Contains(body, "Nothing to clean up.\n\n## ") || strings.Count(journal, "\n- ") < 4 {
				t.Errorf("closing or journal:\n%s", body)
			}
		})
	}
}

func TestSetSectionRefuses(t *testing.T) {
	b := open(t, "en")
	n, err := b.NewTask(NewTaskInput{Title: "Paint the kitchen"})
	if err != nil {
		t.Fatal(err)
	}
	id := n.Header.ID
	before := read(t, b, n.Path)
	for _, c := range []struct{ section, text, want string }{
		{"budget", "x", "no section"},
		{"analysis", "   ", "empty"},
		{"analysis", "Fine.\n## Plan\nsneaky", "heading"},
		{"analysis", "# Big", "heading"},
		{"plan", "Order:\n- [ ] Buy the paint", "task_step"},
		{"analysis", "```\nopen", "code block"},
	} {
		if _, err := b.SetSection(id, c.section, c.text); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s %q: %v", c.section, c.text, err)
		}
	}
	// A heading inside a code block is not one.
	if _, err := b.SetSection(id, "analysis", "Seen in the file:\n```\n## not a heading\n```"); err != nil {
		t.Errorf("heading in code: %v", err)
	}
	if read(t, b, n.Path) == before {
		t.Error("the allowed text was not written")
	}
	if _, err := b.Close(id, "done"); err != nil {
		t.Fatal(err)
	}
	if _, err := b.SetSection(id, "analysis", "late"); err == nil {
		t.Error("a closed task took a new section")
	}
}
