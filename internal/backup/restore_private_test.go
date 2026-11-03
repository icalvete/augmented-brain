package backup

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/setup"
)

// Found in the rehearsal of 0.5.9: a restore left the notes as the umask made them
// (664 in 775), and said nothing of the repositories it did not bring back.
func TestRestorePrivateAndRepos(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	root := t.TempDir()
	remote := filepath.Join(root, "remote.git")
	if out, err := exec.Command("git", "init", "-q", "--bare", "-b", "main", remote).CombinedOutput(); err != nil {
		t.Fatal(err, string(out))
	}
	home1 := filepath.Join(root, "h1")
	t.Setenv("HOME", home1)
	if err := os.MkdirAll(filepath.Join(home1, "work", "site"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg1 := filepath.Join(home1, "cfg")
	if _, err := setup.Init(cfg1, setup.Options{Space: "work", WorkFolder: "~/work", Memory: "~/brain"}); err != nil {
		t.Fatal(err)
	}
	if err := config.SetSpaceList(cfg1, "work", "git_allowed_repos", []string{"~/work/site"}); err != nil {
		t.Fatal(err)
	}
	c1, _ := config.Load(cfg1)
	if err := Setup(c1, c1.Spaces["work"], remote); err != nil {
		t.Fatal(err)
	}
	c1, _ = config.Load(cfg1)
	if _, err := Run(c1, c1.Spaces["work"], "one"); err != nil {
		t.Fatal(err)
	}

	home2 := filepath.Join(root, "h2")
	t.Setenv("HOME", home2)
	cfg2 := filepath.Join(home2, "cfg")
	s, err := Restore(cfg2, RestoreOptions{URL: remote})
	if err != nil {
		t.Fatal(err)
	}
	if len(s.NotRestored) != 1 || filepath.Base(s.NotRestored[0]) != "site" {
		t.Errorf("the repositories not brought back: %v", s.NotRestored)
	}
	c2, _ := config.Load(cfg2)
	if r := c2.Spaces["work"].GitAllowedRepos; len(r) != 0 {
		t.Errorf("a restore brought back git_allowed_repos: %v", r)
	}
	if runtime.GOOS == "windows" {
		return
	}
	for _, p := range []string{s.Memory, filepath.Join(s.Memory, "MEMORY.md")} {
		fi, err := os.Stat(p)
		if err != nil {
			t.Fatal(err)
		}
		if fi.Mode().Perm()&0o077 != 0 {
			t.Errorf("%s is %v after a restore, readable by others", p, fi.Mode().Perm())
		}
	}
}
