package hook

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/state"
)

// transcript writes a Claude Code transcript whose answers came from these models.
func transcript(t *testing.T, models ...string) string {
	t.Helper()
	var b strings.Builder
	b.WriteString(`{"type":"user","message":{"role":"user","content":"hi"}}` + "\n")
	for _, m := range models {
		line, _ := json.Marshal(map[string]any{"type": "assistant", "message": map[string]any{"model": m, "content": "ok"}})
		b.Write(line)
		b.WriteString("\n")
	}
	p := filepath.Join(t.TempDir(), "s.jsonl")
	if err := os.WriteFile(p, []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// With a preferred model, Stop warns once per session and model when the last answer
// came from another one; without one, it never looks.
func TestModelWarningAfterAnswer(t *testing.T) {
	cfgDir, work := setupSpace(t)
	clock(t, time.Date(2026, 11, 6, 10, 0, 0, 0, time.Local))
	stop := func(path string) string {
		o := run(t, func(in *bytes.Reader, out *bytes.Buffer) error { return Stop(in, out, cfgDir) },
			map[string]any{"cwd": work, "session_id": "s1", "transcript_path": path})
		if o == nil {
			return ""
		}
		return o["additionalContext"].(string)
	}
	sonnet := transcript(t, "claude-opus-5-5", "claude-sonnet-5-5", "<synthetic>")
	if got := stop(sonnet); got != "" {
		t.Errorf("no preference set, yet: %s", got)
	}
	if err := config.SaveSession(cfgDir, config.Session{PreferredModel: "opus"}); err != nil {
		t.Fatal(err)
	}
	got := stop(sonnet)
	if !strings.Contains(got, "claude-sonnet-5-5") || !strings.Contains(got, "/model") {
		t.Errorf("warning: %q", got)
	}
	if again := stop(sonnet); again != "" {
		t.Errorf("warned twice: %s", again)
	}
	if got := stop(transcript(t, "claude-opus-5-5")); got != "" {
		t.Errorf("opus is fine: %s", got)
	}
	if got := stop(transcript(t, "claude-haiku-5-5")); !strings.Contains(got, "haiku") {
		t.Errorf("a change to another model is warned: %q", got)
	}
}

// At the start, a model Claude Code names is checked at once, and what it said (or that
// it said nothing) is recorded.
func TestModelWarningAtStart(t *testing.T) {
	cfgDir, work := setupSpace(t)
	clock(t, time.Date(2026, 11, 6, 10, 0, 0, 0, time.Local))
	if err := config.SaveSession(cfgDir, config.Session{PreferredModel: "opus"}); err != nil {
		t.Fatal(err)
	}
	ss := func(in *bytes.Reader, out *bytes.Buffer) error { return SessionStart(in, out, cfgDir) }
	o := run(t, ss, map[string]any{"cwd": work, "source": "startup", "session_id": "a", "model": "claude-sonnet-5-5"})
	if text := o["additionalContext"].(string); !strings.Contains(text, "started with model claude-sonnet-5-5") {
		t.Errorf("no warning at start:\n%s", text)
	}
	if ls, ok := state.LastStartModel(cfgDir); !ok || ls.Model != "claude-sonnet-5-5" {
		t.Errorf("last start: %+v %v", ls, ok)
	}
	// The answer after it is not warned again for the same model.
	o = run(t, func(in *bytes.Reader, out *bytes.Buffer) error { return Stop(in, out, cfgDir) },
		map[string]any{"cwd": work, "session_id": "a", "transcript_path": transcript(t, "claude-sonnet-5-5")})
	if o != nil && strings.Contains(o["additionalContext"].(string), "sonnet") {
		t.Error("warned again after the first answer")
	}
	o = run(t, ss, map[string]any{"cwd": work, "source": "startup", "session_id": "b"})
	if strings.Contains(o["additionalContext"].(string), "started with model") {
		t.Error("warned with no model")
	}
	if ls, _ := state.LastStartModel(cfgDir); ls.Model != "" || ls.Source != "startup" {
		t.Errorf("a start with no model: %+v", ls)
	}
}
