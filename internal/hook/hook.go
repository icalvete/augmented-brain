// Package hook answers Claude Code's hooks: SessionStart loads the space of the folder
// the session starts in, and PreToolUse runs the guard before a command or a write.
//
// A hook must never break the assistant: when its input or the configuration cannot be
// read, it falls back to the general rules (and lets everything else through).
package hook

import (
	"encoding/json"
	"io"
	"os"
	"sort"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/guard"
	"github.com/icalvete/augmented-brain/internal/session"
)

type input struct {
	Cwd       string         `json:"cwd"`
	ToolName  string         `json:"tool_name"`
	ToolInput map[string]any `json:"tool_input"`
}

type specific struct {
	HookEventName            string `json:"hookEventName"`
	AdditionalContext        string `json:"additionalContext,omitempty"`
	PermissionDecision       string `json:"permissionDecision,omitempty"`
	PermissionDecisionReason string `json:"permissionDecisionReason,omitempty"`
}

type output struct {
	HookSpecificOutput specific `json:"hookSpecificOutput"`
}

func read(in io.Reader) input {
	var i input
	_ = json.NewDecoder(in).Decode(&i)
	if i.Cwd == "" {
		i.Cwd, _ = os.Getwd()
	}
	return i
}

func write(out io.Writer, o output) error {
	return json.NewEncoder(out).Encode(o)
}

// load reads the configuration and the space of dir. Both may be nil.
func load(cfgDir, dir string) (*config.Config, *config.Space) {
	c, err := config.Load(cfgDir)
	if err != nil {
		return nil, nil
	}
	return c, c.SpaceFor(dir)
}

// SessionStart adds the space's profile, rules and index to the session.
func SessionStart(in io.Reader, out io.Writer, cfgDir string) error {
	i := read(in)
	c, s := load(cfgDir, i.Cwd)
	var text string
	switch {
	case c == nil:
		text = "augmented brain is installed but not set up yet: no memory is loaded. The user can run `augmented-brain init`."
	case s == nil:
		text = session.NoSpace(c, i.Cwd)
	default:
		text = session.Context(c, s)
	}
	return write(out, output{specific{HookEventName: "SessionStart", AdditionalContext: text}})
}

// PreToolUse runs the guard on a Bash command or on the text of a file write.
func PreToolUse(in io.Reader, out io.Writer, cfgDir string) error {
	i := read(in)
	var p guard.Policy
	if _, s := load(cfgDir, i.Cwd); s != nil {
		p = guard.Policy{GitAllowedRepos: s.GitAllowedRepos,
			ProductionAWSProfiles: s.ProductionAWSProfiles, UntouchableHosts: s.UntouchableHosts}
	}
	var d guard.Decision
	switch i.ToolName {
	case "Bash":
		cmd, _ := i.ToolInput["command"].(string)
		d = p.Command(cmd)
	case "Write", "Edit", "MultiEdit", "NotebookEdit":
		d = guard.Write(written(i.ToolInput))
	default:
		return nil
	}
	decision := map[guard.Verdict]string{guard.Deny: "deny", guard.Ask: "ask"}[d.Verdict]
	if decision == "" {
		return nil
	}
	return write(out, output{specific{HookEventName: "PreToolUse",
		PermissionDecision: decision, PermissionDecisionReason: "augmented brain: " + d.Reason}})
}

// written collects the text a write would put in the file: every string in the tool's
// input except the path and the text being replaced.
func written(v any) string {
	var parts []string
	var walk func(key string, v any)
	walk = func(key string, v any) {
		switch x := v.(type) {
		case string:
			if key != "file_path" && key != "notebook_path" && key != "old_string" {
				parts = append(parts, x)
			}
		case map[string]any:
			keys := make([]string, 0, len(x))
			for k := range x {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				walk(k, x[k])
			}
		case []any:
			for _, e := range x {
				walk(key, e)
			}
		}
	}
	walk("", v)
	out := ""
	for _, p := range parts {
		out += p + "\n"
	}
	return out
}
