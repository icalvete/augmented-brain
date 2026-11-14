package doctor

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/remote"
	"github.com/icalvete/augmented-brain/internal/setup"
	"github.com/icalvete/augmented-brain/internal/state"
	"github.com/icalvete/augmented-brain/internal/tailnet"
)

func find(checks []Check, text string) *Check {
	for i := range checks {
		if strings.Contains(checks[i].What, text) {
			return &checks[i]
		}
	}
	return nil
}

func allTools(string) (string, error) { return "/usr/bin/tool", nil }
func noTools(string) (string, error)  { return "", errors.New("not found") }
func yes() (bool, error)              { return true, nil }
func no() (bool, error)               { return false, nil }

func denies([]byte) ([]byte, error) {
	return []byte(`{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny"}}`), nil
}

func TestNoConfiguration(t *testing.T) {
	checks := Run(Env{ConfigDir: t.TempDir(), WorkDir: "/", LookPath: noTools})
	if Worst(checks) != Failed || !strings.Contains(checks[0].Fix, "augmented-brain init") {
		t.Errorf("checks = %+v", checks)
	}
	if c := find(checks, "Claude Code"); c == nil || c.Status != Failed || !strings.Contains(c.Fix, "claude.ai/install.sh") {
		t.Errorf("missing Claude Code must fail with its install command: %+v", checks)
	}
	if c := find(checks, "git"); c == nil || c.Status != Failed || !strings.Contains(c.Fix, GitInstall()) {
		t.Errorf("missing git must fail with its install command: %+v", checks)
	}
}

func TestHealthyAndBroken(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	work := filepath.Join(home, "work")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(home, "cfg")
	if _, err := setup.Init(cfg, setup.Options{Space: "work", WorkFolder: work, Memory: "~/brain"}); err != nil {
		t.Fatal(err)
	}

	checks := Run(Env{ConfigDir: cfg, WorkDir: work, LookPath: allTools, Plugin: yes, Hook: denies})
	if w := Worst(checks); w > Info {
		t.Fatalf("a fresh setup is not healthy: %+v", checks)
	}
	if c := find(checks, "Claude will ask you"); c == nil || c.Status != Info {
		t.Errorf("a blank profile is news, not a problem: %+v", checks)
	}
	if find(checks, `belongs to space "work"`) == nil {
		t.Errorf("current space not reported: %+v", checks)
	}

	// Outside any space: news, with where the spaces are.
	if c := find(Run(Env{ConfigDir: cfg, WorkDir: home, LookPath: allTools, Plugin: yes, Hook: denies}), "outside your spaces"); c == nil || c.Status != Info || !strings.Contains(c.What, "work (~/work)") {
		t.Errorf("outside a space must say where the spaces are, as news: %+v", c)
	}

	// A new valid note makes the index stale.
	note := "---\nid: area-web\ntype: area\ntitle: Web\n---\n"
	if err := os.WriteFile(filepath.Join(home, "brain", "areas", "area-web.md"), []byte(note), 0o644); err != nil {
		t.Fatal(err)
	}
	if c := find(Run(Env{ConfigDir: cfg, WorkDir: work, LookPath: allTools, Plugin: yes, Hook: denies}), "out of date"); c == nil || c.Status != Warning {
		t.Error("stale index not reported")
	}

	// A broken note fails.
	if err := os.WriteFile(filepath.Join(home, "brain", "tasks", "task-x.md"), []byte("no header"), 0o644); err != nil {
		t.Fatal(err)
	}
	checks = Run(Env{ConfigDir: cfg, WorkDir: work, LookPath: allTools, Plugin: yes, Hook: denies})
	if c := find(checks, "not valid"); c == nil || c.Status != Failed || Worst(checks) != Failed {
		t.Errorf("broken note not reported: %+v", checks)
	}
	if c := find(checks, "not valid"); c != nil && strings.Contains(c.Fix, "augmented-brain update") {
		t.Errorf("a note with no header does not ask to update: %+v", c)
	}

	// A field only a newer version knows asks to update.
	newer := "---\nid: area-new\ntype: area\ntitle: New\ncolour: blue\n---\n"
	if err := os.WriteFile(filepath.Join(home, "brain", "areas", "area-new.md"), []byte(newer), 0o644); err != nil {
		t.Fatal(err)
	}
	checks = Run(Env{ConfigDir: cfg, WorkDir: work, LookPath: allTools, Plugin: yes, Hook: denies})
	if c := find(checks, "not valid"); c == nil || !strings.Contains(c.Fix, "augmented-brain update") || !strings.Contains(c.Fix, "1 of them") {
		t.Errorf("a newer note must ask to update: %+v", c)
	}
}

// Two spaces and no default one (a configuration from before 0.2.9): no backup keeps the
// profile, and doctor says how to fix it.
func TestNoDefaultSpace(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg := filepath.Join(home, "cfg")
	if _, err := setup.Init(cfg, setup.Options{Space: "work", WorkFolder: "~/work", Memory: "~/brain/work"}); err != nil {
		t.Fatal(err)
	}
	if _, err := setup.AddSpace(cfg, setup.Options{Space: "home", WorkFolder: "~/home", Memory: "~/brain/home"}); err != nil {
		t.Fatal(err)
	}
	run := func() *Check {
		return find(Run(Env{ConfigDir: cfg, WorkDir: home, LookPath: allTools, Plugin: yes, Hook: denies}), "no default space")
	}
	if c := run(); c != nil {
		t.Errorf("a default space was chosen, but doctor complains: %+v", c)
	}

	file := filepath.Join(cfg, "config.toml")
	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var kept []string
	for _, l := range strings.Split(string(raw), "\n") {
		if !strings.HasPrefix(l, "default_space") {
			kept = append(kept, l)
		}
	}
	if err := os.WriteFile(file, []byte(strings.Join(kept, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	if c := run(); c == nil || c.Status != Warning || !strings.Contains(c.Fix, `default_space = "home"`) {
		t.Errorf("no default space must warn, with the line to add: %+v", c)
	}
}

func TestPluginCheck(t *testing.T) {
	if c := plugin(Env{LookPath: allTools, Plugin: no}); len(c) != 1 || c[0].Status != Failed || c[0].Fix != "augmented-brain update --plugin" {
		t.Errorf("missing plugin: %+v", c)
	}
	if c := plugin(Env{LookPath: allTools, Plugin: yes, Hook: denies}); len(c) != 2 || c[0].Status != OK || c[1].Status != OK {
		t.Errorf("installed plugin: %+v", c)
	}
	if c := plugin(Env{LookPath: noTools}); len(c) != 0 {
		t.Errorf("without Claude Code the plugin is not checked: %+v", c)
	}
}

// The guard is tried for real: a hook that cannot run, or lets a risky command through,
// fails the check.
func TestGuardCheck(t *testing.T) {
	if c := guardWorks(Env{Lang: "en", Hook: denies}); c.Status != OK {
		t.Errorf("working guard: %+v", c)
	}
	silent := func([]byte) ([]byte, error) { return nil, nil }
	missing := func([]byte) ([]byte, error) { return nil, errors.New("executable file not found") }
	garbled := func([]byte) ([]byte, error) { return []byte("panic: oops"), nil }
	for _, h := range []func([]byte) ([]byte, error){silent, missing, garbled} {
		if c := guardWorks(Env{Lang: "es", Hook: h}); c.Status != Failed || !strings.Contains(c.Fix, "~/.local/bin") {
			t.Errorf("broken guard: %+v", c)
		}
	}
	// A hook that cannot run blocks everything in Claude Code; one that answers without
	// blocking lets risky commands through. The message says which.
	if c := guardWorks(Env{Lang: "en", Hook: missing}); !strings.Contains(c.What, "stops every command") {
		t.Errorf("missing program: %+v", c)
	}
	if c := guardWorks(Env{Lang: "en", Hook: garbled}); !strings.Contains(c.What, "stops every command") {
		t.Errorf("garbled answer: %+v", c)
	}
	if c := guardWorks(Env{Lang: "en", Hook: silent}); !strings.Contains(c.What, "go through unchecked") {
		t.Errorf("silent hook: %+v", c)
	}
}

// After a restore that brought no guard lists, doctor warns until they are given again.
func TestNoListsAfterRestore(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg := filepath.Join(home, "cfg")
	if _, err := setup.Init(cfg, setup.Options{Space: "work", WorkFolder: "~/work", Memory: "~/brain/work"}); err != nil {
		t.Fatal(err)
	}
	run := func() *Check {
		return find(Run(Env{ConfigDir: cfg, WorkDir: home, LookPath: allTools, Plugin: yes, Hook: denies}), "restored from a backup")
	}
	if c := run(); c != nil {
		t.Errorf("not restored, but doctor says so: %+v", c)
	}
	if err := os.WriteFile(filepath.Join(cfg, ".ask-safety"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if c := run(); c == nil || c.Status != Warning {
		t.Errorf("no warning after a restore without lists: %+v", c)
	}
}

// guard.toml: a repository both untouchable and allowed is worth a look, a file that
// cannot be read is a warning, and lists changed outside the panel are said.
func TestGuardFileChecks(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	work := filepath.Join(home, "work")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(home, "cfg")
	if _, err := setup.Init(cfg, setup.Options{Space: "work", WorkFolder: work, Memory: "~/brain"}); err != nil {
		t.Fatal(err)
	}
	env := Env{ConfigDir: cfg, WorkDir: work, LookPath: allTools, Plugin: yes, Hook: denies}
	repo := filepath.Join(work, "core")
	if err := config.SetSpaceList(cfg, "work", "git_allowed_repos", []string{repo}); err != nil {
		t.Fatal(err)
	}
	if err := config.SaveGuard(cfg, config.Guard{UntouchableRepos: []string{repo}}); err != nil {
		t.Fatal(err)
	}
	checks := Run(env)
	if c := find(checks, "is untouchable and also in"); c == nil || c.Status != Warning {
		t.Errorf("both lists: %+v", checks)
	}
	if c := find(checks, "1 untouchable repositories"); c == nil {
		t.Errorf("count: %+v", checks)
	}
	if err := os.WriteFile(filepath.Join(cfg, config.GuardFile), []byte("untouchable_repos = ["), 0o600); err != nil {
		t.Fatal(err)
	}
	if c := find(Run(env), "guard.toml cannot be read"); c == nil || c.Status != Warning {
		t.Error("a broken guard.toml is not a warning")
	}
	// A misspelled setting is ignored, not refused: doctor names it and the right name.
	if err := os.WriteFile(filepath.Join(cfg, config.GuardFile), []byte("untouchable_repo = [\"~/work/core\"]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if c := find(Run(env), "guard.toml has settings this version does not know, and ignores them: untouchable_repo (did you mean untouchable_repos?)"); c == nil || c.Status != Warning || !strings.Contains(c.Fix, "guard.toml") {
		t.Errorf("unknown setting in guard.toml: %+v", c)
	}
	if err := os.WriteFile(filepath.Join(cfg, config.SessionFile), []byte("memory_trus = \"ask\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if c := find(Run(env), "session.toml has settings"); c == nil || !strings.Contains(c.What, "memory_trust?") {
		t.Errorf("unknown setting in session.toml: %+v", c)
	}
	_ = os.Remove(filepath.Join(cfg, config.SessionFile))
	_ = config.SaveGuard(cfg, config.Guard{})
	if c := find(Run(env), "does not know"); c != nil {
		t.Errorf("a clean guard.toml: %+v", c)
	}
	if err := state.WriteSeal(cfg, "not-the-current-one"); err != nil {
		t.Fatal(err)
	}
	if c := find(Run(env), "not the ones saved"); c == nil || c.Status != Warning {
		t.Error("lists changed outside the panel are not said")
	}
}

// With remote access set up, doctor gives a line per space and warns of a stopped
// service, an untrusted folder and linger off; with it not set up, no line.
func TestRemoteLines(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg := filepath.Join(home, "cfg")
	if _, err := setup.Init(cfg, setup.Options{Space: "work", WorkFolder: "~/work", Memory: "~/brain/work"}); err != nil {
		t.Fatal(err)
	}
	units := filepath.Join(home, "units")
	sys := &remote.System{
		Run: func(name string, args ...string) (string, error) {
			if name == "loginctl" {
				return "Linger=no\n", nil
			}
			if len(args) > 1 && args[1] == "is-active" {
				return "inactive\n", errors.New("exit status 3")
			}
			return "", nil
		},
		Hostname: func() (string, error) { return "box", nil }, GOOS: "linux", User: "lucia",
		UnitDir: units, ClaudeJSON: filepath.Join(home, "none.json"),
	}
	env := Env{ConfigDir: cfg, WorkDir: home, LookPath: allTools, Plugin: yes, Hook: denies, Remote: sys}
	if c := find(Run(env), "remote access"); c != nil {
		t.Errorf("not set up, yet doctor says: %+v", c)
	}
	if err := os.WriteFile(filepath.Join(cfg, remote.SettingsFile), []byte("enabled = true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(units, 0o755); err != nil {
		t.Fatal(err)
	}
	unit := remote.Service{Space: "work", Name: "box-work", Folder: filepath.Join(cfg, "remote", "work")}.UnitText("/bin/claude", "default", "/bin")
	if err := os.WriteFile(filepath.Join(units, "augmented-brain-remote-work.service"), []byte(unit), 0o644); err != nil {
		t.Fatal(err)
	}
	c := find(Run(env), "remote access, space work")
	if c == nil || c.Status != Warning || !strings.Contains(c.What, "box-work") ||
		!strings.Contains(c.Fix, "claude") || !strings.Contains(c.Fix, "enable-linger lucia") {
		t.Errorf("remote line = %+v", c)
	}
}

// The remote panel gets a line only once it has been used, and doctor warns of an
// address of the panel left published with no remote panel running.
func TestPanelRemoteLine(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg := filepath.Join(home, "cfg")
	if _, err := setup.Init(cfg, setup.Options{Space: "work", WorkFolder: "~/work", Memory: "~/brain/work"}); err != nil {
		t.Fatal(err)
	}
	served := "{}"
	tn := &tailnet.Tailnet{
		Run: func(name string, args ...string) (string, error) {
			switch strings.Join(args, " ") {
			case "status --json":
				return `{"BackendState":"Running","Self":{"DNSName":"box.example.ts.net.","UserID":1},"User":{"1":{"LoginName":"lucia@example.com"}},"CertDomains":["box.example.ts.net"]}`, nil
			case "serve status --json":
				return served, nil
			}
			return "", errors.New("unexpected")
		},
		LookPath: func(string) (string, error) { return "/usr/bin/tailscale", nil }, GOOS: "linux",
	}
	env := Env{ConfigDir: cfg, WorkDir: home, LookPath: allTools, Plugin: yes, Hook: denies, Remote: &remote.System{GOOS: "darwin", Hostname: func() (string, error) { return "box", nil }}, Tailnet: tn}
	if c := find(Run(env), "remote panel"); c != nil {
		t.Errorf("never used, yet doctor says: %+v", c)
	}
	if err := tailnet.MarkUsed(cfg); err != nil {
		t.Fatal(err)
	}
	if c := find(Run(env), "remote panel"); c == nil || c.Status != OK || !strings.Contains(c.What, "box.example.ts.net") {
		t.Errorf("ready: %+v", c)
	}
	served = `{"TCP":{"8443":{"HTTPS":true}},"Web":{"box.example.ts.net:8443":{"Handlers":{"/":{"Proxy":"http://127.0.0.1:4321"}}}}}`
	// A proxy to a local port the panel never recorded is the user's own (P-03).
	if c := find(Run(env), "remote panel"); c == nil || c.Status != OK {
		t.Errorf("the user's own proxy taken for a leftover: %+v", c)
	}
	if err := tailnet.Record(cfg, 4321); err != nil {
		t.Fatal(err)
	}
	if c := find(Run(env), "remote panel"); c == nil || c.Status != Warning || !strings.Contains(c.Fix, "tailscale serve --https=8443 off") {
		t.Errorf("leftover: %+v", c)
	}
	// While a remote panel runs, that address is its own.
	if err := os.WriteFile(filepath.Join(cfg, "panel.json"), []byte(`{"pid":1,"port":4321,"token":"x","remote":{"host":"box.example.ts.net:8443","owner":"lucia@example.com"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if c := find(Run(env), "remote panel"); c == nil || c.Status != OK {
		t.Errorf("running: %+v", c)
	}
}

// A backup address that holds a password (an earlier version kept it as it came) is
// named, with the address to set up again, without the password.
func TestBackupPassword(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	work := filepath.Join(home, "work")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(home, "cfg")
	if _, err := setup.Init(cfg, setup.Options{Space: "work", WorkFolder: work, Memory: "~/brain"}); err != nil {
		t.Fatal(err)
	}
	if err := config.SetSpaceKey(cfg, "work", "backup", "https://ana:Sup3rS3cret@example.com/u/r.git"); err != nil {
		t.Fatal(err)
	}
	c, err := config.Load(cfg)
	if err != nil {
		t.Fatal(err)
	}
	checks := space(c, c.Spaces["work"])
	w := find(checks, "password or a token")
	if w == nil || w.Status != Warning || !strings.Contains(w.Fix, "backup setup https://ana@example.com/u/r.git") || strings.Contains(w.What+w.Fix, "Sup3rS3cret") {
		t.Errorf("checks = %+v", checks)
	}
	if err := config.SetSpaceKey(cfg, "work", "backup", "https://ana@example.com/u/r.git"); err != nil {
		t.Fatal(err)
	}
	c, _ = config.Load(cfg)
	if w := find(space(c, c.Spaces["work"]), "password or a token"); w != nil {
		t.Errorf("a clean address was named: %+v", w)
	}
}

// The user's rules go whole to every session start: doctor says how much they take,
// with the longest, shared ones included and their History left out.
func TestRulesSize(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	work := filepath.Join(home, "work")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(home, "cfg")
	if _, err := setup.Init(cfg, setup.Options{Space: "work", WorkFolder: work, Memory: "~/brain"}); err != nil {
		t.Fatal(err)
	}
	c, err := config.Load(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if find(space(c, c.Spaces["work"]), "rules") != nil {
		t.Error("a line about rules with none")
	}
	write := func(path, text string) {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(c.Spaces["work"].Memory, "rules", "rule-long.md"), "---\nid: rule-long\ntype: rule\ntitle: Long one\n---\n"+strings.Repeat("x", 500)+"\n\n## History\n"+strings.Repeat("y", 5000)+"\n")
	write(filepath.Join(c.SharedDir(), "rules", "rule-short.md"), "---\nid: rule-short\ntype: rule\ntitle: Short one\n---\nBe brief.\n")
	r := find(space(c, c.Spaces["work"]), "rules (2,")
	if r == nil || r.Status != Info || !strings.Contains(r.What, "the longest: Long one (") || !strings.Contains(r.What, "; Short one (") {
		t.Fatalf("rules line: %+v", r)
	}
	if !strings.Contains(r.What, "take 540 characters") { // the History's 5,000 do not count
		t.Errorf("the History was counted: %s", r.What)
	}
}

// A Claude Code older than 2.1.295 lets every command through when the guard fails:
// doctor says so; a recent one, or one it cannot ask, adds nothing.
func TestClaudeVersion(t *testing.T) {
	answer := func(s string, err error) func() (string, error) { return func() (string, error) { return s, err } }
	if c, ok := claudeVersion(Env{Lang: "en", ClaudeVersion: answer("2.1.200 (Claude Code)\n", nil)}); !ok || c.Status != Warning || !strings.Contains(c.What, "2.1.200") {
		t.Errorf("old: %v %+v", ok, c)
	}
	for _, v := range []string{"2.1.295 (Claude Code)", "2.1.322 (Claude Code)", "3.0.0", "nonsense", ""} {
		if c, ok := claudeVersion(Env{Lang: "en", ClaudeVersion: answer(v, nil)}); ok {
			t.Errorf("%q: %+v", v, c)
		}
	}
	if _, ok := claudeVersion(Env{Lang: "en", ClaudeVersion: answer("", errors.New("no claude"))}); ok {
		t.Error("an error says nothing")
	}
}
