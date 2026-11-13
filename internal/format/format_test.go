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

	r, err := EnsureNotes(mem, cfg, "work", "", now)
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
	r, err = EnsureNotes(mem, cfg, "work", "", now.Add(time.Minute))
	if err != nil || r.From != 2 || r.To != 3 || len(r.Done) != 1 || r.Done[0] != "add" {
		t.Fatalf("again: %+v %v", r, err)
	}
	if v, _ := Of(mem); v != 3 {
		t.Errorf("version %d, want 3", v)
	}
	if r, err := EnsureNotes(mem, cfg, "work", "", now); err != nil || r.Migrated() {
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
	if err := CheckNotes(mem, ""); !errors.As(err, &newer) || !strings.Contains(err.Error(), "augmented-brain update") {
		t.Errorf("CheckNotes: %v", err)
	}
	t.Setenv("AB_LANGUAGE", "es")
	if err := CheckNotes(mem, ""); err == nil || !strings.Contains(err.Error(), "lo escribió un augmented brain más nuevo (formato 7; este conoce hasta el 1)") {
		t.Errorf("CheckNotes, in Spanish: %v", err)
	}
	t.Setenv("AB_LANGUAGE", "en")
	if _, err := EnsureNotes(mem, t.TempDir(), "work", "", time.Now()); !errors.As(err, &newer) {
		t.Errorf("EnsureNotes: %v", err)
	}
	// The user's language, when the caller knows it, wins over the system's.
	if err := CheckNotes(mem, "es"); err == nil || !strings.Contains(err.Error(), "lo escribió un augmented brain más nuevo") {
		t.Errorf("CheckNotes in the user's Spanish: %v", err)
	}
	if _, err := EnsureNotes(mem, t.TempDir(), "work", "es", time.Now()); err == nil || !strings.Contains(err.Error(), "lo escribió") {
		t.Errorf("EnsureNotes in the user's Spanish: %v", err)
	}
	if v, _ := Of(mem); v != 7 {
		t.Errorf("the version was changed to %d", v)
	}
}

// What a migration that fails says is in the user's language, and keeps the error.
func TestMigrationErrorsInTheUsersLanguage(t *testing.T) {
	boom := errors.New("boom")
	old := NotesMigrations
	t.Cleanup(func() { NotesMigrations = old })
	NotesMigrations = []Migration{{From: 1, What: "a step", Run: func(string) error { return boom }}}
	mem, cfg := t.TempDir(), t.TempDir()
	_, err := EnsureNotes(mem, cfg, "work", "es", time.Now())
	if err == nil || !strings.Contains(err.Error(), "al pasar "+mem+" del formato 1 al 2 (a step): boom; hay una copia de antes en") || !errors.Is(err, boom) {
		t.Errorf("failed step, in Spanish: %v", err)
	}
	if err := Write(mem, 1); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mem, MetaDir, File), []byte("x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := EnsureNotes(mem, cfg, "work", "en", time.Now()); err == nil || !strings.Contains(err.Error(), `"x" is not a format version`) {
		t.Errorf("unreadable version, in English: %v", err)
	}
}

// A folder that says no version is recorded as the current one, with no copy.
func TestEnsureRecordsTheImpliedVersion(t *testing.T) {
	mem, cfg := t.TempDir(), t.TempDir()
	r, err := EnsureNotes(mem, cfg, "work", "", time.Now())
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
	r, err := EnsureConfig(cfg, 1, func(v int) error { got = v; return nil }, "", time.Now())
	if err != nil || got != 2 {
		t.Fatalf("%+v %v, wrote %d", r, err, got)
	}
	for f, want := range map[string]bool{"config.toml": true, "profile.md": true, "state": false, "migrations": false} {
		if _, err := os.Stat(filepath.Join(r.Copy, f)); (err == nil) != want {
			t.Errorf("%s in the copy: %v, want %v", f, err == nil, want)
		}
	}
}

// A migration that stops half way, run again and again (twice in the same second),
// keeps the one copy made before it started: the notes as they were, links included.
// No other whole copy is made, and none holds the notes half migrated.
func TestInterruptedMigrationKeepsOneCopy(t *testing.T) {
	mem, cfg := t.TempDir(), t.TempDir()
	note := filepath.Join(mem, "tasks", "task-a.md")
	if err := os.MkdirAll(filepath.Dir(note), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(note, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}
	links := os.Symlink("task-a.md", filepath.Join(mem, "tasks", "link.md")) == nil
	t.Cleanup(func() { NotesMigrations = nil })
	NotesMigrations = []Migration{{From: 1, What: "half", Run: func(dir string) error {
		if err := os.WriteFile(filepath.Join(dir, "tasks", "task-a.md"), []byte("MIGRATED HALF"), 0o644); err != nil {
			return err
		}
		return errors.New("cut short")
	}}}
	now := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	var copies []string
	for _, at := range []time.Time{now, now.Add(time.Minute), now.Add(time.Minute)} {
		r, err := EnsureNotes(mem, cfg, "work", "", at)
		if err == nil || !strings.Contains(err.Error(), "cut short") {
			t.Fatalf("a failing step: %v", err)
		}
		copies = append(copies, r.Copy)
	}
	entries, _ := os.ReadDir(filepath.Join(cfg, CopiesDir))
	if len(entries) != 1 {
		t.Fatalf("%d copies, want 1: %v", len(entries), entries)
	}
	for _, c := range copies[1:] {
		if c != copies[0] {
			t.Errorf("a run again pointed at %s, not the first copy %s", c, copies[0])
		}
	}
	if data, _ := os.ReadFile(filepath.Join(copies[0], "tasks", "task-a.md")); string(data) != "original" {
		t.Errorf("the copy holds %q, want the notes from before", data)
	}
	if _, err := os.Stat(filepath.Join(copies[0], CompleteFile)); err != nil {
		t.Errorf("the copy is not marked finished: %v", err)
	}
	if links {
		if to, err := os.Readlink(filepath.Join(copies[0], "tasks", "link.md")); err != nil || to != "task-a.md" {
			t.Errorf("the link in the copy: %q %v", to, err)
		}
	}
}

// A copy cut short (no mark that it was finished) is never written into, nor taken as
// the copy from before: the next run makes its own, in a folder of its own, even when
// the time gives it the same name.
func TestUnfinishedCopyIsLeftAlone(t *testing.T) {
	mem, cfg := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(mem, "note.md"), []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	unfinished := filepath.Join(cfg, CopiesDir, "notes-work-1-"+now.Format("20060102-150405"))
	if err := os.MkdirAll(unfinished, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(unfinished, "half.md"), []byte("half"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { NotesMigrations = nil })
	NotesMigrations = []Migration{{From: 1, What: "x", Run: func(string) error { return nil }}}
	r, err := EnsureNotes(mem, cfg, "work", "", now)
	if err != nil {
		t.Fatal(err)
	}
	if r.Copy != unfinished+"-2" {
		t.Errorf("the copy went to %s, want %s-2", r.Copy, unfinished)
	}
	if _, err := os.Stat(filepath.Join(unfinished, "note.md")); err == nil {
		t.Error("the copy was made over the unfinished one")
	}
	if data, _ := os.ReadFile(filepath.Join(r.Copy, "note.md")); string(data) != "original" {
		t.Errorf("the copy holds %q", data)
	}
}
