package importer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/format"
	"github.com/icalvete/augmented-brain/internal/memory"
	"github.com/icalvete/augmented-brain/internal/setup"
)

// spaces sets up two spaces, work and home, in a temporary home.
func spaces(t *testing.T) (*config.Config, string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg := filepath.Join(home, "cfg")
	for i, sp := range []string{"work", "home"} {
		work := filepath.Join(home, sp)
		if err := os.MkdirAll(filepath.Join(work, "code", "app"), 0o755); err != nil {
			t.Fatal(err)
		}
		o := setup.Options{Language: "en", Space: sp, WorkFolder: work, Memory: filepath.Join(work, "notes"), Today: "2026-10-15"}
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
	c, err := config.Load(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return c, home
}

// sourceDir writes a memory shaped like Claude Code's.
func sourceDir(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, text := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestImport(t *testing.T) {
	c, home := spaces(t)
	src := sourceDir(t, map[string]string{
		"MEMORY.md":        "# Index\n- [Ship the app](task-ship.md) — soon\n- [Platform](area-platform.md)\n",
		"area-platform.md": "---\nname: area-platform\ndescription: The platform\nmetadata:\n  type: project\n---\nWhat I look after.\n",
		"area-family.md":   "---\nname: area-family\ndescription: Family things\nmetadata:\n  type: project\n---\n",
		"task-ship.md": "---\nname: task-ship\ndescription: Ship the app\nmetadata:\n  type: project\n  area: area-platform\n  repos: [code/app, code/gone]\n" +
			"  related: [lesson-small-steps, task-school]\n  originSessionId: abc\n  colour: blue\n---\n## Notes\n- 2026-10-01 — Started.\n",
		"task-ready.md": "---\nname: task-ready\ndescription: Prepared by a step of the user's own\nstatus: execution\nwaiting_on: nobody\nnext: Deploy\n" +
			"watch:\n  - {date: 2026-11-08, what: Check the deploy}\nmetadata:\n  area: area-platform\n---\n",
		"task-school.md":                 "---\nname: task-school\ndescription: School trip\nmetadata:\n  type: project\n  area: area-family\n---\n",
		"lesson-small-steps.md":          "---\nname: lesson-small-steps\ndescription: Small steps win\nmetadata:\n  type: feedback\n---\nBody.\n",
		"prefer-short.md":                "---\nname: prefer-short\ndescription: Short answers\nmetadata:\n  type: feedback\n---\n",
		"user-me.md":                     "---\nname: user-me\ndescription: Who I am\nmetadata:\n  type: user\n---\n",
		"broken.md":                      "---\nname: [unclosed\n---\n",
		"archive/task-ship-journal-1.md": "old entries\n",
	})
	o := Options{Map: map[string]string{"area-family": "home", "*": "work"}, ReposBase: filepath.Join(home, "work"),
		Language: "en", Today: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)}
	p, err := MakePlan(src, c, o)
	if err != nil {
		t.Fatal(err)
	}
	space := map[string]string{}
	notes := map[string]*memory.Note{}
	for _, it := range p.Items {
		space[it.Source] = it.Space
		if it.Note != nil {
			notes[it.Note.Header.ID] = it.Note
		}
	}
	for src, want := range map[string]string{"task-ship.md": "work", "task-school.md": "home", "area-family.md": "home",
		"lesson-small-steps.md": "work", "archive/task-ship-journal-1.md": "work", "user-me.md": "", "MEMORY.md": "", "broken.md": ""} {
		if space[src] != want {
			t.Errorf("%s goes to %q, want %q", src, space[src], want)
		}
	}
	ship := notes["task-ship"].Header
	if ship.Title != "Ship the app" || ship.Type != "task" || ship.Status != "parked" || ship.WaitingOn != "me" || ship.Next == "" {
		t.Errorf("task-ship: %+v", ship)
	}
	if strings.Join(ship.Repos, ",") != "code/app,code/gone" || strings.Join(ship.Related, ",") != "lesson-small-steps" {
		t.Errorf("task-ship repos %v, related %v (task-school is in the other space)", ship.Repos, ship.Related)
	}
	if notes["lesson-small-steps"].Header.Type != "lesson" || notes["prefer-short"].Header.Type != "rule" {
		t.Error("the prefix wins over Claude Code's kind; feedback without a prefix is a rule")
	}
	ready := notes["task-ready"].Header
	if ready.Status != "execution" || ready.WaitingOn != "nobody" || ready.Next != "Deploy" || len(ready.Watch) != 1 || ready.Watch[0].Date != "2026-11-08" {
		t.Errorf("fields of augmented brain in the source are kept: %+v", ready)
	}
	issues := map[string]bool{}
	for _, is := range p.Issues {
		issues[is.Kind+" "+is.Source+" "+is.Detail] = true
	}
	for _, want := range []string{
		"cross-link task-ship.md task-school", "dropped-field task-ship.md metadata.colour", "to-profile user-me.md ",
		"not-a-note MEMORY.md ", "task-defaults task-ship.md ",
	} {
		if !issues[want] {
			t.Errorf("no issue %q in %v", want, p.Issues)
		}
	}
	if issues["dropped-field task-ship.md metadata.originSessionId"] {
		t.Error("Claude Code's own bookkeeping is dropped without a word")
	}
	found := false
	for k := range issues {
		found = found || strings.HasPrefix(k, "unreadable broken.md")
	}
	if !found {
		t.Errorf("broken.md is not reported: %v", p.Issues)
	}

	// Apply writes valid notes, copies the archive and leaves the source alone.
	before, _ := os.ReadFile(filepath.Join(src, "task-ship.md"))
	done, err := Apply(p, c, "en")
	if err != nil {
		t.Fatal(err)
	}
	if done.Notes["work"] != 5 || done.Notes["home"] != 2 || done.Copies["work"] != 1 {
		t.Errorf("applied: %+v", done)
	}
	for _, sp := range []string{"work", "home"} {
		m, problems, err := memory.Load(c.Spaces[sp].Memory)
		if err != nil || len(problems) > 0 {
			t.Errorf("space %s after the import: %v %v", sp, problems, err)
		}
		if sp == "work" && m.Notes["task-ship"] == nil {
			t.Error("task-ship was not written")
		}
	}
	if _, err := os.Stat(filepath.Join(c.Spaces["work"].Memory, "archive", "task-ship-journal-1.md")); err != nil {
		t.Error("the archive file was not copied")
	}
	if after, _ := os.ReadFile(filepath.Join(src, "task-ship.md")); string(after) != string(before) {
		t.Error("the source was changed")
	}

	// A second run finds every id taken and writes nothing.
	again, err := MakePlan(src, c, o)
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range again.Items {
		if it.Note != nil {
			t.Errorf("%s would be written again", it.Source)
		}
	}
}

func TestImportNeedsAMapWithSeveralSpaces(t *testing.T) {
	c, _ := spaces(t)
	if _, err := MakePlan(sourceDir(t, nil), c, Options{}); err == nil || !strings.Contains(err.Error(), "2 spaces") {
		t.Errorf("no map with two spaces: %v", err)
	}
	if _, err := MakePlan(sourceDir(t, nil), c, Options{Map: map[string]string{"*": "nowhere"}}); err == nil {
		t.Error("a map to a space that does not exist was taken")
	}
}

// Notes written by a newer program are never written into.
func TestImportRefusesNewerNotes(t *testing.T) {
	c, _ := spaces(t)
	if err := format.Write(c.Spaces["work"].Memory, format.Notes()+1); err != nil {
		t.Fatal(err)
	}
	src := sourceDir(t, map[string]string{"task-ship.md": "---\nname: task-ship\ndescription: Ship it\nmetadata:\n  type: project\n---\n"})
	p, err := MakePlan(src, c, Options{Map: map[string]string{"*": "work"}, Language: "en"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(p, c, "en"); err == nil {
		t.Fatal("notes of a newer format were written into")
	}
	if _, err := os.Stat(filepath.Join(c.Spaces["work"].Memory, "tasks", "task-ship.md")); err == nil {
		t.Error("task-ship was written")
	}
}
