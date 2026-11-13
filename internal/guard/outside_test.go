package guard

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/icalvete/augmented-brain/internal/i18n"
)

// generalOnly are the reasons only a general rule gives: outside the spaces, none of
// them may come back. The rules about repositories (git.deny, publish, merge…) are
// general only for repositories no list names, so the rows below check them.
var generalOnly = []string{"guard.root", "guard.root.admin", "guard.root.changed", "guard.root.container",
	"guard.root.exception", "guard.root.machinectl", "guard.root.runuser", "guard.root.ssh", "guard.root.su",
	"guard.root.systemdrun", "guard.powershell.bash", "guard.secret.command", "guard.secret.inner",
	"guard.secret.write", "guard.awskey"}

// generalAny are every reason a general rule may give, including those that hold
// outside for a repository of the lists.
var generalAny = append(slices2(generalOnly), "guard.git.deny", "guard.git.publish", "guard.publish",
	"guard.merge.cd", "guard.merge.remote", "guard.merge.unread", "guard.merge.admin", "guard.dispatch",
	"guard.dispatch.bypass", "guard.dispatch.id", "guard.api", "guard.api.dispatch", "guard.api.graphql",
	"guard.gh.alias", "guard.gh.rerun")

func slices2(s []string) []string { return append([]string{}, s...) }

var (
	reasonOnce sync.Once
	reasonRe   map[string][]*regexp.Regexp
)

// reasonIs reports whether a reason was written from one of keys, in either language,
// alone or inside another one (the script x runs a command that needs it: …).
func reasonIs(reason string, keys []string) bool {
	reasonOnce.Do(func() {
		reasonRe = map[string][]*regexp.Regexp{}
		verbs := regexp.MustCompile(`%[-+# 0-9.]*[svdqx]`)
		for _, k := range generalAny {
			for _, tpl := range i18n.All(k) {
				parts := verbs.Split(tpl, -1)
				for i := range parts {
					parts[i] = regexp.QuoteMeta(parts[i])
				}
				reasonRe[k] = append(reasonRe[k], regexp.MustCompile(`(?s)`+strings.Join(parts, `.*`)))
			}
		}
	})
	for _, k := range keys {
		for _, re := range reasonRe[k] {
			if re.MatchString(reason) {
				return true
			}
		}
	}
	return false
}

// checkOutside runs a command as a session outside every space would (no space holds
// the policy's folder) and compares it with the answer inside: never stricter, never a
// reason only a general rule gives, and the same answer whenever no general rule gave
// the one inside.
func checkOutside(t *testing.T, p Policy, cmd string, inside Decision) {
	t.Helper()
	q := p
	q.Outside = true
	q.SpaceDirs = []string{filepath.Join(filepath.Dir(cmp2(p.Cwd)), "a-space-elsewhere")}
	got := q.Command(cmd)
	if got.Verdict > inside.Verdict {
		t.Errorf("outside %q = %v, stricter than inside %v (%s)", cmd, got.Verdict, inside.Verdict, got.Reason)
	}
	if got.Verdict != Allow && reasonIs(got.Reason, generalOnly) {
		t.Errorf("outside %q gives a general rule's answer: %s", cmd, got.Reason)
	}
	if inside.Verdict != Allow && !reasonIs(inside.Reason, generalAny) && got.Verdict != inside.Verdict {
		t.Errorf("outside %q = %v, inside %v by a rule that holds outside (%s)", cmd, got.Verdict, inside.Verdict, inside.Reason)
	}
}

func cmp2(s string) string {
	if s == "" {
		return os.TempDir()
	}
	return s
}

// outsidePolicy is pinnedPolicy for a session outside the spaces, with one space at
// ~/space, a repository at ~/free that no list names, a clone of the listed
// ~/repos/mine at ~/clone, and a protected file.
func outsidePolicy(t *testing.T) (Policy, string) {
	t.Helper()
	p := pinnedPolicy(t)
	home := p.Cwd
	remote := func(dir, url string) {
		if err := os.MkdirAll(filepath.Join(home, dir, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
		cfg := "[remote \"origin\"]\n\turl = " + url + "\n"
		if err := os.WriteFile(filepath.Join(home, dir, ".git", "config"), []byte(cfg), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	remote("repos/mine", "git@github.com:me/mine.git")
	remote("repos/never", "https://github.com/me/never.git")
	remote("free", "git@github.com:me/free.git")
	remote("clone", "https://github.com/me/mine")
	if err := os.MkdirAll(filepath.Join(home, "space", "app"), 0o755); err != nil {
		t.Fatal(err)
	}
	p.Outside = true
	p.SpaceDirs = []string{filepath.Join(home, "space")}
	p.Protected = []string{filepath.Join(home, "cfg", "guard.toml")}
	return p, home
}

func TestOutsideVerdicts(t *testing.T) {
	p, home := outsidePolicy(t)
	tok := "gh" + "p_" + strings.Repeat("a", 36)
	rows := []pinned{
		// General rules let it through.
		{"su_do id", Allow},
		{"su_do apt install x", Allow},
		{"ssh web02 su_do id", Allow},
		{"echo " + tok, Allow},
		{"AWS_SECRET_ACCESS_KEY=abc make", Allow},
		{"powershell -Command Get-Date", Allow},
		{"bash ~/root.sh", Allow},
		// What the lists name still holds.
		{"ssh db01 ls", Ask},
		{"aws s3 ls --profile prod", Ask},
		{"AWS_PROFILE=prod make deploy", Ask},
		{"git -C ~/repos/mine com_mit -m x", Allow},
		{"git -C ~/repos/ask com_mit -m x", Ask},
		{"git -C ~/repos/never com_mit -m x", Deny},
		{"git -C ~/repos/never pu_sh", Deny},
		// A repository nobody listed.
		{"git -C ~/free com_mit -m x", Allow},
		{"git -C ~/free pu_sh", Allow},
		{"git -C ~/free pu_sh --force origin main", Allow},
		{"cd ~/free && git com_mit -m x", Allow},
		{"git -C ~/free send-pack x", Allow},
		// ...unless it reaches one that is listed: a clone, or a push there.
		{"git -C ~/clone pu_sh", Deny},
		{"git -C ~/free pu_sh git@github.com:me/mine.git main", Deny},
		{"git -C ~/free pu_sh https://github.com/me/never", Deny},
		{"git -C ~/free pu_sh ~/repos/never", Deny},
		{"git -C ~/free pu_sh ../repos/mine main", Deny},
		// gh and glab.
		{"cd ~/free && gh pr merge 3", Allow},
		{"cd ~/free && gh release create v1", Allow},
		{"cd ~/free && gh workflow run release", Allow},
		{"cd ~/free && gh api repos/other/x/merges -X POST", Allow},
		{"gh pr merge 3 -R me/mine", Deny},
		{"gh pr merge https://github.com/me/never/pull/3", Deny},
		{"GH_REPO=me/mine gh release create v1", Deny},
		{"gh api repos/me/mine/merges -X POST", Deny},
		{"gh api graphql -f query='mutation { mergePullRequest(input: {}) { clientMutationId } }'", Deny},
		{"glab api projects/123/repository/commits -X POST", Deny},
		{"cd ~/repos/never && gh pr merge 3", Deny},
		// Its own files and the home folder that holds the notes.
		{"echo x > ~/cfg/guard.toml", Ask},
		{"rm -rf ~", Deny},
		{"rm -rf /tmp/build", Allow},
		// A line that reaches into a space is read whole, with every rule.
		{"cd ~/space/app && su_do id", Deny},
		{"su_do id; cd ~/space", Deny},
		{"git -C ~/space/app com_mit -m x", Deny},
		{"echo hi > ~/space/f; su_do id", Deny},
		{"bash ~/space/app/x.sh; echo " + tok, Deny},
		{"cd \"$X\" && su_do id", Deny},
		{"cd - && su_do id", Deny},
		// What the guard cannot tell still asks.
		{"$(echo ssh) db01", Ask},
	}
	for _, r := range rows {
		cmd := strings.ReplaceAll(unspell(r.cmd), "~/", home+"/")
		if got := p.Command(cmd); got.Verdict != r.want {
			t.Errorf("outside %q = %v, want %v (%s)", r.cmd, got.Verdict, r.want, got.Reason)
		}
	}
}

func TestOutsideWrite(t *testing.T) {
	p, home := outsidePolicy(t)
	tok := "gh" + "p_" + strings.Repeat("a", 36)
	if d := p.WriteFile(filepath.Join(home, "notes.txt"), tok); d.Verdict != Allow {
		t.Errorf("a secret outside: %v (%s)", d.Verdict, d.Reason)
	}
	if d := p.WriteFile(filepath.Join(home, "space", "app", "x.env"), tok); d.Verdict != Deny {
		t.Errorf("a secret inside a space: %v", d.Verdict)
	}
	if d := p.WriteFile(filepath.Join(home, "cfg", "guard.toml"), "x"); d.Verdict != Ask {
		t.Errorf("its own file outside: %v", d.Verdict)
	}
	in := p
	in.Outside = false
	if d := in.WriteFile(filepath.Join(home, "notes.txt"), tok); d.Verdict != Deny {
		t.Errorf("a secret with the whole machine: %v", d.Verdict)
	}
}

// Every program with a rule, alone, as root or with a secret, outside: no general
// rule's answer comes back, and with a production profile none goes through.
func TestOutsideEveryRule(t *testing.T) {
	p, _ := outsidePolicy(t)
	su := "su" + "do "
	for name := range rules {
		for _, cmd := range []string{name, su + name, name + " x y", "AWS_PROFILE=prod " + name} {
			d := p.Command(cmd)
			if d.Verdict != Allow && reasonIs(d.Reason, generalOnly) {
				t.Errorf("outside %q: %s", cmd, d.Reason)
			}
			if strings.HasPrefix(cmd, "AWS_PROFILE=prod ") && d.Verdict == Allow {
				t.Errorf("outside %q goes through", cmd)
			}
		}
	}
}

func TestAPIRepo(t *testing.T) {
	for path, want := range map[string]struct {
		repo string
		ok   bool
	}{
		"repos/me/mine/merges":                {"me/mine", true},
		"/repos/me/mine":                      {"me/mine", true},
		"projects/grp%2Fapp/repository/files": {"grp/app", true},
		"projects/123/repository/commits":     {"", false},
		"projects/%zz/x":                      {"", false},
		"graphql":                             {"", false},
		"repos":                               {"", false},
		"user/emails":                         {"", true},
	} {
		repo, ok := apiRepo(path)
		if repo != want.repo || ok != want.ok {
			t.Errorf("apiRepo(%q) = %q %v, want %q %v", path, repo, ok, want.repo, want.ok)
		}
	}
}

// The corners: a list entry with a *, a command on another machine, a repository the
// guard reaches through GIT_DIR, a pull request it cannot read.
func TestOutsideCorners(t *testing.T) {
	p, home := outsidePolicy(t)
	p.GitAllowedRepos = []string{filepath.Join(home, "repos", "mi*")} // ~/repos/mine, by its pattern
	rows := []pinned{
		{"git -C ~/clone pu_sh", Deny},                   // a clone of a repository matched by a *
		{"ssh web02 git -C /srv/app com_mit -m x", Deny}, // another machine: its folders are not known
		{"cd ~/free && GIT_DIR=/x/.git gh pr merge 3", Deny},
		{"cd ~/free && gh pr merge \"$PR\"", Deny},
		{"cd ~/free && gh api graphql -F query=@q.graphql", Ask},
		{"cd ~/free && glab api projects/123/repository/commits -X POST", Deny},
	}
	for _, r := range rows {
		cmd := strings.ReplaceAll(unspell(r.cmd), "~/", home+"/")
		if got := p.Command(cmd); got.Verdict != r.want {
			t.Errorf("outside %q = %v, want %v (%s)", r.cmd, got.Verdict, r.want, got.Reason)
		}
	}
	if p.freeRepo("") {
		t.Error("an unknown folder counts as free")
	}
	q := p
	q.outside = true
	q.lost = true
	if q.freeRepo("relative") {
		t.Error("a folder lost after a cd counts as free")
	}
}
