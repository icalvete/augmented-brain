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
	if got := taskFolder(s, []string{"site"}); got != "" {
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
	if got := taskFolder(s, []string{"site"}); got != work {
		t.Errorf("taskFolder = %q, want %q", got, work)
	}
}
