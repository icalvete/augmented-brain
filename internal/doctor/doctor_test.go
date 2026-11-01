package doctor

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/setup"
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
	if c := plugin(Env{LookPath: allTools, Plugin: no}); len(c) != 1 || c[0].Status != Failed || !strings.Contains(c[0].Fix, "claude plugin install") {
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
	for _, h := range []func([]byte) ([]byte, error){silent, missing} {
		if c := guardWorks(Env{Lang: "es", Hook: h}); c.Status != Failed || !strings.Contains(c.Fix, "~/.local/bin") {
			t.Errorf("broken guard: %+v", c)
		}
	}
}
