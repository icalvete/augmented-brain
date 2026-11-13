package guard

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// What the guard reads of a git worktree, without running git, it reads where git
// does: HEAD in the worktree's own folder, the config (aliases, remotes) in its
// repository's. A real worktree, made by git.
func TestGitWorktreeReads(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	gitRun := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=t", "-c", "user.email=t@example.com"}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	repo := filepath.Join(home, "app")
	tree := filepath.Join(home, "app-fix")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	gitRun(repo, "init", "-q", "-b", "main")
	gitRun(repo, "commit", "-q", "--allow-empty", "-m", "start")
	gitRun(repo, "remote", "add", "origin", "https://example.com/me/app.git")
	gitRun(repo, "config", "alias.up", "pu"+"sh") // spelled apart, as in the other tests
	gitRun(repo, "worktree", "add", "-q", "-b", "fix", tree)

	own, common, ok := gitDirs(tree)
	if !ok || own == common {
		t.Fatalf("gitDirs(worktree) = %q, %q, %v: want its own folder and the repository's", own, common, ok)
	}
	if got := filepath.Clean(common); got != filepath.Join(repo, ".git") {
		t.Errorf("common = %q, want %q", got, filepath.Join(repo, ".git"))
	}
	if got := headBranch(tree); got != "fix" {
		t.Errorf("headBranch(worktree) = %q, want fix", got)
	}
	if got := headBranch(repo); got != "main" {
		t.Errorf("headBranch(repo) = %q, want main", got)
	}
	if got := remoteURLs(tree); !slices.Equal(got, []string{"https://example.com/me/app.git"}) {
		t.Errorf("remoteURLs(worktree) = %q", got)
	}
	if got := mainRepo(tree); got != repo {
		t.Errorf("mainRepo(worktree) = %q, want %q", got, repo)
	}
	if got := mainRepo(repo); got != "" {
		t.Errorf("mainRepo(repo) = %q, want empty", got)
	}
	if got := gitDirOf(tree); filepath.Clean(got) != filepath.Join(repo, ".git") {
		t.Errorf("gitDirOf(worktree) = %q", got)
	}
	// The repository's alias counts in its worktree: up is a push there.
	p := Policy{Cwd: home}
	treeCmd := filepath.ToSlash(tree)
	if got := p.Command("git -C " + treeCmd + " up").Verdict; got != Deny {
		t.Errorf("an alias of the repository, in its worktree: %v, want Deny", got)
	}
	if got := p.Command("git -C " + treeCmd + " status").Verdict; got != Allow {
		t.Errorf("git status in the worktree: %v, want Allow", got)
	}

	// No .git, or a .git file that does not say where: nothing to read.
	if _, _, ok := gitDirs(home); ok {
		t.Error("gitDirs(no .git) is ok")
	}
	odd := filepath.Join(home, "odd")
	if err := os.MkdirAll(odd, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(odd, ".git"), []byte("not a gitdir line\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, ok := gitDirs(odd); ok {
		t.Error("gitDirs(.git without gitdir:) is ok")
	}
	if got := gitDirOf(odd); got != filepath.Join(odd, ".git") {
		t.Errorf("gitDirOf falls back to %q", got)
	}
	if headBranch(odd) != "" || remoteURLs(odd) != nil || mainRepo(odd) != "" {
		t.Error("an odd .git file was read")
	}
}

// One reader of git's config for the aliases and the remotes.
func TestReadGitConfig(t *testing.T) {
	text := `# a comment
[core]
	bare = false
[alias] ci = commit -v
[alias]
	st = status ; a comment
	lg = "log --oneline" # another
	flag
[remote "origin"]
	url = https://example.com/a.git
[remote.old]
	url = https://example.com/old.git
[remote "inline"] url = https://example.com/inline.git
[remoteish]
	url = https://example.com/not.git
[broken
	url = https://example.com/broken.git
`
	var got []string
	readGitConfig(strings.NewReader(text), func(section, key, value string) {
		got = append(got, section+"|"+key+"|"+value)
	})
	want := []string{"core|bare|false", "alias|ci|commit -v", "alias|st|status", "alias|lg|log --oneline",
		`remote "origin"|url|https://example.com/a.git`, "remote.old|url|https://example.com/old.git",
		`remote "inline"|url|https://example.com/inline.git`, "remoteish|url|https://example.com/not.git",
		"|url|https://example.com/broken.git"}
	if !slices.Equal(got, want) {
		t.Errorf("readGitConfig:\n got %q\nwant %q", got, want)
	}
	if v, ok := parseAlias(strings.NewReader(text), "CI"); !ok || v != "commit -v" {
		t.Errorf("parseAlias(ci) = %q, %v", v, ok)
	}
	if _, ok := parseAlias(strings.NewReader(text), "flag"); ok {
		t.Error("a key with no value is not an alias")
	}
	urls := parseRemotes(strings.NewReader(text))
	wantURLs := []string{"https://example.com/a.git", "https://example.com/old.git", "https://example.com/inline.git"}
	if !slices.Equal(urls, wantURLs) {
		t.Errorf("parseRemotes = %q, want %q", urls, wantURLs)
	}
}
