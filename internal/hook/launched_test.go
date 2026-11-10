package hook

import (
	"bytes"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/remote"
)

// A session opened by `augmented-brain session start` carries its task in the
// environment, and its start has the assistant read that task first. Any other session
// hears nothing about it.
func TestLaunchedSessionReadsItsTask(t *testing.T) {
	cfgDir, work := setupSpace(t)
	old := Getenv
	t.Cleanup(func() { Getenv = old })
	ss := func(in *bytes.Reader, out *bytes.Buffer) error { return SessionStartPart(in, out, cfgDir, 1) }

	Getenv = func(k string) string { return map[string]string{remote.TaskEnv: "task-new-website"}[k] }
	text := run(t, ss, map[string]any{"cwd": work})["additionalContext"].(string)
	if !strings.Contains(text, "task-new-website") || !strings.Contains(text, "task_get") {
		t.Errorf("launched session start:\n%s", text)
	}

	Getenv = func(string) string { return "" }
	text = run(t, ss, map[string]any{"cwd": work})["additionalContext"].(string)
	if strings.Contains(text, "This session was opened for task") {
		t.Errorf("a normal session heard about a launch:\n%s", text)
	}
}

// Every session on Linux knows the launcher exists and how to use it: once one told the
// user there was no way to open a session for a task. Elsewhere the launcher does not
// work, so nothing is said.
func TestEverySessionKnowsTheLauncher(t *testing.T) {
	cfgDir, work := setupSpace(t)
	oldEnv, oldOS := Getenv, GOOS
	t.Cleanup(func() { Getenv, GOOS = oldEnv, oldOS })
	Getenv = func(string) string { return "" }
	ss := func(in *bytes.Reader, out *bytes.Buffer) error { return SessionStartPart(in, out, cfgDir, 1) }

	GOOS = "linux"
	text := run(t, ss, map[string]any{"cwd": work})["additionalContext"].(string)
	for _, want := range []string{"augmented-brain session start <task>", "session list", "session stop <task>", "--folder"} {
		if !strings.Contains(text, want) {
			t.Errorf("Linux session start lacks %q:\n%s", want, text)
		}
	}

	for _, goos := range []string{"darwin", "windows"} {
		GOOS = goos
		text = run(t, ss, map[string]any{"cwd": work})["additionalContext"].(string)
		if strings.Contains(text, "session start <task>") {
			t.Errorf("%s session start names the launcher, which does not work there:\n%s", goos, text)
		}
	}
}
