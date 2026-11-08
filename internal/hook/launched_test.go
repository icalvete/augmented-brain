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
	if strings.Contains(text, "session start`") {
		t.Errorf("a normal session heard about a launch:\n%s", text)
	}
}
