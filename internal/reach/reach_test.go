package reach

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/state"
)

func setup(t *testing.T, reach string) (string, *config.Config, string) {
	t.Helper()
	home, _ := filepath.EvalSymlinks(t.TempDir())
	t.Setenv("HOME", home)
	cfgDir := t.TempDir()
	body := "[spaces.work]\nmemory = \"~/notes\"\nwork_folders = [\"~/work\"]\n"
	if err := os.WriteFile(filepath.Join(cfgDir, config.FileName), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := config.SaveGuard(cfgDir, config.Guard{Reach: reach}); err != nil {
		t.Fatal(err)
	}
	c, err := config.Load(cfgDir)
	if err != nil {
		t.Fatal(err)
	}
	return cfgDir, c, home
}

func seal(t *testing.T, cfgDir string, c *config.Config) {
	t.Helper()
	g, err := config.LoadGuard(cfgDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := state.WriteSeal(cfgDir, c.GuardFingerprint(g)); err != nil {
		t.Fatal(err)
	}
}

func TestOutside(t *testing.T) {
	cfgDir, c, home := setup(t, config.ReachSpaces)
	out, in := filepath.Join(home, "tmp"), filepath.Join(home, "work", "app")
	if Outside(cfgDir, c, out) {
		t.Error("spaces counts before the user accepted it (no seal)")
	}
	seal(t, cfgDir, c)
	if !Outside(cfgDir, c, out) || Outside(cfgDir, c, in) {
		t.Error("sealed spaces: outside and inside mixed up")
	}
	if Outside(cfgDir, nil, out) {
		t.Error("no configuration counts as spaces")
	}
}

func TestMachineAndUnsealed(t *testing.T) {
	cfgDir, c, home := setup(t, config.ReachMachine)
	seal(t, cfgDir, c)
	if Outside(cfgDir, c, filepath.Join(home, "tmp")) {
		t.Error("machine is outside")
	}
	// Changed by hand to spaces after the seal: not taken until the user accepts it.
	if err := config.SaveGuard(cfgDir, config.Guard{Reach: config.ReachSpaces}); err != nil {
		t.Fatal(err)
	}
	if OnlySpaces(cfgDir, c) {
		t.Error("a change nobody accepted switches the guard")
	}
}
