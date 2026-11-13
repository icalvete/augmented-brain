package backup

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/setup"
)

// Built in pieces, so this file holds no token-shaped text.
var token = "ghp" + "_" + strings.Repeat("a1B2", 9)

// Found in the review of 2026-11-13 (S-06): a password or a token in the address was
// kept as it came, in config.toml and inside the backup itself.
func TestAddressWithPassword(t *testing.T) {
	for _, c := range []struct{ in, clean string }{
		{"https://ana:Sup3rS3cret@127.0.0.1:18443/u/repo.git", "https://ana@127.0.0.1:18443/u/repo.git"},
		{"https://x:" + token + "@github.com/u/r.git", "https://x@github.com/u/r.git"},
		{"https://" + token + "@github.com/u/r.git", "https://github.com/u/r.git"},
		{"https://ana:@h/r.git", "https://ana@h/r.git"},
		{token + "@github.com:u/r.git", "github.com:u/r.git"},
	} {
		if !HasPassword(c.in) {
			t.Errorf("HasPassword(%s) = false", c.in)
		}
		if got := WithoutPassword(c.in); got != c.clean {
			t.Errorf("WithoutPassword(%s) = %s, want %s", c.in, got, c.clean)
		}
		if err := checkURL(c.in); err == nil || strings.Contains(err.Error(), "Sup3rS3cret") || strings.Contains(err.Error(), token) {
			t.Errorf("checkURL(%s) = %v", c.in, err)
		}
	}
	for _, ok := range []string{"https://github.com/u/r.git", "https://ana@h/r.git", "git@gitlab.com:lucia/brain.git",
		"ssh://git@h:22/r.git", "/srv/backup.git", `C:\backup\work.git`, "file:///srv/backup.git"} {
		if HasPassword(ok) || WithoutPassword(ok) != ok || checkURL(ok) != nil {
			t.Errorf("%s: HasPassword %v, WithoutPassword %s, checkURL %v", ok, HasPassword(ok), WithoutPassword(ok), checkURL(ok))
		}
	}
}

// Setup refuses such an address and writes nothing; a backup of a space whose
// configuration already holds one (written by an earlier version) carries it without
// the password.
func TestSetupRefusesAPassword(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, "work"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(home, "cfg")
	if _, err := setup.Init(cfg, setup.Options{Space: "work", WorkFolder: "~/work", Memory: "~/brain"}); err != nil {
		t.Fatal(err)
	}
	c, _ := config.Load(cfg)
	s := c.Spaces["work"]
	if err := Setup(c, s, "https://ana:Sup3rS3cret@example.com/u/r.git"); err == nil || !strings.Contains(err.Error(), "gh auth setup-git") {
		t.Errorf("an address with a password was accepted: %v", err)
	}
	if b, _ := os.ReadFile(filepath.Join(cfg, config.FileName)); strings.Contains(string(b), "backup") {
		t.Errorf("config.toml gained a backup:\n%s", b)
	}
	if _, err := os.Stat(filepath.Join(s.Memory, ".git")); err == nil {
		t.Error("the notes became a repository")
	}

	s.Backup = "https://ana:Sup3rS3cret@example.com/u/r.git"
	if err := writeMeta(c, s); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(s.Memory, MetaDir, config.FileName))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "Sup3rS3cret") || !strings.Contains(string(b), "https://ana@example.com/u/r.git") {
		t.Errorf("the backup's copy of the configuration:\n%s", b)
	}
	if s.Backup != "https://ana:Sup3rS3cret@example.com/u/r.git" {
		t.Errorf("writeMeta changed the space: %s", s.Backup)
	}
}
