package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/session"
)

// The plugin runs one SessionStart hook per part of the session text: as many as the
// program has, 1 to session.Parts. Found in the review of 2026-11-03: they had to be
// kept in step by hand.
func TestPluginParts(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "plugin", "hooks", "hooks.json"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if n := strings.Count(text, "hook session-start "); n != session.Parts {
		t.Errorf("the plugin runs %d session-start parts, the program has %d", n, session.Parts)
	}
	for n := 1; n <= session.Parts; n++ {
		if !strings.Contains(text, fmt.Sprintf("hook session-start %d\"", n)) {
			t.Errorf("the plugin does not ask for part %d", n)
		}
	}
}

// A plugin newer than the program may ask for a part the program does not have: it
// gets an empty part and the session starts, never an error.
func TestPartBeyondTheProgram(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run([]string{"hook", "session-start", fmt.Sprint(session.Parts + 3)}, strings.NewReader("{}"), &out, &errOut); code != 0 {
		t.Errorf("exit %d: %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), `"hookEventName":"SessionStart"`) {
		t.Errorf("output: %s", out.String())
	}
}

// The plugin comes from the repository's main branch, so it can reach a machine before
// the program that knows its hooks (found on 2026-11-07). A program that does not know
// a hook answers 2, which for UserPromptSubmit throws the user's message away. So the
// hooks that only inform end in `|| true`; the guard (PreToolUse) never does: if the
// guard cannot run, the command must be stopped, not let through.
func TestNewHooksTolerateAnOldProgram(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "plugin", "hooks", "hooks.json"))
	if err != nil {
		t.Fatal(err)
	}
	var plugin struct {
		Hooks map[string][]struct {
			Hooks []struct {
				Command string `json:"command"`
			} `json:"hooks"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal(data, &plugin); err != nil {
		t.Fatal(err)
	}
	commands := func(event string) []string {
		var out []string
		for _, m := range plugin.Hooks[event] {
			for _, h := range m.Hooks {
				out = append(out, h.Command)
			}
		}
		return out
	}
	ups := commands("UserPromptSubmit")
	if len(ups) != 1 || !strings.HasSuffix(ups[0], "|| true") || !strings.Contains(ups[0], "hook user-prompt-submit") {
		t.Errorf("UserPromptSubmit: %q", ups)
	}
	for _, c := range commands("PreToolUse") {
		if strings.Contains(c, "|| true") {
			t.Errorf("the guard must fail closed: %q", c)
		}
	}
	// An unknown hook still exits 2, so a future guard hook that reaches an old
	// program blocks instead of letting everything through.
	var out, errOut bytes.Buffer
	if code := run([]string{"hook", "some-future-hook"}, strings.NewReader("{}"), &out, &errOut); code != 2 {
		t.Errorf("unknown hook: exit %d", code)
	}
	// And the command as written, run by sh against a program that does not know the
	// hook, ends well.
	if runtime.GOOS == "windows" {
		return
	}
	bin := t.TempDir()
	old := "#!/bin/sh\necho 'usage: augmented-brain hook session-start|pre-tool-use|stop' >&2\nexit 2\n"
	if err := os.WriteFile(filepath.Join(bin, "augmented-brain"), []byte(old), 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("sh", "-c", ups[0])
	cmd.Env = append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+"/usr/bin:/bin", "HOME="+t.TempDir())
	cmd.Stdin = strings.NewReader(`{"prompt":"hi"}`)
	if got, err := cmd.CombinedOutput(); err != nil || len(got) != 0 {
		t.Errorf("with an old program: %v, output %q", err, got)
	}
}
