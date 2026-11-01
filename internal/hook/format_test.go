package hook

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/format"
)

// A new space is written in this program's format; when a newer program adds a step,
// the next session runs it and tells the assistant, once.
func TestSessionStartMigrates(t *testing.T) {
	cfgDir, work := setupSpace(t)
	c, _ := config.Load(cfgDir)
	mem := c.Spaces["work"].Memory
	if v, err := format.Of(mem); err != nil || v != 1 {
		t.Fatalf("a new space: format %d %v", v, err)
	}
	if data, _ := os.ReadFile(filepath.Join(mem, format.MetaDir, format.File)); string(data) != "1\n" {
		t.Errorf("version file: %q", data)
	}
	if data, _ := os.ReadFile(filepath.Join(cfgDir, config.FileName)); strings.Contains("\n"+string(data), "\nformat") {
		t.Errorf("format 1 must not be written to config.toml: older releases refuse it:\n%s", data)
	}

	t.Cleanup(func() { format.NotesMigrations = nil })
	format.NotesMigrations = []format.Migration{{From: 1, What: "split the inbox", Run: func(string) error { return nil }}}
	ss := func(in *bytes.Reader, out *bytes.Buffer) error { return SessionStart(in, out, cfgDir) }
	text := run(t, ss, map[string]any{"cwd": work})["additionalContext"].(string)
	if !strings.Contains(text, "moved the notes of space work from format 1 to 2 (split the inbox)") {
		t.Errorf("no migration note:\n%s", text)
	}
	if v, _ := format.Of(mem); v != 2 {
		t.Errorf("format %d, want 2", v)
	}
	if text := run(t, ss, map[string]any{"cwd": work})["additionalContext"].(string); strings.Contains(text, "from format") {
		t.Error("the note came back in the next session")
	}

	format.NotesMigrations = nil // this program is now older than the notes
	text = run(t, ss, map[string]any{"cwd": work})["additionalContext"].(string)
	if !strings.Contains(text, "written by a newer augmented brain") {
		t.Errorf("newer notes:\n%s", text)
	}
}
