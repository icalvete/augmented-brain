package hook

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/config"
)

// codexCall is a PreToolUse input as Codex CLI 0.170.0 sends it (measured with a
// scripted model): Claude Code's shape, with a turn_id, and bypassPermissions in
// `codex exec` whatever its approval policy.
func codexCall(cwd, tool string, input map[string]any, mode string) map[string]any {
	return map[string]any{
		"session_id": "01a1f41f-146d-7e32-b577-324fb4a897ca", "turn_id": "01a1f41f-14a7-7de1-8a6c-d5ccd736ca77",
		"transcript_path": "/tmp/rollout.jsonl", "cwd": cwd, "hook_event_name": "PreToolUse", "model": "gpt-5.5",
		"permission_mode": mode, "tool_name": tool, "tool_input": input, "tool_use_id": "call1",
	}
}

func decision(o map[string]any) (string, string) {
	if o == nil {
		return "", ""
	}
	d, _ := o["permissionDecision"].(string)
	r, _ := o["permissionDecisionReason"].(string)
	return d, r
}

// In Codex the guard never answers "ask": Codex runs such a command without asking
// anyone. A git push to a repository the lists do not allow and sudo are refused, and
// whatever Claude Code would ask about is refused too, in any permission mode.
func TestPreToolUseCodex(t *testing.T) {
	cfgDir, work := setupSpace(t)
	c, _ := config.Load(cfgDir)
	asked := filepath.Join(work, "asked")
	c.Spaces["work"].GitAskRepos = []string{asked}
	if err := c.Save(); err != nil {
		t.Fatal(err)
	}
	codex := func(in *bytes.Reader, out *bytes.Buffer) error { return PreToolUseFor(in, out, cfgDir, Codex) }
	claude := func(in *bytes.Reader, out *bytes.Buffer) error { return PreToolUse(in, out, cfgDir) }
	bash := func(cmd string) map[string]any { return map[string]any{"command": cmd} }
	slash := filepath.ToSlash

	for _, mode := range []string{"bypassPermissions", "default", ""} {
		for _, c := range []struct {
			cmd, codex, claude string
		}{
			{"ls", "", ""},
			{"sudo id", "deny", "deny"},
			{"git push origin main", "deny", "deny"},
			{"git -C " + slash(filepath.Join(work, "repo")) + " push", "", ""},
			// What Claude Code asks about outside bypassPermissions.
			{"git -C " + slash(asked) + " push", "deny", "ask"},
			{"aws s3 ls --profile prod", "deny", "ask"},
			{"gh workflow run release", "deny", "ask"},
		} {
			want := c.claude
			if mode == "bypassPermissions" && want == "ask" {
				want = "deny"
			}
			if d, r := decision(run(t, codex, codexCall(work, "Bash", bash(c.cmd), mode))); d != c.codex {
				t.Errorf("codex, %s, %q: %q (%s), want %q", mode, c.cmd, d, r, c.codex)
			}
			if d, r := decision(run(t, claude, codexCall(work, "Bash", bash(c.cmd), mode))); d != want {
				t.Errorf("claude, %s, %q: %q (%s), want %q", mode, c.cmd, d, r, want)
			}
		}
	}

	// The refusal says it is Codex that cannot ask, and keeps what would have been asked.
	_, r := decision(run(t, codex, codexCall(work, "Bash", bash("aws s3 ls --profile prod"), "default")))
	if !strings.HasPrefix(r, "augmented brain: ") || !strings.Contains(r, "Codex cannot") || !strings.Contains(r, "prod") {
		t.Errorf("reason: %q", r)
	}
}

// Codex's apply_patch, as its own tool or named in a shell command, is a write of
// every file it names: a secret in the lines it adds, or a protected file, stops it.
func TestPreToolUseCodexPatch(t *testing.T) {
	cfgDir, work := setupSpace(t)
	codex := func(in *bytes.Reader, out *bytes.Buffer) error { return PreToolUseFor(in, out, cfgDir, Codex) }
	key := "-----BEGIN " + "PRIVATE KEY-----"
	guardFile := filepath.Join(cfgDir, config.GuardFile)
	patch := func(body string) string { return "*** Begin Patch\n" + body + "*** End Patch\n" }

	cases := []struct {
		name, tool, text, want string
	}{
		{"add a plain file", "apply_patch", patch("*** Add File: hello.txt\n+hi\n"), ""},
		{"update a plain file", "apply_patch", patch("*** Update File: hello.txt\n@@\n-hi\n+hello\n"), ""},
		{"add a secret", "apply_patch", patch("*** Add File: id.txt\n+" + key + "\n"), "deny"},
		{"secret in a later file", "apply_patch", patch("*** Add File: a.txt\n+a\n*** Update File: b.txt\n@@\n+" + key + "\n"), "deny"},
		{"remove a secret", "apply_patch", patch("*** Update File: id.txt\n@@\n-" + key + "\n+gone\n"), ""},
		{"the guard's lists", "apply_patch", patch("*** Add File: " + guardFile + "\n+x = 1\n"), "deny"},
		{"delete the guard's lists", "apply_patch", patch("*** Delete File: " + guardFile + "\n"), "deny"},
		{"move onto the guard's lists", "apply_patch", patch("*** Update File: x.toml\n*** Move to: " + guardFile + "\n@@\n+y\n"), "deny"},
		{"Codex's hooks", "apply_patch", patch("*** Add File: " + filepath.Join(os.Getenv("HOME"), ".codex", "hooks.json") + "\n+{}\n"), "deny"},
		{"the project's Codex settings", "apply_patch", patch("*** Update File: .codex/config.toml\n@@\n+[features]\n"), "deny"},
		{"no patch", "apply_patch", "hello", "deny"},
		{"names no file", "apply_patch", patch(""), "deny"},
		{"shell: plain file", "Bash", "apply_patch <<'EOF'\n" + patch("*** Add File: viashell.txt\n+hi\n") + "EOF\n", ""},
		{"shell: secret", "Bash", "apply_patch <<'EOF'\n" + patch("*** Add File: viashell.txt\n+"+key+"\n") + "EOF\n", "deny"},
		{"shell: after cd", "Bash", "cd " + filepath.ToSlash(cfgDir) + " && apply_patch <<'EOF'\n" + patch("*** Add File: "+config.GuardFile+"\n+x\n") + "EOF\n", "deny"},
		{"shell: later in the command", "Bash", "true; applypatch <<'EOF'\n" + patch("*** Delete File: "+filepath.ToSlash(guardFile)+"\n") + "EOF\n", "deny"},
		{"shell: names no file", "Bash", "apply_patch <<'EOF'\n" + patch("") + "EOF\n", "deny"},
	}
	for _, c := range cases {
		d, r := decision(run(t, codex, codexCall(work, c.tool, map[string]any{"command": c.text}, "bypassPermissions")))
		if d != c.want {
			t.Errorf("%s: %q (%s), want %q", c.name, d, r, c.want)
		}
	}
	// A patch that is not text cannot be read: refused.
	if d, _ := decision(run(t, codex, codexCall(work, "apply_patch", map[string]any{"command": 7}, "bypassPermissions"))); d != "deny" {
		t.Errorf("patch not text: %q", d)
	}
}

// Codex names this program's MCP server augmented_brain: the tools that save what
// every session answers to are refused there when the user wants to be asked.
func TestPreToolUseCodexMemoryTools(t *testing.T) {
	cfgDir, work := setupSpace(t)
	if err := config.SaveSession(cfgDir, config.Session{MemoryTrust: config.TrustAsk}); err != nil {
		t.Fatal(err)
	}
	codex := func(in *bytes.Reader, out *bytes.Buffer) error { return PreToolUseFor(in, out, cfgDir, Codex) }
	rule := map[string]any{"title": "Short answers", "body": "Keep them short."}
	for _, mode := range []string{"bypassPermissions", "default"} {
		for tool, want := range map[string]string{
			"mcp__augmented_brain__rule_new":      "deny",
			"mcp__augmented_brain__profile_save":  "deny",
			"mcp__augmented_brain__safety_save":   "deny",
			"mcp__augmented_brain__task_journal":  "",
			"mcp__augmented_brain__where_am_i":    "",
			"mcp__other_brain__rule_new":          "",
			"mcp__augmented-brain__rule_new":      "deny",
			"mcp__not_augmented_brain__rule_new2": "",
		} {
			if d, r := decision(run(t, codex, codexCall(work, tool, rule, mode))); d != want {
				t.Errorf("%s in %s: %q (%s), want %q", tool, mode, d, r, want)
			}
		}
	}
}
