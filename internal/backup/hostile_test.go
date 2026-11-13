package backup

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/setup"
)

// tamper changes the configuration saved in the backup at remote, as someone with
// access to it could, and pushes the change.
func tamper(t *testing.T, root, remote, cfg string) {
	t.Helper()
	dir := filepath.Join(root, "tamper")
	if out, err := runGit("clone", "-q", "--", remote, dir); err != nil {
		t.Fatal(err, out)
	}
	if err := os.WriteFile(filepath.Join(dir, MetaDir, config.FileName), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "-A"}, append(ident, "com"+"mit", "-q", "-m", "x"), {"push", "-q", "origin", "HEAD:main"}} {
		if _, err := git(dir, args...); err != nil {
			t.Fatal(err)
		}
	}
}

// A backup cannot choose where its files land: one that asks for ~/.config/autostart,
// whose files the desktop runs at login, goes to the usual place, and says so.
func TestRestoreRefusesAPlaceItMayNotChoose(t *testing.T) {
	needGit(t)
	root, _ := filepath.EvalSymlinks(t.TempDir())
	remote := backedUp(t, root)
	tamper(t, root, remote, "language = \"en\"\n\n[spaces.work]\nmemory = \"~/.config/autostart\"\nwork_folders = [\"~/.config\"]\n")

	home := filepath.Join(root, "new")
	t.Setenv("HOME", home)
	s, err := Restore(filepath.Join(home, "cfg"), RestoreOptions{URL: remote})
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(home, "augmented-brain", "work", "notes"); s.Memory != want {
		t.Errorf("memory: %s, want %s", s.Memory, want)
	}
	if want := []string{filepath.Join(home, "augmented-brain", "work")}; !slices.Equal(s.WorkFolders, want) {
		t.Errorf("work folders: %v, want %v", s.WorkFolders, want)
	}
	if want := []string{"~/.config", "~/.config/autostart"}; !slices.Equal(s.Unsafe, want) {
		t.Errorf("unsafe: %v, want %v", s.Unsafe, want)
	}
	if _, err := os.Stat(filepath.Join(home, ".config")); !os.IsNotExist(err) {
		t.Errorf("~/.config was created: %v", err)
	}
	c, err := config.Load(filepath.Join(home, "cfg"))
	if err != nil || c.Spaces["work"] == nil || c.Spaces["work"].Memory != s.Memory {
		t.Errorf("configuration: %+v, %v", c, err)
	}

	// The user may still choose any folder.
	mine := filepath.Join(home, ".mine")
	s, err = Restore(filepath.Join(root, "cfg2"), RestoreOptions{URL: remote, Memory: mine, WorkFolder: "~/.work"})
	if err != nil {
		t.Fatal(err)
	}
	if s.Memory != mine || len(s.Unsafe) != 0 {
		t.Errorf("a folder the user chose was not kept: %s, %v", s.Memory, s.Unsafe)
	}
}

// A space's name becomes a folder and a line of the configuration: one that is not a
// valid name is refused before anything is written.
func TestRestoreRefusesABadName(t *testing.T) {
	needGit(t)
	root, _ := filepath.EvalSymlinks(t.TempDir())
	remote := backedUp(t, root)
	tamper(t, root, remote, "language = \"en\"\n\n[spaces.\"../../x\"]\nmemory = \"~/brain\"\nwork_folders = [\"~/work\"]\n")
	home := filepath.Join(root, "new")
	t.Setenv("HOME", home)
	_, err := Restore(filepath.Join(home, "cfg"), RestoreOptions{URL: remote})
	if err == nil || !strings.Contains(err.Error(), "not a valid name") {
		t.Fatalf("want a refusal, got %v", err)
	}
	if _, err := os.Stat(home); !os.IsNotExist(err) {
		t.Errorf("something was written: %v", err)
	}
}

func TestSafePlace(t *testing.T) {
	root, _ := filepath.EvalSymlinks(t.TempDir())
	home := filepath.Join(root, "home")
	t.Setenv("HOME", home)
	for p, want := range map[string]bool{
		"~/augmented-brain/work/notes": true,
		"~/projects/.notes":            true, // hidden, but not at the top
		"~":                            false,
		"":                             false,
		"~/.config/autostart":          false,
		"~/.ssh":                       false,
		"~/Library/LaunchAgents":       false,
		"~/AppData/Roaming":            false,
		"~/code/app/.git/hooks":        false,
		"~/../elsewhere":               false,
		filepath.Join(root, "outside"): false,
	} {
		if got := safePlace(p); got != want {
			t.Errorf("safePlace(%q) = %v, want %v", p, got, want)
		}
	}
}

// A repository made with `git init --bare` may name master as its HEAD; the backups
// went to main all along, and a restore finds them there.
func TestRestoreFromARepositoryWhoseHeadIsNotMain(t *testing.T) {
	needGit(t)
	root, _ := filepath.EvalSymlinks(t.TempDir())
	remote := filepath.Join(root, "master.git")
	if out, err := exec.Command("git", "init", "-q", "--bare", "-b", "master", remote).CombinedOutput(); err != nil {
		t.Fatal(err, string(out))
	}
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
	t.Setenv("HOME", filepath.Join(root, "new"))
	if _, err := Restore(filepath.Join(root, "new", "cfg"), RestoreOptions{URL: remote}); err != nil {
		t.Fatal(err)
	}

	// An empty repository holds no backup, and the restore says so in words.
	empty := filepath.Join(root, "empty.git")
	if out, err := exec.Command("git", "init", "-q", "--bare", "-b", "master", empty).CombinedOutput(); err != nil {
		t.Fatal(err, string(out))
	}
	_, err := Restore(filepath.Join(root, "new", "cfg3"), RestoreOptions{URL: empty, Memory: filepath.Join(root, "new", "m3")})
	if err == nil || !strings.Contains(err.Error(), "no branch main") {
		t.Errorf("want a plain error, got %v", err)
	}
}

// The user's global git settings that suit their code (every commit signed, a hook for
// every repository) do not stop the backup of the notes.
func TestBackupIgnoresTheUsersGitSettings(t *testing.T) {
	c, s := oneSpace(t)
	dir := t.TempDir()
	hooks := filepath.Join(dir, "hooks")
	if err := os.MkdirAll(hooks, 0o755); err != nil {
		t.Fatal(err)
	}
	ran := filepath.Join(dir, "hook-ran")
	hook := "#!/bin/sh\necho ran > '" + filepath.ToSlash(ran) + "'\nexit 1\n"
	if err := os.WriteFile(filepath.Join(hooks, "pre-commit"), []byte(hook), 0o755); err != nil {
		t.Fatal(err)
	}
	global := filepath.Join(dir, "gitconfig")
	settings := "[commit]\n\tgpgsign = true\n[gpg]\n\tprogram = augmented-brain-no-such-gpg\n[core]\n\thooksPath = " + filepath.ToSlash(hooks) + "\n"
	if err := os.WriteFile(global, []byte(settings), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", global)
	if err := os.WriteFile(filepath.Join(s.Memory, "inbox.md"), []byte("# Inbox\n\n- 2026-11-13 — a thought\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	res, err := Run(c, s, "h")
	if err != nil {
		t.Fatal(err)
	}
	if !res.Committed {
		t.Error("nothing was committed")
	}
	if _, err := os.Stat(ran); !os.IsNotExist(err) {
		t.Error("the user's global hook ran inside the notes")
	}
	// The user's own repositories still follow them: Git (session start's worktrees)
	// does not change them.
	if out, _ := Git(s.Memory, "config", "commit.gpgsign"); strings.TrimSpace(out) != "true" {
		t.Errorf("Git changed the user's settings: %q", out)
	}
}

// A backup that began and never ended (the computer went off) is noticed; one that
// ends clears it.
func TestCutShort(t *testing.T) {
	cfg := t.TempDir()
	now := time.Now()
	RecordStarted(cfg, "work", now.Add(-time.Hour))
	if _, ok := LastStatus(cfg, "work"); ok {
		t.Error("a backup that only began is not a last status")
	}
	if began, cut := CutShort(cfg, "work", now); !cut || began.IsZero() {
		t.Errorf("not cut short: %v %v", began, cut)
	}
	RecordStarted(cfg, "work", now)
	if _, cut := CutShort(cfg, "work", now); cut {
		t.Error("a backup that just began is not cut short")
	}
	RecordStatusWarning(cfg, "work", nil, "", now)
	if _, cut := CutShort(cfg, "work", now.Add(2*time.Hour)); cut {
		t.Error("a backup that ended is not cut short")
	}
}

// The last good backup is days old and the notes changed since: the automatic backup is
// not running, and the user hears of it. Fresh, or with nothing new, nothing is said.
func TestBehind(t *testing.T) {
	c, s := oneSpace(t)
	if _, err := Run(c, s, "h"); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	old := Status{OK: true, Time: now.Add(-4 * 24 * time.Hour)}
	if old.Behind(s.Memory, now) {
		t.Error("nothing changed, yet it is behind")
	}
	if err := os.WriteFile(filepath.Join(s.Memory, "inbox.md"), []byte("# Inbox\n\n- 2026-11-13 — new\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !old.Behind(s.Memory, now) {
		t.Error("a change not backed up for 4 days was not noticed")
	}
	if (Status{OK: true, Time: now.Add(-time.Hour)}).Behind(s.Memory, now) {
		t.Error("a recent backup is not behind")
	}
	// Committed but not pushed counts too.
	for _, args := range [][]string{{"add", "-A"}, append(ident, "com"+"mit", "-q", "-m", "x")} {
		if _, err := git(s.Memory, args...); err != nil {
			t.Fatal(err)
		}
	}
	if !old.Behind(s.Memory, now) {
		t.Error("a commit not pushed was not noticed")
	}
}
