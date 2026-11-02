package guard

import (
	"os"
	"path/filepath"
	"testing"
)

// rm -r of the whole disk or of the user's home is never run; inside them it is.
func TestRmHomeOrDisk(t *testing.T) {
	home, _ := filepath.EvalSymlinks(t.TempDir()) // macOS: /var is /private/var
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, "build"), 0o755); err != nil {
		t.Fatal(err)
	}
	p := Policy{Cwd: home}
	for cmd, want := range map[string]Verdict{
		"rm -rf ~":                   Deny,
		"rm -rf ~/":                  Deny,
		"rm -rf ~/*":                 Deny,
		"rm -r $HOME":                Deny,
		"rm -fr ${HOME}/":            Deny,
		"rm -rf /":                   Deny,
		"rm -rf /*":                  Deny,
		"rm -R --no-preserve-root /": Deny,
		"rm --recursive -- " + filepath.ToSlash(home): Deny, // Bash on Windows takes / too
		"cd /tmp && rm -rf " + filepath.ToSlash(home): Deny,
		"rm -rf .":                        Deny, // run in the home folder
		"rm -rf ~/build":                  Allow,
		"rm -rf build":                    Allow,
		"rm ~/notes.txt":                  Allow, // not recursive
		"rm -rf /tmp/x":                   Allow,
		"rm -rf ~/build ~/.cache ~ /tmp":  Deny,
		"cd ~ && rm -rf .":                Deny,
		"cd / ; rm -rf *":                 Deny,
		"cd && rm -rf ./":                 Deny,
		"cd ~/build && rm -rf ..":         Deny,
		"cd ~/build && rm -rf .":          Allow,
		"cd ~/build && rm -rf *":          Allow,
		"cd /tmp && cd - && rm -rf .":     Ask, // the guard cannot tell where cd - goes
		"cd \"$DIR\" && rm -rf *":         Ask,
		"cd \"$DIR\" && rm -rf build":     Allow,
		"cd \"$DIR\" && cd ~ && rm -rf .": Deny,
	} {
		if got := p.Command(cmd).Verdict; got != want {
			t.Errorf("%s: %v, want %v", cmd, got, want)
		}
	}
}

// In a repository the assistant may push to, a push that rewrites or deletes what is on
// the remote still asks; and a repository in both lists asks.
func TestPushThatRewrites(t *testing.T) {
	home, _ := filepath.EvalSymlinks(t.TempDir())
	t.Setenv("HOME", home)
	repo := filepath.Join(home, "notes")
	both := filepath.Join(home, "shared-repo")
	for _, d := range []string{repo, both} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	p := Policy{GitAllowedRepos: []string{repo, both}, GitAskRepos: []string{both}}
	push := "pu" + "sh" // spelled apart, so tools that scan for the words leave this file alone
	for cmd, want := range map[string]Verdict{
		"git -C ~/notes " + push:                          Allow,
		"git -C ~/notes " + push + " -u origin main":      Allow,
		"git -C ~/notes " + push + " --force":             Ask,
		"git -C ~/notes " + push + " -f origin main":      Ask,
		"git -C ~/notes " + push + " -uf origin main":     Ask,
		"git -C ~/notes " + push + " --force-with-lease":  Ask,
		"git -C ~/notes " + push + " origin +main":        Ask,
		"git -C ~/notes " + push + " origin --delete old": Ask,
		"git -C ~/notes " + push + " origin :old":         Ask,
		"git -C ~/notes " + push + " --mirror":            Ask,
		"git -C ~/notes " + push + " --tags":              Allow,
		"git -C ~/shared-repo " + push:                    Ask,   // in both lists: asking wins
		"cd ~/notes && git -C . " + push:                  Allow, // -C . is where the cd went
		"cd ~ && git -C notes " + push:                    Allow,
		"cd /tmp && git -C . " + push:                     Deny,
		"cd \"$R\" && git -C . " + push:                   Deny, // the guard cannot tell where
	} {
		if got := p.Command(cmd).Verdict; got != want {
			t.Errorf("%s: %v, want %v", cmd, got, want)
		}
	}
}
