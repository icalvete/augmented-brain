package config

import (
	"os"
	"path/filepath"
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
	in := Guard{UntouchableRepos: []string{"/srv/core", "/srv/core", "/srv/cli-*"}}
	if err := SaveGuard(dir, in); err != nil {
		t.Fatal(err)
	}
	g, err := LoadGuard(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.UntouchableRepos) != 2 || g.UntouchableRepos[0] != "/srv/core" || g.UntouchableRepos[1] != "/srv/cli-*" {
		t.Fatalf("got %v", g.UntouchableRepos)
	}
	if fi, _ := os.Stat(filepath.Join(dir, GuardFile)); fi.Mode().Perm() != 0o600 {
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
