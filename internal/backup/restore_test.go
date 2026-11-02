package backup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/setup"
)

// backedUp makes a space work on an old machine under root, backs it up to a bare
// repository and returns that repository.
func backedUp(t *testing.T, root string) string {
	t.Helper()
	remote := bare(t, root)
	t.Setenv("HOME", filepath.Join(root, "old"))
	cfg := filepath.Join(root, "old", "cfg")
	if _, err := setup.Init(cfg, setup.Options{Space: "work", WorkFolder: "~/work", Memory: "~/brain"}); err != nil {
		t.Fatal(err)
	}
	c, _ := config.Load(cfg)
	if err := Setup(c, c.Spaces["work"], remote); err != nil {
		t.Fatal(err)
	}
	c, _ = config.Load(cfg)
	if _, err := Run(c, c.Spaces["work"], "old"); err != nil {
		t.Fatal(err)
	}
	return remote
}

// leftovers lists the restore's own folders (".<name>.restoring", ".<name>.before-restore")
// left in dir.
func leftovers(t *testing.T, dir string) []string {
	t.Helper()
	entries, _ := os.ReadDir(dir)
	var left []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".restoring") || strings.HasSuffix(e.Name(), ".before-restore") {
			left = append(left, e.Name())
		}
	}
	return left
}

// A repository that is not a backup, or not there at all, is refused before anything
// is written on this machine.
func TestRestoreNotABackup(t *testing.T) {
	needGit(t)
	root, _ := filepath.EvalSymlinks(t.TempDir())
	t.Setenv("HOME", filepath.Join(root, "home"))

	// A repository with a file, but no saved configuration.
	other := bare(t, filepath.Join(root, "other"))
	work := filepath.Join(root, "work")
	if _, err := runGit("clone", "-q", other, work); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(work, "README.md"), []byte("not notes\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "-A"}, append(ident, "com"+"mit", "-q", "-m", "x"), {"push", "-q", "origin", "HEAD:main"}} {
		if _, err := git(work, args...); err != nil {
			t.Fatal(err)
		}
	}

	cfg := filepath.Join(root, "home", "cfg")
	mem := filepath.Join(root, "home", "notes")
	for url, want := range map[string]string{
		other:                              "is not an augmented brain backup",
		filepath.Join(root, "nowhere.git"): "cloning",
	} {
		_, err := Restore(cfg, RestoreOptions{URL: url, Memory: mem})
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: want an error with %q, got %v", url, want, err)
		}
		for _, p := range []string{cfg, mem} {
			if _, err := os.Stat(p); !os.IsNotExist(err) {
				t.Errorf("%s: %s was written", url, p)
			}
		}
	}
}

// A memory folder that already holds files is never restored into: the restore stops,
// and the folder, the configuration and the space already there stay as they were.
func TestRestoreIntoAFolderWithNotes(t *testing.T) {
	needGit(t)
	root, _ := filepath.EvalSymlinks(t.TempDir())
	remote := backedUp(t, root)

	home := filepath.Join(root, "new")
	t.Setenv("HOME", home)
	cfg := filepath.Join(home, "cfg")
	if _, err := setup.Init(cfg, setup.Options{Space: "work", WorkFolder: "~/work", Memory: "~/brain"}); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(filepath.Join(cfg, config.FileName))
	mem := filepath.Join(home, "elsewhere")
	if err := os.MkdirAll(mem, 0o755); err != nil {
		t.Fatal(err)
	}
	mine := filepath.Join(mem, "mine.md")
	if err := os.WriteFile(mine, []byte("my own notes\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Restore(cfg, RestoreOptions{URL: remote, Memory: mem}); err == nil || !strings.Contains(err.Error(), "not empty") {
		t.Fatalf("a folder with notes was restored into: %v", err)
	}
	if b, _ := os.ReadFile(mine); string(b) != "my own notes\n" {
		t.Errorf("the user's file was changed: %q", b)
	}
	if entries, _ := os.ReadDir(mem); len(entries) != 1 {
		t.Errorf("the folder was changed: %v", entries)
	}
	if after, _ := os.ReadFile(filepath.Join(cfg, config.FileName)); string(after) != string(before) {
		t.Errorf("the configuration was changed:\n%s", after)
	}
	if !Pristine(filepath.Join(home, "brain")) {
		t.Error("the space already there was touched")
	}
	if left := leftovers(t, home); len(left) > 0 {
		t.Errorf("left behind: %v", left)
	}
}

// A restore into a new folder leaves a repository that backs up to the same URL, and
// nothing of its own behind.
func TestRestoreRoundTrip(t *testing.T) {
	needGit(t)
	root, _ := filepath.EvalSymlinks(t.TempDir())
	remote := backedUp(t, root)

	home := filepath.Join(root, "new")
	t.Setenv("HOME", home)
	cfg := filepath.Join(home, "cfg")
	mem := filepath.Join(home, "restored")
	got, err := Restore(cfg, RestoreOptions{URL: remote, Memory: mem})
	if err != nil {
		t.Fatal(err)
	}
	if got.Memory != mem || got.Backup != remote {
		t.Errorf("restored space: %+v", got)
	}
	if out, err := git(mem, "remote", "get-url", "origin"); err != nil || strings.TrimSpace(out) != remote {
		t.Errorf("origin is %q, want %s (%v)", strings.TrimSpace(out), remote, err)
	}
	if left := leftovers(t, home); len(left) > 0 {
		t.Errorf("left behind: %v", left)
	}
	c, err := config.Load(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Run(c, c.Spaces["work"], "new"); err != nil {
		t.Errorf("backup after the restore: %v", err)
	}
}
