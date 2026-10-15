package backup

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/memory"
	"github.com/icalvete/augmented-brain/internal/setup"
)

// The whole story: a space on one machine is backed up, the machine is lost, and the
// space comes back on a new one with its notes, configuration and profile.
func TestBackupAndRestore(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	root := t.TempDir()
	remote := filepath.Join(root, "remote.git")
	if out, err := exec.Command("git", "init", "-q", "--bare", "-b", "main", remote).CombinedOutput(); err != nil {
		t.Fatal(err, string(out))
	}

	// Machine one.
	home1 := filepath.Join(root, "home1")
	t.Setenv("HOME", home1)
	if err := os.MkdirAll(filepath.Join(home1, "work"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg1 := filepath.Join(home1, "cfg")
	if _, err := setup.Init(cfg1, setup.Options{Language: "es", Space: "work", WorkFolder: "~/work", Memory: "~/brain"}); err != nil {
		t.Fatal(err)
	}
	c, _ := config.Load(cfg1)
	s := c.Spaces["work"]
	if _, err := Run(c, s, "one"); err == nil || !strings.Contains(err.Error(), "backup setup") {
		t.Errorf("a backup with no repository must explain how to set it up: %v", err)
	}
	if err := Setup(c, s, remote); err != nil {
		t.Fatal(err)
	}
	note := "---\nid: lesson-x\ntype: lesson\ntitle: Something learned\n---\nBody.\n"
	if err := os.WriteFile(filepath.Join(s.Memory, "knowledge", "lesson-x.md"), []byte(note), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := Run(c, s, "one")
	if err != nil || !res.Committed || !res.Pushed {
		t.Fatalf("first backup: %+v %v", res, err)
	}
	if res, err := Run(c, s, "one"); err != nil || res.Committed {
		t.Errorf("a backup with no changes must not commit: %+v %v", res, err)
	}

	// A secret stops the backup.
	secret := "-----BEGIN " + "PRIVATE KEY-----"
	leak := filepath.Join(s.Memory, "knowledge", "leak.md")
	if err := os.WriteFile(leak, []byte(secret), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(c, s, "one"); err == nil || !strings.Contains(err.Error(), "leak.md") {
		t.Errorf("a secret was not stopped: %v", err)
	}
	os.Remove(leak)

	// Machine two, empty.
	home2 := filepath.Join(root, "home2")
	t.Setenv("HOME", home2)
	cfg2 := filepath.Join(home2, "cfg")
	got, err := Restore(cfg2, RestoreOptions{URL: remote})
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "work" || got.Memory != filepath.Join(home2, "augmented-brain", "work") {
		t.Errorf("restored space: %+v", got)
	}
	c2, err := config.Load(cfg2)
	if err != nil {
		t.Fatal(err)
	}
	if c2.Language != "es" || c2.Spaces["work"].Backup != remote {
		t.Errorf("restored config: %+v %+v", c2, c2.Spaces["work"])
	}
	if s2 := c2.SpaceFor(filepath.Join(home2, "work")); s2 == nil || s2.Name != "work" {
		t.Error("the work folder does not select the restored space")
	}
	if _, err := os.Stat(filepath.Join(cfg2, "profile.md")); err != nil {
		t.Error("profile not restored")
	}
	m, problems, err := memory.Load(got.Memory)
	if err != nil || len(problems) > 0 {
		t.Fatal(err, problems)
	}
	if _, ok := m.Notes["lesson-x"]; !ok {
		t.Errorf("notes not restored: %v", m.IDs())
	}

	// The restored space keeps backing up to the same repository.
	if err := os.WriteFile(filepath.Join(got.Memory, "inbox.md"), []byte("# Buzón\n\n- 2026-10-15 — from machine two\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if res, err := Run(c2, c2.Spaces["work"], "two"); err != nil || !res.Pushed {
		t.Errorf("backup after restore: %+v %v", res, err)
	}

	// Restoring the same space twice is refused.
	if _, err := Restore(cfg2, RestoreOptions{URL: remote, Memory: filepath.Join(home2, "other")}); err == nil {
		t.Error("a second restore of the same space was accepted")
	}
}
