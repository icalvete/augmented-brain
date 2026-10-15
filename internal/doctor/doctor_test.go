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

func TestNoConfiguration(t *testing.T) {
	checks := Run(Env{ConfigDir: t.TempDir(), WorkDir: "/", LookPath: noTools})
	if Worst(checks) != Failed || !strings.Contains(checks[0].Fix, "augmented-brain init") {
		t.Errorf("checks = %+v", checks)
	}
	if c := find(checks, "Claude Code"); c == nil || c.Status != Warning {
		t.Errorf("missing tool not reported: %+v", checks)
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

	checks := Run(Env{ConfigDir: cfg, WorkDir: work, LookPath: allTools})
	if w := Worst(checks); w != OK {
		t.Fatalf("a fresh setup is not healthy: %+v", checks)
	}
	if find(checks, `belongs to space "work"`) == nil {
		t.Errorf("current space not reported: %+v", checks)
	}

	// Outside any space: a warning, not a failure.
	if c := find(Run(Env{ConfigDir: cfg, WorkDir: home, LookPath: allTools}), "belongs to no space"); c == nil || c.Status != Warning {
		t.Error("outside a space must be a warning")
	}

	// A new valid note makes the index stale.
	note := "---\nid: area-web\ntype: area\ntitle: Web\n---\n"
	if err := os.WriteFile(filepath.Join(home, "brain", "areas", "area-web.md"), []byte(note), 0o644); err != nil {
		t.Fatal(err)
	}
	if c := find(Run(Env{ConfigDir: cfg, WorkDir: work, LookPath: allTools}), "out of date"); c == nil || c.Status != Warning {
		t.Error("stale index not reported")
	}

	// A broken note fails.
	if err := os.WriteFile(filepath.Join(home, "brain", "tasks", "task-x.md"), []byte("no header"), 0o644); err != nil {
		t.Fatal(err)
	}
	checks = Run(Env{ConfigDir: cfg, WorkDir: work, LookPath: allTools})
	if c := find(checks, "not valid"); c == nil || c.Status != Failed || Worst(checks) != Failed {
		t.Errorf("broken note not reported: %+v", checks)
	}
}
