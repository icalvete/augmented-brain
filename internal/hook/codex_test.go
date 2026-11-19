package hook

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/session"
)

// The Codex start gives the space's text in parts that each fit Codex's cut, in the
// work folder and in the notes folder, without what only Claude Code has, and with
// the note on what differs there.
func TestCodexStart(t *testing.T) {
	cfgDir, work := setupSpace(t)
	c, err := config.Load(cfgDir)
	if err != nil {
		t.Fatal(err)
	}
	notes := c.Spaces["work"].Memory
	part := func(dir string, n int) string {
		o := run(t, func(in *bytes.Reader, out *bytes.Buffer) error { return CodexStartPart(in, out, cfgDir, n) },
			map[string]any{"cwd": dir, "session_id": "codex-" + filepath.Base(dir), "source": "startup"})
		text, _ := o["additionalContext"].(string)
		return text
	}
	for _, dir := range []string{filepath.Join(work, "repo"), filepath.Join(notes, "tasks")} {
		all := ""
		for n := 1; n <= session.Parts; n++ {
			p := part(dir, n)
			if len(p) > session.CodexBudget {
				t.Errorf("%s, part %d: %d bytes", dir, n, len(p))
			}
			all += p
		}
		for _, want := range []string{"space \"work\"", "## In Codex", "command guard runs here only if", "task_section", "## Rules", "# Memory index"} {
			if !strings.Contains(all, want) {
				t.Errorf("%s: the Codex start lacks %q", dir, want)
			}
		}
		for _, claudeOnly := range []string{"start_session", "Remote Control", "Your preferred assistant"} {
			if strings.Contains(all, claudeOnly) {
				t.Errorf("%s: the Codex start says %q", dir, claudeOnly)
			}
		}
	}
	outside := filepath.Dir(work)
	if p := part(outside, 1); !strings.Contains(p, "belongs to no space") {
		t.Errorf("outside, part 1: %q", p)
	}
	if p := part(outside, 2); p != "" {
		t.Errorf("outside, part 2: %q", p)
	}
}
