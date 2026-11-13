package hook

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/state"
)

// The guard cannot be switched off or softened (Rev 29a, closing S-01 and S-03 of the
// search for ways around it, Rev 29). What only shows in the hook: which files it
// protects, and what the lists say once something changed them outside the panel.

// verdict is the hook's permissionDecision for one tool call: "" when it lets it through.
func verdict(t *testing.T, cfgDir, work, tool string, input map[string]any) string {
	t.Helper()
	o := run(t, func(in *bytes.Reader, out *bytes.Buffer) error { return PreToolUse(in, out, cfgDir) },
		map[string]any{"cwd": work, "tool_name": tool, "tool_input": input})
	if o == nil {
		return ""
	}
	return o["permissionDecision"].(string)
}

// S-01: Claude Code reads the plugin's hooks from its own copy in
// ~/.claude/plugins/cache/<marketplace>/<plugin>/<version>/, and the hooks run the
// program from ~/.local/bin: changing either switches the guard off.
func TestGuardCannotBeSwitchedOff(t *testing.T) {
	cfgDir, work := setupSpace(t)
	old := Getenv
	t.Cleanup(func() { Getenv = old })
	Getenv = func(string) string { return "" }
	home := os.Getenv("HOME")
	cached := filepath.Join(home, ".claude", "plugins", "cache", "augmented-brain", "brain", "0.23.0-beta", "hooks", "hooks.json")
	program := filepath.Join(home, ".local", "bin", programName())
	installed := filepath.Join(home, ".claude", "plugins", "installed_plugins.json")
	for _, f := range []string{cached, program, installed} {
		if err := os.MkdirAll(filepath.Dir(f), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(f, []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	slash := filepath.ToSlash
	seal := filepath.Join(cfgDir, state.Dir, state.SealFile)
	for _, c := range []struct {
		what, tool string
		input      map[string]any
		want       string
	}{
		{"Write the cached hooks.json", "Write", map[string]any{"file_path": cached, "content": "{}"}, "ask"},
		{"Edit the cached hooks.json", "Edit", map[string]any{"file_path": cached, "old_string": "{", "new_string": "{ "}, "ask"},
		{"Write a hooks.json of another version", "Write", map[string]any{"file_path": filepath.Join(filepath.Dir(filepath.Dir(filepath.Dir(cached))), "9.9.9", "hooks", "hooks.json"), "content": "{}"}, "ask"},
		{"Write the program", "Write", map[string]any{"file_path": program, "content": "#!/bin/sh\n"}, "ask"},
		{"Write installed_plugins.json", "Write", map[string]any{"file_path": installed, "content": "{}"}, "ask"},
		{"Write the seal", "Write", map[string]any{"file_path": seal, "content": "x\n"}, "ask"},
		{"cp over the program", "Bash", map[string]any{"command": "cp /tmp/x ~/.local/bin/augmented-brain"}, "ask"},
		{"cp -t into the program's folder", "Bash", map[string]any{"command": "cp -t ~/.local/bin /tmp/augmented-brain"}, "ask"},
		{"> the cached hooks.json", "Bash", map[string]any{"command": "echo '{}' > " + slash(cached)}, "ask"},
		{"rm the plugin's cache", "Bash", map[string]any{"command": "rm -rf ~/.claude/plugins/cache/augmented-brain"}, "ask"},
		{"Write settings.json", "Write", map[string]any{"file_path": filepath.Join(home, ".claude", "settings.json"), "content": "{}"}, "ask"},
		// Reading them, and writing next to them, go through.
		{"cat the cached hooks.json", "Bash", map[string]any{"command": "cat " + slash(cached)}, ""},
		{"run the program", "Bash", map[string]any{"command": "augmented-brain version"}, ""},
		{"Write another plugin's file", "Write", map[string]any{"file_path": filepath.Join(home, ".claude", "plugins", "cache", "other", "x", "1.0", "hooks", "hooks.json"), "content": "{}"}, ""},
	} {
		if got := verdict(t, cfgDir, work, c.tool, c.input); got != c.want {
			t.Errorf("%s: %q, want %q", c.what, got, c.want)
		}
	}
}

// S-03: when the lists change outside the panel, a root exception for this machine
// does not count, and a production profile or untouchable host taken out of them no
// longer lets a command through: until the user accepts the lists, those ask.
func TestListsChangedOutsideThePanelDoNotSoften(t *testing.T) {
	cfgDir, work := setupSpace(t)
	old := Getenv
	t.Cleanup(func() { Getenv = old })
	Getenv = func(string) string { return "" }
	startSession(t, cfgDir, work, "startup") // the user opens a session: the lists are sealed
	su := "su" + "do id"
	bash := func(cmd string) string {
		return verdict(t, cfgDir, work, "Bash", map[string]any{"command": cmd})
	}
	if got := bash(su); got != "deny" {
		t.Fatalf("sealed: %q", got)
	}

	// What python3 -c could do to config.toml without the guard asking.
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
	for _, row := range []struct{ cmd, want string }{
		{push, "ask"},
		{su, "deny"},
		{"aws --profile prod s3 ls", "ask"},
		{"ssh db01", "ask"},
		{"ls", ""},
	} {
		if got := bash(row.cmd); got != row.want {
			t.Errorf("lists changed, %q: %q, want %q", row.cmd, got, row.want)
		}
	}

	// The user starts a session: the lists are the ones they know, and count.
	startSession(t, cfgDir, work, "startup")
	for _, row := range []struct{ cmd, want string }{
		{su, "ask"},
		{"aws --profile prod s3 ls", ""},
	} {
		if got := bash(row.cmd); got != row.want {
			t.Errorf("lists accepted, %q: %q, want %q", row.cmd, got, row.want)
		}
	}
}
