package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// AGENTS.md is loaded into every agent's session: it stays short, and its map names
// every package, so a new one cannot be forgotten. The detail lives in dev/.
func TestAgentsHandbook(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(docRoot, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	if n := len(strings.Fields(text)); n > 2000 {
		t.Errorf("AGENTS.md has %d words; keep it under 2000 and move the detail to dev/", n)
	}
	dirs, err := os.ReadDir(filepath.Join(docRoot, "internal"))
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range dirs {
		if d.IsDir() && !strings.Contains(text, "`internal/"+d.Name()+"`") {
			t.Errorf("AGENTS.md's map does not name internal/%s", d.Name())
		}
	}
	// Its links, and those of dev/, lead to files that exist.
	link := regexp.MustCompile(`\]\(([^)#]+)(#[^)]*)?\)`)
	pages, _ := filepath.Glob(filepath.Join(docRoot, "dev", "*.md"))
	pages = append(pages, filepath.Join(docRoot, "AGENTS.md"))
	for _, p := range pages {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range link.FindAllStringSubmatch(string(b), -1) {
			if strings.Contains(m[1], "://") {
				continue
			}
			if _, err := os.Stat(filepath.Join(filepath.Dir(p), m[1])); err != nil {
				t.Errorf("%s links to %s, which does not exist", p, m[1])
			}
		}
	}
}
