package hook

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/icalvete/augmented-brain/internal/config"
)

// Ways around the guard that only show in the hook (Rev 29 of the review of November
// 2026; ids as in its report). As in guard.TestKnownBypasses, each check pins today's
// answer and says the one it should give: when a fix changes it, the test fails and
// the check moves next to the guard's other hook tests with its new answer.

// decision is the hook's permissionDecision for one tool call: "" when it lets it through.
func decision(t *testing.T, cfgDir, work, tool string, input map[string]any) string {
	t.Helper()
	o := run(t, func(in *bytes.Reader, out *bytes.Buffer) error { return PreToolUse(in, out, cfgDir) },
		map[string]any{"cwd": work, "tool_name": tool, "tool_input": input})
	if o == nil {
		return ""
	}
	return o["permissionDecision"].(string)
}

// S-01: Claude Code reads the plugin's hooks from its own copy in
// ~/.claude/plugins/cache/<marketplace>/<plugin>/<version>/, not from the marketplace
// folder the guard protects, and runs the program from ~/.local/bin. Writing either
// switches the guard off, and nothing asks.
func TestBypassGuardSwitchedOff(t *testing.T) {
	cfgDir, work := setupSpace(t)
	old := Getenv
	t.Cleanup(func() { Getenv = old })
	Getenv = func(string) string { return "" }
	home := os.Getenv("HOME")
	cached := filepath.Join(home, ".claude", "plugins", "cache", "augmented-brain", "brain", "0.23.0-beta", "hooks", "hooks.json")
	program := filepath.Join(home, ".local", "bin", "augmented-brain")
	for _, f := range []string{cached, program} {
		if err := os.MkdirAll(filepath.Dir(f), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(f, []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, c := range []struct {
		what, tool  string
		input       map[string]any
		today, want string
	}{
		{"Write the cached hooks.json", "Write", map[string]any{"file_path": cached, "content": "{}"}, "", "ask"},
		{"Edit the cached hooks.json", "Edit", map[string]any{"file_path": cached, "old_string": "{", "new_string": "{ "}, "", "ask"},
		{"Write the program", "Write", map[string]any{"file_path": program, "content": "#!/bin/sh\n"}, "", "ask"},
		{"cp over the program", "Bash", map[string]any{"command": "cp /tmp/x ~/.local/bin/augmented-brain"}, "", "ask"},
		{"> the cached hooks.json", "Bash", map[string]any{"command": "echo '{}' > " + filepath.ToSlash(cached)}, "", "ask"},
		// The plain form, for comparison: the settings are protected.
		{"Write settings.json", "Write", map[string]any{"file_path": filepath.Join(home, ".claude", "settings.json"), "content": "{}"}, "ask", "ask"},
	} {
		if got := decision(t, cfgDir, work, c.tool, c.input); got != c.today {
			t.Errorf("S-01 %s: %q, it was %q (should be %q): if that is a fix, move the check", c.what, got, c.today, c.want)
		}
	}
}

// S-03: when the lists change outside the panel, only commit, push and merge go back
// to asking (ConfigChanged). A root exception for this machine, or a production
// profile or an untouchable host taken out, applies at once: becoming root asks
// instead of being denied, and aws with the profile that was production goes through.
func TestBypassListsChangedOutsideThePanel(t *testing.T) {
	cfgDir, work := setupSpace(t)
	old := Getenv
	t.Cleanup(func() { Getenv = old })
	Getenv = func(string) string { return "" }
	startSession(t, cfgDir, work, "startup") // the user opens a session: the lists are sealed
	su := "su" + "do id"
	bash := func(cmd string) string {
		return decision(t, cfgDir, work, "Bash", map[string]any{"command": cmd})
	}
	if got := bash(su); got != "deny" {
		t.Fatalf("sealed: %q", got)
	}
	if got := bash("aws --profile prod s3 ls"); got != "ask" {
		t.Fatalf("sealed: aws %q", got)
	}

	// What python3 -c, cp -t or rsync could do to config.toml without the guard asking.
	host, _ := os.Hostname()
	if host == "" {
		t.Skip("no host name")
	}
	c, err := config.Load(cfgDir)
	if err != nil {
		t.Fatal(err)
	}
	c.RootAsk = []string{host + " until 2099-12-31"}
	c.Spaces["work"].ProductionAWSProfiles = nil
	if err := c.Save(); err != nil {
		t.Fatal(err)
	}
	push := "git -C " + filepath.ToSlash(filepath.Join(work, "repo")) + " pu" + "sh"
	for _, row := range []struct{ cmd, today, want string }{
		{push, "ask", "ask"}, // the lists changed: what they let through asks
		{su, "ask", "deny"},
		{"aws --profile prod s3 ls", "", "ask"},
	} {
		if got := bash(row.cmd); got != row.today {
			t.Errorf("S-03 lists changed, %q: %q, it was %q (should be %q): if that is a fix, move the check", row.cmd, got, row.today, row.want)
		}
	}
}
