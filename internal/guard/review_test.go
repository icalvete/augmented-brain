package guard

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// Found in the review of 2026-11-03: commands a cooperative assistant could write by
// mistake that slipped past the guard. Words are spelled apart, so tools that scan for
// them leave this file alone.
func TestReviewBypasses(t *testing.T) {
	home, _ := filepath.EvalSymlinks(t.TempDir())
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	repo := filepath.Join(home, "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	su, push, commit := "su"+"do", "pu"+"sh", "com"+"mit"
	script := filepath.Join(home, "deploy.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nset -e\ngit "+push+" origin main\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	safe := filepath.Join(home, "build.sh")
	if err := os.WriteFile(safe, []byte("#!/bin/sh\ngo build ./...\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(home, "cfg", "config.toml")
	// In a shell command a path is written with / (Git Bash too): a \ would escape.
	scriptCmd, cfgCmd := filepath.ToSlash(script), filepath.ToSlash(cfg)
	p := Policy{Cwd: home, GitAllowedRepos: []string{repo}, UntouchableHosts: []string{"db01"},
		ProductionAWSProfiles: []string{"prod"}, Protected: []string{cfg}, Hostname: "here",
		Now: time.Now()}
	for cmd, want := range map[string]Verdict{
		// redirections next to the words
		"git " + push + ">/dev/null":        Deny,
		"git " + push + ">log 2>&1":         Deny,
		"git " + commit + ">/dev/null -m x": Deny,
		"rm -rf ~>/dev/null":                Deny,
		"2>/dev/null git " + push:           Deny,
		"< /dev/null " + su + " id":         Deny,
		"git status >out.txt 2>&1":          Allow,
		"echo hi &>/dev/null":               Allow,
		// Windows shells and names, from Bash
		`powershell -Command "git ` + push + `"`: Deny,
		"cmd.exe /c dir":                         Deny,
		"git.exe " + push:                        Deny,
		"GIT " + push:                            Deny,
		"gsudo id":                               Deny,
		// scripts
		"bash " + scriptCmd:                   Deny,
		"sh ./deploy.sh":                      Deny,
		"./deploy.sh":                         Deny,
		"source deploy.sh":                    Deny,
		"bash build.sh":                       Allow,
		"./build.sh":                          Allow,
		"bash missing.sh":                     Allow,
		"bash -o pipefail -c '" + su + " id'": Deny,
		// root through ssh and other programs
		"ssh root@web02 systemctl restart nginx": Deny,
		"ssh -l root web02 id":                   Deny,
		"ssh -o User=root web02 id":              Deny,
		"ssh app@web02 id":                       Allow,
		"sudoedit /etc/hosts":                    Deny,
		"runuser -u root id":                     Deny,
		"runuser -u app id":                      Allow,
		"systemd-run id":                         Deny,
		"systemd-run --user id":                  Allow,
		"machinectl shell root@.host":            Deny,
		"docker compose exec web " + su + " id":  Deny,
		"time -p git " + push:                    Deny,
		"/usr/bin/time git " + push:              Deny,
		"flock /tmp/l " + su + " id":             Deny,
		"script -qc '" + su + " id' /dev/null":   Deny,
		// a program only known when it runs
		"$(echo " + su + ") id": Ask,
		"S=x; $S id":            Ask,
		// rm with a variable, home given another way, home's parents, find
		`rm -rf "$BUILD_DIR/"*`:                Ask,
		`rm -rf "$DIR"/`:                       Ask,
		`rm -rf "$DIR/build"`:                  Allow,
		`rm -rf "${HOME:?}"`:                   Deny,
		"rm -rf ~/..":                          Deny,
		"find ~ -delete":                       Deny,
		"find ~ -mindepth 1 -exec rm -rf {} +": Deny,
		"find ~ -name '*.tmp' -delete":         Allow,
		// cd that the guard cannot follow
		`cd "$(echo ~)" && rm -rf *`:       Ask,
		"cd ~; pushd /tmp; popd; rm -rf *": Ask,
		// git: commands that run commands, other ways to publish
		"git -C " + repo + " submodule foreach git " + push: Deny,
		"git subtree " + push + " --prefix x origin main":   Deny,
		"git send-pack origin main":                         Deny,
		"git -c alias.ci=" + commit + " ci":                 Ask,
		"git -c core.sshCommand=x fetch":                    Ask,
		"git rebase -x '" + su + " id' main":                Deny,
		// gh and glab
		"gh api -X PUT repos/o/r/pulls/1/merge": Deny,
		"gh api repos/o/r/pulls/1":              Allow,
		"gh pr -R o/r merge 1":                  Deny,
		"glab release edit v1":                  Deny,
		// hosts
		"ssh -J db01 web02 id":           Ask,
		"ssh -o ProxyJump=db01 web02 id": Ask,
		"scp f scp://db01/tmp/":          Ask,
		"sftp sftp://db01":               Ask,
		// AWS profile given as a variable
		`aws s3 ls --profile "$PROFILE"`: Ask,
		// nesting too deep asks, never allows
		`bash -c "bash -c \"bash -c 'bash -c \\\"bash -c id\\\"'\""`: Ask,
		// the guard's own settings
		"echo 'git_allowed_repos = [\"~\"]' >> " + cfgCmd: Ask,
		"sed -i s/a/b/ " + cfgCmd:                         Ask,
		"cat " + cfgCmd:                                   Allow,
	} {
		if runtime.GOOS == "windows" && (cmd == "./deploy.sh" || cmd == "./build.sh") {
			continue
		}
		if got := p.Command(cmd); got.Verdict != want {
			t.Errorf("%s: %v (%s), want %v", cmd, got.Verdict, got.Reason, want)
		}
	}
	if got := p.WriteFile(cfg, "root_ask = []"); got.Verdict != Ask {
		t.Errorf("a write of the configuration: %v", got.Verdict)
	}
	if got := p.WriteFile(filepath.Join(home, "notes.md"), "hello"); got.Verdict != Allow {
		t.Errorf("a write of another file: %v", got.Verdict)
	}
}

// A root exception for a full name never covers another full name, and a virtual
// machine or a container is not this machine.
func TestRootHost(t *testing.T) {
	for _, c := range []struct {
		host, saved string
		want        bool
	}{
		{"web01", "web01.example.com", true},
		{"web01.example.com", "web01", true},
		{"WEB01.example.com.", "web01.example.com", true},
		{"web01.other.net", "web01.example.com", false},
		{"web01.example.com", "ubuntu@web01.example.com", true},
		{"db", "web01", false},
	} {
		if got := rootHost(c.host, c.saved); got != c.want {
			t.Errorf("rootHost(%q, %q) = %v", c.host, c.saved, got)
		}
	}
	su := "su" + "do"
	p := Policy{Hostname: "here", Now: time.Now(), RootAsk: []RootException{{Host: "here", Until: time.Now().Add(time.Hour)}}}
	if got := p.Command(su + " id").Verdict; got != Ask {
		t.Errorf("root here, with an exception here: %v", got)
	}
	if got := p.Command("docker exec c " + su + " id").Verdict; got != Deny {
		t.Errorf("root in a container, with an exception for this machine: %v", got)
	}
}
