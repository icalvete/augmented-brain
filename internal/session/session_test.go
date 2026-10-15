package session

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/setup"
)

func space(t *testing.T, lang string) (*config.Config, *config.Space) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	work := filepath.Join(home, "work")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	cfgDir := filepath.Join(home, "cfg")
	if _, err := setup.Init(cfgDir, setup.Options{Language: lang, Space: "work", WorkFolder: work, Memory: "~/brain"}); err != nil {
		t.Fatal(err)
	}
	c, err := config.Load(cfgDir)
	if err != nil {
		t.Fatal(err)
	}
	return c, c.Spaces["work"]
}

func TestContext(t *testing.T) {
	c, s := space(t, "es")
	rule := "---\nid: rule-db\ntype: rule\ntitle: Las bases de datos no se tocan\n---\nNunca reiniciar db01 sin permiso.\n"
	if err := os.WriteFile(filepath.Join(s.Memory, "rules", "rule-db.md"), []byte(rule), 0o644); err != nil {
		t.Fatal(err)
	}
	text := Context(c, s)
	for _, want := range []string{
		`space "work"`,
		"## Who the user is", "Idioma: español.", // the profile body, without its header
		"### Cómo trabajamos", "### Seguridad", // common rules in the user's language
		"### Las bases de datos no se tocan\nNunca reiniciar db01 sin permiso.",
		"# Índice de la memoria", "[General](areas/area-general.md)",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("context lacks %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "type: profile") {
		t.Error("a header leaked into the context")
	}
}

func TestContextBudget(t *testing.T) {
	c, s := space(t, "en")
	long := strings.Repeat("Explain every step before doing it. ", 60)
	for i := 0; i < 8; i++ {
		rule := fmt.Sprintf("---\nid: rule-%d\ntype: rule\ntitle: Rule %d\n---\n%s\n", i, i, long)
		if err := os.WriteFile(filepath.Join(s.Memory, "rules", fmt.Sprintf("rule-%d.md", i)), []byte(rule), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	text := Context(c, s)
	if n := utf8.RuneCountInString(text); n > Budget {
		t.Fatalf("context is %d characters, over the %d budget", n, Budget)
	}
	if !strings.Contains(text, "read each file before acting") || !strings.Contains(text, "rule-3.md") {
		t.Errorf("long rules are not listed by path:\n%s", text)
	}
	if strings.Contains(text, long) {
		t.Error("a long rule body is still inlined")
	}

	// A huge index is cut, with a note saying where the whole one is.
	for i := 0; i < 400; i++ {
		task := fmt.Sprintf("---\nid: task-%03d\ntype: task\ntitle: Task number %03d with a long title\nstatus: plan\nwaiting_on: me\nnext: do the next thing for task %03d\n---\n", i, i, i)
		if err := os.WriteFile(filepath.Join(s.Memory, "tasks", fmt.Sprintf("task-%03d.md", i)), []byte(task), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	text = Context(c, s)
	if n := utf8.RuneCountInString(text); n > Budget || !strings.Contains(text, "The index was cut") {
		t.Errorf("big index: %d characters, cut note present: %v", n, strings.Contains(text, "The index was cut"))
	}
}
