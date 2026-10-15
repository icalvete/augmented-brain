package hook

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/setup"
)

func setupSpace(t *testing.T) (cfgDir, work string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	work = filepath.Join(home, "work")
	if err := os.MkdirAll(filepath.Join(work, "repo"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfgDir = filepath.Join(home, "cfg")
	if _, err := setup.Init(cfgDir, setup.Options{Space: "work", WorkFolder: work, Memory: "~/brain"}); err != nil {
		t.Fatal(err)
	}
	c, _ := config.Load(cfgDir)
	c.Spaces["work"].ProductionAWSProfiles = []string{"prod"}
	c.Spaces["work"].GitAllowedRepos = []string{filepath.Join(work, "repo")}
	if err := c.Save(); err != nil {
		t.Fatal(err)
	}
	return cfgDir, work
}

func run(t *testing.T, f func(*bytes.Reader, *bytes.Buffer) error, in map[string]any) map[string]any {
	t.Helper()
	data, _ := json.Marshal(in)
	var out bytes.Buffer
	if err := f(bytes.NewReader(data), &out); err != nil {
		t.Fatal(err)
	}
	if out.Len() == 0 {
		return nil
	}
	var o map[string]any
	if err := json.Unmarshal(out.Bytes(), &o); err != nil {
		t.Fatalf("output is not JSON: %s", out.String())
	}
	return o["hookSpecificOutput"].(map[string]any)
}

func TestSessionStart(t *testing.T) {
	cfgDir, work := setupSpace(t)
	ss := func(in *bytes.Reader, out *bytes.Buffer) error { return SessionStart(in, out, cfgDir) }

	o := run(t, ss, map[string]any{"cwd": filepath.Join(work, "repo"), "hook_event_name": "SessionStart"})
	if o["hookEventName"] != "SessionStart" || !strings.Contains(o["additionalContext"].(string), `space "work"`) {
		t.Errorf("in a space: %v", o)
	}
	o = run(t, ss, map[string]any{"cwd": "/"})
	if !strings.Contains(o["additionalContext"].(string), "belongs to no space") {
		t.Errorf("outside: %v", o)
	}
	none := func(in *bytes.Reader, out *bytes.Buffer) error { return SessionStart(in, out, t.TempDir()) }
	if o := run(t, none, map[string]any{"cwd": "/"}); !strings.Contains(o["additionalContext"].(string), "augmented-brain init") {
		t.Errorf("not set up: %v", o)
	}
}

func TestPreToolUse(t *testing.T) {
	cfgDir, work := setupSpace(t)
	pre := func(in *bytes.Reader, out *bytes.Buffer) error { return PreToolUse(in, out, cfgDir) }
	key := "-----BEGIN " + "PRIVATE KEY-----"

	cases := []struct {
		in   map[string]any
		want string // "" means let through, with no output at all
	}{
		{map[string]any{"cwd": work, "tool_name": "Bash", "tool_input": map[string]any{"command": "ls"}}, ""},
		{map[string]any{"cwd": work, "tool_name": "Bash", "tool_input": map[string]any{"command": "git push"}}, "deny"},
		{map[string]any{"cwd": work, "tool_name": "Bash", "tool_input": map[string]any{"command": "git -C " + filepath.Join(work, "repo") + " push"}}, ""},
		{map[string]any{"cwd": work, "tool_name": "Bash", "tool_input": map[string]any{"command": "aws s3 ls --profile prod"}}, "ask"},
		{map[string]any{"cwd": "/", "tool_name": "Bash", "tool_input": map[string]any{"command": "aws s3 ls --profile prod"}}, ""}, // outside the space
		{map[string]any{"cwd": "/", "tool_name": "Bash", "tool_input": map[string]any{"command": "sudo id"}}, "deny"},              // general rule
		{map[string]any{"cwd": work, "tool_name": "Write", "tool_input": map[string]any{"file_path": "/x", "content": key}}, "deny"},
		{map[string]any{"cwd": work, "tool_name": "Edit", "tool_input": map[string]any{"file_path": "/x", "old_string": key, "new_string": "removed"}}, ""},
		{map[string]any{"cwd": work, "tool_name": "MultiEdit", "tool_input": map[string]any{"file_path": "/x", "edits": []any{map[string]any{"old_string": "a", "new_string": key}}}}, "deny"},
		{map[string]any{"cwd": work, "tool_name": "Read", "tool_input": map[string]any{"file_path": "/etc/passwd"}}, ""},
	}
	for _, c := range cases {
		o := run(t, pre, c.in)
		got := ""
		if o != nil {
			got, _ = o["permissionDecision"].(string)
			if !strings.HasPrefix(o["permissionDecisionReason"].(string), "augmented brain: ") {
				t.Errorf("reason: %v", o)
			}
		}
		if got != c.want {
			t.Errorf("%v: got %q, want %q", c.in, got, c.want)
		}
	}

	// Broken input never breaks the assistant.
	var out bytes.Buffer
	if err := PreToolUse(strings.NewReader("not json"), &out, cfgDir); err != nil || out.Len() != 0 {
		t.Errorf("broken input: err=%v out=%q", err, out.String())
	}
}

// Found in the review before 0.1: a broken configuration must be named as such, not
// sent to init (which refuses because the file exists).
func TestSessionStartBrokenConfig(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, config.FileName), []byte("[spaces.work\nmemory = "), 0o600); err != nil {
		t.Fatal(err)
	}
	ss := func(in *bytes.Reader, out *bytes.Buffer) error { return SessionStart(in, out, dir) }
	text := run(t, ss, map[string]any{"cwd": "/"})["additionalContext"].(string)
	if !strings.Contains(text, "cannot read its configuration") || strings.Contains(text, "not set up yet") {
		t.Errorf("broken config: %s", text)
	}
}
