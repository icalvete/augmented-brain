package backup

import (
	"errors"
	"os"
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
	best := "" // the longest prefix that matches: one command may start another's
	for prefix := range f.out {
		if strings.HasPrefix(cmd+" ", prefix) && len(prefix) > len(best) {
			best = prefix
		}
	}
	if best == "" {
		return "", nil
	}
	if f.fail[best] {
		return f.out[best], errors.New("exit status 1")
	}
	return f.out[best], nil
}

// fakeMachine makes gh "gh" and this computer "Lucia-PC.local" for one test.
func fakeMachine(t *testing.T) {
	oldGH, oldHost := ghCommand, hostname
	ghCommand = func() string { return "gh" }
	hostname = func() (string, error) { return "Lucia-PC.local", nil }
	t.Cleanup(func() { ghCommand, hostname = oldGH, oldHost })
}

func TestGitHubRepo(t *testing.T) {
	fakeMachine(t)
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
	repo, err := GitHubRepo("trabajo", f.run, nil)
	if err != nil || repo.URL != "https://github.com/someone/augmented-brain-trabajo.git" || repo.Taken != "" {
		t.Fatalf("create: %+v %v", repo, err)
	}
	if got := strings.Join(f.calls, "\n"); !strings.Contains(got, "gh repo create augmented-brain-trabajo --private") || !strings.Contains(got, "gh auth setup-git") {
		t.Errorf("calls:\n%s", got)
	}
	// It exists, private and empty (made by hand for this, never used): used.
	f = &ghFake{out: map[string]string{"gh repo view": `{"url":"https://github.com/someone/augmented-brain-trabajo","visibility":"PRIVATE","isEmpty":true}`}}
	if repo, err := GitHubRepo("trabajo", f.run, nil); err != nil || repo.URL != "https://github.com/someone/augmented-brain-trabajo.git" || repo.Taken != "" {
		t.Errorf("an empty one: %+v %v", repo, err)
	}
}

// A repository with the usual name and something in it is most likely another
// computer's backup: never used for a new one. The backup goes to a name with this
// computer's, and the answer says which one was left alone.
func TestGitHubRepoLeavesAnotherBackupAlone(t *testing.T) {
	fakeMachine(t)
	for _, vis := range []string{"PRIVATE", "PUBLIC"} {
		f := &ghFake{out: map[string]string{
			"gh repo view augmented-brain-personal ":                `{"url":"https://github.com/someone/augmented-brain-personal","visibility":"` + vis + `","isEmpty":false}`,
			"gh repo view augmented-brain-personal-lucia-pc-local ": "no such repo",
			"gh repo create": "https://github.com/someone/augmented-brain-personal-lucia-pc-local\n",
		}, fail: map[string]bool{"gh repo view augmented-brain-personal-lucia-pc-local ": true}}
		repo, err := GitHubRepo("personal", f.run, nil)
		if err != nil || repo.URL != "https://github.com/someone/augmented-brain-personal-lucia-pc-local.git" || repo.Taken != "https://github.com/someone/augmented-brain-personal" {
			t.Fatalf("%s: %+v %v", vis, repo, err)
		}
		if got := strings.Join(f.calls, "\n"); strings.Contains(got, "gh repo create augmented-brain-personal --") {
			t.Errorf("%s: the taken name was created again:\n%s", vis, got)
		}
	}
	// An earlier gh that does not know isEmpty: the repository counts as not empty.
	f := &ghFake{out: map[string]string{
		"gh repo view augmented-brain-personal --json url,visibility,isEmpty": `Unknown JSON field: "isEmpty"`,
		"gh repo view augmented-brain-personal --json url,visibility ":        `{"url":"https://github.com/someone/augmented-brain-personal","visibility":"PRIVATE"}`,
		"gh repo view augmented-brain-personal-lucia-pc-local ":               "no such repo",
		"gh repo create": "https://github.com/someone/augmented-brain-personal-lucia-pc-local\n",
	}, fail: map[string]bool{
		"gh repo view augmented-brain-personal --json url,visibility,isEmpty": true,
		"gh repo view augmented-brain-personal-lucia-pc-local ":               true,
	}}
	if repo, err := GitHubRepo("personal", f.run, nil); err != nil || repo.Taken == "" || !strings.HasSuffix(repo.URL, "-lucia-pc-local.git") {
		t.Errorf("an earlier gh: %+v %v", repo, err)
	}
	// Both names taken: nothing is used, and the answer says restore is for bringing
	// one of them here.
	f = &ghFake{out: map[string]string{"gh repo view": `{"url":"https://github.com/someone/x","visibility":"PRIVATE","isEmpty":false}`}}
	_, err := GitHubRepo("personal", f.run, nil)
	var taken *ErrRepoTaken
	if !errors.As(err, &taken) || len(taken.Taken) != 2 || !strings.Contains(err.Error(), "augmented-brain restore") {
		t.Errorf("both taken: %v", err)
	}
	if got := strings.Join(f.calls, "\n"); strings.Contains(got, "repo create") || strings.Contains(got, "setup-git") {
		t.Errorf("both taken, yet:\n%s", got)
	}
}

func TestRepoNames(t *testing.T) {
	old := hostname
	t.Cleanup(func() { hostname = old })
	for host, want := range map[string]string{
		"Lucia-PC.local":        "augmented-brain-w-lucia-pc-local",
		"ÑAndú_42":              "augmented-brain-w-and-42",
		"---":                   "",
		strings.Repeat("a", 40): "augmented-brain-w-" + strings.Repeat("a", 30),
	} {
		hostname = func() (string, error) { return host, nil }
		names := repoNames("w")
		if names[0] != "augmented-brain-w" || (want == "" && len(names) != 1) || (want != "" && (len(names) != 2 || names[1] != want)) {
			t.Errorf("%q: %v", host, names)
		}
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
	if st, err := os.Stat(filepath.Join(cfg, "state", loginFile)); err != nil || st.Mode().Perm() != 0o600 { // windows-ok: fakeGH skips on Windows
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
