package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// failHook makes the hook named name panic for the length of the test.
func failHook(t *testing.T, name string) {
	t.Helper()
	i := slices.IndexFunc(hookEvents, func(e hookEvent) bool { return e.name == name })
	if i < 0 {
		t.Fatalf("no hook %s", name)
	}
	old := hookEvents[i].run
	t.Cleanup(func() { hookEvents[i].run = old })
	hookEvents[i].run = func(io.Reader, io.Writer, string, int) error {
		var m map[string]int
		m["boom"] = 1 // a nil map: the kind of bug a parser could have
		return nil
	}
}

// A panic in pre-tool-use refuses the call in plain words, with exit code 0 and the
// trace in a file, never on stderr for the assistant to read.
func TestHookPanicPreToolUse(t *testing.T) {
	cfg := t.TempDir()
	t.Setenv("AB_CONFIG_DIR", cfg)
	failHook(t, "pre-tool-use")
	var out, errOut bytes.Buffer
	code := cmdHook([]string{"pre-tool-use"}, strings.NewReader(`{"tool_name":"Bash","tool_input":{"command":"ls"}}`), &out, &errOut)
	if code != 0 {
		t.Errorf("exit code %d", code)
	}
	if !strings.Contains(out.String(), `"permissionDecision":"deny"`) || strings.Contains(out.String(), "goroutine") {
		t.Errorf("output: %s", out.String())
	}
	if errOut.Len() > 0 {
		t.Errorf("stderr: %s", errOut.String())
	}
	trace, err := os.ReadFile(filepath.Join(cfg, "state", "hook-panic.log"))
	if err != nil || !strings.Contains(string(trace), "goroutine") {
		t.Errorf("trace file: %v %.200s", err, trace)
	}
}

// A panic in stop exits 0 with one line: exit code 2 there would make the assistant go
// on, again and again.
func TestHookPanicStop(t *testing.T) {
	t.Setenv("AB_CONFIG_DIR", t.TempDir())
	failHook(t, "stop")
	var out, errOut bytes.Buffer
	if code := cmdHook([]string{"stop"}, strings.NewReader(`{}`), &out, &errOut); code != 0 {
		t.Errorf("exit code %d", code)
	}
	if out.Len() > 0 {
		t.Errorf("output: %s", out.String())
	}
	if lines := strings.Count(strings.TrimSpace(errOut.String()), "\n"); lines > 0 {
		t.Errorf("stderr has %d lines: %s", lines+1, errOut.String())
	}
}

type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) { return 0, errors.New("no space left on device") }

// A deny that cannot be written blocks with exit code 2: an empty answer with code 0
// would let the call through.
func TestHookUnwrittenDenyBlocks(t *testing.T) {
	cfg := t.TempDir()
	t.Setenv("AB_CONFIG_DIR", cfg)
	var errOut bytes.Buffer
	code := cmdHook([]string{"pre-tool-use"}, strings.NewReader(`{"tool_name":"Bash","tool_input":{"command":"sudo id"}}`), brokenWriter{}, &errOut)
	if code != 2 || !strings.Contains(errOut.String(), "no space left") {
		t.Errorf("exit code %d, stderr %q", code, errOut.String())
	}
	// Nothing to say (an allow) writes nothing, so nothing fails.
	if code := cmdHook([]string{"pre-tool-use"}, strings.NewReader(`{"tool_name":"Bash","tool_input":{"command":"ls"}}`), brokenWriter{}, &errOut); code != 0 {
		t.Errorf("allow: exit code %d", code)
	}
}

// With no configuration folder at all, the guard still runs and asks; the other hooks
// do nothing rather than write into the folder Claude runs in.
func TestHookNoConfigFolder(t *testing.T) {
	work := t.TempDir()
	t.Chdir(work)
	t.Setenv("AB_CONFIG_DIR", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")
	t.Setenv("AppData", "")
	var out, errOut bytes.Buffer
	code := cmdHook([]string{"pre-tool-use"}, strings.NewReader(`{"tool_name":"Bash","tool_input":{"command":"ssh db01 uptime"}}`), &out, &errOut)
	if code != 0 || !strings.Contains(out.String(), `"permissionDecision":"ask"`) {
		t.Errorf("guard: exit code %d, output %s, stderr %s", code, out.String(), errOut.String())
	}
	out.Reset()
	if code := cmdHook([]string{"stop"}, strings.NewReader(`{}`), &out, &errOut); code != 0 || out.Len() > 0 {
		t.Errorf("stop: exit code %d, output %s", code, out.String())
	}
	if entries, _ := os.ReadDir(work); len(entries) > 0 {
		t.Errorf("the hooks wrote in the working folder: %v", entries)
	}
}
