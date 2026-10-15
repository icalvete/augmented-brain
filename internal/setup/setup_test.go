package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/memory"
)

func TestInitAndAddSpace(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	for _, d := range []string{"work", "personal"} {
		if err := os.MkdirAll(filepath.Join(home, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	cfgDir := filepath.Join(home, ".config", "augmented-brain")

	res, err := Init(cfgDir, Options{Language: "es", Space: "work", WorkFolder: "~/work",
		Memory: "~/augmented-brain/work", Today: "2026-10-15"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Created) == 0 {
		t.Fatal("nothing created")
	}

	c, err := config.Load(cfgDir)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(filepath.Join(cfgDir, config.FileName))
	if !strings.Contains(string(raw), `"~/augmented-brain/work"`) || !strings.Contains(string(raw), `language = "es"`) {
		t.Errorf("config file:\n%s", raw)
	}
	if s := c.SpaceFor(filepath.Join(home, "work", "repo")); s == nil || s.Name != "work" {
		t.Errorf("work folder does not select the space: %v", s)
	}
	profile, _ := os.ReadFile(filepath.Join(cfgDir, "profile.md"))
	if !strings.Contains(string(profile), "title: Perfil") || !strings.Contains(string(profile), "created: 2026-10-15") {
		t.Errorf("profile:\n%s", profile)
	}

	mem := filepath.Join(home, "augmented-brain", "work")
	m, problems, err := memory.Load(mem)
	if err != nil || len(problems) > 0 {
		t.Fatal(err, problems)
	}
	if _, ok := m.Notes["area-general"]; !ok {
		t.Errorf("starter area missing: %v", m.IDs())
	}
	for _, f := range []string{"inbox.md", "MEMORY.md", "tasks", "knowledge", "rules", "reviews", "archive"} {
		if _, err := os.Stat(filepath.Join(mem, f)); err != nil {
			t.Errorf("%s missing", f)
		}
	}
	index, _ := os.ReadFile(filepath.Join(mem, "MEMORY.md"))
	if !strings.Contains(string(index), "## Áreas") {
		t.Errorf("index not in Spanish:\n%s", index)
	}

	// A second init must refuse, and touch nothing.
	if _, err := Init(cfgDir, Options{Space: "x", WorkFolder: "~/personal", Memory: "~/m"}); err == nil {
		t.Error("second init did not refuse")
	}

	// Adding a space keeps the first one.
	if _, err := AddSpace(cfgDir, Options{Space: "personal", WorkFolder: "~/personal",
		Memory: "~/augmented-brain/personal"}); err != nil {
		t.Fatal(err)
	}
	c, _ = config.Load(cfgDir)
	if got := c.SpaceNames(); len(got) != 2 {
		t.Errorf("spaces = %v", got)
	}

	// An existing file is never overwritten.
	area := filepath.Join(mem, "areas", "area-general.md")
	if err := os.WriteFile(area, []byte("---\nid: area-general\ntype: area\ntitle: Mine\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := createSpace("es", mem, "2026-10-16"); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(area); !strings.Contains(string(b), "title: Mine") {
		t.Error("an existing note was overwritten")
	}
}

func TestAddSpaceErrors(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, "work", "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfgDir := filepath.Join(home, "cfg")
	if _, err := Init(cfgDir, Options{Space: "work", WorkFolder: "~/work", Memory: "~/m/work"}); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		o    Options
		want string
	}{
		{Options{Space: "Work!", WorkFolder: "~/work", Memory: "~/m/x"}, "lowercase"},
		{Options{Space: "work", WorkFolder: "~/work", Memory: "~/m/x"}, "already a space"},
		{Options{Space: "x", WorkFolder: "~/m/work/MEMORY.md", Memory: "~/m/x"}, "is a file"},
		{Options{Space: "x", WorkFolder: "~/work/sub", Memory: "~/m/x"}, "already belongs"},
		{Options{Space: "x", WorkFolder: "~", Memory: "~/m/work"}, "memory folder"},
	}
	for _, c := range cases {
		_, err := AddSpace(cfgDir, c.o)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%+v: error %v, want it to mention %q", c.o, err, c.want)
		}
	}
}
