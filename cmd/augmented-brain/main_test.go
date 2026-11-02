package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/config"
)

// `check` (and `space`, `backup`) in a folder outside every space use the default space,
// as the hooks and the MCP server do, instead of saying the folder belongs to none.
func TestCheckUsesTheDefaultSpace(t *testing.T) {
	_, cfg := home(t)
	if err := config.SetDefaultSpace(cfg, "personal"); err != nil {
		t.Fatal(err)
	}
	wd, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(wd) })
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	if code := run([]string{"check"}, nil, &out, &errOut); code != 0 {
		t.Fatalf("check outside every space: %d\n%s%s", code, out.String(), errOut.String())
	}
	if !strings.Contains(out.String(), "personal") {
		t.Errorf("check did not use the default space:\n%s", out.String())
	}
}

// `check` warns of a repository in a note's repos: that is in no work folder, without
// failing: the notes are valid, the disk changed.
func TestCheckWarnsOfMissingRepos(t *testing.T) {
	h, _ := home(t)
	work := filepath.Join(h, "augmented-brain", "trabajo")
	areas, _ := filepath.Glob(filepath.Join(work, "notas", "*", "area-*.md"))
	if len(areas) == 0 {
		t.Fatal("no starter area")
	}
	data, _ := os.ReadFile(areas[0])
	data = []byte(strings.Replace(string(data), "\ntype: area\n", "\ntype: area\nrepos: [web-vieja]\n", 1))
	if err := os.WriteFile(areas[0], data, 0o644); err != nil {
		t.Fatal(err)
	}
	wd, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(wd) })
	if err := os.Chdir(work); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	if code := run([]string{"check"}, nil, &out, &errOut); code != 0 {
		t.Fatalf("check: %d\n%s%s", code, out.String(), errOut.String())
	}
	if !strings.Contains(out.String(), "el repositorio web-vieja no está en ninguna carpeta de trabajo (~/augmented-brain/trabajo)") {
		t.Errorf("check:\n%s", out.String())
	}
}

// The help speaks the configuration's language, or the system's before there is one.
func TestUsageLanguage(t *testing.T) {
	t.Setenv("AB_CONFIG_DIR", t.TempDir())
	t.Setenv("AB_LANGUAGE", "es")
	var out, errOut bytes.Buffer
	if code := run([]string{"nope"}, nil, &out, &errOut); code != 2 || !strings.Contains(errOut.String(), "no conozco el comando") || !strings.Contains(errOut.String(), "Uso:") {
		t.Errorf("no configuration, Spanish system: %d\n%s", code, errOut.String())
	}
	home(t) // a Spanish configuration wins over an English system
	t.Setenv("AB_LANGUAGE", "en")
	if run([]string{"help"}, nil, &out, &errOut); !strings.Contains(out.String(), "Uso:") {
		t.Errorf("Spanish configuration:\n%s", out.String())
	}
}
