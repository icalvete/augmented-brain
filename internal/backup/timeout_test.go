package backup

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/filelock"
	"github.com/icalvete/augmented-brain/internal/setup"
)

// oneSpace makes a space backed up to a bare repository, and returns it loaded.
func oneSpace(t *testing.T) (*config.Config, *config.Space) {
	t.Helper()
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
	return c, c.Spaces["work"]
}

// A remote that never answers (a network that half works): the backup gives up after
// GitTimeout and says why, instead of hanging and holding the backup lock for good.
func TestBackupGivesUpOnAHangingRemote(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the hanging remote is a shell script")
	}
	c, s := oneSpace(t)
	// The remote becomes an ssh address whose "ssh" never answers.
	hang := filepath.Join(t.TempDir(), "hang.sh")
	if err := os.WriteFile(hang, []byte("#!/bin/sh\nexec sleep 60\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_SSH_COMMAND", hang)
	if _, err := git(s.Memory, "remote", "set-url", "origin", "ssh://example.invalid/brain.git"); err != nil {
		t.Fatal(err)
	}
	old := GitTimeout
	GitTimeout = time.Second
	defer func() { GitTimeout = old }()

	start := time.Now()
	_, err := Run(c, s, "one")
	if err == nil || !strings.Contains(err.Error(), "without an answer") {
		t.Fatalf("a hanging remote: %v", err)
	}
	if d := time.Since(start); d > 20*time.Second {
		t.Errorf("it took %s to give up", d)
	}
	// The lock is free again: the next backup does not wait.
	unlock, err := filelock.LockWithin(filepath.Join(c.Dir, backupLock), 0)
	if err != nil {
		t.Fatalf("the lock stayed held: %v", err)
	}
	unlock()
}

// A backup that finds another one holding the lock waits LockWait at most, then says so.
func TestBackupDoesNotWaitForeverForAnother(t *testing.T) {
	c, s := oneSpace(t)
	unlock, err := filelock.Lock(filepath.Join(c.Dir, backupLock))
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	old := LockWait
	LockWait = 200 * time.Millisecond
	defer func() { LockWait = old }()
	if _, err := Run(c, s, "one"); err == nil || !strings.Contains(err.Error(), "another backup") {
		t.Errorf("waiting behind a stuck backup: %v", err)
	}
}

// The messages of a backup that gave up speak the configuration's language, also when
// the git call that got no answer is deep inside it.
func TestBackupMessagesInSpanish(t *testing.T) {
	c, s := oneSpace(t)
	c.Language = "es"
	unlock, err := filelock.Lock(filepath.Join(c.Dir, backupLock))
	if err != nil {
		t.Fatal(err)
	}
	old := LockWait
	LockWait = 200 * time.Millisecond
	defer func() { LockWait = old }()
	if _, err := Run(c, s, "one"); err == nil || !strings.Contains(err.Error(), "otra copia lleva más de") {
		t.Errorf("Run, another backup running: %v", err)
	}
	if _, err := GoBack(c, s, "HEAD", "one"); err == nil || !strings.Contains(err.Error(), "otra copia lleva más de") {
		t.Errorf("GoBack, another backup running: %v", err)
	}
	unlock()

	none := &config.Space{Name: "casa"}
	if _, err := Run(c, none, "one"); err == nil || !strings.Contains(err.Error(), "el espacio casa aún no tiene copia") {
		t.Errorf("Run, no backup: %v", err)
	}
	if _, err := GoBack(c, none, "HEAD", "one"); err == nil || !strings.Contains(err.Error(), "el espacio casa no tiene copia") {
		t.Errorf("GoBack, no backup: %v", err)
	}

	// A git call that got no answer, wrapped as git's errors are.
	inner := fmt.Errorf("git push: %w: ", &NoAnswerError{After: 3 * time.Minute, Err: errors.New("signal: killed")})
	if got := inner.Error(); !strings.Contains(got, "stopped after 3m0s without an answer") {
		t.Errorf("before Localize, English: %s", got)
	}
	if got := Localize(inner, "es").Error(); !strings.Contains(got, "parada tras 3m0s sin respuesta") {
		t.Errorf("after Localize: %s", got)
	}
	if Localize(nil, "es") != nil {
		t.Error("Localize(nil) is not nil")
	}
}
