package importer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/icalvete/augmented-brain/internal/memory"
)

// The captures of inbox.md go to the inbox of the space of the rest, dated, and only
// once; docs/<id>/ goes, as it is, with the note of that id.
func TestImportInboxAndDocs(t *testing.T) {
	c, _ := spaces(t)
	src := sourceDir(t, map[string]string{
		"inbox.md":                  "# Buzón\n\n## Pendiente de procesar\n- 2026-10-30 — Mirar el runner\n- sin fecha\n  - (una sublínea, no es captura)\nTexto suelto.\n",
		"task-ship.md":              "---\nname: task-ship\ndescription: Ship it\nmetadata:\n  type: project\n---\n",
		"task-school.md":            "---\nname: task-school\ndescription: School trip\nmetadata:\n  type: project\n  area: area-family\n---\n",
		"area-family.md":            "---\nname: area-family\ndescription: Family\nmetadata:\n  type: project\n---\n",
		"docs/task-ship/plan.md":    "# Plan\n",
		"docs/task-school/list.csv": "a,b\n",
		"docs/task-gone/x.md":       "orphan\n",
	})
	o := Options{Map: map[string]string{"area-family": "home", "*": "work"}, Language: "en",
		Today: time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)}
	p, err := MakePlan(src, c, o)
	if err != nil {
		t.Fatal(err)
	}
	space := map[string]string{}
	for _, it := range p.Items {
		space[it.Source] = it.Space
	}
	for src, want := range map[string]string{"inbox.md": "work", "docs/task-ship/plan.md": "work",
		"docs/task-school/list.csv": "home", "docs/task-gone/x.md": "work"} {
		if space[src] != want {
			t.Errorf("%s goes to %q, want %q", src, space[src], want)
		}
	}
	if n := p.Count()["work"]["inbox"]; n != 2 {
		t.Errorf("inbox captures counted: %d", n)
	}
	done, err := Apply(p, c, "en")
	if err != nil {
		t.Fatal(err)
	}
	if done.Inbox["work"] != 2 || done.Copies["home"] != 1 {
		t.Errorf("applied: %+v", done)
	}
	inbox, _ := os.ReadFile(filepath.Join(c.Spaces["work"].Memory, "inbox.md"))
	for _, want := range []string{"\n- 2026-10-30 — Mirar el runner\n", "\n- 2026-11-02 — sin fecha\n"} {
		if !strings.Contains(string(inbox), want) {
			t.Errorf("inbox lacks %q:\n%s", want, inbox)
		}
	}
	if strings.Contains(string(inbox), "sublínea") {
		t.Errorf("an indented line was taken for a capture:\n%s", inbox)
	}
	if data, err := os.ReadFile(filepath.Join(c.Spaces["home"].Memory, "docs", "task-school", "list.csv")); err != nil || string(data) != "a,b\n" {
		t.Errorf("docs not copied as they were: %v %q", err, data)
	}
	// The documents are not notes: the space still has no problems.
	if _, problems, err := memory.LoadSpace(c.Spaces["work"].Memory, c.SharedDir()); err != nil || len(problems) > 0 {
		t.Errorf("work after importing docs/: %v %v", problems, err)
	}
	// A second run adds no capture twice, and the files of notes imported before still
	// go with them, not to the space of the rest.
	again, _ := MakePlan(src, c, o)
	for _, it := range again.Items {
		if it.Copy {
			t.Errorf("on a second run, %s is copied again (to %q)", it.Source, it.Space)
		}
	}
	present := map[string]string{}
	for _, is := range again.Issues {
		if is.Kind == Present {
			present[is.Source] = is.Detail
		}
	}
	if present["docs/task-school/list.csv"] != "home" || present["docs/task-ship/plan.md"] != "work" {
		t.Errorf("on a second run, the files already there: %v", present)
	}
	if done, _ := Apply(again, c, "en"); done.Inbox["work"] != 0 {
		t.Errorf("captures added twice: %+v", done)
	}
}

// A note with a secret in it is imported, and the report says so: the backup would
// refuse it later.
func TestImportFlagsSecrets(t *testing.T) {
	c, _ := spaces(t)
	key := "AK" + "IA" + "QWERTYUIOPASDFGH" // built here, so this file holds no key-shaped text
	src := sourceDir(t, map[string]string{
		"task-keys.md":             "---\nname: task-keys\ndescription: Keys\nmetadata:\n  type: project\n---\nThe key: " + key + "\n",
		"docs/task-keys/env.txt":   "KEY=" + key + "\n",
		"archive/task-keys-old.md": "Old key: " + key + "\n",
	})
	p, err := MakePlan(src, c, Options{Map: map[string]string{"*": "work"}, Language: "en"})
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{"task-keys.md", "docs/task-keys/env.txt", "archive/task-keys-old.md"} {
		found := false
		for _, is := range p.Issues {
			found = found || (is.Kind == Secret && is.Source == file && is.Detail == "an AWS access key")
		}
		if !found {
			t.Errorf("secret in %s not reported: %+v", file, p.Issues)
		}
	}
	if p.Count()["work"]["task"] != 1 {
		t.Errorf("the note not imported: %v", p.Count())
	}
}
