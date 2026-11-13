package hook

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/icalvete/augmented-brain/internal/testenv"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/icalvete/augmented-brain/internal/backup"
	"github.com/icalvete/augmented-brain/internal/claudecode"
	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/hostmem"
	"github.com/icalvete/augmented-brain/internal/setup"
	"github.com/icalvete/augmented-brain/internal/state"
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
	h, _ := o["hookSpecificOutput"].(map[string]any)
	return h
}

// runAll is run, returning the whole output (systemMessage included), or nil.
func runAll(t *testing.T, f func(*bytes.Reader, *bytes.Buffer) error, in map[string]any) map[string]any {
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
	return o
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
	// In parts: 1 has the heading, then the index and the common rules; the others say
	// nothing outside a space.
	part := func(n int) func(*bytes.Reader, *bytes.Buffer) error {
		return func(in *bytes.Reader, out *bytes.Buffer) error { return SessionStartPart(in, out, cfgDir, n) }
	}
	if o := run(t, part(1), map[string]any{"cwd": filepath.Join(work, "repo")}); !strings.Contains(o["additionalContext"].(string), "part 1 of 3") {
		t.Errorf("part 1: %v", o)
	}
	if o := run(t, part(2), map[string]any{"cwd": filepath.Join(work, "repo")}); !strings.Contains(o["additionalContext"].(string), "part 2 of 3") {
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
	testenv.CleanGit()
	LatestVersion = func() (string, error) { return "", nil }
	state.ScanProc = false // the Claude sessions open on the machine running the tests are not theirs
	Processes = nil        // nor is the memory of the window running them
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
	// Merging a pull request in a repository the assistant may push to, unless what the
	// command inherits points gh at another repository.
	repo := filepath.Join(work, "repo")
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".git", "config"), []byte("[remote \"origin\"]\n\turl = git@github.com:acme/repo.git\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	Getenv = func(string) string { return "" }
	merge := "cd " + filepath.ToSlash(repo) + " && gh pr merge 1 --squash"
	if got := decide(work, merge); got != "" {
		t.Errorf("merge in an allowed repository: %q", got)
	}
	if got := decide(repo, "gh pr merge 1 --squash"); got != "deny" {
		t.Errorf("merge without the repository written in the command: %q", got)
	}
	Getenv = func(k string) string { return map[string]string{"GH_REPO": "acme/other"}[k] }
	if got := decide(work, merge); got != "deny" {
		t.Errorf("inherited GH_REPO of another repository: %q", got)
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

// A session open since before an update still runs the old program: the start of a
// new one asks the assistant to say so.
func TestSessionStartOldServers(t *testing.T) {
	cfgDir, work := setupSpace(t)
	sd := filepath.Join(cfgDir, state.ServersDir)
	if err := os.MkdirAll(sd, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sd, strconv.Itoa(os.Getppid())), []byte("0.5.10-alpha\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	in := bytes.NewReader([]byte(`{"session_id":"s1","cwd":"` + filepath.ToSlash(work) + `"}`))
	if err := SessionStart(in, &out, cfgDir); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "1 other Claude sessions on this machine still run an earlier augmented brain") {
		t.Errorf("no word of the old session:\n%s", out.String())
	}
}

// After /clear a session keeps its MCP server: when that server is an old one, the
// start says so of this session (its parent is this session's Claude), apart from the
// others. Where the hook does not know its Claude, it counts as another, as before.
func TestSessionStartOwnOldServer(t *testing.T) {
	cfgDir, work := setupSpace(t)
	sd := filepath.Join(cfgDir, state.ServersDir)
	if err := os.MkdirAll(sd, 0o700); err != nil {
		t.Fatal(err)
	}
	server := os.Getppid() // a live process, as the server must be
	if err := os.WriteFile(filepath.Join(sd, strconv.Itoa(server)), []byte("0.5.10-alpha\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	useProcs(t, &fakeProcs{procs: map[int]hostmem.Proc{server: {PID: server, PPID: 200, Name: "augmented-brain"}}}, 300)
	old := Getenv
	t.Cleanup(func() { Getenv = old })
	start := func(claudePID string) string {
		Getenv = func(k string) string { return map[string]string{"CLAUDE_PID": claudePID}[k] }
		ss := func(in *bytes.Reader, out *bytes.Buffer) error { return SessionStartPart(in, out, cfgDir, 1) }
		return run(t, ss, map[string]any{"cwd": work})["additionalContext"].(string)
	}
	text := start("200")
	if !strings.Contains(text, "This session itself still runs an earlier augmented brain") || strings.Contains(text, "other Claude sessions") {
		t.Errorf("its own old server, not told as its own:\n%s", text)
	}
	text = start("999")
	if strings.Contains(text, "This session itself") || !strings.Contains(text, "1 other Claude sessions") {
		t.Errorf("another session's old server, told as its own:\n%s", text)
	}
	text = start("")
	if strings.Contains(text, "This session itself") || !strings.Contains(text, "1 other Claude sessions") {
		t.Errorf("no CLAUDE_PID: %s", text)
	}

	// A session opened for a task hears of its own old server, never of the others.
	launched(t, cfgDir, "abcdef12", "task-website", work)
	ss := func(in *bytes.Reader, out *bytes.Buffer) error { return SessionStartPart(in, out, cfgDir, 1) }
	task := func(claudePID string) string {
		Getenv = func(k string) string { return map[string]string{"CLAUDE_PID": claudePID}[k] }
		return run(t, ss, map[string]any{"cwd": work, "session_id": "abcdef12-0000-4000-8000-000000000000"})["additionalContext"].(string)
	}
	if text := task("999"); strings.Contains(text, "earlier augmented brain") {
		t.Errorf("a task's session hears of other sessions to restart:\n%s", text)
	}
	if text := task("200"); !strings.Contains(text, "This session itself still runs an earlier augmented brain") {
		t.Errorf("a task's session does not hear of its own old server:\n%s", text)
	}
}

// The guard's reasons come in the user's language: the configuration's, and the
// system's when there is no configuration to read it from.
func TestPreToolUseLanguage(t *testing.T) {
	cfgDir, work := setupSpace(t)
	reason := func(cfg, cmd string) string {
		pre := func(in *bytes.Reader, out *bytes.Buffer) error { return PreToolUse(in, out, cfg) }
		o := run(t, pre, map[string]any{"cwd": work, "tool_name": "Bash", "tool_input": map[string]any{"command": cmd}})
		if o == nil {
			return ""
		}
		return o["permissionDecisionReason"].(string)
	}
	setLanguage := func(lang string) {
		c, err := config.Load(cfgDir)
		if err != nil {
			t.Fatal(err)
		}
		c.Language = lang
		if err := c.Save(); err != nil {
			t.Fatal(err)
		}
	}
	setLanguage("en")
	t.Setenv("AB_LANGUAGE", "es")
	if r := reason(cfgDir, "sudo id"); !strings.Contains(r, "this would run as root") {
		t.Errorf("configuration in English: %q", r)
	}
	setLanguage("es")
	t.Setenv("AB_LANGUAGE", "en")
	for cmd, want := range map[string]string{
		"sudo id":                  "esto se ejecutaría como root",
		"aws s3 ls --profile prod": "puede usar el perfil de producción prod",
		"git push":                 "`git push` es cosa del usuario",
	} {
		if r := reason(cfgDir, cmd); !strings.Contains(r, want) {
			t.Errorf("configuration in Spanish, %q: %q", cmd, r)
		}
	}
	t.Setenv("AB_LANGUAGE", "es")
	if r := reason(t.TempDir(), "sudo id"); !strings.Contains(r, "esto se ejecutaría como root") {
		t.Errorf("no configuration, system in Spanish: %q", r)
	}
	var out bytes.Buffer
	if err := PreToolUse(strings.NewReader("not json"), &out, t.TempDir()); err != nil || !strings.Contains(out.String(), "no se pudo leer la llamada") {
		t.Errorf("broken input, system in Spanish: %q", out.String())
	}
}

// guard.toml, loaded by the hook: an untouchable repository is denied even where the
// space allows it, the panel's token file and pages are out of the assistant's reach,
// and the guard's own files ask before a write.
func TestPreToolUseGuardFile(t *testing.T) {
	cfgDir, work := setupSpace(t)
	old := Getenv
	t.Cleanup(func() { Getenv = old })
	Getenv = func(string) string { return "" }
	repo := filepath.Join(work, "repo")
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	pre := func(in *bytes.Reader, out *bytes.Buffer) error { return PreToolUse(in, out, cfgDir) }
	decide := func(tool string, input map[string]any) string {
		o := run(t, pre, map[string]any{"cwd": work, "tool_name": tool, "tool_input": input})
		if o == nil {
			return ""
		}
		return o["permissionDecision"].(string)
	}
	push := "git -C " + filepath.ToSlash(repo) + " push"
	if got := decide("Bash", map[string]any{"command": push}); got != "" {
		t.Fatalf("before guard.toml, the allowed repository: %q", got)
	}
	if err := config.SaveGuard(cfgDir, config.Guard{UntouchableRepos: []string{repo}}); err != nil {
		t.Fatal(err)
	}
	if got := decide("Bash", map[string]any{"command": push}); got != "deny" {
		t.Errorf("untouchable repository: %q", got)
	}
	// a guard.toml that cannot be read: what the lists would cover asks every time
	if err := os.WriteFile(filepath.Join(cfgDir, config.GuardFile), []byte("untouchable_repos = ["), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := decide("Bash", map[string]any{"command": "ssh web01 uptime"}); got != "ask" {
		t.Errorf("broken guard.toml: %q", got)
	}
	// The untouchable list is unknown: a push the other lists allow asks, and the
	// merge of a pull request too, until guard.toml is fixed.
	if got := decide("Bash", map[string]any{"command": push}); got != "ask" {
		t.Errorf("push with guard.toml broken: %q", got)
	}
	if got := decide("Bash", map[string]any{"command": "git -C " + filepath.ToSlash(repo) + " commit -m x"}); got != "ask" {
		t.Errorf("commit with guard.toml broken: %q", got)
	}
	if got := decide("Bash", map[string]any{"command": "cd " + filepath.ToSlash(repo) + " && gh pr merge 12"}); got != "ask" {
		t.Errorf("merge with guard.toml broken: %q", got)
	}
	if err := config.SaveGuard(cfgDir, config.Guard{}); err != nil {
		t.Fatal(err)
	}

	token := filepath.Join(cfgDir, "panel.json")
	if got := decide("Read", map[string]any{"file_path": token}); got != "deny" {
		t.Errorf("Read panel.json: %q", got)
	}
	if got := decide("Grep", map[string]any{"path": cfgDir, "pattern": "token"}); got != "deny" {
		t.Errorf("Grep in the configuration folder: %q", got)
	}
	if got := decide("Read", map[string]any{"file_path": filepath.Join(cfgDir, config.FileName)}); got != "" {
		t.Errorf("Read config.toml: %q", got)
	}
	for _, cmd := range []string{
		"cat " + token,
		"cd " + cfgDir + " && cat panel.json",
		"jq .token " + cfgDir + "/panel.*",
		"curl -s http://127.0.0.1:41234/s/0123456789abcdef0123456789abcdef/",
	} {
		if got := decide("Bash", map[string]any{"command": cmd}); got != "deny" {
			t.Errorf("%s: %q", cmd, got)
		}
	}
	for _, cmd := range []string{"grep -rn panel.json internal docs", "ls " + cfgDir, "augmented-brain panel", "curl -s http://127.0.0.1:8080/health"} {
		if got := decide("Bash", map[string]any{"command": cmd}); got != "" {
			t.Errorf("%s should pass: %q", cmd, got)
		}
	}
	for _, f := range []string{config.GuardFile, config.SecretsFile, "remote.toml", config.FileName} {
		if got := decide("Write", map[string]any{"file_path": filepath.Join(cfgDir, f), "content": "x = 1\n"}); got != "ask" {
			t.Errorf("write %s: %q", f, got)
		}
	}
}

// A change to the guard's lists that the user did not see (not made in the panel, not
// there when the session started) makes a push that the lists allow ask instead, until
// the user accepts the lists.
func TestPreToolUseGuardChangeAsks(t *testing.T) {
	cfgDir, work := setupSpace(t)
	old := Getenv
	t.Cleanup(func() { Getenv = old })
	Getenv = func(string) string { return "" }
	repo := filepath.Join(work, "repo")
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	seal := func() {
		t.Helper()
		c, err := config.Load(cfgDir)
		if err != nil {
			t.Fatal(err)
		}
		g, _ := config.LoadGuard(cfgDir)
		if err := state.WriteSeal(cfgDir, c.GuardFingerprint(g)); err != nil {
			t.Fatal(err)
		}
	}
	push := "git -C " + filepath.ToSlash(repo) + " push"
	decide := func() string {
		o := run(t, func(in *bytes.Reader, out *bytes.Buffer) error { return PreToolUse(in, out, cfgDir) },
			map[string]any{"cwd": work, "tool_name": "Bash", "tool_input": map[string]any{"command": push}})
		if o == nil {
			return ""
		}
		return o["permissionDecision"].(string)
	}
	if got := decide(); got != "" {
		t.Fatalf("no seal yet: %q", got)
	}
	seal()
	if got := decide(); got != "" {
		t.Fatalf("lists as sealed: %q", got)
	}
	if err := config.SetTopList(cfgDir, "untouchable_hosts", []string{"db99.example.com"}); err != nil {
		t.Fatal(err)
	}
	if got := decide(); got != "ask" {
		t.Errorf("a list changed since the seal: %q", got)
	}
	seal() // the user accepts them (the panel does this when it saves)
	if got := decide(); got != "" {
		t.Errorf("after accepting: %q", got)
	}
}

// A space a session of another space wrote in is backed up when any session ends, once.
func TestSessionEndBacksUpWrittenFromOutside(t *testing.T) {
	cfgDir, work := setupSpace(t)
	t.Setenv("AB_NO_AUTO_BACKUP", "")
	other := filepath.Join(filepath.Dir(work), "me")
	if err := os.MkdirAll(other, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := setup.AddSpace(cfgDir, setup.Options{Space: "personal", WorkFolder: other, Memory: filepath.Join(filepath.Dir(work), "brain-personal")}); err != nil {
		t.Fatal(err)
	}
	if err := config.SetSpaceKey(cfgDir, "personal", "backup", "git@example.com:me/personal.git"); err != nil {
		t.Fatal(err)
	}
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
	end(work) // work has no backup, personal was not written: nothing
	if len(started) != 0 {
		t.Fatalf("started: %v", started)
	}
	if err := state.MarkOutside(cfgDir, "personal", time.Now()); err != nil {
		t.Fatal(err)
	}
	end(work)
	end(work)
	if strings.Join(started, ",") != "personal" {
		t.Errorf("started: %v", started)
	}
}

// The plugin Claude Code reads from the configuration folder holds the hooks, the
// guard among them: changing it asks, as changing the guard's lists does.
func TestPreToolUseLocalPluginAsks(t *testing.T) {
	cfgDir, work := setupSpace(t)
	old := Getenv
	t.Cleanup(func() { Getenv = old })
	Getenv = func(string) string { return "" }
	pre := func(in *bytes.Reader, out *bytes.Buffer) error { return PreToolUse(in, out, cfgDir) }
	hooks := filepath.Join(claudecode.LocalDir(cfgDir), "plugin", "hooks", "hooks.json")
	o := run(t, pre, map[string]any{"cwd": work, "tool_name": "Write",
		"tool_input": map[string]any{"file_path": hooks, "content": "{}"}})
	if o == nil || o["permissionDecision"] != "ask" {
		t.Errorf("writing the local plugin's hooks: %v", o)
	}
}
