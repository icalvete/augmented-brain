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

	"github.com/icalvete/augmented-brain/internal/state"
)

type ghFake struct {
	calls []string
	out   map[string]string
	fail  map[string]bool
}

func (f *ghFake) run(name string, args ...string) (string, error) {
	cmd := name + " " + strings.Join(args, " ")
	f.calls = append(f.calls, cmd)
	for prefix, out := range f.out {
		if strings.HasPrefix(cmd, prefix) {
			if f.fail[prefix] {
				return out, errors.New("exit status 1")
			}
			return out, nil
		}
	}
	return "", nil
}

func TestGitHubRepo(t *testing.T) {
	if _, err := exec.LookPath("gh"); err != nil {
		t.Skip("gh not installed: GitHubRepo answers with the steps before calling it")
	}
	// Not logged in: the steps, not an error to show as such.
	f := &ghFake{out: map[string]string{"gh auth status": "not logged in"}, fail: map[string]bool{"gh auth status": true}}
	_, err := GitHubRepo("trabajo", f.run, nil)
	var steps *ErrNoGitHub
	if !errors.As(err, &steps) || !strings.Contains(steps.Steps, "gh auth login --web") {
		t.Errorf("not logged in: %v", err)
	}
	// Logged in, no repository yet: it is created private, and git uses gh's login.
	f = &ghFake{out: map[string]string{
		"gh repo view":   "no such repo",
		"gh repo create": "https://github.com/someone/augmented-brain-trabajo\n",
	}, fail: map[string]bool{"gh repo view": true}}
	url, err := GitHubRepo("trabajo", f.run, nil)
	if err != nil || url != "https://github.com/someone/augmented-brain-trabajo.git" {
		t.Fatalf("create: %q %v", url, err)
	}
	if got := strings.Join(f.calls, "\n"); !strings.Contains(got, "gh repo create augmented-brain-trabajo --private") || !strings.Contains(got, "gh auth setup-git") {
		t.Errorf("calls:\n%s", got)
	}
	// It exists and is public: refused.
	f = &ghFake{out: map[string]string{"gh repo view": `{"url":"https://github.com/someone/augmented-brain-trabajo","visibility":"PUBLIC"}`}}
	if _, err := GitHubRepo("trabajo", f.run, nil); err == nil || !strings.Contains(err.Error(), "not private") {
		t.Errorf("a public repository was accepted: %v", err)
	}
	// It exists and is private: reused.
	f = &ghFake{out: map[string]string{"gh repo view": `{"url":"https://github.com/someone/augmented-brain-trabajo","visibility":"PRIVATE"}`}}
	if url, err := GitHubRepo("trabajo", f.run, nil); err != nil || !strings.HasSuffix(url, "augmented-brain-trabajo.git") {
		t.Errorf("reuse: %q %v", url, err)
	}
}

// Not logged in, with a login starter: the login is started for the user, and the
// answer carries its code instead of telling them to open a terminal.
func TestGitHubRepoStartsLogin(t *testing.T) {
	old := ghCommand
	ghCommand = func() string { return "gh" }
	t.Cleanup(func() { ghCommand = old })
	f := &ghFake{out: map[string]string{"gh auth status": "not logged in"}, fail: map[string]bool{"gh auth status": true}}
	started := &Login{Code: "ABCD-1234", URL: "https://github.com/login/device"}
	_, err := GitHubRepo("trabajo", f.run, func() (*Login, error) { return started, nil })
	var steps *ErrNoGitHub
	if !errors.As(err, &steps) || steps.Login != started || !strings.Contains(steps.Steps, "ABCD-1234") || !strings.Contains(steps.Steps, "github.com/login/device") {
		t.Fatalf("login: %#v", err)
	}
	if strings.Contains(steps.Steps, "gh auth login --web") {
		t.Errorf("the steps still send the user to a terminal: %s", steps.Steps)
	}
	// It could not start: the steps by hand, with the reason.
	_, err = GitHubRepo("trabajo", f.run, func() (*Login, error) { return nil, errors.New("no network") })
	if !errors.As(err, &steps) || steps.Login != nil || !strings.Contains(steps.Steps, "no network") || !strings.Contains(steps.Steps, "gh auth login --web") {
		t.Errorf("failed start: %#v", err)
	}
	// No gh at all: how to install it, and the way without a GitHub account.
	ghCommand = func() string { return "" }
	_, err = GitHubRepo("trabajo", f.run, func() (*Login, error) { t.Fatal("login without gh"); return nil, nil })
	if !errors.As(err, &steps) || !strings.Contains(steps.Steps, "github.com/signup") || !strings.Contains(steps.Steps, "backup setup <url>") {
		t.Errorf("no gh: %#v", err)
	}
}

// fakeGH writes a gh that runs the shell script body. Unix only: the start, the wait
// and the code are the same code on Windows, and a batch file there would test cmd.
func fakeGH(t *testing.T, body string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("a fake gh is a shell script")
	}
	p := filepath.Join(t.TempDir(), "gh")
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestStartLogin(t *testing.T) {
	cfg := t.TempDir()
	gh := fakeGH(t, `echo "! First copy your one-time code: WXYZ-9876" >&2
echo "Open this URL to continue in your web browser: https://github.com/login/device" >&2
exec sleep 30`)
	l, err := startLogin(cfg, gh, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if p, err := os.FindProcess(l.PID); err == nil {
			_ = p.Kill()
		}
	})
	if l.Code != "WXYZ-9876" || l.URL != "https://github.com/login/device" || !state.Alive(l.PID) {
		t.Fatalf("login: %+v", l)
	}
	// While gh waits, every session and the panel get the same code, not a new login.
	again, err := startLogin(cfg, gh, 5*time.Second)
	if err != nil || again.PID != l.PID || again.Code != l.Code {
		t.Errorf("second start: %+v %v", again, err)
	}
	if cur := CurrentLogin(cfg); cur == nil || cur.Code != l.Code {
		t.Errorf("current: %+v", cur)
	}
	if st, err := os.Stat(filepath.Join(cfg, "state", loginFile)); err != nil || st.Mode().Perm() != 0o600 {
		t.Errorf("state file: %v %v", st, err)
	}
	// Once gh is gone, the code is too.
	p, _ := os.FindProcess(l.PID)
	_ = p.Kill()
	for i := 0; i < 50 && state.Alive(l.PID); i++ {
		time.Sleep(20 * time.Millisecond)
	}
	if cur := CurrentLogin(cfg); cur != nil {
		t.Errorf("a finished login is still offered: %+v", cur)
	}
}

func TestStartLoginFails(t *testing.T) {
	// gh says why and stops: the reason comes back.
	gh := fakeGH(t, `echo "error connecting to github.com" >&2; exit 1`)
	if _, err := startLogin(t.TempDir(), gh, 5*time.Second); err == nil || !strings.Contains(err.Error(), "error connecting to github.com") {
		t.Errorf("failing gh: %v", err)
	}
	// gh never says a code: given up on, and stopped.
	gh = fakeGH(t, `exec sleep 30`)
	start := time.Now()
	if _, err := startLogin(t.TempDir(), gh, 300*time.Millisecond); err == nil || !strings.Contains(err.Error(), "did not answer") {
		t.Errorf("silent gh: %v", err)
	}
	if time.Since(start) > 5*time.Second {
		t.Errorf("waited %v", time.Since(start))
	}
	if _, err := startLogin(t.TempDir(), "", time.Second); err == nil {
		t.Error("no gh: no error")
	}
}
