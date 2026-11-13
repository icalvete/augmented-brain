package guard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeRepo makes a repository at dir by hand (no git needed): a .git folder whose
// config has the given remotes.
func fakeRepo(t *testing.T, dir string, remotes map[string]string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := "[core]\n\tbare = false\n"
	for name, url := range remotes {
		cfg += "[remote \"" + name + "\"]\n\turl = " + url + "\n\tfetch = +refs/heads/*:refs/remotes/" + name + "/*\n"
	}
	if err := os.WriteFile(filepath.Join(dir, ".git", "config"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
}

// fakeWorktree makes wt a linked worktree of the repository at repo, as git does.
func fakeWorktree(t *testing.T, repo, wt string) {
	t.Helper()
	admin := filepath.Join(repo, ".git", "worktrees", filepath.Base(wt))
	for _, d := range []string{admin, wt} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(admin, "commondir"), []byte("../..\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wt, ".git"), []byte("gitdir: "+admin+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// Merging a pull request goes by the repository lists, as a push does: the repository
// is the folder gh or glab runs in, and one the command names must be its remote.
func TestMergeLikePush(t *testing.T) {
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	app := filepath.Join(home, "code", "app")
	ops := filepath.Join(home, "code", "ops")
	lab := filepath.Join(home, "code", "lab")
	other := filepath.Join(home, "code", "other")
	fakeRepo(t, app, map[string]string{"origin": "git@github.com:Acme/app.git", "upstream": "https://github.com/upstream-org/app"})
	fakeRepo(t, ops, map[string]string{"origin": "ssh://git@github.com:22/acme/ops.git"})
	fakeRepo(t, lab, map[string]string{"origin": "git@gitlab.example.com:team/sub/lab.git"})
	fakeRepo(t, other, map[string]string{"origin": "git@github.com:acme/other.git"})
	fakeWorktree(t, app, filepath.Join(home, "wt", "app-fix"))
	if err := os.MkdirAll(filepath.Join(app, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}

	base := Policy{
		GitAllowedRepos: []string{app, lab},
		GitAskRepos:     []string{ops},
	}
	cases := []struct {
		cwd  string
		env  map[string]string
		cmd  string
		want Verdict
	}{
		// allowed, asked, denied: the folder it runs in
		{app, nil, "gh pr merge 12 --squash", Allow},
		{app, nil, "gh pr merge 12 --auto --rebase --delete-branch", Allow},
		{ops, nil, "gh pr merge 12", Ask},
		{other, nil, "gh pr merge 12", Deny},
		{home, nil, "gh pr merge 12", Deny},
		{"", nil, "gh pr merge 12", Deny},
		{filepath.Join(app, "sub"), nil, "gh pr merge 12", Deny}, // a folder inside it is not the repository, as with git -C
		{filepath.Join(home, "wt", "app-fix"), nil, "gh pr merge 12", Allow},
		// a cd in the command, as a push follows it
		{home, nil, "cd code/app && gh pr merge 12", Allow},
		{home, nil, "cd ~/code/ops && gh pr merge 12", Ask},
		{home, nil, "cd ~/code/other && gh pr merge 12", Deny},
		{app, nil, "cd \"$DIR\" && gh pr merge 12", Deny}, // a folder the guard cannot tell
		{app, nil, "cd - && gh pr merge 12", Deny},
		{app, nil, "(cd ../other && gh pr merge 12)", Deny},
		{app, nil, "ssh web01 gh pr merge 12", Deny}, // another machine
		// -R and the address of the pull request must be a remote of the repository
		{app, nil, "gh pr merge 12 -R acme/app", Allow},
		{app, nil, "gh pr merge 12 --repo ACME/APP", Allow},
		{app, nil, "gh pr merge 12 --repo=github.com/acme/app", Allow},
		{app, nil, "gh pr merge 12 -Racme/app", Allow},
		{app, nil, "gh pr merge 12 -R upstream-org/app", Allow},
		{app, nil, "gh pr merge 12 -R https://github.com/acme/app.git", Allow},
		{app, nil, "gh pr merge https://github.com/acme/app/pull/12", Allow},
		{app, nil, "gh pr merge 12 -R acme/other", Deny},
		{app, nil, "gh pr merge 12 -R evil/app", Deny},
		{app, nil, "gh pr merge 12 -R ghe.example.com/acme/app", Deny},
		{app, nil, "gh pr merge https://github.com/acme/other/pull/12", Deny},
		{app, nil, "gh pr merge https://github.com/acme/app/issues/12", Deny}, // not an address it can read
		{app, nil, "gh pr merge 12 -b 'see https://github.com/acme/other/pull/1'", Allow},
		{app, nil, "GH_REPO=acme/other gh pr merge 12", Deny},
		{app, map[string]string{"GH_REPO": "acme/other"}, "gh pr merge 12", Deny},
		{app, map[string]string{"GH_REPO": "acme/app"}, "gh pr merge 12", Allow},
		{app, map[string]string{"GH_HOST": "ghe.example.com"}, "gh pr merge 12 -R acme/app", Deny},
		{app, nil, "GIT_DIR=../other/.git gh pr merge 12", Deny},
		{app, map[string]string{"GIT_WORK_TREE": other}, "gh pr merge 12", Deny},
		{ops, nil, "gh pr merge 12 -R acme/ops", Ask},
		{ops, nil, "gh pr merge 12 -R acme/app", Deny},
		// --admin skips the branch's protection: always asked
		{app, nil, "gh pr merge 12 --admin", Ask},
		{app, nil, "gh pr merge 12 --admin -R acme/app", Ask},
		{ops, nil, "gh pr merge 12 --admin", Ask},
		{other, nil, "gh pr merge 12 --admin", Deny},
		{app, nil, "gh pr merge 12 --admin -R acme/other", Deny},
		// glab
		{lab, nil, "glab mr merge 7", Allow},
		{lab, nil, "glab mr accept 7 --squash", Allow},
		{lab, nil, "glab mr merge 7 -R gitlab.example.com/team/sub/lab", Allow},
		{lab, nil, "glab mr merge 7 -R team/sub/lab", Deny}, // no host: gitlab.com, not this remote
		{lab, map[string]string{"GITLAB_HOST": "gitlab.example.com"}, "glab mr merge 7 -R team/sub/lab", Allow},
		{lab, nil, "glab mr merge https://gitlab.example.com/team/sub/lab/-/merge_requests/7", Allow},
		{lab, nil, "glab mr merge https://gitlab.example.com/team/other/-/merge_requests/7", Deny},
		{lab, nil, "GITLAB_REPO=team/other glab mr merge 7", Deny},
		{other, nil, "glab mr merge 7", Deny},
		{ops, nil, "glab mr accept 7", Ask},
		// publishing stays for the user, in any repository
		{app, nil, "gh release create v1.0.0", Deny},
		{app, nil, "gh release upload v1.0.0 a.tgz", Deny},
		{app, nil, "gh release edit v1.0.0", Deny},
		{lab, nil, "glab release create v1.0.0", Deny},
		{app, nil, "gh api -X PUT repos/acme/app/pulls/12/merge", Deny},
		{app, nil, "gh api repos/acme/app/releases -f tag_name=v1", Deny},
		{app, nil, "gh api -X PATCH repos/acme/app/git/refs/heads/main", Deny},
		{app, nil, "gh pr view 12", Allow},
		// starting a release workflow by hand publishes a version, so the user is asked; other workflows go on
		{app, nil, "gh workflow run release", Ask},
		{app, nil, "gh workflow run release.yml -R acme/app", Ask},
		{app, nil, "gh workflow run --ref main .github/workflows/Release.yaml", Ask},
		{app, nil, "gh workflow run -R acme/app -f x=1 release", Ask},
		{app, nil, "gh api -X POST repos/acme/app/actions/workflows/release.yml/dispatches -f ref=main", Ask},
		{app, nil, "gh api repos/acme/app/actions/workflows/release.yml/dispatches --input body.json", Ask},
		{app, nil, "gh workflow run 123456", Ask},
		{app, nil, "gh api -X POST repos/acme/app/actions/workflows/123456/dispatches -f ref=main", Ask},
		{app, nil, "gh workflow run deploy-site -R acme/app", Allow},
		{app, nil, "gh api -X POST repos/acme/app/actions/workflows/deploy.yml/dispatches -f ref=main", Allow},
		{app, nil, "gh api repos/acme/app/actions/workflows/release.yml", Allow},
		{app, nil, "gh workflow view release", Allow},
		{app, nil, "gh run list --workflow release", Allow},
	}
	// The folder of each case is written in the command, with a cd, from the home folder;
	// cases in the home folder or in none run as written.
	in := func(dir, cmd string) string { return "cd '" + filepath.ToSlash(dir) + "' && " + cmd }
	for _, c := range cases {
		p := base
		p.Cwd, p.Env = home, c.env
		cmd := c.cmd
		switch c.cwd {
		case home:
		case "":
			p.Cwd = ""
		default:
			cmd = in(c.cwd, cmd)
		}
		if d := p.Command(cmd); d.Verdict != c.want {
			t.Errorf("%q (env %v): %v (%s), want %v", cmd, c.env, d.Verdict, d.Reason, c.want)
		}
	}

	// The session's folder alone never counts, as a push counts only git -C: the
	// command must name the repository with a cd.
	for _, c := range []struct {
		cwd, cmd string
		want     Verdict
	}{
		{app, "gh pr merge 12", Deny},
		{ops, "gh pr merge 12", Deny},
		{lab, "glab mr merge 7", Deny},
		// The repository named in the command itself counts, as a cd would: Claude Code
		// drops a cd to the folder the session is already in before the guard sees it.
		{app, "gh pr merge 12 -R acme/app", Allow},
		{app, "gh pr merge 12 --repo=Acme/app --rebase", Allow},
		{app, "gh pr merge https://github.com/acme/app/pull/12", Allow},
		{ops, "gh pr merge 12 -R acme/ops", Ask},
		{app, "gh pr merge 12 -R acme/other", Deny},    // not this folder's repository
		{other, "gh pr merge 12 -R acme/other", Deny},  // its repository, but not in the lists
		{app, "GH_REPO=acme/app gh pr merge 12", Deny}, // a variable is not the command's words
		{filepath.Join(app, "sub"), "gh pr merge 12 -R acme/app", Deny},
		{app, "cd . && gh pr merge 12", Allow}, // named, if only as .
		{app, "cd && gh pr merge 12", Deny},    // the home folder, not named
		{app, "cd ~/code/app && popd && gh pr merge 12", Deny},
		{filepath.Join(app, "sub"), "cd .. && gh pr merge 12", Allow},
	} {
		p := base
		p.Cwd = c.cwd
		if d := p.Command(c.cmd); d.Verdict != c.want {
			t.Errorf("%q in %s: %v (%s), want %v", c.cmd, c.cwd, d.Verdict, d.Reason, c.want)
		}
	}

	// The reasons the user reads name the repository, in their language.
	p := base
	p.Cwd = home
	if r := p.Command(in(ops, "gh pr merge 12")).Reason; !strings.Contains(r, "~/code/ops") || !strings.Contains(r, "merges a pull request") {
		t.Errorf("ask reason: %s", r)
	}
	p.Lang = "es"
	if r := p.Command(in(ops, "glab mr merge 7")).Reason; !strings.Contains(r, "~/code/ops") || !strings.Contains(r, "fusiona una pull request") {
		t.Errorf("Spanish ask reason: %s", r)
	}
	p.Lang = ""
	if r := p.Command(in(app, "gh pr merge 12 --admin")).Reason; !strings.Contains(r, "~/code/app") || !strings.Contains(r, "protection") {
		t.Errorf("admin reason: %s", r)
	}
	// Where merging is not let through, the reason is the one of every publishing command.
	if r := p.Command(in(other, "gh pr merge 12")).Reason; !strings.Contains(r, "publishes or merges, like a git push") {
		t.Errorf("deny reason: %s", r)
	}
	// In a listed repository without the cd, the reason says how to write it.
	p.Cwd = app
	if r := p.Command("gh pr merge 12").Reason; !strings.Contains(r, "cd <repo> && gh pr merge") {
		t.Errorf("no cd reason: %s", r)
	}
	p.Lang = "es"
	if r := p.Command("gh pr merge 12").Reason; !strings.Contains(r, "escríbelo como `cd <repo> && gh pr merge") {
		t.Errorf("Spanish no cd reason: %s", r)
	}
}

func TestParseRemotes(t *testing.T) {
	cfg := `# a comment
[core]
	url = not-a-remote
[remote "origin"]
	url = "git@github.com:acme/app.git" ; the main one
	URL=https://github.com/acme/app
[remote.old]
	url = https://example.com/x/y # old form
[remote]
	url = nameless
[branch "main"]
	remote = origin
`
	got := strings.Join(parseRemotes(strings.NewReader(cfg)), " ")
	if want := "git@github.com:acme/app.git https://github.com/acme/app https://example.com/x/y"; got != want {
		t.Errorf("parseRemotes = %q, want %q", got, want)
	}
}

func TestSameRepo(t *testing.T) {
	for _, c := range []struct {
		named, remote string
		want          bool
	}{
		{"acme/app", "git@github.com:acme/app.git", true},
		{"acme/app", "https://user:pw@github.com/acme/app/", true},
		{"github.com/acme/app", "ssh://git@github.com:2222/acme/app.git", true},
		{"acme/app", "git@ghe.example.com:acme/app.git", false}, // no host is github.com
		{"ghe.example.com/acme/app", "git@ghe.example.com:acme/app.git", true},
		{"acme/app", "/srv/git/acme/app.git", false}, // a local path has no host
		{"acme/ap", "git@github.com:acme/app.git", false},
		{"", "git@github.com:acme/app.git", false},
	} {
		if got := sameRepo(c.named, c.remote, "github.com"); got != c.want {
			t.Errorf("sameRepo(%q, %q) = %v, want %v", c.named, c.remote, got, c.want)
		}
	}
}

// Found in use: `gh pr merge` written in the text of another argument (the body of a
// pull request, a commit message) is not a merge. Only the words the shell would run
// count, as for a push; and in a real merge, the text given to an option (a body, a
// subject) is not an address or an option of the command.
func TestMergeOnlyInCommandWords(t *testing.T) {
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	app := filepath.Join(home, "code", "app")
	lab := filepath.Join(home, "code", "lab")
	fakeRepo(t, app, map[string]string{"origin": "git@github.com:acme/app.git"})
	fakeRepo(t, lab, map[string]string{"origin": "git@gitlab.com:team/lab.git"})
	p := Policy{Cwd: home, GitAllowedRepos: []string{app, lab}}
	in := func(dir, cmd string) string { return "cd '" + filepath.ToSlash(dir) + "' && " + cmd }
	for _, c := range []struct {
		cmd  string
		want Verdict
	}{
		// gh pr merge only as text: never a merge, wherever it runs
		{`gh pr create --title "Fix" --body "When CI is green, gh pr merge 5 --admin"`, Allow},
		{`gh pr create --title Fix --body 'then: gh pr merge 5 -R evil/app'`, Allow},
		{"gh pr create --title Fix --body \"$(cat <<'EOF'\n## Next\nRun `gh pr merge 5` (after review).\nEOF\n)\"", Allow},
		{`gh pr comment 5 --body "gh pr merge 5"`, Allow},
		{`glab mr create --title Fix --description "then glab mr merge 7"`, Allow},
		{`echo "gh pr merge 5"`, Allow},
		{"git -C '" + filepath.ToSlash(app) + "' commit -m \"Docs: how to gh pr merge 5\"", Allow},
		{in(app, `gh pr create --body "gh pr merge 5"`), Allow},
		{in(app, `gh pr merge 12 --body "--admin, as in gh pr merge 5 --admin"`), Allow},
		{in(app, `gh pr merge 12 --body "see https://github.com/acme/other/pull/3"`), Allow},
		{in(app, `gh pr merge 12 --body="see https://github.com/acme/other/pull/3"`), Allow},
		{in(app, `gh pr merge 12 -b"see https://github.com/acme/other/pull/3"`), Allow},
		{in(app, `gh pr merge 12 -t "Merge https://github.com/acme/other/pull/3"`), Allow},
		{in(lab, `glab mr merge 7 -m "see https://gitlab.com/team/other/-/merge_requests/3"`), Allow},
		{in(lab, `glab mr merge 7 --squash-message="see https://gitlab.com/team/other/-/merge_requests/3"`), Allow},
		// the merges themselves still count
		{in(app, "gh pr merge 12 --admin"), Ask},
		{in(app, "gh pr merge 12 -R evil/app"), Deny},
		{in(app, "gh pr merge -m https://github.com/acme/other/pull/3"), Deny}, // gh's -m is --merge, with no value
		{"gh pr merge 5", Deny},
		{"gh pr create --body x && gh pr merge 5", Deny},
		{"gh pr create --body \"`gh pr merge 5`\"", Deny}, // backquotes run, even inside double quotes
	} {
		if d := p.Command(c.cmd); d.Verdict != c.want {
			t.Errorf("%q: %v (%s), want %v", c.cmd, d.Verdict, d.Reason, c.want)
		}
	}
	// A commit that is not allowed is denied as a commit, not as a merge.
	if r := p.Command(`git commit -m "then gh pr merge 5"`).Reason; !strings.Contains(r, "`git commit` is for the user") {
		t.Errorf("commit reason: %s", r)
	}
}
