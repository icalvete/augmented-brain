package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The line editor finds the top of the file and a space's table, and a key by its name
// only (not a longer key that starts the same, nor a comment).
func TestLineEditor(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "# format = 9\nlanguage = \"en\"\n\n[spaces.work]\nbackup_x = 1\nbackup\t= \"b\"\n\n[spaces.\"my space\"]\nmemory = \"/m\"\n")
	f, err := openLines(dir)
	if err != nil {
		t.Fatal(err)
	}
	if start, end, ok := f.table(""); !ok || start != 0 || end != 3 {
		t.Errorf("top: %d %d %v", start, end, ok)
	}
	start, end, ok := f.table("work")
	if !ok || start != 4 || end != 7 {
		t.Errorf("work: %d %d %v", start, end, ok)
	}
	if i := f.find(start, end, "backup"); i != 5 {
		t.Errorf("backup at %d, want 5", i)
	}
	if i := f.find(0, 3, "format"); i != -1 {
		t.Errorf("a comment was taken for format: %d", i)
	}
	if start, end, ok := f.table("my space"); !ok || start != 8 || end != len(f.lines) {
		t.Errorf("quoted name: %d %d %v", start, end, ok)
	}
	if _, _, ok := f.table("nope"); ok {
		t.Error("a missing space was found")
	}
}

// A whole file always says work_folders, even empty; a space added to a file leaves
// every empty list out. Both write paths under the home folder with ~.
func TestForFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := t.TempDir()
	c := &Config{Language: "en", Dir: dir, Spaces: map[string]*Space{
		"a": {Name: "a", Memory: filepath.Join(home, "a"), GitAllowedRepos: []string{}},
	}}
	if err := c.Save(); err != nil {
		t.Fatal(err)
	}
	if err := AddSpace(dir, &Space{Name: "b", Memory: filepath.Join(home, "b")}); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, FileName))
	text := string(data)
	want := "[spaces.a]\nmemory = \"~/a\"\nwork_folders = []\n\n[spaces.b]\nmemory = \"~/b\"\n"
	if !strings.HasSuffix(text, want) {
		t.Errorf("file:\n%s\nwant it to end with:\n%s", text, want)
	}
}

// An edit whose result would not load never reaches the disk, not even for a moment:
// the guard reads the file at every command. One that loads is written once.
func TestEditChecksBeforeWriting(t *testing.T) {
	dir := t.TempDir()
	text := "language = \"en\"\n\n[spaces.work]\nmemory = \"/m\"\n"
	write(t, dir, text)
	var wrote []string
	orig := writeAtomic
	t.Cleanup(func() { writeAtomic = orig })
	writeAtomic = func(path, text string) error {
		wrote = append(wrote, text)
		return orig(path, text)
	}
	if err := SetDefaultSpace(dir, "nope"); err == nil {
		t.Fatal("a default space that does not exist was accepted")
	}
	if len(wrote) != 0 {
		t.Errorf("a text that does not load was written: %q", wrote)
	}
	if data, _ := os.ReadFile(filepath.Join(dir, FileName)); string(data) != text {
		t.Errorf("the file changed: %q", data)
	}
	if err := SetDefaultSpace(dir, "work"); err != nil {
		t.Fatal(err)
	}
	if len(wrote) != 1 || !strings.Contains(wrote[0], `default_space = "work"`) {
		t.Errorf("written: %q", wrote)
	}
	if c, err := Load(dir); err != nil || c.DefaultSpace != "work" {
		t.Errorf("loaded back: %v %v", c, err)
	}
}
