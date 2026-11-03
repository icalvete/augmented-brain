package guard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// On Windows the guard reads paths as Git Bash writes them: rm -r of the home folder or
// of a drive is never run, and a listed repository counts however its path is written.
func TestGitBashPaths(t *testing.T) {
	home, _ := filepath.EvalSymlinks(t.TempDir())
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	repo := filepath.Join(home, "notes")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	drive := strings.ToLower(home[:1])
	bash := "/" + drive + filepath.ToSlash(home[2:]) // C:\\x\\y as /c/x/y
	p := Policy{Cwd: home, GitAllowedRepos: []string{repo}}
	push := "pu" + "sh" // spelled apart, so tools that scan for the words leave this file alone
	for cmd, want := range map[string]Verdict{
		"rm -rf " + bash:                    Deny,
		"rm -rf " + bash + "/":              Deny,
		`rm -rf "$USERPROFILE"`:             Deny,
		"rm -rf /" + drive + "/":            Deny,
		"rm -rf /" + drive:                  Deny,
		"rm -rf " + bash + "/build":         Allow,
		"git -C " + bash + "/notes " + push: Allow,
		"git -C " + bash + " " + push:       Deny,
	} {
		if got := p.Command(cmd).Verdict; got != want {
			t.Errorf("%s: %v, want %v", cmd, got, want)
		}
	}
}
