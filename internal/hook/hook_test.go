package hook

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/icalvete/augmented-brain/internal/backup"
	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/setup"
)

func setupSpace(t *testing.T) (cfgDir, work string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	work = filepath.Join(home, "work")
	if err := os.MkdirAll(filepath.Join(work, "repo"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfgDir = filepath.Join(home, "cfg")
	if _, err := setup.Init(cfgDir, setup.Options{Space: "work", WorkFolder: work, Memory: "~/brain"}); err != nil {
		t.Fatal(err)
	}
	c, _ := config.Load(cfgDir)
	c.Spaces["work"].ProductionAWSProfiles = []string{"prod"}
	c.Spaces["work"].GitAllowedRepos = []string{filepath.Join(work, "repo")}
	if err := c.Save(); err != nil {
		t.Fatal(err)
	}
	return cfgDir, work
}

func run(t *testing.T, f func(*bytes.Reader, *bytes.Buffer) error, in map[string]any) map[string]any {
	t.Helper()
	data, _ := json.Marshal(in)
	var out bytes.Buffer
	if err := f(bytes.NewReader(data), &out); err != nil {
		t.Fatal(err)
	}
	if out.Len() == 0 {
		return nil
	}
	var o map[string]any
	if err := json.Unmarshal(out.Bytes(), &o); err != nil {
		t.Fatalf("output is not JSON: %s", out.String())
	}
	return o["hookSpecificOutput"].(map[string]any)
}

func TestSessionStart(t *testing.T) {
	cfgDir, work := setupSpace(t)
	ss := func(in *bytes.Reader, out *bytes.Buffer) error { return SessionStart(in, out, cfgDir) }

	o := run(t, ss, map[string]any{"cwd": filepath.Join(work, "repo"), "hook_event_name": "SessionStart"})
	if o["hookEventName"] != "SessionStart" || !strings.Contains(o["additionalContext"].(string), `space "work"`) {
		t.Errorf("in a space: %v", o)
	}
	o = run(t, ss, map[string]any{"cwd": "/"})
	if !strings.Contains(o["additionalContext"].(string), "outside the user's spaces") {
		t.Errorf("outside: %v", o)
	}
	// In parts: 1 has the heading, the others say nothing outside a space.
	part := func(n int) func(*bytes.Reader, *bytes.Buffer) error {
		return func(in *bytes.Reader, out *bytes.Buffer) error { return SessionStartPart(in, out, cfgDir, n) }
	}
	if o := run(t, part(1), map[string]any{"cwd": filepath.Join(work, "repo")}); !strings.Contains(o["additionalContext"].(string), "part 1 of 2") {
		t.Errorf("part 1: %v", o)
	}
	if o := run(t, part(2), map[string]any{"cwd": filepath.Join(work, "repo")}); !strings.Contains(o["additionalContext"].(string), "part 2 of 2") {
		t.Errorf("part 2: %v", o)
	}
	if o := run(t, part(2), map[string]any{"cwd": "/"}); o["additionalContext"] != nil {
		t.Errorf("part 2 outside the spaces: %v", o)
	}
	none := func(in *bytes.Reader, out *bytes.Buffer) error { return SessionStart(in, out, t.TempDir()) }
	if o := run(t, none, map[string]any{"cwd": "/"}); !strings.Contains(o["additionalContext"].(string), "augmented-brain init") {
		t.Errorf("not set up: %v", o)
	}
}

func TestPreToolUse(t *testing.T) {
	cfgDir, work := setupSpace(t)
	pre := func(in *bytes.Reader, out *bytes.Buffer) error { return PreToolUse(in, out, cfgDir) }
	key := "-----BEGIN " + "PRIVATE KEY-----"

	cases := []struct {
		in   map[string]any
		want string // "" means let through, with no output at all
	}{
		{map[string]any{"cwd": work, "tool_name": "Bash", "tool_input": map[string]any{"command": "ls"}}, ""},
		{map[string]any{"cwd": work, "tool_name": "Bash", "tool_input": map[string]any{"command": "git push"}}, "deny"},
		// In Bash a path is written with forward slashes, which git takes on Windows too.
		{map[string]any{"cwd": work, "tool_name": "Bash", "tool_input": map[string]any{"command": "git -C " + filepath.ToSlash(filepath.Join(work, "repo")) + " push"}}, ""},
		{map[string]any{"cwd": work, "tool_name": "Bash", "tool_input": map[string]any{"command": "aws s3 ls --profile prod"}}, "ask"},
		// The repositories of a space are allowed from a session anywhere, like the profiles.
		{map[string]any{"cwd": "/", "tool_name": "Bash", "tool_input": map[string]any{"command": "git -C " + filepath.ToSlash(filepath.Join(work, "repo")) + " push"}}, ""},
		{map[string]any{"cwd": work, "tool_name": "PowerShell", "tool_input": map[string]any{"command": "git push origin main"}}, "deny"},
		{map[string]any{"cwd": work, "tool_name": "PowerShell", "tool_input": map[string]any{"command": "Get-ChildItem"}}, "deny"},
		{map[string]any{"cwd": "/", "tool_name": "Bash", "tool_input": map[string]any{"command": "aws s3 ls --profile prod"}}, "ask"}, // machine-wide, outside the space too
		{map[string]any{"cwd": "/", "tool_name": "Bash", "tool_input": map[string]any{"command": "sudo id"}}, "deny"},                 // general rule
		{map[string]any{"cwd": work, "tool_name": "Write", "tool_input": map[string]any{"file_path": "/x", "content": key}}, "deny"},
		{map[string]any{"cwd": work, "tool_name": "Edit", "tool_input": map[string]any{"file_path": "/x", "old_string": key, "new_string": "removed"}}, ""},
		{map[string]any{"cwd": work, "tool_name": "MultiEdit", "tool_input": map[string]any{"file_path": "/x", "edits": []any{map[string]any{"old_string": "a", "new_string": key}}}}, "deny"},
		{map[string]any{"cwd": work, "tool_name": "Read", "tool_input": map[string]any{"file_path": "/etc/passwd"}}, ""},
		// Monitor runs a command in the background: checked exactly like Bash.
		{map[string]any{"cwd": work, "tool_name": "Monitor", "tool_input": map[string]any{"description": "watch", "timeout_ms": 20000, "command": "tail -f app.log | grep ERROR"}}, ""},
		{map[string]any{"cwd": work, "tool_name": "Monitor", "tool_input": map[string]any{"command": "while true; do git push; sleep 30; done"}}, "deny"},
		{map[string]any{"cwd": work, "tool_name": "Monitor", "tool_input": map[string]any{"command": "aws logs tail app --follow --profile prod"}}, "ask"},
		{map[string]any{"cwd": "/", "tool_name": "Monitor", "tool_input": map[string]any{"command": "sudo journalctl -f"}}, "deny"},
		// A WebSocket watch runs no program.
		{map[string]any{"cwd": work, "tool_name": "Monitor", "tool_input": map[string]any{"ws": map[string]any{"url": "wss://example.invalid/feed"}}}, ""},
	}
	for _, c := range cases {
		o := run(t, pre, c.in)
		got := ""
		if o != nil {
			got, _ = o["permissionDecision"].(string)
			if !strings.HasPrefix(o["permissionDecisionReason"].(string), "augmented brain: ") {
				t.Errorf("reason: %v", o)
			}
		}
		if got != c.want {
			t.Errorf("%v: got %q, want %q", c.in, got, c.want)
		}
	}

	// Broken input is refused, never let through unchecked, and is no error.
	var out bytes.Buffer
	if err := PreToolUse(strings.NewReader("not json"), &out, cfgDir); err != nil || !strings.Contains(out.String(), `"deny"`) {
		t.Errorf("broken input: err=%v out=%q", err, out.String())
	}
}

// Found in the review before 0.1: a broken configuration must be named as such, not
// sent to init (which refuses because the file exists).
func TestSessionStartBrokenConfig(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, config.FileName), []byte("[spaces.work\nmemory = "), 0o600); err != nil {
		t.Fatal(err)
	}
	ss := func(in *bytes.Reader, out *bytes.Buffer) error { return SessionStart(in, out, dir) }
	text := run(t, ss, map[string]any{"cwd": "/"})["additionalContext"].(string)
	if !strings.Contains(text, "cannot read its configuration") || strings.Contains(text, "not set up yet") {
		t.Errorf("broken config: %s", text)
	}
}

// No test asks GitHub: the latest version is whatever each test says.
func TestMain(m *testing.M) {
	LatestVersion = func() (string, error) { return "", nil }
	os.Exit(m.Run())
}

// A newer version out: the session asks the assistant to say so and offer to update.
func TestSessionStartUpdateNotice(t *testing.T) {
	cfgDir, work := setupSpace(t)
	t.Setenv("AB_NO_UPDATE_CHECK", "")
	ss := func(in *bytes.Reader, out *bytes.Buffer) error { return SessionStart(in, out, cfgDir) }
	old := LatestVersion
	t.Cleanup(func() { LatestVersion = old })

	LatestVersion = func() (string, error) { return "99.0.0", nil }
	o := run(t, ss, map[string]any{"cwd": work})
	if text := o["additionalContext"].(string); !strings.Contains(text, "augmented brain 99.0.0 is available") || !strings.Contains(text, "`augmented-brain update`") {
		t.Errorf("no update notice:\n%s", text)
	}

	// Up to date (the answer is cached for a day, so start from a new configuration).
	cfgDir, work = setupSpace(t)
	LatestVersion = func() (string, error) { return "0.0.0", nil }
	o = run(t, func(in *bytes.Reader, out *bytes.Buffer) error { return SessionStart(in, out, cfgDir) }, map[string]any{"cwd": work})
	if strings.Contains(o["additionalContext"].(string), "is available") {
		t.Error("an up-to-date install must not hear about updates")
	}
}

// When a session ends, its space is backed up, but only if it has a backup set up.
func TestSessionEnd(t *testing.T) {
	cfgDir, work := setupSpace(t)
	t.Setenv("AB_NO_AUTO_BACKUP", "")
	var started []string
	old := StartBackup
	StartBackup = func(space string) error { started = append(started, space); return nil }
	t.Cleanup(func() { StartBackup = old })
	end := func(cwd string) {
		data, _ := json.Marshal(map[string]any{"cwd": cwd})
		if err := SessionEnd(bytes.NewReader(data), cfgDir); err != nil {
			t.Fatal(err)
		}
	}
	end(work)
	if len(started) != 0 {
		t.Errorf("a space without a backup was backed up: %v", started)
	}
	if err := config.SetSpaceKey(cfgDir, "work", "backup", "git@example.com:me/brain.git"); err != nil {
		t.Fatal(err)
	}
	end(work)
	end("/") // outside every space, with no default space: nothing
	if strings.Join(started, ",") != "work" {
		t.Errorf("started: %v", started)
	}
	t.Setenv("AB_NO_AUTO_BACKUP", "1")
	end(work)
	if len(started) != 1 {
		t.Error("AB_NO_AUTO_BACKUP=1 must turn it off")
	}
}

// A failed backup is told to the user at the start of the next session.
func TestSessionStartBackupFailed(t *testing.T) {
	cfgDir, work := setupSpace(t)
	if err := config.SetSpaceKey(cfgDir, "work", "backup", "git@example.com:me/brain.git"); err != nil {
		t.Fatal(err)
	}
	backup.RecordStatusWarning(cfgDir, "work", errors.New("permission denied (publickey)"), "", time.Date(2026, 10, 16, 18, 0, 0, 0, time.UTC))
	ss := func(in *bytes.Reader, out *bytes.Buffer) error { return SessionStart(in, out, cfgDir) }
	o := run(t, ss, map[string]any{"cwd": work})
	if text := o["additionalContext"].(string); !strings.Contains(text, "The last backup of space work, at 2026-10-16 18:00, failed: permission denied (publickey)") {
		t.Errorf("no warning about the failed backup:\n%s", text)
	}
	backup.RecordStatusWarning(cfgDir, "work", nil, "", time.Now())
	o = run(t, ss, map[string]any{"cwd": work})
	if strings.Contains(o["additionalContext"].(string), "failed") {
		t.Error("a backup that worked again must not be reported")
	}
}

// The production profiles and untouchable hosts apply in every space, whichever space
// saved them; the AWS profile a command inherits counts; and a configuration that
// cannot be read makes those commands ask.
func TestPreToolUseMachineWide(t *testing.T) {
	cfgDir, work := setupSpace(t)
	other := filepath.Join(os.Getenv("HOME"), "personal")
	if _, err := setup.AddSpace(cfgDir, setup.Options{Space: "personal", WorkFolder: other, Memory: "~/brain-personal"}); err != nil {
		t.Fatal(err)
	}
	if err := config.SetTopList(cfgDir, "untouchable_hosts", []string{"db01.example.com"}); err != nil {
		t.Fatal(err)
	}
	pre := func(in *bytes.Reader, out *bytes.Buffer) error { return PreToolUse(in, out, cfgDir) }
	decide := func(cwd, cmd string) string {
		o := run(t, pre, map[string]any{"cwd": cwd, "tool_name": "Bash", "tool_input": map[string]any{"command": cmd}})
		if o == nil {
			return ""
		}
		return o["permissionDecision"].(string)
	}
	for _, cwd := range []string{work, other, "/"} {
		if got := decide(cwd, "aws s3 ls --profile prod"); got != "ask" { // saved in the work space
			t.Errorf("%s: production profile: %q", cwd, got)
		}
		if got := decide(cwd, "ssh DB01 uptime"); got != "ask" {
			t.Errorf("%s: untouchable host: %q", cwd, got)
		}
	}
	old := Getenv
	t.Cleanup(func() { Getenv = old })
	Getenv = func(k string) string { return map[string]string{"AWS_PROFILE": "prod"}[k] }
	if got := decide(other, "aws s3 ls"); got != "ask" {
		t.Errorf("inherited AWS_PROFILE=prod: %q", got)
	}
	Getenv = func(string) string { return "" }

	broken := t.TempDir()
	if err := os.WriteFile(filepath.Join(broken, config.FileName), []byte("[spaces.work\nmemory = "), 0o600); err != nil {
		t.Fatal(err)
	}
	pre = func(in *bytes.Reader, out *bytes.Buffer) error { return PreToolUse(in, out, broken) }
	if got := decide(work, "ssh web01 uptime"); got != "ask" {
		t.Errorf("broken configuration: %q", got)
	}
	if got := decide(work, "ls"); got != "" {
		t.Errorf("broken configuration, ls: %q", got)
	}
}
