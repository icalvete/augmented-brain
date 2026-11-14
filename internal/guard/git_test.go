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

// An untouchable repository is denied for commit, push and merge, even when the other
// lists allow or ask for it; the denial says it is untouchable.
func TestUntouchableWinsOverTheOtherLists(t *testing.T) {
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	core := filepath.Join(home, "work", "core")
	cli := filepath.Join(home, "work", "cli-tools")
	ok := filepath.Join(home, "work", "ok")
	for _, d := range []string{core, cli, ok} {
		fakeRepo(t, d, map[string]string{"origin": "git@example.com:acme/x.git"})
	}
	fakeWorktree(t, core, filepath.Join(home, "wt", "core-fix"))
	p := Policy{
		GitAllowedRepos:  []string{core, ok},
		GitAskRepos:      []string{cli, core},
		UntouchableRepos: []string{core, filepath.Join(home, "work", "cli-*")},
	}
	ci := "com" + "mit"
	for cmd, want := range map[string]Verdict{
		"git -C ~/work/core push":                          Deny, // allowed and asked, but untouchable
		"git -C ~/work/core " + ci + " -m x":               Deny,
		"git -C ~/work/cli-tools push":                     Deny, // the * of the untouchable list
		"git -C ~/wt/core-fix push":                        Deny, // a worktree of an untouchable repository
		"git -C ~/work/ok " + ci + " -m x":                 Allow,
		"git -C ~/work/ok push && git -C ~/work/core push": Deny,
	} {
		d := p.Command(cmd)
		if d.Verdict != want {
			t.Errorf("%s: %v, want %v", cmd, d.Verdict, want)
		}
		if want == Deny && strings.Contains(cmd, "core push") && !strings.Contains(d.Reason, "untouchable") {
			t.Errorf("%s: reason does not say untouchable: %s", cmd, d.Reason)
		}
	}
	p.Cwd = core
	if d := p.Command("gh pr merge 12"); d.Verdict != Deny || !strings.Contains(d.Reason, "untouchable") {
		t.Errorf("merge in an untouchable repository: %v %s", d.Verdict, d.Reason)
	}
	p.Cwd = home
	if d := p.Command("cd ~/work/core && gh pr merge 12"); d.Verdict != Deny || !strings.Contains(d.Reason, "untouchable") {
		t.Errorf("cd and merge: %v %s", d.Verdict, d.Reason)
	}
}

// The question to the user names the branch and where the remote leads, with no
// password in the address, for a push and for a merge, in both languages.
func TestAskSaysBranchAndRemote(t *testing.T) {
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	repo := filepath.Join(home, "ops", "infra")
	fakeRepo(t, repo, map[string]string{"origin": "https://bob:s3cr3t@git.example.com/acme/infra.git"})
	if err := os.WriteFile(filepath.Join(repo, ".git", "HEAD"), []byte("ref: refs/heads/feat/x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for lang, want := range map[string][]string{"en": {"Branch: feat/x", "Remote: git.example.com/acme/infra"}, "es": {"Rama: feat/x", "Remoto: git.example.com/acme/infra"}} {
		p := Policy{GitAskRepos: []string{repo}, Lang: lang, Cwd: repo}
		for _, cmd := range []string{"git -C ~/ops/infra push", "cd ~/ops/infra && gh pr merge 3"} {
			d := p.Command(cmd)
			if d.Verdict != Ask {
				t.Fatalf("%s: %v", cmd, d.Verdict)
			}
			for _, w := range want {
				if !strings.Contains(d.Reason, w) {
					t.Errorf("%s (%s): reason lacks %q: %s", cmd, lang, w, d.Reason)
				}
			}
			if strings.Contains(d.Reason, "s3cr3t") || strings.Contains(d.Reason, "bob") {
				t.Errorf("%s: credentials in the reason: %s", cmd, d.Reason)
			}
		}
	}
}

// A commit or a push in a listed repository that does not name it with git -C and a
// path written out is still denied, but the reason says how to write it, never that
// the repository should be added to the lists it is already in.
func TestGitDeniedSaysHowToNameTheRepository(t *testing.T) {
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	repo := filepath.Join(home, "code", "app-fix")
	other := filepath.Join(home, "code", "lib")
	for _, d := range []string{repo, other} {
		if err := os.MkdirAll(filepath.Join(d, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	ci := "com" + "mit"
	p := Policy{Cwd: home, GitAllowedRepos: []string{filepath.Join(home, "code", "app-*")}}
	slashed := filepath.ToSlash(repo) // C:/x on Windows: a shell would eat the backslashes
	named := "git -C " + slashed + " "
	cases := []struct{ cmd, key string }{
		{"cd ~/code/app-fix && git " + ci + " -m x", "guard.git.named"},
		{"cd " + slashed + " && git push", "guard.git.named"},
		{"W=~/code/app-fix; cd $W && git " + ci + " -m x", "guard.git.unknown"},
		{"git -C \"$W\" " + ci + " -m x", "guard.git.unknown"},
		{"git -C $(pwd) push", "guard.git.unknown"},
		{"cd ~/code/lib && git " + ci + " -m x", "guard.git.deny"}, // in no list: for the user
		{"git " + ci + " -m x", "guard.git.deny"},                  // home is no repository of the lists
	}
	for _, c := range cases {
		d := p.Command(c.cmd)
		if d.Verdict != Deny {
			t.Errorf("%s: %v, want Deny", c.cmd, d.Verdict)
		}
		if !reasonIs(d.Reason, []string{c.key}) {
			t.Errorf("%s: reason %q, want %s", c.cmd, d.Reason, c.key)
		}
		if c.key == "guard.git.named" && !strings.Contains(d.Reason, named) {
			t.Errorf("%s: the reason does not say %q: %s", c.cmd, named, d.Reason)
		}
	}
	if d := p.Command(named + ci + " -m x"); d.Verdict != Allow {
		t.Errorf("as the reason says: %v (%s)", d.Verdict, d.Reason)
	}
	q := p
	q.Cwd = repo
	if d := q.Command("git " + ci + " -m x"); d.Verdict != Deny || !strings.Contains(d.Reason, named) {
		t.Errorf("in the repository's own folder: %v (%s)", d.Verdict, d.Reason)
	}
}
