package backup

import (
	"errors"
	"os/exec"
	"strings"
	"testing"
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
	_, err := GitHubRepo("trabajo", f.run)
	var steps *ErrNoGitHub
	if !errors.As(err, &steps) || !strings.Contains(steps.Steps, "gh auth login --web") {
		t.Errorf("not logged in: %v", err)
	}
	// Logged in, no repository yet: it is created private, and git uses gh's login.
	f = &ghFake{out: map[string]string{
		"gh repo view":   "no such repo",
		"gh repo create": "https://github.com/someone/augmented-brain-trabajo\n",
	}, fail: map[string]bool{"gh repo view": true}}
	url, err := GitHubRepo("trabajo", f.run)
	if err != nil || url != "https://github.com/someone/augmented-brain-trabajo.git" {
		t.Fatalf("create: %q %v", url, err)
	}
	if got := strings.Join(f.calls, "\n"); !strings.Contains(got, "gh repo create augmented-brain-trabajo --private") || !strings.Contains(got, "gh auth setup-git") {
		t.Errorf("calls:\n%s", got)
	}
	// It exists and is public: refused.
	f = &ghFake{out: map[string]string{"gh repo view": `{"url":"https://github.com/someone/augmented-brain-trabajo","visibility":"PUBLIC"}`}}
	if _, err := GitHubRepo("trabajo", f.run); err == nil || !strings.Contains(err.Error(), "not private") {
		t.Errorf("a public repository was accepted: %v", err)
	}
	// It exists and is private: reused.
	f = &ghFake{out: map[string]string{"gh repo view": `{"url":"https://github.com/someone/augmented-brain-trabajo","visibility":"PRIVATE"}`}}
	if url, err := GitHubRepo("trabajo", f.run); err != nil || !strings.HasSuffix(url, "augmented-brain-trabajo.git") {
		t.Errorf("reuse: %q %v", url, err)
	}
}
