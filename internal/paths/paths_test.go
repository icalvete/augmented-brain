package paths

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// The macOS case, reproduced anywhere: a folder reached through a symlink, and paths
// below it that do not exist yet, must resolve to the same real place.
func TestResolveThroughASymlink(t *testing.T) {
	root := t.TempDir()
	real := filepath.Join(root, "private", "var")
	if err := os.MkdirAll(real, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "var")
	if err := os.Symlink(real, link); err != nil {
		if runtime.GOOS == "windows" {
			t.Skip("symlinks need Developer Mode or an administrator on Windows:", err)
		}
		t.Fatal(err)
	}
	realRoot, _ := filepath.EvalSymlinks(root)

	exists, _ := Resolve(link)
	missing, _ := Resolve(filepath.Join(link, "work", "project", "src"))
	if exists != filepath.Join(realRoot, "private", "var") {
		t.Errorf("existing: %s", exists)
	}
	if missing != filepath.Join(realRoot, "private", "var", "work", "project", "src") {
		t.Errorf("missing: %s", missing)
	}
	if !Within(missing, exists) {
		t.Error("a missing folder under a symlinked one is not inside it")
	}

	t.Setenv("HOME", link)
	t.Setenv("USERPROFILE", link) // where Windows takes the home folder from
	if got := Tilde(filepath.Join(link, "brain")); got != "~/brain" {
		t.Errorf("Tilde through the link = %s", got)
	}
	if got := Tilde(filepath.Join(real, "brain")); got != "~/brain" {
		t.Errorf("Tilde of the real path = %s", got)
	}
	if got, _ := Resolve("~/x"); got != filepath.Join(realRoot, "private", "var", "x") {
		t.Errorf("~ = %s", got)
	}
}

func TestSame(t *testing.T) {
	a := filepath.Join(t.TempDir(), "notes")
	if !Same(a, a) || Same(a, filepath.Join(a, "sub")) || Same(a, a+"x") {
		t.Error("Same: wrong answer for the same, a child and a sibling folder")
	}
	if runtime.GOOS == "windows" && !Same(a, strings.ToUpper(a)) {
		t.Error("on Windows, the same folder in other case must be the same")
	}
}
