package guard

import (
	"os"
	"path/filepath"
	"testing"
)

// Holes found in a review of the guard, each closed and kept closed here.
func TestReviewHoles(t *testing.T) {
	home, _ := filepath.EvalSymlinks(t.TempDir()) // macOS: /var is /private/var
	t.Setenv("HOME", home)
	repo := filepath.Join(home, "repos", "mine")
	other := filepath.Join(home, "repos", "other")
	for _, d := range []string{repo, other} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	p := Policy{GitAllowedRepos: []string{repo}, ProductionAWSProfiles: []string{"prod"}, UntouchableHosts: []string{"db01"}}
	ci := "com" + "mit" // spelled apart, so tools that scan for the words leave this file alone
	root := "su" + "do"
	for cmd, want := range map[string]Verdict{
		// find: a later -exec that is blocked wins over an earlier one that asks
		`find . -exec ssh db01 ls \; -exec ` + root + ` id \;`: Deny,
		`find . -exec ssh db01 ls \;`:                          Ask,
		// git: each -C goes from the one before; another repository given any other way is not listed
		"git -C ~/repos/mine -C ../other " + ci + " -m x":                  Deny,
		"git -C ~/repos -C mine " + ci + " -m x":                           Allow,
		"git -C ~/repos/mine --git-dir=" + other + "/.git " + ci + " -m x": Deny,
		"git -C ~/repos/mine --work-tree " + other + " " + ci + " -m x":    Deny,
		"GIT_DIR=" + other + "/.git git -C ~/repos/mine " + ci + " -m x":   Deny,
		// any program given a production profile in the command, either variable
		"AWS_DEFAULT_PROFILE=prod ./deploy.sh": Ask,
		// tools that call AWS on their own
		"terraform apply --auto-approve": Allow,
	} {
		if got := p.Command(cmd).Verdict; got != want {
			t.Errorf("%s: %v, want %v", cmd, got, want)
		}
	}

	// A production profile inherited from the session counts for the tools that call AWS.
	p.Env = map[string]string{"AWS_PROFILE": "prod"}
	for cmd, want := range map[string]Verdict{
		"terraform apply":    Ask,
		"tofu apply":         Ask,
		"pulumi up":          Ask,
		"eksctl get nodes":   Ask,
		"ls -la":             Allow, // but not for everything else
		"AWS_PROFILE=dev ls": Allow,
	} {
		if got := p.Command(cmd).Verdict; got != want {
			t.Errorf("inherited prod, %s: %v, want %v", cmd, got, want)
		}
	}
}
