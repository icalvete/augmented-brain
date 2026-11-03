package repostat

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

func mkdirs(t *testing.T, root string, dirs ...string) {
	t.Helper()
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(d)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
}

func rels(t *testing.T, root string, found []string) []string {
	t.Helper()
	var out []string
	for _, p := range found {
		r, err := filepath.Rel(root, p)
		if err != nil || strings.HasPrefix(r, "..") {
			t.Fatalf("%s is outside %s", p, root)
		}
		out = append(out, filepath.ToSlash(r))
	}
	slices.Sort(out)
	return out
}

// Find finds nested repositories, stops at the first on each path, skips heavy and
// hidden folders, and keeps to its depth.
func TestFind(t *testing.T) {
	root := t.TempDir()
	mkdirs(t, root,
		"a/.git",                      // depth 1
		"a/inner/.git",                // inside a repository: not looked for
		"group/b/.git",                // depth 2
		"group/sub/c/.git",            // depth 3
		"group/sub/deep/d/.git",       // depth 4: too deep
		"node_modules/pkg/.git",       // dependencies
		"vendor/lib/.git",             // dependencies
		".hidden/e/.git",              // hidden
		"skipme/f/.git",               // the notes, say
		"plain/folder/without/a/repo", // nothing
	)
	// A worktree or submodule has a .git file.
	mkdirs(t, root, "wt")
	if err := os.WriteFile(filepath.Join(root, "wt", ".git"), []byte("gitdir: /elsewhere\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	skip := func(p string) bool { return p == filepath.Join(root, "skipme") }
	found, more := Find(root, 3, 100, skip)
	if got, want := rels(t, root, found), []string{"a", "group/b", "group/sub/c", "wt"}; !slices.Equal(got, want) || more {
		t.Errorf("Find = %v (more %v), want %v", got, more, want)
	}
	if found, _ := Find(root, 1, 100, skip); !slices.Equal(rels(t, root, found), []string{"a", "wt"}) {
		t.Errorf("depth 1: %v", rels(t, root, found))
	}
	// The root itself is not one of its repositories.
	mkdirs(t, root, ".git")
	if found, _ := Find(root, 3, 100, skip); slices.Contains(found, root) {
		t.Error("the root was listed")
	}
}

// At most limit repositories, shallowest first, and it says it stopped.
func TestFindLimit(t *testing.T) {
	root := t.TempDir()
	mkdirs(t, root, "x/deep/.git", "r1/.git", "r2/.git", "r3/.git")
	found, more := Find(root, 3, 2, nil)
	if len(found) != 2 || !more {
		t.Fatalf("Find = %v, %v", found, more)
	}
	for _, p := range found {
		if filepath.Base(filepath.Dir(p)) == "x" {
			t.Errorf("a deeper repository came before shallower ones: %v", found)
		}
	}
	if _, more := Find(root, 3, 4, nil); more {
		t.Error("exactly the limit is not more")
	}
}

// A link is never followed: neither one to a repository outside the folder, nor one
// to a folder holding repositories, nor a .git that is a link.
func TestFindFollowsNoLinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symbolic links need privileges on Windows")
	}
	root, outside := t.TempDir(), t.TempDir()
	mkdirs(t, outside, "repo/.git", "group/other/.git")
	mkdirs(t, root, "mine/.git", "fake")
	for name, target := range map[string]string{
		"link-repo":  filepath.Join(outside, "repo"),
		"link-group": filepath.Join(outside, "group"),
	} {
		if err := os.Symlink(target, filepath.Join(root, name)); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(filepath.Join(outside, "repo", ".git"), filepath.Join(root, "fake", ".git")); err != nil {
		t.Fatal(err)
	}
	found, _ := Find(root, 3, 100, nil)
	if got := rels(t, root, found); !slices.Equal(got, []string{"mine"}) {
		t.Errorf("Find = %v, want only mine", got)
	}
}

func TestCleanRemote(t *testing.T) {
	for raw, want := range map[string]string{
		"https://user:s3cret@example.com/me/site.git":     "example.com/me/site",
		"https://token@example.com:8443/me/site?x=1#frag": "example.com/me/site",
		"git@example.com:me/site.git":                     "example.com/me/site",
		"ssh://git@example.com:22/group/sub/site.git\n":   "example.com/group/sub/site",
		"example.com:me/site":                             "example.com/me/site",
		"/srv/git/site.git":                               "site",
		"file:///srv/git/site.git":                        "site",
		"":                                                "",
		"https://a b@%zz/":                                "",
	} {
		if got := CleanRemote(raw); got != want {
			t.Errorf("CleanRemote(%q) = %q, want %q", raw, got, want)
		}
		if got := CleanRemote(raw); strings.Contains(got, "s3cret") || strings.Contains(got, "@") {
			t.Errorf("CleanRemote(%q) kept a credential: %q", raw, got)
		}
	}
}
