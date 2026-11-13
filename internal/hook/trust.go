package hook

import (
	"strings"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/memory"
)

// memoryTools are the tools of this program's MCP server that save what every session
// answers to: a rule, the profile, the guard's lists. With memory_trust = "ask" in
// session.toml the user is asked before each one. The plugin's second PreToolUse
// matcher names the same ones (a test compares them).
var memoryTools = []string{"rule_new", "note_retire", "profile_save", "safety_save"}

// memoryServer is how the name of this program's MCP server ends, as Claude Code
// writes it in a tool's name: mcp__augmented-brain__rule_new when the server is added
// by hand, mcp__plugin_brain_augmented-brain__rule_new through the plugin.
const memoryServer = "augmented-brain"

// memoryTool is which of memoryTools a tool's name is, or "".
func memoryTool(name string) string {
	rest, ok := strings.CutPrefix(name, "mcp__")
	if !ok {
		return ""
	}
	at := strings.LastIndex(rest, "__")
	if at < 0 || !strings.HasSuffix(rest[:at], memoryServer) {
		return ""
	}
	for _, t := range memoryTools {
		if rest[at+2:] == t {
			return t
		}
	}
	return ""
}

// trustDecision is the answer to a call of one of memoryTools: nothing with
// memory_trust = "tell" (the tool saves and the user is told afterwards), a question
// with "ask". A session.toml that cannot be read asks too: the level is unknown.
// Where Claude Code may skip the question (bypassPermissions) the call is refused,
// as everything else that must not go without the user's yes.
func trustDecision(cfgDir, lang string, i input, tool string) (decision, reason string) {
	if cfgDir == "" {
		return "", ""
	}
	if s, err := config.LoadSession(cfgDir); err == nil && s.Trust() == config.TrustTell {
		return "", ""
	}
	if tool == "note_retire" {
		// Only a rule reaches every session: a reference or a lesson goes without a
		// question. An id that is not text, or of no known type, is asked about: what
		// it retires cannot be told.
		if id, isText := i.ToolInput["id"].(string); isText && memory.TypeOfID(id) != "" && memory.TypeOfID(id) != "rule" {
			return "", ""
		}
	}
	what := i18n.T(lang, "guard.trust.what."+tool)
	if i.PermissionMode == "bypassPermissions" {
		return "deny", i18n.T(lang, "guard.trust.deny", what)
	}
	return "ask", i18n.T(lang, "guard.trust.ask", what)
}

// language is the configuration's language when it loads, the system's when not.
func language(cfgDir string) string {
	if c, err := config.Load(cfgDir); err == nil && c.Language != "" {
		return c.Language
	}
	return i18n.FromEnv()
}
