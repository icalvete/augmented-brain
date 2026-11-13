package hook

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/config"
)

// With memory_trust = "ask", the tools that save what every session answers to ask
// the user first; with "tell", or nothing set, they go through as before.
func TestPreToolUseMemoryTrust(t *testing.T) {
	cfgDir, work := setupSpace(t)
	pre := func(in *bytes.Reader, out *bytes.Buffer) error { return PreToolUse(in, out, cfgDir) }
	call := func(tool string, input map[string]any, mode string) (string, string) {
		o := run(t, pre, map[string]any{"cwd": work, "tool_name": tool, "tool_input": input, "permission_mode": mode})
		if o == nil {
			return "", ""
		}
		return o["permissionDecision"].(string), o["permissionDecisionReason"].(string)
	}
	const plugin, byHand = "mcp__plugin_brain_augmented-brain__", "mcp__augmented-brain__"
	rule := map[string]any{"title": "Short answers", "body": "Keep them short."}

	for _, level := range []string{"", config.TrustTell} {
		if err := config.SaveSession(cfgDir, config.Session{MemoryTrust: level}); err != nil {
			t.Fatal(err)
		}
		for _, tool := range memoryTools {
			if d, _ := call(plugin+tool, rule, "default"); d != "" {
				t.Errorf("memory_trust %q, %s: %q, want nothing", level, tool, d)
			}
		}
	}

	if err := config.SaveSession(cfgDir, config.Session{PreferredModel: "opus", MemoryTrust: config.TrustAsk}); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		tool  string
		input map[string]any
		mode  string
		want  string
	}{
		{plugin + "rule_new", rule, "default", "ask"},
		{byHand + "rule_new", rule, "default", "ask"},
		{plugin + "rule_new", rule, "auto", "ask"},
		{plugin + "rule_new", rule, "bypassPermissions", "deny"},
		{plugin + "profile_save", map[string]any{"about": "x"}, "default", "ask"},
		{plugin + "safety_save", map[string]any{"untouchable_hosts": []any{"db1"}}, "default", "ask"},
		{plugin + "note_retire", map[string]any{"id": "rule-short-answers", "reason": "old"}, "default", "ask"},
		{plugin + "note_retire", map[string]any{"id": "odd-name", "reason": "old"}, "default", "ask"},
		{plugin + "note_retire", map[string]any{"id": 7}, "default", "ask"},
		// A reference or a lesson does not reach every session.
		{plugin + "note_retire", map[string]any{"id": "lesson-x", "reason": "old"}, "default", ""},
		{plugin + "note_retire", map[string]any{"id": "ref-x", "reason": "old"}, "default", ""},
		// Tools the level does not cover, and other servers' tools of the same name.
		{plugin + "knowledge_new", rule, "default", ""},
		{plugin + "task_journal", map[string]any{"id": "task-x", "entry": "y"}, "default", ""},
		{"mcp__other__rule_new", rule, "default", ""},
		{"rule_new", rule, "default", ""},
	} {
		if d, r := call(c.tool, c.input, c.mode); d != c.want {
			t.Errorf("%s %v in %s: %q (%s), want %q", c.tool, c.input, c.mode, d, r, c.want)
		}
	}

	// The question says what is saved and where the level is, in the user's language.
	if _, r := call(plugin+"rule_new", rule, "default"); !strings.Contains(r, "saves a rule") || !strings.Contains(r, "memory_trust") {
		t.Errorf("question: %q", r)
	}
	if _, r := call(plugin+"profile_save", rule, "bypassPermissions"); !strings.Contains(r, "nothing was saved") {
		t.Errorf("refusal: %q", r)
	}

	// A level this program does not know, or a file it cannot read, asks.
	for _, content := range []string{"memory_trust = \"stricter\"\n", "memory_trust = [\n"} {
		if err := os.WriteFile(filepath.Join(cfgDir, config.SessionFile), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		if d, _ := call(plugin+"rule_new", rule, "default"); d != "ask" {
			t.Errorf("session.toml %q: %q, want ask", content, d)
		}
	}
}

// The plugin's second PreToolUse matcher names the tools trustDecision answers for,
// and matches them however Claude Code names the server.
func TestMatcherNamesTheMemoryTools(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "plugin", "hooks", "hooks.json"))
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`"matcher":\s*"(mcp__[^"]*)"`).FindSubmatch(data)
	if m == nil {
		t.Fatal("no matcher for the memory tools in hooks.json")
	}
	if want := "mcp__.*" + memoryServer + "__(" + strings.Join(memoryTools, "|") + ")"; string(m[1]) != want {
		t.Errorf("matcher %q, want %q", m[1], want)
	}
	re := regexp.MustCompile("^(?:" + string(m[1]) + ")$")
	for _, tool := range memoryTools {
		for _, prefix := range []string{"mcp__plugin_brain_augmented-brain__", "mcp__augmented-brain__"} {
			if !re.MatchString(prefix+tool) || memoryTool(prefix+tool) != tool {
				t.Errorf("%s%s: not matched", prefix, tool)
			}
		}
	}
}
