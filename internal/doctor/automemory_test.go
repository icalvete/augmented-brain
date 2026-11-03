package doctor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/icalvete/augmented-brain/internal/claudecode"
	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/setup"
)

// Found in the rehearsal of 0.5.10: Claude Code started a memory of its own for the work
// folder, next to the space's. doctor says so until it is turned off.
func TestAutoMemory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	work := filepath.Join(home, "work")
	cfg := filepath.Join(home, "cfg")
	if _, err := setup.Init(cfg, setup.Options{Space: "work", WorkFolder: work, Memory: "~/brain"}); err != nil {
		t.Fatal(err)
	}
	claude := filepath.Join(home, ".claude")
	env := Env{ConfigDir: cfg, WorkDir: work, LookPath: allTools, Plugin: yes, Hook: denies, ClaudeDir: claude}
	if c := find(Run(env), "memory of its own"); c != nil {
		t.Errorf("warned with no second memory: %+v", c)
	}
	c, err := config.Load(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(claude, "projects", claudecode.ProjectSlug(c.Spaces["work"].WorkFolders[0]), "memory"), 0o700); err != nil {
		t.Fatal(err)
	}
	if c := find(Run(env), "memory of its own"); c == nil || c.Status != Warning {
		t.Errorf("a second memory is not named: %+v", c)
	}
	if err := os.WriteFile(filepath.Join(claude, "settings.json"), []byte(`{"autoMemoryEnabled": false}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if c := find(Run(env), "memory of its own"); c != nil {
		t.Errorf("warned with the auto memory off: %+v", c)
	}
}
