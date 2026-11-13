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
	// The second space chooses the default one, whose backup carries the profile.
	if c.DefaultSpace != "personal" {
		t.Errorf("default space = %q, want personal", c.DefaultSpace)
	}

	// An existing file is never overwritten.
	area := filepath.Join(mem, "areas", "area-general.md")
	if err := os.WriteFile(area, []byte("---\nid: area-general\ntype: area\ntitle: Mine\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := createSpace("es", mem, filepath.Join(home, "cfg", "shared"), "2026-10-16"); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(area); !strings.Contains(string(b), "title: Mine") {
		t.Error("an existing note was overwritten")
	}
}

// Without a space named personal, the space that was there first stays the default, and
// a third space changes nothing.
func TestAddSpaceKeepsTheFirstAsDefault(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfgDir := filepath.Join(home, "cfg")
	if _, err := Init(cfgDir, Options{Space: "home", WorkFolder: "~/home", Memory: "~/m/home"}); err != nil {
		t.Fatal(err)
	}
	for _, step := range []struct{ space, chosen string }{{"job", "home"}, {"club", ""}} {
		res, err := AddSpace(cfgDir, Options{Space: step.space, WorkFolder: "~/" + step.space, Memory: "~/m/" + step.space})
		if err != nil {
			t.Fatal(err)
		}
		if res.DefaultSpace != step.chosen {
			t.Errorf("adding %s: chose %q, want %q", step.space, res.DefaultSpace, step.chosen)
		}
	}
	if c, _ := config.Load(cfgDir); c.DefaultSpace != "home" {
		t.Errorf("default space = %q, want home", c.DefaultSpace)
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
		{Options{Space: "shared", WorkFolder: "~/shared", Memory: "~/m/shared"}, "shared by every space"},
		{Options{Space: "x", WorkFolder: "~/m/work/MEMORY.md", Memory: "~/m/x"}, "is a file"},
		{Options{Space: "x", WorkFolder: "~/work/sub", Memory: "~/m/x"}, "already belongs"},
		{Options{Space: "x", WorkFolder: "~", Memory: "~/m/work"}, "memory folder"},
		// Notes around the work: uninstall --all would have taken the work with them.
		{Options{Space: "x", WorkFolder: "~/proj", Memory: "~/proj"}, "holds the work folder"},
		{Options{Space: "x", WorkFolder: "~/code/proj", Memory: "~/code"}, "holds the work folder"},
		{Options{Space: "x", WorkFolder: "~/proj", Memory: "~"}, "holds the work folder"},
		{Options{Space: "x", WorkFolder: "~/proj", Memory: "~/work"}, "a work folder of space"},
	}
	for _, c := range cases {
		_, err := AddSpace(cfgDir, c.o)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%+v: error %v, want it to mention %q", c.o, err, c.want)
		}
	}
}

// Several spaces at once: config.toml is written only at the end. A second space that
// fails (as a Ctrl+C would stop it) leaves no configuration, so running it again starts
// over; the first time, init wrote the first space and the installer never ran it again.
func TestInitSpacesAllOrNothing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfgDir := filepath.Join(home, "cfg")
	work := Options{Language: "en", Space: "work", WorkFolder: "~/ab/work", Memory: "~/ab/work/notes"}
	personal := Options{Language: "en", Space: "personal", WorkFolder: "~/ab/personal", Memory: "~/ab/personal/notes"}

	// The second space cannot be made: its work folder is a file.
	if err := os.MkdirAll(filepath.Join(home, "ab"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "ab", "personal"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := InitSpaces(cfgDir, []Options{work, personal}, "personal"); err == nil {
		t.Fatal("a space that failed was not reported")
	}
	if _, err := os.Stat(filepath.Join(cfgDir, config.FileName)); !os.IsNotExist(err) {
		t.Fatalf("a configuration with a space missing was left: %v", err)
	}

	// Fixed, it runs again from the start, over the folders already made.
	if err := os.Remove(filepath.Join(home, "ab", "personal")); err != nil {
		t.Fatal(err)
	}
	res, err := InitSpaces(cfgDir, []Options{work, personal}, "personal")
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 2 {
		t.Errorf("%d results", len(res))
	}
	c, err := config.Load(cfgDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Spaces) != 2 || c.DefaultSpace != "personal" {
		t.Errorf("spaces %v, default %q", c.SpaceNames(), c.DefaultSpace)
	}

	// With a configuration there, it changes nothing.
	if _, err := InitSpaces(cfgDir, []Options{work}, ""); err == nil || !strings.Contains(err.Error(), "already a configuration") {
		t.Errorf("over a configuration: %v", err)
	}
	// A default space that is not one of them is refused.
	if _, err := InitSpaces(filepath.Join(home, "cfg2"), []Options{{Language: "en", Space: "x", WorkFolder: "~/x", Memory: "~/x/notes"}}, "personal"); err == nil {
		t.Error("an unknown default space was taken")
	}
	if _, err := os.Stat(filepath.Join(home, "cfg2", config.FileName)); !os.IsNotExist(err) {
		t.Errorf("a refused configuration was written: %v", err)
	}
}
