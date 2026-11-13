package hook

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/config"
)

// A call the guard can parse but not read the way it expects is refused, never let
// through unchecked: Claude Code changing the shape of a call must not switch it off.
func TestPreToolUseOddShapes(t *testing.T) {
	cfgDir, work := setupSpace(t)
	for _, c := range []struct {
		name  string
		stdin string
		want  string
	}{
		{"Bash with no command", `{"tool_name":"Bash","tool_input":{}}`, "deny"},
		{"Bash with tool_input null", `{"tool_name":"Bash","tool_input":null}`, "deny"},
		{"command as a list", `{"tool_name":"Bash","tool_input":{"command":["sudo","id"]}}`, "deny"},
		{"command null", `{"tool_name":"Bash","tool_input":{"command":null}}`, "deny"},
		{"Monitor with a command not text", `{"tool_name":"Monitor","tool_input":{"command":5}}`, "deny"},
		{"Monitor with no command (a WebSocket)", `{"tool_name":"Monitor","tool_input":{"url":"wss://x"}}`, ""},
		{"Write with a path not text", `{"tool_name":"Write","tool_input":{"file_path":["a"],"content":"x"}}`, "deny"},
		{"NotebookEdit with a path not text", `{"tool_name":"NotebookEdit","tool_input":{"notebook_path":1}}`, "deny"},
		{"Read with a path not text", `{"tool_name":"Read","tool_input":{"file_path":{}}}`, "deny"},
		{"Grep with no path (the folder)", `{"tool_name":"Grep","tool_input":{"pattern":"x"}}`, ""},
		{"a tool the guard does not check", `{"tool_name":"WebFetch","tool_input":{"url":"x"}}`, ""},
	} {
		var in map[string]any
		if err := json.Unmarshal([]byte(c.stdin), &in); err != nil {
			t.Fatal(err)
		}
		in["cwd"] = work
		data, _ := json.Marshal(in)
		var out bytes.Buffer
		if err := PreToolUse(bytes.NewReader(data), &out, cfgDir); err != nil {
			t.Fatal(err)
		}
		got := ""
		if out.Len() > 0 {
			var o struct {
				H map[string]any `json:"hookSpecificOutput"`
			}
			if err := json.Unmarshal(out.Bytes(), &o); err != nil {
				t.Fatalf("%s: not JSON: %s", c.name, out.String())
			}
			got, _ = o.H["permissionDecision"].(string)
		}
		if got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
}

// The plugin's matcher and the tools toolDecision checks are the same: a tool added to
// one and not the other is either never seen or never checked.
func TestMatcherNamesTheCheckedTools(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "plugin", "hooks", "hooks.json"))
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`"PreToolUse":\s*\[\s*\{\s*"matcher":\s*"([^"]*)"`).FindSubmatch(data)
	if m == nil {
		t.Fatal("no PreToolUse matcher in hooks.json")
	}
	matcher := strings.Split(string(m[1]), "|")
	slices.Sort(matcher)
	cases := slices.Clone(toolCases)
	slices.Sort(cases)
	if !slices.Equal(matcher, cases) {
		t.Errorf("matcher %v, toolDecision %v", matcher, cases)
	}
}

// With no configuration folder known, the guard asks about what its lists cover and
// never takes a config.toml of the working folder for its own.
func TestPreToolUseNoConfigFolder(t *testing.T) {
	work := t.TempDir()
	if err := os.WriteFile(filepath.Join(work, "config.toml"), []byte("[tool]\nname = \"x\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(work)
	decide := func(tool string, input map[string]any) (string, string) {
		o := run(t, func(in *bytes.Reader, out *bytes.Buffer) error { return PreToolUse(in, out, "") },
			map[string]any{"cwd": work, "tool_name": tool, "tool_input": input})
		if o == nil {
			return "", ""
		}
		return o["permissionDecision"].(string), o["permissionDecisionReason"].(string)
	}
	if got, why := decide("Bash", map[string]any{"command": "ssh db01 uptime"}); got != "ask" || !strings.Contains(why, "AB_CONFIG_DIR") {
		t.Errorf("ssh: %q (%s), want ask saying the folder is unknown", got, why)
	}
	if got, why := decide("Write", map[string]any{"file_path": filepath.Join(work, "config.toml"), "content": "x"}); got != "" {
		t.Errorf("the project's own config.toml: %q (%s)", got, why)
	}
}

// config.toml gone but the guard's lists or state still there: a broken configuration,
// not one never set up, so the guard asks.
func TestPreToolUseConfigGone(t *testing.T) {
	cfgDir, work := setupSpace(t)
	if err := config.SaveGuard(cfgDir, config.Guard{}); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(cfgDir, config.FileName)); err != nil {
		t.Fatal(err)
	}
	o := run(t, func(in *bytes.Reader, out *bytes.Buffer) error { return PreToolUse(in, out, cfgDir) },
		map[string]any{"cwd": work, "tool_name": "Bash", "tool_input": map[string]any{"command": "aws s3 ls --profile prod"}})
	if o == nil || o["permissionDecision"] != "ask" {
		t.Errorf("production profile with config.toml gone: %v", o)
	}
	// A folder never set up applies nothing, as before.
	empty := t.TempDir()
	o = run(t, func(in *bytes.Reader, out *bytes.Buffer) error { return PreToolUse(in, out, empty) },
		map[string]any{"cwd": work, "tool_name": "Bash", "tool_input": map[string]any{"command": "aws s3 ls --profile prod"}})
	if o != nil {
		t.Errorf("never set up: %v", o)
	}
}
