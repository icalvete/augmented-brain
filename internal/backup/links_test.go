package backup

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Found in the review of 2026-11-03: a backup someone changed by hand could hold a
// symbolic link (inbox.md -> ~/.bashrc), so a capture would land outside the notes. A
// backup with a link is never restored, and an address that is an option is refused.
func TestRestoreRefusesLinks(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	if runtime.GOOS == "windows" {
		t.Skip("symbolic links need extra rights on Windows")
	}
	root := t.TempDir()
	t.Setenv("HOME", filepath.Join(root, "home"))
	repo := filepath.Join(root, "evil")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../../victim.txt", filepath.Join(repo, "inbox.md")); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"init", "-q", "-b", "main"}, {"add", "-A"}, append(append([]string{}, ident...), "com"+"mit", "-q", "-m", "x")} {
		if out, err := git(repo, args...); err != nil {
			t.Fatal(err, out)
		}
	}
	_, err := Restore(filepath.Join(root, "cfg"), RestoreOptions{URL: repo})
	if err == nil || !strings.Contains(err.Error(), "symbolic links (inbox.md)") {
		t.Errorf("a backup with a link: %v", err)
	}
	if _, err := Restore(filepath.Join(root, "cfg"), RestoreOptions{URL: "--upload-pack=touch x"}); err == nil || !strings.Contains(err.Error(), "not the address") {
		t.Errorf("an address that is an option: %v", err)
	}
}
