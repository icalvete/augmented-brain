package main

import (
	"bytes"
	"encoding/json"
	"errors"
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
		if !strings.Contains(text, fmt.Sprintf("hook session-start %d;", n)) {
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

// Someone may install the plugin from Claude Code's plugin menu without the program
// (found on 2026-11-14: every hook failed with "augmented-brain: not found" and the
// assistant was never told why). Then the first part of the session start tells the
// user and the assistant how to install it, the other hooks say nothing and end well,
// and the guard still fails closed. A program that is there keeps its exit code.
func TestHooksWithoutTheProgram(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the hooks run under sh; tried on Linux and macOS")
	}
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
	root, err := filepath.Abs(filepath.Join("..", "..", "plugin"))
	if err != nil {
		t.Fatal(err)
	}
	empty := t.TempDir()
	run := func(command, bin, lang string) (string, int) {
		cmd := exec.Command("sh", "-c", command)
		cmd.Env = []string{"PATH=" + bin + ":/usr/bin:/bin", "HOME=" + t.TempDir(), "CLAUDE_PLUGIN_ROOT=" + root, "LANG=" + lang}
		cmd.Stdin = strings.NewReader("{}")
		out, err := cmd.Output()
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return string(out), exit.ExitCode()
		}
		if err != nil {
			t.Fatal(err)
		}
		return string(out), 0
	}
	told := 0
	for event, groups := range plugin.Hooks {
		for _, g := range groups {
			for _, h := range g.Hooks {
				out, code := run(h.Command, empty, "en_US.UTF-8")
				if event == "PreToolUse" {
					if code == 0 {
						t.Errorf("the guard let a command through without the program: %q", h.Command)
					}
					continue
				}
				if code != 0 {
					t.Errorf("%s: exit %d without the program: %q", event, code, h.Command)
				}
				if !strings.HasSuffix(h.Command, "session-start 1") {
					if out != "" {
						t.Errorf("%s says something without the program: %q", event, out)
					}
					continue
				}
				told++
				for lang, want := range map[string]string{"en_US.UTF-8": "program is missing", "es_ES.UTF-8": "falta el programa"} {
					out, _ := run(h.Command, empty, lang)
					var o struct {
						SystemMessage string `json:"systemMessage"`
						Specific      struct {
							Event   string `json:"hookEventName"`
							Context string `json:"additionalContext"`
						} `json:"hookSpecificOutput"`
					}
					if err := json.Unmarshal([]byte(out), &o); err != nil {
						t.Fatalf("%s: not JSON: %v\n%s", lang, err, out)
					}
					if !strings.Contains(o.SystemMessage, want) || !strings.Contains(o.SystemMessage, "install.sh | sh") || !strings.Contains(o.SystemMessage, "install.ps1 | iex") {
						t.Errorf("%s: the user reads %q", lang, o.SystemMessage)
					}
					if o.Specific.Event != "SessionStart" || !strings.Contains(o.Specific.Context, "install.sh | sh") {
						t.Errorf("%s: the assistant reads %+v", lang, o.Specific)
					}
				}
			}
		}
	}
	if told != 1 {
		t.Errorf("%d hooks tell how to install the program, want 1", told)
	}
	// A program that is there and fails keeps its exit code: only "not found" (127)
	// goes to missing.sh.
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "augmented-brain"), []byte("#!/bin/sh\nexit 3\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	stop := plugin.Hooks["Stop"][0].Hooks[0].Command
	if out, code := run(stop, bin, "en_US.UTF-8"); code != 3 || out != "" {
		t.Errorf("a failing program: exit %d, output %q", code, out)
	}
}
