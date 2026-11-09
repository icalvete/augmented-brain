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
