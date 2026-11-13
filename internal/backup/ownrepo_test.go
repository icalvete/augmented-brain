package backup

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/setup"
)

// Notes that lost their .git inside another repository: the backup refuses, and that
// repository gets no commit and no push (F-14 of the November 2026 review).
func TestBackupOnlyInItsOwnRepository(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	root, _ := filepath.EvalSymlinks(t.TempDir())
	run := func(dir string, args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_CEILING_DIRECTORIES="+root)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatal(args, err, string(out))
		}
		return strings.TrimSpace(string(out))
	}

	// The outer repository: the work folder, with a remote and changes not saved.
	outerRemote := filepath.Join(root, "outer.git")
	run(root, "init", "-q", "--bare", "-b", "main", outerRemote)
	home := filepath.Join(root, "home")
	work := filepath.Join(home, "work")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	run(work, "init", "-q", "-b", "main")
	run(work, "remote", "add", "origin", outerRemote)
	if err := os.WriteFile(filepath.Join(work, "code.txt"), []byte("v1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run(work, "add", "-A")
	run(work, "-c", "user.name=t", "-c", "user.email=t@example.invalid", "commit", "-q", "-m", "first")
	run(work, "push", "-q", "origin", "main")
	if err := os.WriteFile(filepath.Join(work, "code.txt"), []byte("v2, not saved\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	headBefore := run(work, "rev-parse", "HEAD")
	remoteBefore := run(outerRemote, "rev-parse", "main")

	// The notes inside it, set up with a backup, then their .git gone.
	t.Setenv("HOME", home)
	cfg := filepath.Join(home, "cfg")
	if _, err := setup.Init(cfg, setup.Options{Language: "en", Space: "work", WorkFolder: "~/work", Memory: "~/work/notes"}); err != nil {
		t.Fatal(err)
	}
	c, _ := config.Load(cfg)
	s := c.Spaces["work"]
	backupRemote := filepath.Join(root, "notes.git")
	run(root, "init", "-q", "--bare", "-b", "main", backupRemote)
	if err := Setup(c, s, backupRemote); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(c, s, "one"); err != nil {
		t.Fatalf("a backup in its own repository: %v", err)
	}
	if err := os.RemoveAll(filepath.Join(s.Memory, ".git")); err != nil {
		t.Fatal(err)
	}

	_, err := Run(c, s, "one")
	if err == nil || !strings.Contains(err.Error(), "backup setup") {
		t.Errorf("a backup of notes with no .git of their own must refuse and say how to fix it: %v", err)
	}
	if _, err := GoBack(c, s, "HEAD", "one"); err == nil || !strings.Contains(err.Error(), "backup setup") {
		t.Errorf("going back with no .git of their own must refuse too: %v", err)
	}
	if h := run(work, "rev-parse", "HEAD"); h != headBefore {
		t.Errorf("the outer repository got a commit: %s", run(work, "log", "--oneline", "-3"))
	}
	if r := run(outerRemote, "rev-parse", "main"); r != remoteBefore {
		t.Errorf("the outer repository's remote got a push")
	}
	if st := run(work, "status", "--porcelain"); !strings.Contains(st, "code.txt") || strings.Contains(st, "A ") {
		t.Errorf("the outer repository's changes must stay as they were, unstaged: %q", st)
	}

	// A .git that is there but belongs to the outer repository's subfolder is refused too.
	if err := os.WriteFile(filepath.Join(s.Memory, ".git"), []byte("gitdir: "+filepath.Join(work, ".git")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(c, s, "one"); err == nil {
		t.Error("notes whose .git points to another repository must not be backed up")
	}
	if h := run(work, "rev-parse", "HEAD"); h != headBefore {
		t.Error("the outer repository got a commit through a borrowed .git")
	}
}
