package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestReleaseCommit checks scripts/release-commit.sh, which picks the commit of main a
// release is made from: the one that set VERSION or last changed that version's section
// of the changelog, never a pull request merged later with its line under Unreleased.
// The history copies 0.22.0-beta's: the release merged (#130), then a change it should
// include (#129), then the fix of the changelog that says so (#131), then the next
// change (#132), which waits for the next version.
func TestReleaseCommit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("a shell script")
	}
	script, err := filepath.Abs(filepath.Join("..", "..", "scripts", "release-commit.sh"))
	if err != nil {
		t.Fatal(err)
	}
	repo := t.TempDir()
	git := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@example.com",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@example.com", "GIT_CONFIG_NOSYSTEM=1", "HOME="+repo)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	write := func(name, text string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(repo, name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	changelog := func(unreleased, v2 string) string {
		return "# Changelog\n\n## Unreleased\n" + unreleased + "\n## 2.0\n" + v2 + "\n## 1.0\n- first\n"
	}
	// merge makes a pull request's merge commit on main: a branch with one commit.
	merge := func(name string, files map[string]string) string {
		t.Helper()
		git("checkout", "-q", "-b", name)
		for f, text := range files {
			write(f, text)
		}
		git("add", "-A")
		git("commit", "-q", "-m", name)
		git("checkout", "-q", "main")
		git("merge", "-q", "--no-ff", "-m", "Merge "+name, name)
		return git("rev-parse", "HEAD")
	}
	release := func(at string) string {
		t.Helper()
		cmd := exec.Command("sh", script, at)
		cmd.Dir = repo
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("release-commit.sh %s: %v\n%s", at, err, out)
		}
		return strings.TrimSpace(string(out))
	}

	git("init", "-q", "-b", "main")
	write("VERSION", "1.0\n")
	write("CHANGELOG.md", "# Changelog\n\n## Unreleased\n\n## 1.0\n- first\n")
	git("add", "-A")
	git("commit", "-q", "-m", "1.0")

	merge("feature-a", map[string]string{"a.go": "a\n"})
	v2 := merge("release-2", map[string]string{"VERSION": "2.0\n", "CHANGELOG.md": changelog("", "- a\n")})
	late := merge("feature-b", map[string]string{"b.go": "b\n"})
	if got := release(late); got != v2 {
		t.Errorf("a change merged after the release, its changelog untouched: got %s, want the release's commit %s", got, v2)
	}
	fix := merge("changelog-fix", map[string]string{"CHANGELOG.md": changelog("", "- a\n- b\n")})
	next := merge("feature-c", map[string]string{"c.go": "c\n", "CHANGELOG.md": changelog("- c\n", "- a\n- b\n")})
	if got := release(next); got != fix {
		t.Errorf("a change listed under Unreleased: got %s, want the changelog fix %s", got, fix)
	}
	if got := release(v2); got != v2 {
		t.Errorf("at the release itself: got %s, want %s", got, v2)
	}
}
