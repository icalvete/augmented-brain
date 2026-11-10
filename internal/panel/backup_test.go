package panel

import (
	"html"
	"net/http"
	"net/url"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/backup"
	"github.com/icalvete/augmented-brain/internal/config"
)

// The Backup page says which spaces have no copy, sets one up (a repository the person
// has, or one made on their GitHub), starts a copy in the background, lists the last
// copies, and only shows through the network.
func TestBackupPage(t *testing.T) {
	s, cfg := fixture(t, "en")
	var started []string
	s.backupNow = func(space string) error { started = append(started, space); return nil }
	made := filepath.Join(t.TempDir(), "made.git")
	s.github = func(space string) (string, error) { return made, nil }

	page := html.UnescapeString(s.get(t, "/backup?s=work").Body.String())
	for _, want := range []string{"2 spaces have no backup", "Set it up on my GitHub", "augmented-brain-work", "I already have a private repository"} {
		if !strings.Contains(page, want) {
			t.Errorf("the page lacks %q", want)
		}
	}
	if !strings.Contains(s.get(t, "/today?s=work").Body.String(), `/backup?s=work">Backup</a>`) {
		t.Error("Backup is not in the menu")
	}

	// A repository the person has.
	own := filepath.Join(t.TempDir(), "own.git")
	if out, err := exec.Command("git", "init", "-q", "--bare", own).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v %s", err, out)
	}
	w := s.post(t, "/backup/setup", url.Values{"s": {"work"}, "url": {own}})
	if w.Code != http.StatusSeeOther || !strings.Contains(w.Header().Get("Location"), "/backup?") {
		t.Fatalf("setup: %d %s", w.Code, w.Body)
	}
	if c, _ := config.Load(cfg); c.Spaces["work"].Backup != own {
		t.Errorf("not set up: %q", c.Spaces["work"].Backup)
	}
	if len(started) != 1 || started[0] != "work" {
		t.Errorf("the first copy did not start: %v", started)
	}
	// Moving it is for a terminal.
	if w := s.post(t, "/backup/setup", url.Values{"s": {"work"}, "url": {"/elsewhere.git"}}); w.Code != http.StatusBadRequest {
		t.Errorf("moving a backup from the panel: %d", w.Code)
	}
	if w := s.post(t, "/backup/setup", url.Values{"s": {"work"}}); w.Code != http.StatusBadRequest {
		t.Errorf("making a GitHub repository for a space that has a backup: %d", w.Code)
	}

	// One made on GitHub.
	if w := s.post(t, "/backup/setup", url.Values{"s": {"home"}}); w.Code != http.StatusSeeOther {
		t.Fatalf("setup on GitHub: %d %s", w.Code, w.Body)
	}
	if c, _ := config.Load(cfg); c.Spaces["home"].Backup != made {
		t.Errorf("GitHub's repository not set: %q", c.Spaces["home"].Backup)
	}

	// The last copies are the backup's commits.
	mem := filepath.Join(filepath.Dir(cfg), "brain-work")
	for _, args := range [][]string{{"add", "-A"}, {"-c", "user.name=t", "-c", "user.email=t@example.com", "commit", "-q", "-m", "backup laptop-of-lucia 2026-11-10 10:00"}} {
		if out, err := backup.Git(mem, args...); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	page = html.UnescapeString(s.get(t, "/backup?s=work&started=work").Body.String())
	for _, want := range []string{"Last copies", "from laptop-of-lucia", "A copy has started", "Back up now", "augmented-brain restore " + own} {
		if !strings.Contains(page, want) {
			t.Errorf("the page lacks %q", want)
		}
	}
	if strings.Contains(page, "have no backup") {
		t.Error("every space has a backup now, but the page says otherwise")
	}

	if w := s.post(t, "/backup/run", url.Values{"s": {"work"}}); w.Code != http.StatusSeeOther {
		t.Fatalf("back up now: %d", w.Code)
	}
	if len(started) != 3 || started[2] != "work" {
		t.Errorf("back up now did not start a copy: %v", started)
	}

	// Through the network: shown, no buttons, posts refused.
	const rhost, owner = "box.example.ts.net:8443", "lucia@example.com"
	s.remote.Store(&Remote{Host: rhost, Owner: owner})
	w = s.do(t, req{path: s.base() + "/backup?s=work", host: rhost, owner: owner})
	if w.Code != http.StatusOK || strings.Contains(w.Body.String(), "Back up now") || !strings.Contains(w.Body.String(), "from another device") {
		t.Errorf("remote page: %d, button or note wrong", w.Code)
	}
	for _, p := range []string{"/backup/run", "/backup/setup"} {
		form := url.Values{"s": {"work"}, "csrf": {s.csrf}}
		if w := s.do(t, req{method: http.MethodPost, path: s.base() + p, host: rhost, owner: owner, origin: "https://" + rhost, form: form}); w.Code != http.StatusBadRequest {
			t.Errorf("remote %s: %d", p, w.Code)
		}
	}
	if len(started) != 3 {
		t.Errorf("a remote post started a copy: %v", started)
	}
}

// Without gh ready, the page says how to get it, in the person's language.
func TestBackupSetupWithoutGitHub(t *testing.T) {
	s, cfg := fixture(t, "es")
	s.backupNow = func(string) error { return nil }
	s.github = func(string) (string, error) { return "", &backup.ErrNoGitHub{Steps: "steps"} }
	w := s.post(t, "/backup/setup", url.Values{"s": {"work"}})
	if w.Code != http.StatusBadRequest || !strings.Contains(html.UnescapeString(w.Body.String()), "gh auth login") {
		t.Errorf("no steps: %d", w.Code)
	}
	if c, _ := config.Load(cfg); c.Spaces["work"].Backup != "" {
		t.Error("a backup was set up without a repository")
	}
	if w := s.post(t, "/backup/run", url.Values{"s": {"work"}}); w.Code != http.StatusBadRequest {
		t.Errorf("a copy of a space with no backup: %d", w.Code)
	}
}
