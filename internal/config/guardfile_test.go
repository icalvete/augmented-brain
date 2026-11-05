package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestGuardFileMissingIsEmpty(t *testing.T) {
	g, err := LoadGuard(t.TempDir())
	if err != nil || len(g.UntouchableRepos) != 0 {
		t.Fatalf("missing file: %v %v", g, err)
	}
}

func TestGuardFileRoundTrip(t *testing.T) {
	dir := t.TempDir()
	// Paths of this system, resolved as Load resolves them (on Windows "/srv/core" gains a drive).
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	core, cli := filepath.Join(base, "core"), filepath.Join(base, "cli-*")
	in := Guard{UntouchableRepos: []string{core, core, cli}}
	if err := SaveGuard(dir, in); err != nil {
		t.Fatal(err)
	}
	g, err := LoadGuard(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.UntouchableRepos) != 2 || g.UntouchableRepos[0] != core || g.UntouchableRepos[1] != cli {
		t.Fatalf("got %v", g.UntouchableRepos)
	}
	if fi, _ := os.Stat(filepath.Join(dir, GuardFile)); runtime.GOOS != "windows" && fi.Mode().Perm() != 0o600 { // Windows has no such modes
		t.Fatalf("mode %v", fi.Mode())
	}
}

func TestGuardFileBrokenIsAnError(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, GuardFile), []byte("untouchable_repos = ["), 0o600)
	if _, err := LoadGuard(dir); err == nil {
		t.Fatal("a file that cannot be read must be an error")
	}
}
