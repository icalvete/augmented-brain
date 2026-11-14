package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGuardReach(t *testing.T) {
	dir := t.TempDir()
	if err := SaveGuard(dir, Guard{Reach: ReachSpaces}); err != nil {
		t.Fatal(err)
	}
	g, err := LoadGuard(dir)
	if err != nil || !g.OnlySpaces() {
		t.Fatalf("spaces: %+v %v", g, err)
	}
	// Without reach the file says nothing of it, as earlier programs wrote it.
	if err := SaveGuard(dir, Guard{}); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(dir, GuardFile))
	if strings.Contains(string(b), "reach") {
		t.Errorf("default written: %s", b)
	}
	if g, _ := LoadGuard(dir); g.OnlySpaces() {
		t.Error("default reads as spaces")
	}
	// A value it does not know is a broken file: the guard asks rather than guess.
	if err := os.WriteFile(filepath.Join(dir, GuardFile), []byte("reach = \"space\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadGuard(dir); err == nil {
		t.Error("an unknown reach loads")
	}
}

// The default keeps the hash earlier programs compute; only spaces changes it.
func TestFingerprintReach(t *testing.T) {
	c := &Config{Spaces: map[string]*Space{}}
	base := c.GuardFingerprint(Guard{})
	if got := c.GuardFingerprint(Guard{Reach: ReachMachine}); got != base {
		t.Error("machine changes the hash")
	}
	if got := c.GuardFingerprint(Guard{Reach: ReachSpaces}); got == base {
		t.Error("spaces leaves the hash as it was")
	}
}

func TestInSpace(t *testing.T) {
	home, _ := filepath.EvalSymlinks(t.TempDir())
	t.Setenv("HOME", home)
	cfgDir := t.TempDir()
	write(t, cfgDir, `
[spaces.work]
memory = "~/notes/work"
work_folders = ["~/work"]
`)
	c, err := Load(cfgDir)
	if err != nil {
		t.Fatal(err)
	}
	for rel, want := range map[string]bool{"work": true, "work/app": true, "notes/work": true,
		"notes/work/tasks": true, "notes": false, "elsewhere": false, "": false, "workshop": false} {
		if got := c.InSpace(filepath.Join(home, rel)); got != want {
			t.Errorf("InSpace(%q) = %v, want %v", rel, got, want)
		}
	}
	if got := c.SpaceDirs(); len(got) != 2 {
		t.Errorf("SpaceDirs = %v", got)
	}
}

// doctor and the session start name unknown settings of guard.toml: reach is known.
func TestReachIsAKnownSetting(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, GuardFile), []byte("reach = \"spaces\"\nreech = 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	us := UnknownSettings(dir)
	if len(us) != 1 || us[0].Key != "reech" {
		t.Errorf("unknown settings: %+v", us)
	}
}
