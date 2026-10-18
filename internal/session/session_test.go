package session

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/icalvete/augmented-brain/internal/brain"
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

// Found in the review before 0.1: a huge profile, hundreds of space rules and emojis
// must never push the session context over the budget.
func TestContextNeverOverBudget(t *testing.T) {
	c, s := space(t, "en")
	profile := "---\nid: profile\ntype: profile\ntitle: Profile\n---\n" + strings.Repeat("I like short answers 🙂. ", 600) + "\n"
	if err := os.WriteFile(filepath.Join(c.Dir, "profile.md"), []byte(profile), 0o644); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 300; i++ {
		rule := fmt.Sprintf("---\nid: rule-%03d\ntype: rule\ntitle: A rule with a fairly long title number %03d\n---\nbody\n", i, i)
		if err := os.WriteFile(filepath.Join(s.Memory, "rules", fmt.Sprintf("rule-%03d.md", i)), []byte(rule), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	text := Context(c, s)
	if n := size(text); n > Budget {
		t.Fatalf("context is %d units, over %d", n, Budget)
	}
	if !strings.Contains(text, "The profile was cut to fit") || !strings.Contains(text, "### How we work") {
		t.Errorf("the profile note or the common rules are missing")
	}
	if utf8.RuneCountInString(text) > size(text) {
		t.Error("size must never count less than the characters")
	}
}

// While the profile is blank the session asks the assistant to meet the user first; once
// it is filled in, it does not.
func TestFirstSession(t *testing.T) {
	c, s := space(t, "es")
	if text := Context(c, s); !strings.Contains(text, "## First session: get to know the user") || !strings.Contains(text, "the language the user writes in (Spanish if you cannot tell yet)") {
		t.Errorf("a blank profile must start with the first-session note:\n%s", text)
	}
	profile := filepath.Join(c.Dir, "profile.md")
	data, err := os.ReadFile(profile)
	if err != nil {
		t.Fatal(err)
	}
	filled := strings.Replace(string(data), "- Nombre:", "- Nombre: Ana", 1)
	if err := os.WriteFile(profile, []byte(filled), 0o644); err != nil {
		t.Fatal(err)
	}
	if text := Context(c, s); strings.Contains(text, "First session") {
		t.Error("a filled profile must not ask to meet the user again")
	}
}

// Outside every space the assistant must tell the user, with the folders to open.
func TestNoSpace(t *testing.T) {
	c, s := space(t, "es")
	text := NoSpace(c, "/tmp/scratch")
	for _, want := range []string{"/tmp/scratch", "outside the user's spaces", "Spanish if you cannot tell", "work: " + s.WorkFolders[0], "Choose…"} {
		if !strings.Contains(text, want) {
			t.Errorf("no-space text lacks %q:\n%s", want, text)
		}
	}
}

func TestContextDueAndReview(t *testing.T) {
	c, s := space(t, "en")
	Now = func() time.Time { return time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC) }
	defer func() { Now = time.Now }()
	b, err := brain.Open(c, s)
	if err != nil {
		t.Fatal(err)
	}
	b.Now = func() time.Time { return time.Date(2026, 10, 10, 9, 0, 0, 0, time.UTC) }
	if _, err := b.SaveProfile(brain.ProfileInput{Name: "Dana", About: "Runs a bakery's IT."}); err != nil {
		t.Fatal(err)
	}
	task, _ := b.NewTask(brain.NewTaskInput{Title: "Certificates"})
	if _, err := b.AddWatch(task.Header.ID, "2026-10-18", "renewed?"); err != nil {
		t.Fatal(err)
	}
	if err := b.Capture("call the insurance", "2026-10-21"); err != nil {
		t.Fatal(err)
	}

	text := Context(c, s)
	for _, want := range []string{
		"## Due",
		"- 2026-10-18 (2 days overdue): renewed? [" + task.Header.ID + "]",
		"- 2026-10-21 (tomorrow): call the insurance [inbox]",
		"The weekly review is due: 10 days since the last one, with 1 captures in the inbox",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("context lacks %q:\n%s", want, text)
		}
	}

	// A blank profile means the first session: the interview, nothing else.
	c2, s2 := space(t, "en")
	if text := Context(c2, s2); strings.Contains(text, "## Due") || strings.Contains(text, "weekly review is due") {
		t.Errorf("the first session must only run the interview:\n%s", text)
	}
}
