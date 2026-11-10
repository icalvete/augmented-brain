package hook

import (
	"bytes"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/remote"
)

// A launched session learns its task from its first message, never from the
// environment: Claude's daemon hands every background session the environment of
// whoever started the daemon, so a task id there may be another session's. Its start
// says nothing of a task, whatever the environment holds.
func TestStartIgnoresTaskInEnvironment(t *testing.T) {
	cfgDir, work := setupSpace(t)
	old := Getenv
	t.Cleanup(func() { Getenv = old })
	ss := func(in *bytes.Reader, out *bytes.Buffer) error { return SessionStartPart(in, out, cfgDir, 1) }
	Getenv = func(k string) string { return map[string]string{remote.TaskEnv: "task-new-website"}[k] }
	text := run(t, ss, map[string]any{"cwd": work})["additionalContext"].(string)
	if strings.Contains(text, "task-new-website") {
		t.Errorf("the start believed the environment's task:\n%s", text)
	}
}

// The first message of a launched session has it read its task and get going.
func TestFirstMessageNamesTheTask(t *testing.T) {
	for _, l := range []string{"en", "es"} {
		msg := i18n.T(l, "session.first", "task-new-website")
		if !strings.Contains(msg, "task-new-website") || !strings.Contains(msg, "task_get") {
			t.Errorf("%s: %s", l, msg)
		}
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
