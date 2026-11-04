package format

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestOfAndWrite(t *testing.T) {
	dir := t.TempDir()
	if v, err := Of(dir); err != nil || v != 1 {
		t.Errorf("no version file: %d %v, want 1", v, err)
	}
	if err := Write(dir, 3); err != nil {
		t.Fatal(err)
	}
	if v, err := Of(dir); err != nil || v != 3 {
		t.Errorf("after writing 3: %d %v", v, err)
	}
	if err := os.WriteFile(filepath.Join(dir, MetaDir, File), []byte("two\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Of(dir); err == nil {
		t.Error("a version that is not a number was taken")
	}
}

// Two steps run in order, after a copy, writing the version after each; a step that
// fails leaves the version at the last one done, and running again goes on from there.
func TestEnsureNotes(t *testing.T) {
	mem, cfg := t.TempDir(), t.TempDir()
	note := filepath.Join(mem, "tasks", "task-a.md")
	if err := os.MkdirAll(filepath.Dir(note), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(note, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	fail := true
	t.Cleanup(func() { NotesMigrations = nil })
	NotesMigrations = []Migration{
		{From: 1, What: "rename", Run: func(dir string) error {
			return os.WriteFile(filepath.Join(dir, "tasks", "task-a.md"), []byte("new"), 0o644)
		}},
		{From: 2, What: "add", Run: func(dir string) error {
			if fail {
				return errors.New("disk full")
			}
			return os.WriteFile(filepath.Join(dir, "added.md"), nil, 0o644)
		}},
	}
	now := time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)

	r, err := EnsureNotes(mem, cfg, "work", now)
	if err == nil || !strings.Contains(err.Error(), "disk full") || !strings.Contains(err.Error(), r.Copy) {
		t.Fatalf("a failing step: %v", err)
	}
	if v, _ := Of(mem); v != 2 {
		t.Errorf("after a failure in step 2: version %d, want 2", v)
	}
	if data, _ := os.ReadFile(filepath.Join(r.Copy, "tasks", "task-a.md")); string(data) != "old" {
		t.Errorf("the copy holds %q, want the notes from before", data)
	}

	fail = false
	r, err = EnsureNotes(mem, cfg, "work", now.Add(time.Minute))
	if err != nil || r.From != 2 || r.To != 3 || len(r.Done) != 1 || r.Done[0] != "add" {
		t.Fatalf("again: %+v %v", r, err)
	}
	if v, _ := Of(mem); v != 3 {
		t.Errorf("version %d, want 3", v)
	}
	if r, err := EnsureNotes(mem, cfg, "work", now); err != nil || r.Migrated() {
		t.Errorf("nothing left to do: %+v %v", r, err)
	}
}

func TestNewerNotesAreLeftAlone(t *testing.T) {
	t.Setenv("AB_LANGUAGE", "en")
	mem := t.TempDir()
	if err := Write(mem, 7); err != nil {
		t.Fatal(err)
	}
	var newer *NewerError
	if err := CheckNotes(mem); !errors.As(err, &newer) || !strings.Contains(err.Error(), "augmented-brain update") {
		t.Errorf("CheckNotes: %v", err)
	}
	t.Setenv("AB_LANGUAGE", "es")
	if err := CheckNotes(mem); err == nil || !strings.Contains(err.Error(), "lo escribió un augmented brain más nuevo (formato 7; este conoce hasta el 1)") {
		t.Errorf("CheckNotes, in Spanish: %v", err)
	}
	t.Setenv("AB_LANGUAGE", "en")
	if _, err := EnsureNotes(mem, t.TempDir(), "work", time.Now()); !errors.As(err, &newer) {
		t.Errorf("EnsureNotes: %v", err)
	}
	if v, _ := Of(mem); v != 7 {
		t.Errorf("the version was changed to %d", v)
	}
}

// A folder that says no version is recorded as the current one, with no copy.
func TestEnsureRecordsTheImpliedVersion(t *testing.T) {
	mem, cfg := t.TempDir(), t.TempDir()
	r, err := EnsureNotes(mem, cfg, "work", time.Now())
	if err != nil || r.Migrated() || r.Copy != "" {
		t.Fatalf("%+v %v", r, err)
	}
	if _, err := os.Stat(filepath.Join(mem, MetaDir, File)); err != nil {
		t.Error("the version was not recorded")
	}
	if _, err := os.Stat(filepath.Join(cfg, CopiesDir)); !os.IsNotExist(err) {
		t.Error("a copy was made with nothing to migrate")
	}
}

// The copy of the configuration folder leaves out earlier copies and the state.
func TestEnsureConfigCopy(t *testing.T) {
	cfg := t.TempDir()
	for _, f := range []string{"config.toml", "profile.md", "state/work.json", "migrations/old/config.toml"} {
		p := filepath.Join(cfg, f)
		_ = os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(f), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() { ConfigMigrations = nil })
	ConfigMigrations = []Migration{{From: 1, What: "x", Run: func(string) error { return nil }}}
	got := 0
	r, err := EnsureConfig(cfg, 1, func(v int) error { got = v; return nil }, time.Now())
	if err != nil || got != 2 {
		t.Fatalf("%+v %v, wrote %d", r, err, got)
	}
	for f, want := range map[string]bool{"config.toml": true, "profile.md": true, "state": false, "migrations": false} {
		if _, err := os.Stat(filepath.Join(r.Copy, f)); (err == nil) != want {
			t.Errorf("%s in the copy: %v, want %v", f, err == nil, want)
		}
	}
}
