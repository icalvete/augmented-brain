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
