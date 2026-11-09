package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/icalvete/augmented-brain/internal/config"
)

// A bare repository is never a task's folder: it holds no files, and Claude does not
// trust it. On 9-nov a session was sent to one; the worktree next to it is the folder.
func TestTaskFolderSkipsBare(t *testing.T) {
	wf := t.TempDir()
	bare := filepath.Join(wf, "site")
	if err := os.MkdirAll(filepath.Join(bare, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bare, ".git", "config"), []byte("[core]\n\trepositoryformatversion = 0\n\tbare = true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := &config.Space{WorkFolders: []string{wf}}
	if got := taskFolder(s, []string{"site"}, nil); got != "" {
		t.Errorf("taskFolder chose the bare repository %s", got)
	}
	// The same repository found deeper, as a working tree, is the folder.
	work := filepath.Join(wf, "code", "site")
	if err := os.MkdirAll(filepath.Join(work, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(work, ".git", "config"), []byte("[core]\n\tbare = false\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := taskFolder(s, []string{"site"}, nil); got != work {
		t.Errorf("taskFolder = %q, want %q", got, work)
	}
}

// A bare repository's worktree is the folder: one Claude trusts first, then by path;
// one outside the space's work folders, or gone, never (9-nov: the repository only
// existed as a bare one, with worktrees named site-<branch>).
func TestTaskFolderUsesBareWorktree(t *testing.T) {
	wf := t.TempDir()
	bare := filepath.Join(wf, "site")
	if err := os.MkdirAll(filepath.Join(bare, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bare, ".git", "config"), []byte("[core]\n\tbare = true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	worktree := func(name, dir string, exists bool) string {
		wt := filepath.Join(dir, name)
		if exists {
			if err := os.MkdirAll(wt, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(wt, ".git"), []byte("gitdir: "+filepath.Join(bare, ".git", "worktrees", name)+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		rec := filepath.Join(bare, ".git", "worktrees", name)
		if err := os.MkdirAll(rec, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(rec, "gitdir"), []byte(filepath.Join(wt, ".git")+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		return wt
	}
	worktree("site-gone", wf, false)
	worktree("site-away", outside, true)
	s := &config.Space{WorkFolders: []string{wf}}
	if got := taskFolder(s, []string{"site"}, nil); got != "" {
		t.Errorf("chose %q: only a gone worktree and one outside the space", got)
	}
	a := worktree("site-a", wf, true)
	b := worktree("site-b", wf, true)
	if got := taskFolder(s, []string{"site"}, nil); got != a {
		t.Errorf("no trusted worktree: got %q, want the first by path %q", got, a)
	}
	trusted := func(p string) bool { return p == b }
	if got := taskFolder(s, []string{"site"}, trusted); got != b {
		t.Errorf("got %q, want the trusted worktree %q", got, b)
	}
}

// With every worktree trusted (trust read through the main repository), the one on the
// default branch wins, then one named after it, never a leftover feature branch by its
// name (9-nov: the session went to site-barewt, on a merged branch).
func TestTaskFolderPrefersDefaultBranch(t *testing.T) {
	wf := t.TempDir()
	bare := filepath.Join(wf, "site")
	gitDir := filepath.Join(bare, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gitDir, "config"), []byte("[core]\n\tbare = true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	add := func(name, head string) string {
		wt := filepath.Join(wf, name)
		rec := filepath.Join(gitDir, "worktrees", name)
		for _, d := range []string{wt, rec} {
			if err := os.MkdirAll(d, 0o755); err != nil {
				t.Fatal(err)
			}
		}
		for f, body := range map[string]string{
			filepath.Join(wt, ".git"):    "gitdir: " + rec + "\n",
			filepath.Join(rec, "gitdir"): filepath.Join(wt, ".git") + "\n",
			filepath.Join(rec, "HEAD"):   head + "\n",
		} {
			if err := os.WriteFile(f, []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		return wt
	}
	s := &config.Space{WorkFolders: []string{wf}}
	all := func(string) bool { return true }
	add("site-barewt", "ref: refs/heads/session-bare-worktree")
	named := add("site-main", "5949f693bc8106d0d115b6ee3f3030bd89d28909")
	if got := taskFolder(s, []string{"site"}, all); got != named {
		t.Errorf("got %q, want the worktree named after main %q", got, named)
	}
	onMain := add("site-zz", "ref: refs/heads/main")
	if got := taskFolder(s, []string{"site"}, all); got != onMain {
		t.Errorf("got %q, want the worktree on main %q", got, onMain)
	}
	// The default branch is the one origin's HEAD names, when git recorded it.
	if err := os.MkdirAll(filepath.Join(gitDir, "refs", "remotes", "origin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gitDir, "refs", "remotes", "origin", "HEAD"), []byte("ref: refs/remotes/origin/trunk\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	onTrunk := add("site-yy", "ref: refs/heads/trunk")
	if got := taskFolder(s, []string{"site"}, all); got != onTrunk {
		t.Errorf("got %q, want the worktree on trunk %q", got, onTrunk)
	}
}
