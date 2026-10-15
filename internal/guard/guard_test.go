package guard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommand(t *testing.T) {
	home := t.TempDir()
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

func TestCommands(t *testing.T) {
	got := commands(`a 'b c' "d \"e\"" f\ g; h | i && j # comment
k`)
	want := [][]string{{"a", "b c", `d "e"`, "f g"}, {"h"}, {"i"}, {"j"}, {"k"}}
	if len(got) != len(want) {
		t.Fatalf("got %q", got)
	}
	for i := range want {
		if strings.Join(got[i], "|") != strings.Join(want[i], "|") {
			t.Errorf("command %d: got %q, want %q", i, got[i], want[i])
		}
	}
}
