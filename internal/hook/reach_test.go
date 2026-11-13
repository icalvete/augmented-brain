package hook

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/state"
)

// keepToSpaces sets reach = "spaces" and, when accepted, seals it as the panel does.
func keepToSpaces(t *testing.T, cfgDir string, accepted bool) {
	t.Helper()
	c, err := config.Load(cfgDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := state.WriteSeal(cfgDir, c.GuardFingerprint(config.Guard{})); err != nil {
		t.Fatal(err)
	}
	g := config.Guard{Reach: config.ReachSpaces}
	if err := config.SaveGuard(cfgDir, g); err != nil {
		t.Fatal(err)
	}
	if accepted {
		if err := state.WriteSeal(cfgDir, c.GuardFingerprint(g)); err != nil {
			t.Fatal(err)
		}
	}
}

func preTool(t *testing.T, cfgDir, cwd, tool string, input map[string]any) string {
	t.Helper()
	h := run(t, func(in *bytes.Reader, out *bytes.Buffer) error { return PreToolUse(in, out, cfgDir) },
		map[string]any{"cwd": cwd, "tool_name": tool, "tool_input": input})
	if h == nil {
		return ""
	}
	d, _ := h["permissionDecision"].(string)
	return d
}

func bash(t *testing.T, cfgDir, cwd, cmd string) string {
	t.Helper()
	return preTool(t, cfgDir, cwd, "Bash", map[string]any{"command": cmd})
}

func TestOutsideIsQuiet(t *testing.T) {
	cfgDir, work := setupSpace(t)
	keepToSpaces(t, cfgDir, true)
	out := filepath.Join(filepath.Dir(work), "elsewhere")
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	_ = state.AddGuardNotice(cfgDir, "lists changed", time.Now())
	for n := 0; n <= 3; n++ {
		o := runAll(t, func(in *bytes.Reader, w *bytes.Buffer) error { return SessionStartPart(in, w, cfgDir, n) },
			map[string]any{"cwd": out, "source": "startup"})
		h, _ := o["hookSpecificOutput"].(map[string]any)
		if text, _ := h["additionalContext"].(string); text != "" || o["systemMessage"] != nil {
			t.Errorf("part %d outside says something: %q %v", n, text, o["systemMessage"])
		}
	}
	for name, f := range map[string]func(*bytes.Reader, *bytes.Buffer) error{
		"UserPromptSubmit": func(in *bytes.Reader, w *bytes.Buffer) error { return UserPromptSubmit(in, w, cfgDir) },
		"Stop":             func(in *bytes.Reader, w *bytes.Buffer) error { return Stop(in, w, cfgDir) },
	} {
		if o := runAll(t, f, map[string]any{"cwd": out, "session_id": "s1"}); o != nil {
			t.Errorf("%s outside: %v", name, o)
		}
	}
	// The notice waits for a session inside a space.
	o := runAll(t, func(in *bytes.Reader, w *bytes.Buffer) error { return UserPromptSubmit(in, w, cfgDir) },
		map[string]any{"cwd": work, "session_id": "s2"})
	if msg, _ := o["systemMessage"].(string); !strings.Contains(msg, "lists changed") {
		t.Errorf("inside, the notice is gone: %v", o)
	}
	// Inside a space, the session still gets its memory.
	h := run(t, func(in *bytes.Reader, w *bytes.Buffer) error { return SessionStartPart(in, w, cfgDir, 1) },
		map[string]any{"cwd": work, "source": "startup"})
	if text, _ := h["additionalContext"].(string); text == "" {
		t.Error("inside a space, part 1 is empty")
	}
}

func TestOutsideGuard(t *testing.T) {
	cfgDir, work := setupSpace(t)
	keepToSpaces(t, cfgDir, true)
	out := filepath.Join(filepath.Dir(work), "elsewhere")
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	su := "su" + "do "
	for _, c := range []struct{ cwd, cmd, want string }{
		{out, su + "id", ""},
		{out, "git commit -m x", ""},
		{out, "aws s3 ls --profile prod", "ask"},
		{out, "cd " + work + " && " + su + "id", "deny"},
		{work, su + "id", "deny"},
		{work, "aws s3 ls --profile prod", "ask"},
	} {
		if got := bash(t, cfgDir, c.cwd, c.cmd); got != c.want {
			t.Errorf("%s in %s = %q, want %q", c.cmd, c.cwd, got, c.want)
		}
	}
	if got := preTool(t, cfgDir, out, "PowerShell", map[string]any{"command": "Get-Date"}); got != "" {
		t.Errorf("PowerShell outside = %q", got)
	}
	if got := preTool(t, cfgDir, work, "PowerShell", map[string]any{"command": "Get-Date"}); got != "deny" {
		t.Errorf("PowerShell inside = %q", got)
	}
	// Its own files stay protected; the project's settings are the user's outside.
	if got := preTool(t, cfgDir, out, "Write", map[string]any{"file_path": filepath.Join(cfgDir, config.GuardFile), "content": "reach = \"machine\"\n"}); got != "ask" {
		t.Errorf("guard.toml from outside = %q", got)
	}
	if got := preTool(t, cfgDir, out, "Write", map[string]any{"file_path": filepath.Join(out, ".claude", "settings.json"), "content": "{}"}); got != "" {
		t.Errorf("the project's settings outside = %q", got)
	}
	if got := preTool(t, cfgDir, work, "Write", map[string]any{"file_path": filepath.Join(work, ".claude", "settings.json"), "content": "{}"}); got != "ask" {
		t.Errorf("the project's settings inside = %q", got)
	}
}

// Written by hand and never accepted, reach counts as the whole machine.
func TestOutsideNeedsTheSeal(t *testing.T) {
	cfgDir, work := setupSpace(t)
	keepToSpaces(t, cfgDir, false)
	out := filepath.Join(filepath.Dir(work), "elsewhere")
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := bash(t, cfgDir, out, "su"+"do id"); got != "deny" {
		t.Errorf("unsealed reach switched the guard: %q", got)
	}
	h := run(t, func(in *bytes.Reader, w *bytes.Buffer) error { return SessionStartPart(in, w, cfgDir, 1) },
		map[string]any{"cwd": out, "source": "resume"})
	if text, _ := h["additionalContext"].(string); text == "" {
		t.Error("unsealed reach: the default space's memory is gone")
	}
}
