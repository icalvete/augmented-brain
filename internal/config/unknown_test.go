package config

import (
	"os"
	"path/filepath"
	"testing"
)

// Settings of guard.toml and session.toml this program does not know are listed, file
// by file, with the known name a misspelling is close to; the files still load.
func TestUnknownSettings(t *testing.T) {
	dir := t.TempDir()
	if got := UnknownSettings(dir); len(got) != 0 {
		t.Fatalf("no files: %+v", got)
	}
	write := func(name, text string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(GuardFile, "untouchable_repo = [\"~/a\"]\nuntouchable_repos = []\n[later]\nx = 1\n")
	write(SessionFile, "memory-trust = \"ask\"\npreferred_model = \"opus\"\nwhatever.deep = 2\n")
	want := []Unknown{
		{GuardFile, "untouchable_repo", "untouchable_repos"},
		{GuardFile, "later", ""},
		{SessionFile, "memory-trust", "memory_trust"},
		{SessionFile, "whatever", ""},
	}
	got := UnknownSettings(dir)
	if len(got) != len(want) {
		t.Fatalf("got %+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("%d: got %+v, want %+v", i, got[i], want[i])
		}
	}
	if _, err := LoadGuard(dir); err != nil {
		t.Errorf("guard.toml with an unknown setting is refused: %v", err)
	}
	if s, err := LoadSession(dir); err != nil || s.PreferredModel != "opus" {
		t.Errorf("session.toml with an unknown setting: %+v, %v", s, err)
	}
	by := UnknownByFile("en", got)
	if len(by) != 2 || by[0] != [2]string{GuardFile, "untouchable_repo (did you mean untouchable_repos?), later"} ||
		by[1][0] != SessionFile {
		t.Errorf("by file: %q", by)
	}
	if es := UnknownByFile("es", got[:1]); es[0][1] != "untouchable_repo (¿querías decir untouchable_repos?)" {
		t.Errorf("Spanish: %q", es)
	}
	// A file that cannot be read says nothing here: LoadGuard reports it.
	write(GuardFile, "untouchable_repos = [")
	if got := UnknownSettings(dir); len(got) != 2 || got[0].File != SessionFile {
		t.Errorf("broken guard.toml: %+v", got)
	}
}

func TestNearest(t *testing.T) {
	known := []string{"preferred_model", "memory_trust"}
	for key, want := range map[string]string{
		"preferred_modle": "preferred_model",
		"Memory_Trust":    "memory_trust",
		"memorytrust":     "memory_trust",
		"model":           "",
		"trust":           "",
	} {
		if got := nearest(key, known); got != want {
			t.Errorf("%s: %q, want %q", key, got, want)
		}
	}
}
