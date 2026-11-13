package hook

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/paths"
	"github.com/icalvete/augmented-brain/internal/state"
)

// sealCase is a space with a repository the assistant may push to and guard lists that
// changed since the user last saw them (the seal is of other lists).
func sealCase(t *testing.T) (cfgDir, work string, decide func() string) {
	t.Helper()
	cfgDir, work = setupSpace(t)
	old := Getenv
	t.Cleanup(func() { Getenv = old })
	Getenv = func(string) string { return "" }
	repo := filepath.Join(work, "repo")
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := state.WriteSeal(cfgDir, "deadbeef"); err != nil {
		t.Fatal(err)
	}
	push := "git -C " + filepath.ToSlash(repo) + " push"
	decide = func() string {
		o := run(t, func(in *bytes.Reader, out *bytes.Buffer) error { return PreToolUse(in, out, cfgDir) },
			map[string]any{"cwd": work, "tool_name": "Bash", "tool_input": map[string]any{"command": push}})
		if o == nil {
			return ""
		}
		return o["permissionDecision"].(string)
	}
	return cfgDir, work, decide
}

func startSession(t *testing.T, cfgDir, work, source string) map[string]any {
	t.Helper()
	return runAll(t, func(in *bytes.Reader, out *bytes.Buffer) error { return SessionStartPart(in, out, cfgDir, 1) },
		map[string]any{"cwd": work, "source": source, "session_id": "s-" + source})
}

// A compaction (or /clear, resume, fork) is not the user opening a session: the lists
// stay unsealed and the push the lists allow keeps asking (G-06).
func TestSessionStartCompactKeepsAsking(t *testing.T) {
	cfgDir, work, decide := sealCase(t)
	if got := decide(); got != "ask" {
		t.Fatalf("lists changed: %q", got)
	}
	for _, src := range []string{"compact", "clear", "resume", "fork"} {
		o := startSession(t, cfgDir, work, src)
		ctx := o["hookSpecificOutput"].(map[string]any)["additionalContext"].(string)
		if !strings.Contains(ctx, "Guard page") && !strings.Contains(ctx, "página Guardia") {
			t.Errorf("%s: no note about the lists", src)
		}
		if got := decide(); got != "ask" {
			t.Errorf("after SessionStart %s the push: %q", src, got)
		}
	}
	// A session the user starts takes them, and the user hears it from Claude Code.
	o := startSession(t, cfgDir, work, "startup")
	if msg, _ := o["systemMessage"].(string); !strings.Contains(msg, "augmented brain") {
		t.Errorf("startup: no message to the user: %v", o)
	}
	if got := decide(); got != "" {
		t.Errorf("after startup the push: %q", got)
	}
}

// A seal that cannot be read is not "no seal": the push asks (G-07).
func TestSealUnreadableAsks(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("file modes")
	}
	cfgDir, _, decide := sealCase(t)
	c, err := config.Load(cfgDir)
	if err != nil {
		t.Fatal(err)
	}
	g, _ := config.LoadGuard(cfgDir)
	if err := state.WriteSeal(cfgDir, c.GuardFingerprint(g)); err != nil {
		t.Fatal(err)
	}
	if got := decide(); got != "" {
		t.Fatalf("sealed lists: %q", got)
	}
	seal := filepath.Join(cfgDir, state.Dir, "guard.seal")
	if err := os.Chmod(seal, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(seal, 0o600) })
	if got := decide(); got != "ask" {
		t.Errorf("seal unreadable: %q", got)
	}
	if err := os.Chmod(seal, 0o600); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(cfgDir, state.Dir)
	if err := os.Chmod(dir, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })
	if got := decide(); got != "ask" {
		t.Errorf("state folder unreadable: %q", got)
	}
}

// What the panel changed in the lists reaches the user once, as Claude Code's own
// message, from whichever hook runs next.
func TestGuardNoticeOnce(t *testing.T) {
	cfgDir, work := setupSpace(t)
	for _, hook := range []struct {
		name string
		f    func(in *bytes.Reader, out *bytes.Buffer) error
	}{
		{"UserPromptSubmit", func(in *bytes.Reader, out *bytes.Buffer) error { return UserPromptSubmit(in, out, cfgDir) }},
		{"Stop", func(in *bytes.Reader, out *bytes.Buffer) error { return Stop(in, out, cfgDir) }},
	} {
		if err := state.AddGuardNotice(cfgDir, "changed in the panel: Allowed repositories: +~/work/core", time.Now()); err != nil {
			t.Fatal(err)
		}
		in := map[string]any{"cwd": work, "session_id": "n1"}
		o := runAll(t, hook.f, in)
		if msg, _ := o["systemMessage"].(string); !strings.Contains(msg, "+~/work/core") {
			t.Errorf("%s: no notice: %v", hook.name, o)
		}
		if o := runAll(t, hook.f, in); o != nil && o["systemMessage"] != nil {
			t.Errorf("%s: shown twice: %v", hook.name, o)
		}
	}
}

// What a tool saved for every session (a rule, the profile) reaches the user from
// Claude Code once, from the next hook, next to a notice about the guard's lists.
func TestToolNoticeOnce(t *testing.T) {
	cfgDir, work := setupSpace(t)
	for _, hook := range []struct {
		name string
		f    func(in *bytes.Reader, out *bytes.Buffer) error
	}{
		{"UserPromptSubmit", func(in *bytes.Reader, out *bytes.Buffer) error { return UserPromptSubmit(in, out, cfgDir) }},
		{"Stop", func(in *bytes.Reader, out *bytes.Buffer) error { return Stop(in, out, cfgDir) }},
		{"SessionStart", func(in *bytes.Reader, out *bytes.Buffer) error { return SessionStart(in, out, cfgDir) }},
	} {
		if err := state.AddToolNotice(cfgDir, "new rule «Deploys» (every space)", time.Now()); err != nil {
			t.Fatal(err)
		}
		if err := state.AddGuardNotice(cfgDir, "changed in the panel: +db01", time.Now()); err != nil {
			t.Fatal(err)
		}
		in := map[string]any{"cwd": work, "session_id": "n1"}
		o := runAll(t, hook.f, in)
		msg, _ := o["systemMessage"].(string)
		if !strings.Contains(msg, "the assistant saved, for every session") || !strings.Contains(msg, "«Deploys»") || !strings.Contains(msg, "+db01") {
			t.Errorf("%s: notice: %q", hook.name, msg)
		}
		if o := runAll(t, hook.f, in); o != nil && o["systemMessage"] != nil {
			t.Errorf("%s: shown twice: %v", hook.name, o)
		}
	}
}

// A setting of guard.toml or session.toml this program does not know is ignored, so
// the session start has the assistant tell the user, with the right name and the file;
// with none, nothing is said.
func TestSessionStartUnknownSettings(t *testing.T) {
	cfgDir, work := setupSpace(t)
	ctx := func() string {
		o := startSession(t, cfgDir, work, "startup")
		return o["hookSpecificOutput"].(map[string]any)["additionalContext"].(string)
	}
	if c := ctx(); strings.Contains(c, "does not know") || strings.Contains(c, "no conoce") {
		t.Fatalf("no unknown settings, yet: %s", c)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, config.GuardFile), []byte("untouchable_repo = []\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	c := ctx()
	for _, want := range []string{"guard.toml has settings", "untouchable_repo (did you mean untouchable_repos?)", paths.Tilde(filepath.Join(cfgDir, config.GuardFile))} {
		if !strings.Contains(c, want) {
			t.Errorf("missing %q in:\n%s", want, c)
		}
	}
}
