package backup

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/memory"
	"github.com/icalvete/augmented-brain/internal/setup"
)

// The whole story: a space on one machine is backed up, the machine is lost, and the
// space comes back on a new one with its notes, configuration and profile.
func TestBackupAndRestore(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	root, _ := filepath.EvalSymlinks(t.TempDir()) // macOS: /var is /private/var
	remote := filepath.Join(root, "remote.git")
	if out, err := exec.Command("git", "init", "-q", "--bare", "-b", "main", remote).CombinedOutput(); err != nil {
		t.Fatal(err, string(out))
	}

	// Machine one.
	home1 := filepath.Join(root, "home1")
	t.Setenv("HOME", home1)
	if err := os.MkdirAll(filepath.Join(home1, "work"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg1 := filepath.Join(home1, "cfg")
	if _, err := setup.Init(cfg1, setup.Options{Language: "es", Space: "work", WorkFolder: "~/work", Memory: "~/brain"}); err != nil {
		t.Fatal(err)
	}
	c, _ := config.Load(cfg1)
	s := c.Spaces["work"]
	if _, err := Run(c, s, "one"); err == nil || !strings.Contains(err.Error(), "backup setup") {
		t.Errorf("a backup with no repository must explain how to set it up: %v", err)
	}
	if err := Setup(c, s, remote); err != nil {
		t.Fatal(err)
	}
	note := "---\nid: lesson-x\ntype: lesson\ntitle: Something learned\n---\nBody.\n"
	if err := os.WriteFile(filepath.Join(s.Memory, "knowledge", "lesson-x.md"), []byte(note), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := Run(c, s, "one")
	if err != nil || !res.Committed || !res.Pushed {
		t.Fatalf("first backup: %+v %v", res, err)
	}
	if res, err := Run(c, s, "one"); err != nil || res.Committed {
		t.Errorf("a backup with no changes must not commit: %+v %v", res, err)
	}

	// A secret stops the backup.
	secret := "-----BEGIN " + "PRIVATE KEY-----"
	leak := filepath.Join(s.Memory, "knowledge", "leak.md")
	if err := os.WriteFile(leak, []byte(secret), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(c, s, "one"); err == nil || !strings.Contains(err.Error(), "leak.md") {
		t.Errorf("a secret was not stopped: %v", err)
	}
	os.Remove(leak)

	// Machine two, empty.
	home2 := filepath.Join(root, "home2")
	t.Setenv("HOME", home2)
	cfg2 := filepath.Join(home2, "cfg")
	got, err := Restore(cfg2, RestoreOptions{URL: remote})
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "work" || got.Memory != filepath.Join(home2, "augmented-brain", "work") {
		t.Errorf("restored space: %+v", got)
	}
	c2, err := config.Load(cfg2)
	if err != nil {
		t.Fatal(err)
	}
	if c2.Language != "es" || c2.Spaces["work"].Backup != remote {
		t.Errorf("restored config: %+v %+v", c2, c2.Spaces["work"])
	}
	if s2 := c2.SpaceFor(filepath.Join(home2, "work")); s2 == nil || s2.Name != "work" {
		t.Error("the work folder does not select the restored space")
	}
	if _, err := os.Stat(filepath.Join(cfg2, "profile.md")); err != nil {
		t.Error("profile not restored")
	}
	m, problems, err := memory.Load(got.Memory)
	if err != nil || len(problems) > 0 {
		t.Fatal(err, problems)
	}
	if _, ok := m.Notes["lesson-x"]; !ok {
		t.Errorf("notes not restored: %v", m.IDs())
	}

	// The restored space keeps backing up to the same repository.
	if err := os.WriteFile(filepath.Join(got.Memory, "inbox.md"), []byte("# Buzón\n\n- 2026-10-15 — from machine two\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if res, err := Run(c2, c2.Spaces["work"], "two"); err != nil || !res.Pushed {
		t.Errorf("backup after restore: %+v %v", res, err)
	}

	// Restoring the same space twice is refused.
	if _, err := Restore(cfg2, RestoreOptions{URL: remote, Memory: filepath.Join(home2, "other")}); err == nil {
		t.Error("a second restore of the same space was accepted")
	}
}

// Found in the review before 0.1: two machines backing up the same space. Different
// notes merge (the generated index too); the same note changed on both stops the backup.
func TestTwoMachines(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	root := t.TempDir()
	remote := filepath.Join(root, "remote.git")
	if out, err := exec.Command("git", "init", "-q", "--bare", "-b", "main", remote).CombinedOutput(); err != nil {
		t.Fatal(err, string(out))
	}
	home1 := filepath.Join(root, "h1")
	t.Setenv("HOME", home1)
	if err := os.MkdirAll(filepath.Join(home1, "work"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg1 := filepath.Join(home1, "cfg")
	if _, err := setup.Init(cfg1, setup.Options{Space: "work", WorkFolder: "~/work", Memory: "~/brain"}); err != nil {
		t.Fatal(err)
	}
	c1, _ := config.Load(cfg1)
	if err := Setup(c1, c1.Spaces["work"], remote); err != nil {
		t.Fatal(err)
	}
	c1, _ = config.Load(cfg1)
	s1 := c1.Spaces["work"]
	if _, err := Run(c1, s1, "one"); err != nil {
		t.Fatal(err)
	}

	home2 := filepath.Join(root, "h2")
	t.Setenv("HOME", home2)
	cfg2 := filepath.Join(home2, "cfg")
	s2, err := Restore(cfg2, RestoreOptions{URL: remote})
	if err != nil {
		t.Fatal(err)
	}
	c2, _ := config.Load(cfg2)

	note := func(dir, id, body string) {
		t.Helper()
		text := "---\nid: " + id + "\ntype: lesson\ntitle: " + id + "\n---\n" + body + "\n"
		if err := os.WriteFile(filepath.Join(dir, "knowledge", id+".md"), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
		m, _, _ := memory.Load(dir)
		if _, err := m.WriteIndex("en"); err != nil { // both machines rebuild the index
			t.Fatal(err)
		}
	}

	// Different notes on each machine: both backups go through.
	note(s2.Memory, "lesson-from-two", "two")
	if _, err := Run(c2, s2, "two"); err != nil {
		t.Fatal(err)
	}
	note(s1.Memory, "lesson-from-one", "one")
	if res, err := Run(c1, s1, "one"); err != nil || !res.Pushed {
		t.Fatalf("machine one after machine two: %+v %v", res, err)
	}
	m1, problems, _ := memory.Load(s1.Memory)
	if len(problems) > 0 || m1.Notes["lesson-from-two"] == nil || m1.Notes["lesson-from-one"] == nil {
		t.Errorf("merge lost notes: %v %v", m1.IDs(), problems)
	}
	idx, _ := os.ReadFile(filepath.Join(s1.Memory, memory.IndexFile))
	if strings.Contains(string(idx), "<<<<<<<") {
		t.Errorf("index left with conflict markers:\n%s", idx)
	}

	// The same note changed differently on both machines: stopped, nothing touched.
	if _, err := Run(c2, s2, "two"); err != nil { // machine two catches up first
		t.Fatal(err)
	}
	note(s1.Memory, "lesson-shared", "version one")
	if _, err := Run(c1, s1, "one"); err != nil {
		t.Fatal(err)
	}
	note(s2.Memory, "lesson-shared", "version two")
	_, err = Run(c2, s2, "two")
	if err == nil || !strings.Contains(err.Error(), "knowledge/lesson-shared.md") {
		t.Fatalf("a real conflict must stop the backup and name the note: %v", err)
	}
	if b, _ := os.ReadFile(filepath.Join(s2.Memory, "knowledge", "lesson-shared.md")); !strings.Contains(string(b), "version two") {
		t.Errorf("the local note was changed:\n%s", b)
	}
	if out, _ := exec.Command("git", "-C", s2.Memory, "status", "--porcelain").Output(); strings.Contains(string(out), "UU") {
		t.Errorf("a merge was left half done:\n%s", out)
	}
}

func TestSetupKeepsAForeignOrigin(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, "work"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(home, "cfg")
	if _, err := setup.Init(cfg, setup.Options{Space: "work", WorkFolder: "~/work", Memory: "~/brain"}); err != nil {
		t.Fatal(err)
	}
	c, _ := config.Load(cfg)
	s := c.Spaces["work"]
	for _, args := range [][]string{{"init", "-q"}, {"remote", "add", "origin", "git@example.com:me/notes.git"}} {
		if out, err := exec.Command("git", append([]string{"-C", s.Memory}, args...)...).CombinedOutput(); err != nil {
			t.Fatal(err, string(out))
		}
	}
	if err := Setup(c, s, filepath.Join(home, "backup.git")); err == nil || !strings.Contains(err.Error(), "does not change it") {
		t.Errorf("a foreign origin was replaced: %v", err)
	}
	if _, err := Restore(cfg, RestoreOptions{URL: "x", Memory: filepath.Join(cfg, "config.toml")}); err == nil {
		t.Error("restore over a file was accepted")
	}
}

// The default layout keeps the notes inside the work folder; a restore puts them in the
// same place, so the new machine looks like the old one.
func TestRestoreKeepsTheLayout(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	root, _ := filepath.EvalSymlinks(t.TempDir())
	remote := filepath.Join(root, "remote.git")
	if out, err := exec.Command("git", "init", "-q", "--bare", "-b", "main", remote).CombinedOutput(); err != nil {
		t.Fatal(err, string(out))
	}
	home1 := filepath.Join(root, "home1")
	t.Setenv("HOME", home1)
	cfg1 := filepath.Join(home1, "cfg")
	o := setup.Options{Language: "es", Space: "trabajo", WorkFolder: "~/augmented-brain/trabajo", Memory: "~/augmented-brain/trabajo/notas"}
	if _, err := setup.Init(cfg1, o); err != nil {
		t.Fatal(err)
	}
	c, _ := config.Load(cfg1)
	if err := Setup(c, c.Spaces["trabajo"], remote); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(c, c.Spaces["trabajo"], "one"); err != nil {
		t.Fatal(err)
	}

	home2 := filepath.Join(root, "home2")
	t.Setenv("HOME", home2)
	got, err := Restore(filepath.Join(home2, "cfg"), RestoreOptions{URL: remote})
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(home2, "augmented-brain", "trabajo", "notas"); got.Memory != want {
		t.Errorf("notes restored in %s, want %s", got.Memory, want)
	}
	if len(got.WorkFolders) != 1 || got.WorkFolders[0] != filepath.Join(home2, "augmented-brain", "trabajo") {
		t.Errorf("work folder %v", got.WorkFolders)
	}
}

// On a new machine the installer has already made the spaces, empty: restoring one of
// them replaces it. A space with anything of the user's is never replaced.
func TestRestoreOverAFreshInstall(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	root, _ := filepath.EvalSymlinks(t.TempDir())
	remote := filepath.Join(root, "remote.git")
	if out, err := exec.Command("git", "init", "-q", "--bare", "-b", "main", remote).CombinedOutput(); err != nil {
		t.Fatal(err, string(out))
	}
	o := func(space string) setup.Options {
		return setup.Options{Language: "es", Space: space, WorkFolder: "~/augmented-brain/" + space, Memory: "~/augmented-brain/" + space + "/notas"}
	}
	// The old machine, with a note, backed up.
	t.Setenv("HOME", filepath.Join(root, "old"))
	cfg1 := filepath.Join(root, "old", "cfg")
	if _, err := setup.Init(cfg1, o("trabajo")); err != nil {
		t.Fatal(err)
	}
	c, _ := config.Load(cfg1)
	s := c.Spaces["trabajo"]
	if err := os.WriteFile(filepath.Join(s.Memory, "inbox.md"), []byte("# Buzón\n\n- 2026-10-16 — from the old machine\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Setup(c, s, remote); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(c, s, "old"); err != nil {
		t.Fatal(err)
	}

	// The new machine, freshly installed: trabajo and personal, untouched.
	home := filepath.Join(root, "new")
	t.Setenv("HOME", home)
	cfg2 := filepath.Join(home, "cfg")
	if _, err := setup.Init(cfg2, o("trabajo")); err != nil {
		t.Fatal(err)
	}
	if _, err := setup.AddSpace(cfg2, o("personal")); err != nil {
		t.Fatal(err)
	}
	if !Pristine(filepath.Join(home, "augmented-brain", "trabajo", "notas")) {
		t.Fatal("a fresh space must count as pristine")
	}
	got, err := Restore(cfg2, RestoreOptions{URL: remote})
	if err != nil {
		t.Fatal(err)
	}
	inbox, _ := os.ReadFile(filepath.Join(got.Memory, "inbox.md"))
	if !strings.Contains(string(inbox), "from the old machine") {
		t.Errorf("notes not restored over the fresh space: %s", inbox)
	}
	c2, err := config.Load(cfg2)
	if err != nil || c2.Spaces["trabajo"].Backup != remote || len(c2.Spaces) != 2 {
		t.Fatalf("config after restore: %+v %v", c2, err)
	}

	// Restoring again over a space that now holds the user's notes is refused.
	if Pristine(got.Memory) {
		t.Error("a restored space is not pristine")
	}
	if _, err := Restore(cfg2, RestoreOptions{URL: remote}); err == nil || !strings.Contains(err.Error(), "notes of its own") {
		t.Errorf("a space with notes was replaced: %v", err)
	}
}
