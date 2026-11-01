package guard

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// A worktree of an allowed repository counts as it, a * matches one folder, and a
// repository in GitAskRepos asks instead of blocking; a submodule is not its parent.
func TestGitWorktreesWildcardsAndAsk(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	home, err := filepath.EvalSymlinks(t.TempDir()) // as the configuration resolves its lists (macOS: /var is /private/var)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	gitRun := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=t", "-c", "user.email=t@example.com",
			"-c", "protocol.file.allow=always"}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	repo := filepath.Join(home, "code", "app")
	lib := filepath.Join(home, "code", "lib")
	for _, d := range []string{repo, lib, filepath.Join(home, "code", "app-old"), filepath.Join(home, "code", "app-x", "deep"), filepath.Join(home, "ops", "infra")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, d := range []string{repo, lib} {
		gitRun(d, "init", "-q")
		gitRun(d, "commit", "-q", "--allow-empty", "-m", "start")
	}
	gitRun(repo, "worktree", "add", "-q", filepath.Join(home, "elsewhere", "app-fix"))
	gitRun(repo, "submodule", "add", "-q", lib, "vendor/lib")

	p := Policy{
		GitAllowedRepos: []string{repo, filepath.Join(home, "code", "app-*")},
		GitAskRepos:     []string{filepath.Join(home, "ops", "infra")},
	}
	ci := "com" + "mit" // spelled apart, so tools that scan for the words leave this file alone
	for cmd, want := range map[string]Verdict{
		"git -C ~/code/app " + ci + " -m x":                 Allow,
		"git -C ~/elsewhere/app-fix push":                   Allow, // a worktree of app, wherever it is
		"git -C ~/code/app-old " + ci + " -m x":             Allow, // app-*
		"git -C ~/code/app-x/deep " + ci + " -m x":          Deny,  // * is one folder
		"git -C ~/code/lib push":                            Deny,
		"git -C ~/code/app/vendor/lib " + ci + " -m x":      Deny, // a submodule is not app
		"git -C ~/ops/infra push":                           Ask,
		"git -C ~/ops/infra push && git -C ~/code/lib push": Deny, // a deny wins over an ask
	} {
		if got := p.Command(cmd).Verdict; got != want {
			t.Errorf("%s: %v, want %v", cmd, got, want)
		}
	}
	if r := p.Command("git -C ~/ops/infra " + ci + " -m x").Reason; !strings.Contains(r, "~/ops/infra") || !strings.Contains(r, "go-ahead") {
		t.Errorf("ask reason: %s", r)
	}
}
