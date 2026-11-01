package guard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommand(t *testing.T) {
	home, _ := filepath.EvalSymlinks(t.TempDir()) // macOS: /var is /private/var
	t.Setenv("HOME", home)
	repo := filepath.Join(home, "repos", "mine")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	p := Policy{
		GitAllowedRepos:       []string{repo},
		ProductionAWSProfiles: []string{"prod"},
		UntouchableHosts:      []string{"db01"},
	}
	// Built at run time so this file holds no secret-shaped text.
	awsKey := "AKI" + "A" + strings.Repeat("Q", 16)
	ghToken := "gh" + "p_" + strings.Repeat("a", 36)

	cases := []struct {
		cmd  string
		want Verdict
	}{
		// everyday commands
		{"ls -la", Allow},
		{"go test ./...", Allow},
		{"git status && git diff --cached", Allow},
		{"git log --oneline -5", Allow},
		{`grep -rn "git commit" docs/`, Allow},                                  // the words, not the command
		{`echo "never run git push here"`, Allow},                               //
		{"printf 'su propia área\\n' > nota.md", Allow},                         // Spanish "su" in an argument
		{"cat > nota.md <<'EOF'\nTiene su propia área.\nsudo nada\nEOF", Allow}, // heredoc body is data
		{"cat <<EOF\nhola\nEOF\nls", Allow},
		// the panel the plugin's /brain:panel command opens
		{"augmented-brain panel", Allow},
		{"augmented-brain panel stop", Allow},
		{"~/.local/bin/augmented-brain panel --no-browser", Allow},
		{`PATH="$PATH:$HOME/.local/bin" augmented-brain panel`, Allow},

		// git
		{"git commit -m x", Deny},
		{"git -c user.name=x commit -m x", Deny},
		{"git push origin main", Deny},
		{"cd ~/repos/mine && git commit -m x", Deny}, // allowed repo, but not written with -C
		{"git -C ~/repos/mine commit -m x", Allow},
		{"git -C ~/repos/mine push", Allow},
		{"git -C ~/repos/other push", Deny},
		{"git -C ~/repos/mine push && git -C ~/repos/other push", Deny},
		{"ls; git push", Deny},
		{"(cd x && git push)", Deny},
		{"echo $(git push)", Deny},

		// root
		{"sudo apt install jq", Deny},
		{"sudo -n true", Deny},
		{"sudo -u root id", Deny},
		{"sudo -i", Deny},
		{"sudo -u deploy id", Allow},
		{"sudo -iu deploy cmk -O", Allow},
		{"sudo --user=deploy id", Allow},
		{"sudo -u deploy sudo id", Deny}, // chained back to root
		{"su -", Deny},
		{"/usr/bin/sudo id", Deny},
		{"cat <<EOF && sudo id\nx\nEOF", Deny}, // the rest of the heredoc line is still a command

		// commands that run other commands
		{`ssh web01 "sudo systemctl restart nginx"`, Deny},
		{`ssh -p 2222 deploy@web01 'sudo -u app ./restart.sh'`, Allow},
		{`bash -c "git push"`, Deny},
		{`env FOO=1 nohup sudo id`, Deny},
		{`timeout 10 sudo id`, Deny},

		// secrets
		{"curl -H 'Authorization: token " + ghToken + "' https://api.example.com", Deny},
		{"AWS_ACCESS_KEY_ID=" + awsKey + " aws s3 ls", Deny},
		{"AWS_SECRET_ACCESS_KEY=abc aws s3 ls", Deny},
		{"AWS_SECRET_ACCESS_KEY=$SECRET aws s3 ls", Allow}, // read from a variable: fine

		// production and untouchable hosts
		{"aws s3 ls --profile prod", Ask},
		{"aws s3 ls --profile=prod", Ask},
		{"AWS_PROFILE=prod aws ec2 describe-instances", Ask},
		{"AWS_PROFILE=prod terraform apply", Ask},
		{"aws s3 ls --profile dev", Allow},
		{"ssh db01 uptime", Ask},
		{"ssh admin@db01", Ask},
		{"ssh db02 uptime", Allow},
		{`ssh db01 "sudo reboot"`, Deny}, // deny wins over ask

		// found in the review before 0.1: compound commands
		{"for f in a b; do git push; done", Deny},
		{"if true; then sudo id; fi", Deny},
		{"{ git push; }", Deny},
		{"if git diff --quiet; then :; else git commit -am wip; fi", Deny},
		{"while read l; do echo $l; done < f", Allow},
		// options and separators
		{"sudo -u app -- git push", Deny},
		{"sudo -u app -- sudo id", Deny},
		{"sudo -u app bash -c 'git push'", Deny},
		{"bash -lc 'git push'", Deny},
		{"sh -ec 'sudo id'", Deny},
		{"bash -lc 'ls -la'", Allow},
		// shells reading a heredoc or a here-string
		{"bash <<'EOF'\ngit push\nEOF", Deny},
		{"ssh web01 bash <<'EOF'\nsudo systemctl restart x\nEOF", Deny},
		{"ssh web01 <<'EOF'\nsudo systemctl restart x\nEOF", Deny},
		{"bash <<< 'git push'", Deny},
		{"cat <<'EOF' > notes.md\ngit push and sudo are words here\nEOF", Allow},
		// substitutions
		{`echo "$(git push)"`, Deny},
		{`out="$(sudo id)"`, Deny},
		{"echo `git push`", Deny},
		{"diff <(sudo cat /etc/shadow) x", Deny},
		{`echo "today is $(date +%F)"`, Allow},
		// exported variables carry over
		{"export AWS_PROFILE=prod; aws s3 ls", Ask},
		{"export AWS_PROFILE=prod && terraform plan", Ask},
		// wrappers whose options take a value
		{"nice -n 10 git push", Deny},
		{"xargs -n 1 git push", Deny},
		{"timeout -k 5 10 git push", Deny},
		{"env -u FOO git push", Deny},
		{"watch -n 5 sudo id", Deny},
		{"find . -name x -exec git push \\;", Deny},
		{"find . -name '*.go' -exec gofmt -l {} +", Allow},
		{"run0 id", Deny},
		{"pkexec id", Deny},
		// git global options with a separate value
		{"git --git-dir .git commit -m x", Deny},
		{"git --work-tree . push", Deny},
		{"git --namespace x push", Deny},
		{"git --no-pager log -1", Allow},
		// no longer false positives
		{"command -v sudo", Allow},
		{"sudo -l", Allow},
		{"sudo -v", Allow},
		{"sudo -p 'pw:' -u app id", Allow},
		{"sudo -e /etc/hosts", Deny},
		// hosts
		{"ssh -B eth0 db01 uptime", Ask},
		{"ssh db01.example.com uptime", Ask},
		{"scp backup.sql db01:/tmp/", Ask},
		{"rsync -a ./ admin@db01:/srv/", Ask},
		{"scp backup.sql web01:/tmp/", Allow},
	}
	for _, c := range cases {
		d := p.Command(c.cmd)
		if d.Verdict != c.want {
			t.Errorf("%q: got %v (%s), want %v", c.cmd, d.Verdict, d.Reason, c.want)
		}
		if d.Verdict != Allow && d.Reason == "" {
			t.Errorf("%q: no reason given", c.cmd)
		}
	}
}

func TestNoPolicy(t *testing.T) {
	var p Policy
	if p.Command("git push").Verdict != Deny || p.Command("sudo id").Verdict != Deny {
		t.Error("the general rules must apply with no space")
	}
	if p.Command("aws s3 ls --profile prod").Verdict != Allow {
		t.Error("with no space there are no production profiles")
	}
}

func TestWrite(t *testing.T) {
	key := "-----BEGIN " + "PRIVATE KEY-----"
	if Write("config:\n  key: "+key).Verdict != Deny {
		t.Error("private key not caught")
	}
	if Write("password: read it from $DB_PASSWORD").Verdict != Allow {
		t.Error("the word password is not a secret")
	}
}

func TestParse(t *testing.T) {
	got := parse(`a 'b c' "d \"e\"" f\ g; h | i && j # comment
k`)
	want := [][]string{{"a", "b c", `d "e"`, "f g"}, {"h"}, {"i"}, {"j"}, {"k"}}
	if len(got) != len(want) {
		t.Fatalf("got %+v", got)
	}
	for i := range want {
		if strings.Join(got[i].words, "|") != strings.Join(want[i], "|") {
			t.Errorf("command %d: got %q, want %q", i, got[i].words, want[i])
		}
	}
	hd := parse("cat <<EOF | wc -l\nline one\nline two\nEOF\nls")
	if len(hd) != 3 || hd[0].stdin != "line one\nline two\n" || hd[2].words[0] != "ls" {
		t.Errorf("heredoc: %+v", hd)
	}
}

func TestRelativeGitDir(t *testing.T) {
	home, _ := filepath.EvalSymlinks(t.TempDir())
	repo := filepath.Join(home, "repo")
	sub := filepath.Join(repo, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	p := Policy{GitAllowedRepos: []string{repo}, Cwd: sub}
	if p.Command("git -C . push").Verdict != Deny {
		t.Error("-C . is the sub folder, not the allowed repo")
	}
	if p.Command("git -C .. push").Verdict != Allow {
		t.Error("-C .. from the sub folder is the allowed repo")
	}
}

// Commands checked against the profile they inherit, the tools that take an AWS profile,
// publishing with gh and glab, shells fed through a pipe and host names.
func TestReviewFixes(t *testing.T) {
	p := Policy{
		ProductionAWSProfiles: []string{"prod", "default-prod"},
		UntouchableHosts:      []string{"db01", "Backup.Example.COM"},
	}
	cases := []struct {
		env  map[string]string
		cmd  string
		want Verdict
	}{
		// the profile an aws command uses without --profile
		{map[string]string{"AWS_PROFILE": "prod"}, "aws s3 ls", Ask},
		{map[string]string{"AWS_DEFAULT_PROFILE": "prod"}, "aws s3 ls", Ask},
		{map[string]string{"AWS_PROFILE": "dev", "AWS_DEFAULT_PROFILE": "prod"}, "aws s3 ls", Ask}, // any candidate counts
		{map[string]string{"AWS_PROFILE": "prod"}, "aws s3 ls --profile dev", Ask},
		{map[string]string{"AWS_PROFILE": "dev"}, "aws s3 ls --profile dev", Allow},
		{nil, "AWS_DEFAULT_PROFILE=prod aws s3 ls --profile dev", Ask},
		{map[string]string{"AWS_PROFILE": "prod"}, "cdk deploy", Ask},
		{map[string]string{"AWS_PROFILE": "dev"}, "cdk deploy", Allow},
		{nil, "aws s3 ls", Allow},
		// cdk, sam, serverless and aws-vault
		{nil, "cdk deploy --profile prod", Ask},
		{nil, "sam deploy --profile=prod", Ask},
		{nil, "serverless deploy --aws-profile prod", Ask},
		{nil, "npx cdk deploy --profile dev", Allow},
		{nil, "npx cdk deploy --profile prod", Ask},
		{nil, "aws-vault exec prod -- aws s3 ls", Ask},
		{nil, "aws-vault exec --duration 1h prod -- terraform plan", Ask},
		{nil, "aws-vault exec dev -- aws s3 ls", Allow},
		{nil, "aws-vault exec dev -- sudo id", Deny},
		// publishing and merging, like a push
		{nil, "gh pr merge 12 --squash", Deny},
		{nil, "gh release create v1.0.0", Deny},
		{nil, "glab mr merge 7", Deny},
		{nil, "gh pr view 12", Allow},
		{nil, "gh pr list", Allow},
		{nil, "glab mr accept 7", Deny},
		{nil, "gh repo sync", Allow},
		{nil, "gh repo sync --branch v1", Allow},
		{nil, "gh repo sync owner/fork", Deny},
		{nil, "gh repo sync --source owner2/repo2", Allow},
		{nil, "gh repo sync owner/repo --source owner2/repo2", Deny},
		// what reaches a shell through a pipe
		{nil, "cat <<'E' | bash\nsudo id\nE", Deny},
		{nil, "echo 'sudo ls' | bash", Deny},
		{nil, "printf 'git push\\n' | sh", Deny},
		{nil, "echo 'sudo ls' | ssh web01", Deny},
		{nil, "cat <<'E' | sudo -u app bash\nls\nE", Allow},
		{nil, "echo 'sudo ls' | grep sudo", Allow},
		{nil, "echo 'sudo id' | (bash)", Deny},
		{nil, "echo 'sudo id' | tee x | bash", Deny},
		{nil, "echo 'sudo id' | { bash; }", Deny},
		{nil, "(echo 'sudo id') | bash", Deny},
		{nil, "echo 'sudo ls' || bash", Allow},
		{nil, "ssh db01 < script.sh", Ask},
		{nil, "ssh web01 < script.sh", Allow},
		// host names: any case, short name and full name both ways, sftp and mosh
		{nil, "ssh DB01 uptime", Ask},
		{nil, "ssh db01.example.com uptime", Ask},
		{nil, "ssh backup uptime", Ask},
		{nil, "ssh backup.example.com", Ask},
		{nil, "ssh backups uptime", Allow},
		{nil, "sftp db01", Ask},
		{nil, "sftp -P 2222 admin@db01", Ask},
		{nil, "sftp web01", Allow},
		{nil, "mosh db01", Ask},
		{nil, "mosh --ssh 'ssh -p 2222' admin@db01 -- htop", Ask},
		{nil, "mosh web01 -- sudo id", Deny},
		{nil, "mosh web01", Allow},
	}
	for _, c := range cases {
		q := p
		q.Env = c.env
		d := q.Command(c.cmd)
		if d.Verdict != c.want {
			t.Errorf("%q (env %v): got %v (%s), want %v", c.cmd, c.env, d.Verdict, d.Reason, c.want)
		}
	}

	// A "default" profile that is production is the one aws uses with nothing set.
	if (Policy{ProductionAWSProfiles: []string{"default"}}).Command("aws s3 ls").Verdict != Ask {
		t.Error("the default profile is not checked")
	}
	if (Policy{ProductionAWSProfiles: []string{"default"}}).Command("sam deploy").Verdict != Ask {
		t.Error("the default profile is not checked for sam")
	}
	if (Policy{ProductionAWSProfiles: []string{"default"}}).Command("aws s3 ls --profile dev").Verdict != Allow {
		t.Error("default counted although a profile was given")
	}
}

// When the configuration cannot be read, what the lists would cover asks every time;
// the rest goes on as usual.
func TestConfigProblem(t *testing.T) {
	p := Policy{ConfigProblem: "config.toml: line 3: bad"}
	for _, cmd := range []string{"aws s3 ls", "ssh web01 uptime", "scp x web01:/tmp", "sftp web01", "mosh web01", "cdk deploy --profile x"} {
		if d := p.Command(cmd); d.Verdict != Ask || !strings.Contains(d.Reason, "cannot read its configuration") {
			t.Errorf("%q: %v (%s)", cmd, d.Verdict, d.Reason)
		}
	}
	for cmd, want := range map[string]Verdict{"ls -la": Allow, "git push": Deny, "ssh web01 sudo id": Deny} {
		if d := p.Command(cmd); d.Verdict != want {
			t.Errorf("%q: %v, want %v", cmd, d.Verdict, want)
		}
	}
}

// The keys of AWS's documentation are examples, not secrets; a real one next to them is.
func TestAWSExampleKeys(t *testing.T) {
	id, secret := examples[0], examples[1]
	if what := FindSecret("aws_access_key_id = " + id + "\naws_secret_access_key = " + secret + "\n"); what != "" {
		t.Errorf("AWS's example keys were taken for secrets: %s", what)
	}
	real := "AKI" + "A" + strings.Repeat("Q", 16)
	if FindSecret("example "+id+" and real "+real) == "" {
		t.Error("a real key next to the example one was missed")
	}
}

// The Monitor tool runs a command in the background and reads its output line by line.
// Its commands tend to be watch loops and pipes; what they run is checked as in Bash.
func TestMonitorShapes(t *testing.T) {
	p := Policy{ProductionAWSProfiles: []string{"prod"}, UntouchableHosts: []string{"db01"}}
	cases := []struct {
		cmd  string
		want Verdict
	}{
		{"tail -f app.log | grep --line-buffered ERROR", Allow},
		{"while true; do git status --short; sleep 30; done", Allow},
		{"for i in 1 2 3; do echo tick $i; sleep 1; done", Allow},
		{"while true; do aws s3 ls --profile prod; sleep 60; done", Ask},
		{"until git push; do sleep 10; done", Deny},
		{"ssh db01 tail -f /var/log/syslog", Ask},
		{"tail -f app.log | sudo tee /var/log/copy", Deny},
	}
	for _, c := range cases {
		if got := p.Command(c.cmd); got.Verdict != c.want {
			t.Errorf("%q: got %v (%s), want %v", c.cmd, got.Verdict, got.Reason, c.want)
		}
	}
}
