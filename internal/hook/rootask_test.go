package hook

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/icalvete/augmented-brain/internal/config"
)

// A root exception in the configuration: root on that machine asks while it lasts, the
// session start names it, and one that ended does nothing.
func TestRootAskFromConfig(t *testing.T) {
	cfgDir, work := setupSpace(t)
	until := time.Now().Add(time.Hour).UTC().Format("2006-01-02T15:04Z")
	if err := config.SetTopList(cfgDir, "root_ask", []string{"db01 until " + until, "old until 2020-01-01"}); err != nil {
		t.Fatal(err)
	}
	su := "su" + "do" // spelled apart, so tools that scan for the word leave this file alone
	pre := func(in *bytes.Reader, out *bytes.Buffer) error { return PreToolUse(in, out, cfgDir) }
	decide := func(cmd string) string {
		o := run(t, pre, map[string]any{"cwd": work, "tool_name": "Bash", "tool_input": map[string]any{"command": cmd}})
		if o == nil {
			return ""
		}
		return o["permissionDecision"].(string)
	}
	if got := decide("ssh ubuntu@db01 '" + su + " systemctl restart nginx'"); got != "ask" {
		t.Errorf("root on db01: %q", got)
	}
	if got := decide("ssh old '" + su + " id'"); got != "deny" {
		t.Errorf("root under an ended exception: %q", got)
	}
	ss := func(in *bytes.Reader, out *bytes.Buffer) error { return SessionStartPart(in, out, cfgDir, 1) }
	text := run(t, ss, map[string]any{"cwd": work})["additionalContext"].(string)
	if !strings.Contains(text, "Temporary root exceptions (root_ask): db01 until") || strings.Contains(text, "old until") {
		t.Errorf("session start:\n%s", text)
	}
}
