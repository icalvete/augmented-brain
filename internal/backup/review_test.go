package backup

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/memory"
	"github.com/icalvete/augmented-brain/internal/setup"
)

func needGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
}

func bare(t *testing.T, dir string) string {
	t.Helper()
	remote := filepath.Join(dir, "remote.git")
	if out, err := exec.Command("git", "init", "-q", "--bare", "-b", "main", remote).CombinedOutput(); err != nil {
		t.Fatal(err, string(out))
	}
	return remote
}

// twoSpaces makes work and personal, personal the default, both backed up.
func twoSpaces(t *testing.T, root string) (*config.Config, string, string) {
	t.Helper()
	t.Setenv("HOME", filepath.Join(root, "home"))
	cfg := filepath.Join(root, "home", "cfg")
	o := func(space string) setup.Options {
		return setup.Options{Space: space, WorkFolder: "~/ab/" + space, Memory: "~/ab/" + space + "/notes"}
	}
	if _, err := setup.Init(cfg, o("work")); err != nil {
		t.Fatal(err)
	}
	if _, err := setup.AddSpace(cfg, o("personal")); err != nil {
		t.Fatal(err)
	}
	if err := config.SetDefaultSpace(cfg, "personal"); err != nil {
		t.Fatal(err)
	}
	if err := config.SetTopList(cfg, "untouchable_hosts", []string{"db01"}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg, "profile.md"), []byte("---\nid: profile\ntype: profile\ntitle: Profile\n---\n- Name: Dana\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	workRemote, personalRemote := bare(t, filepath.Join(root, "w")), bare(t, filepath.Join(root, "p"))
	c, _ := config.Load(cfg)
	if err := Setup(c, c.Spaces["work"], workRemote); err != nil {
		t.Fatal(err)
	}
	if err := Setup(c, c.Spaces["personal"], personalRemote); err != nil {
		t.Fatal(err)
	}
	c, _ = config.Load(cfg)
	return c, workRemote, personalRemote
}

// inBackup reports whether the backup's main branch has the file.
func inBackup(t *testing.T, remote, path string) bool {
	t.Helper()
	out, _ := exec.Command("git", "--git-dir", remote, "ls-tree", "-r", "-z", "--name-only", "main").Output()
	for _, f := range strings.Split(string(out), "\x00") {
		if f == path {
			return true
		}
	}
	return false
}

func show(t *testing.T, remote, path string) (string, bool) {
	t.Helper()
	out, err := exec.Command("git", "--git-dir", remote, "show", "main:"+path).CombinedOutput()
	return string(out), err == nil
}

// The profile and the machine-wide settings travel only in the default space's backup;
// restoring another space alone works without them.
func TestProfileOnlyInTheDefaultSpace(t *testing.T) {
	needGit(t)
	root, _ := filepath.EvalSymlinks(t.TempDir())
	c, workRemote, personalRemote := twoSpaces(t, root)
	// An older version left a copy of the profile in the work space.
	if err := os.MkdirAll(filepath.Join(c.Spaces["work"].Memory, MetaDir), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(c.Spaces["work"].Memory, MetaDir, "profile.md"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{"work", "personal"} {
		if _, err := Run(c, c.Spaces[s], "one"); err != nil {
			t.Fatal(s, err)
		}
	}
	if _, ok := show(t, workRemote, MetaDir+"/profile.md"); ok {
		t.Error("the work backup holds the personal profile")
	}
	if text, ok := show(t, personalRemote, MetaDir+"/profile.md"); !ok || !strings.Contains(text, "Dana") {
		t.Errorf("the default space's backup lacks the profile: %s", text)
	}
	for _, r := range []string{workRemote, personalRemote} {
		if text, _ := show(t, r, MetaDir+"/config.toml"); strings.Contains(text, "db01") {
			t.Errorf("a backup holds the machine-wide lists:\n%s", text)
		}
	}

	// A new machine restores only work: no profile, and nothing fails.
	t.Setenv("HOME", filepath.Join(root, "new"))
	cfg2 := filepath.Join(root, "new", "cfg")
	if _, err := Restore(cfg2, RestoreOptions{URL: workRemote}); err != nil {
		t.Fatalf("restoring a space without the profile: %v", err)
	}
	// The lists are in no backup: the next session will offer to ask for them.
	if _, err := os.Stat(filepath.Join(cfg2, config.AskSafetyFile)); err != nil {
		t.Error("no request to ask the safety questions again")
	}
	// Then personal: the profile comes with it.
	if _, err := Restore(cfg2, RestoreOptions{URL: personalRemote}); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(filepath.Join(cfg2, "profile.md")); err != nil || !strings.Contains(string(data), "Dana") {
		t.Errorf("profile after restoring the default space: %v %s", err, data)
	}
	if c2, err := config.Load(cfg2); err != nil || len(c2.UntouchableHosts) != 0 {
		t.Errorf("lists after restore: %v %v", c2, err)
	}
}

// Two machines that each captured something in the inbox: both lines are kept, and the
// backup goes through without asking anyone to merge.
func TestInboxMergesLineByLine(t *testing.T) {
	needGit(t)
	root, _ := filepath.EvalSymlinks(t.TempDir())
	remote := bare(t, root)
	t.Setenv("HOME", filepath.Join(root, "h1"))
	cfg1 := filepath.Join(root, "h1", "cfg")
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
	t.Setenv("HOME", filepath.Join(root, "h2"))
	cfg2 := filepath.Join(root, "h2", "cfg")
	s2, err := Restore(cfg2, RestoreOptions{URL: remote})
	if err != nil {
		t.Fatal(err)
	}
	c2, _ := config.Load(cfg2)
	capture := func(dir, line string) {
		f, err := os.OpenFile(filepath.Join(dir, "inbox.md"), os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if _, err := f.WriteString(line + "\n"); err != nil {
			t.Fatal(err)
		}
	}
	capture(s2.Memory, "- 2026-10-20 — from two")
	if _, err := Run(c2, s2, "two"); err != nil {
		t.Fatal(err)
	}
	capture(s1.Memory, "- 2026-10-20 — from one")
	if _, err := Run(c1, s1, "one"); err != nil {
		t.Fatalf("both machines captured: %v", err)
	}
	inbox, _ := os.ReadFile(filepath.Join(s1.Memory, "inbox.md"))
	if !strings.Contains(string(inbox), "from one") || !strings.Contains(string(inbox), "from two") || strings.Contains(string(inbox), "<<<<") {
		t.Errorf("inbox after the merge:\n%s", inbox)
	}
}

// Leftovers of a crash and system files stay out; a file too big for GitHub stays out
// too, the rest is backed up, and the user is told which file and how big.
func TestBackupLeavesOutJunkAndBigFiles(t *testing.T) {
	needGit(t)
	root, _ := filepath.EvalSymlinks(t.TempDir())
	remote := bare(t, root)
	t.Setenv("HOME", filepath.Join(root, "h"))
	cfg := filepath.Join(root, "h", "cfg")
	if _, err := setup.Init(cfg, setup.Options{Space: "work", WorkFolder: "~/work", Memory: "~/brain"}); err != nil {
		t.Fatal(err)
	}
	c, _ := config.Load(cfg)
	if err := Setup(c, c.Spaces["work"], remote); err != nil {
		t.Fatal(err)
	}
	c, _ = config.Load(cfg)
	s := c.Spaces["work"]
	old := MaxFileSize
	MaxFileSize = 1 << 20
	t.Cleanup(func() { MaxFileSize = old })
	files := map[string]int{"tasks/task-x.md.tmp": 10, ".DS_Store": 10, "Thumbs.db": 10, "knowledge/video.mp4": 2 << 20,
		"knowledge/small.png": 100, "knowledge/Report [final]!.mp4": 2 << 20}
	secret := []byte("-----BEGIN " + "PRIVATE KEY-----\n")
	for name, size := range files {
		data := make([]byte, size)
		copy(data, secret) // a big file is never committed, so its secret never travels
		if err := os.WriteFile(filepath.Join(s.Memory, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// The small files hold the secret too: the junk is ignored, small.png stops the backup.
	if _, err := Run(c, s, "one"); err == nil || !strings.Contains(err.Error(), "small.png") || strings.Contains(err.Error(), ".mp4") {
		t.Fatalf("secret in small.png: %v", err)
	}
	if err := os.WriteFile(filepath.Join(s.Memory, "knowledge", "small.png"), make([]byte, 100), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := Run(c, s, "one")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.LeftOut) != 2 || res.LeftOut[0] != "knowledge/Report [final]!.mp4 (2 MB)" || !strings.Contains(res.Warning(), "knowledge/video.mp4 (2 MB)") {
		t.Errorf("left out: %v; warning: %s", res.LeftOut, res.Warning())
	}
	for name := range files {
		if strings.HasSuffix(name, ".tmp") || name == ".DS_Store" || name == "Thumbs.db" {
			continue // small, but holding the secret: removed below
		}
		ok := inBackup(t, remote, name)
		if want := name == "knowledge/small.png"; ok != want {
			t.Errorf("%s in the backup: %v, want %v", name, ok, want)
		}
	}
	// A second run changes nothing in the user's git files.
	ignore, _ := os.ReadFile(filepath.Join(s.Memory, ".gitignore"))
	if _, err := Run(c, s, "one"); err != nil {
		t.Fatal(err)
	}
	if again, _ := os.ReadFile(filepath.Join(s.Memory, ".gitignore")); string(again) != string(ignore) || strings.Count(string(again), "*.tmp") != 1 {
		t.Errorf(".gitignore after two runs:\n%s", again)
	}
	// Once the file is small enough, it is backed up again.
	if err := os.WriteFile(filepath.Join(s.Memory, "knowledge", "video.mp4"), []byte("short"), 0o644); err != nil {
		t.Fatal(err)
	}
	if res, err := Run(c, s, "one"); err != nil || len(res.LeftOut) != 1 {
		t.Fatal(res, err)
	}
	if _, ok := show(t, remote, "knowledge/video.mp4"); !ok {
		t.Error("the file that shrank is not backed up")
	}
}

// A restore that fails half way (here: the configuration cannot be written) leaves the
// space it would replace as it was, and nothing behind.
func TestRestoreFailureKeepsTheSpace(t *testing.T) {
	needGit(t)
	if runtime.GOOS == "windows" {
		t.Skip("Windows ignores a folder's read-only mode, so the write cannot be made to fail")
	}
	if os.Getuid() == 0 {
		t.Skip("root writes in read-only folders")
	}
	root, _ := filepath.EvalSymlinks(t.TempDir())
	remote := bare(t, root)
	o := setup.Options{Space: "work", WorkFolder: "~/work", Memory: "~/work/notes"}
	t.Setenv("HOME", filepath.Join(root, "old"))
	cfg1 := filepath.Join(root, "old", "cfg")
	if _, err := setup.Init(cfg1, o); err != nil {
		t.Fatal(err)
	}
	c1, _ := config.Load(cfg1)
	if err := Setup(c1, c1.Spaces["work"], remote); err != nil {
		t.Fatal(err)
	}
	c1, _ = config.Load(cfg1)
	if err := os.WriteFile(filepath.Join(c1.Spaces["work"].Memory, "inbox.md"), []byte("- 2026-10-20 — x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(c1, c1.Spaces["work"], "old"); err != nil {
		t.Fatal(err)
	}

	t.Setenv("HOME", filepath.Join(root, "new"))
	cfg := filepath.Join(root, "new", "cfg")
	if _, err := setup.Init(cfg, o); err != nil {
		t.Fatal(err)
	}
	c, _ := config.Load(cfg)
	mem := c.Spaces["work"].Memory
	if err := os.Chmod(cfg, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(cfg, 0o700) })
	if _, err := Restore(cfg, RestoreOptions{URL: remote}); err == nil || !strings.Contains(err.Error(), "config.toml") {
		t.Fatalf("the restore must fail on the configuration it cannot write: %v", err)
	}
	if !Pristine(mem) {
		t.Error("a failed restore did not put the space back")
	}
	entries, _ := os.ReadDir(filepath.Dir(mem))
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".notes.") {
			t.Errorf("left behind: %s", e.Name())
		}
	}
	os.Chmod(cfg, 0o700)
	if _, err := Restore(cfg, RestoreOptions{URL: remote}); err != nil {
		t.Fatalf("restoring again: %v", err)
	}
	if Pristine(mem) {
		t.Error("the space was not restored")
	}
}

// A backup waits for a write of the assistant (the notes lock) before it reads the notes.
func TestRunTakesTheNotesLock(t *testing.T) {
	needGit(t)
	root, _ := filepath.EvalSymlinks(t.TempDir())
	remote := bare(t, root)
	t.Setenv("HOME", filepath.Join(root, "h"))
	cfg := filepath.Join(root, "h", "cfg")
	if _, err := setup.Init(cfg, setup.Options{Space: "work", WorkFolder: "~/work", Memory: "~/brain"}); err != nil {
		t.Fatal(err)
	}
	c, _ := config.Load(cfg)
	if err := Setup(c, c.Spaces["work"], remote); err != nil {
		t.Fatal(err)
	}
	c, _ = config.Load(cfg)
	unlock, err := lockNotes(cfg)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error)
	go func() { _, err := Run(c, c.Spaces["work"], "one"); done <- err }()
	select {
	case err := <-done:
		t.Fatalf("the backup ran during a write: %v", err)
	case <-time.After(300 * time.Millisecond):
	}
	unlock()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestEscapePattern(t *testing.T) {
	for in, want := range map[string]string{
		"a/Report [final].mp4": `a/Report \[final\].mp4`,
		"#notes!?*.mov":        `\#notes\!\?\*.mov`,
		"x\\y ":                `x\\y\ `,
	} {
		if got := escapePattern(in); got != want {
			t.Errorf("%q: %q, want %q", in, got, want)
		}
	}
}

// A note that was already not valid on one machine does not stop the merge of a
// backup from another; the merge still goes through.
func TestMergeWithAnInvalidNote(t *testing.T) {
	needGit(t)
	root, _ := filepath.EvalSymlinks(t.TempDir())
	remote := bare(t, root)
	t.Setenv("HOME", filepath.Join(root, "h1"))
	cfg1 := filepath.Join(root, "h1", "cfg")
	if _, err := setup.Init(cfg1, setup.Options{Space: "work", WorkFolder: "~/work", Memory: "~/brain"}); err != nil {
		t.Fatal(err)
	}
	c1, _ := config.Load(cfg1)
	if err := Setup(c1, c1.Spaces["work"], remote); err != nil {
		t.Fatal(err)
	}
	c1, _ = config.Load(cfg1)
	s1 := c1.Spaces["work"]
	if err := os.WriteFile(filepath.Join(s1.Memory, "tasks", "task-broken.md"), []byte("no header\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(c1, s1, "one"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", filepath.Join(root, "h2"))
	cfg2 := filepath.Join(root, "h2", "cfg")
	s2, err := Restore(cfg2, RestoreOptions{URL: remote})
	if err != nil {
		t.Fatal(err)
	}
	c2, _ := config.Load(cfg2)
	note := func(dir, id string) {
		text := "---\nid: " + id + "\ntype: lesson\ntitle: " + id + "\n---\n"
		if err := os.WriteFile(filepath.Join(dir, "knowledge", id+".md"), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
		m, _, _ := memory.Load(dir)
		if _, err := m.WriteIndex("en"); err != nil {
			t.Fatal(err)
		}
	}
	note(s2.Memory, "lesson-two")
	if _, err := Run(c2, s2, "two"); err != nil {
		t.Fatal(err)
	}
	note(s1.Memory, "lesson-one")
	if res, err := Run(c1, s1, "one"); err != nil || !res.Pushed {
		t.Fatalf("an old invalid note stopped the merge: %+v %v", res, err)
	}
}

// Restoring with --memory inside the untouched space's folder keeps the restored notes.
func TestRestoreInsideTheOldFolder(t *testing.T) {
	needGit(t)
	root, _ := filepath.EvalSymlinks(t.TempDir())
	remote := bare(t, root)
	o := setup.Options{Space: "work", WorkFolder: "~/work", Memory: "~/brain"}
	t.Setenv("HOME", filepath.Join(root, "old"))
	cfg1 := filepath.Join(root, "old", "cfg")
	if _, err := setup.Init(cfg1, o); err != nil {
		t.Fatal(err)
	}
	c1, _ := config.Load(cfg1)
	if err := Setup(c1, c1.Spaces["work"], remote); err != nil {
		t.Fatal(err)
	}
	c1, _ = config.Load(cfg1)
	if _, err := Run(c1, c1.Spaces["work"], "old"); err != nil {
		t.Fatal(err)
	}
	home := filepath.Join(root, "new")
	t.Setenv("HOME", home)
	cfg := filepath.Join(home, "cfg")
	if _, err := setup.Init(cfg, o); err != nil {
		t.Fatal(err)
	}
	inside := filepath.Join(home, "brain", "restored")
	got, err := Restore(cfg, RestoreOptions{URL: remote, Memory: inside})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(got.Memory, ".git")); err != nil {
		t.Errorf("the restored notes were removed: %v", err)
	}
}

// What was staged is checked whatever the exclude patterns matched: a file that grew
// after the scan is unstaged, and a secret stops the backup.
func TestCheckStaged(t *testing.T) {
	needGit(t)
	dir := t.TempDir()
	if _, err := git(dir, "init", "-q"); err != nil {
		t.Fatal(err)
	}
	old := MaxFileSize
	MaxFileSize = 1 << 10
	t.Cleanup(func() { MaxFileSize = old })
	if err := os.WriteFile(filepath.Join(dir, "big.bin"), make([]byte, 4<<10), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := git(dir, "add", "-A"); err != nil {
		t.Fatal(err)
	}
	big, err := checkStaged(dir)
	if err != nil || len(big) != 1 || big[0] != "big.bin (0 MB)" {
		t.Fatalf("big: %v %v", big, err)
	}
	if out, _ := git(dir, "diff", "--cached", "--name-only"); strings.TrimSpace(out) != "" {
		t.Errorf("still staged: %s", out)
	}
	if err := os.WriteFile(filepath.Join(dir, "n.md"), []byte("-----BEGIN "+"PRIVATE KEY-----"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := git(dir, "add", "n.md"); err != nil {
		t.Fatal(err)
	}
	if _, err := checkStaged(dir); err == nil || !strings.Contains(err.Error(), "n.md") {
		t.Errorf("secret: %v", err)
	}
}

// A configuration edit that fails half way is undone: the file is as it was.
func TestRestoreUndoesAHalfEditedConfig(t *testing.T) {
	needGit(t)
	root, _ := filepath.EvalSymlinks(t.TempDir())
	remote := bare(t, root)
	o := setup.Options{Space: "work", WorkFolder: "~/work", Memory: "~/brain"}
	t.Setenv("HOME", filepath.Join(root, "old"))
	cfg1 := filepath.Join(root, "old", "cfg")
	if _, err := setup.Init(cfg1, o); err != nil {
		t.Fatal(err)
	}
	c1, _ := config.Load(cfg1)
	if err := Setup(c1, c1.Spaces["work"], remote); err != nil {
		t.Fatal(err)
	}
	c1, _ = config.Load(cfg1)
	if _, err := Run(c1, c1.Spaces["work"], "old"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", filepath.Join(root, "new"))
	cfg := filepath.Join(root, "new", "cfg")
	if _, err := setup.Init(cfg, o); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(filepath.Join(cfg, config.FileName))
	old := editSpace
	t.Cleanup(func() { editSpace = old })
	editSpace = func(dir string, s *config.Space) error {
		if err := config.SetSpaceKey(dir, s.Name, "backup", s.Backup); err != nil {
			return err
		}
		return errors.New("disk full")
	}
	if _, err := Restore(cfg, RestoreOptions{URL: remote}); err == nil {
		t.Fatal("the restore did not fail")
	}
	if after, _ := os.ReadFile(filepath.Join(cfg, config.FileName)); string(after) != string(before) {
		t.Errorf("configuration left half edited:\n%s", after)
	}
}
