// Package guard decides, before the assistant runs a command or writes a file, whether
// to let it through, block it, or ask the user first.
//
// It is a safety net against mistakes, not a security boundary: it reads the command as
// text, so a determined program could get around it.
package guard

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Verdict of the guard.
type Verdict int

const (
	Allow Verdict = iota
	Ask           // the user confirms first
	Deny          // blocked; the reason tells the assistant what to do instead
)

// Decision is the guard's answer and the reason given to the assistant.
type Decision struct {
	Verdict Verdict
	Reason  string
}

// Policy is what the guard protects. The lists come from the space of the folder the
// assistant works in; with no space, only the general rules apply.
type Policy struct {
	GitAllowedRepos       []string // absolute paths where the assistant may commit and push
	ProductionAWSProfiles []string // AWS profiles that need the user's confirmation
	UntouchableHosts      []string // hosts that need the user's confirmation for ssh
}

var allow = Decision{Allow, ""}

// Command checks a shell command.
func (p Policy) Command(line string) Decision {
	if what := FindSecret(line); what != "" {
		return Decision{Deny, "the command has " + what + " written in it. Read secrets from the system keychain, an environment variable or a secrets manager, never from the command line, and never ask the user to paste one in the chat."}
	}
	var ask *Decision
	for _, words := range expand(commands(line), 0) {
		d := p.simple(words)
		switch d.Verdict {
		case Deny:
			return d
		case Ask:
			if ask == nil {
				ask = &d
			}
		}
	}
	if ask != nil {
		return *ask
	}
	return allow
}

// expand adds the commands that run inside others: ssh's remote command and the string
// of sh -c, bash -c or eval.
func expand(cmds [][]string, depth int) [][]string {
	out := append([][]string{}, cmds...)
	if depth > 3 {
		return out
	}
	for _, words := range cmds {
		_, name, args := program(words)
		var inner string
		switch name {
		case "ssh":
			if _, rest := sshTarget(args); len(rest) > 0 {
				inner = strings.Join(rest, " ")
			}
		case "sh", "bash", "zsh", "dash":
			for i, a := range args {
				if a == "-c" && i+1 < len(args) {
					inner = args[i+1]
				}
			}
		case "eval":
			inner = strings.Join(args, " ")
		}
		if inner != "" {
			out = append(out, expand(commands(inner), depth+1)...)
		}
	}
	return out
}

func (p Policy) simple(words []string) Decision {
	env, name, args := program(words)
	for _, k := range []string{"AWS_SECRET_ACCESS_KEY", "AWS_ACCESS_KEY_ID", "AWS_SESSION_TOKEN"} {
		if v, ok := env[k]; ok && v != "" && !strings.HasPrefix(v, "$") {
			return Decision{Deny, "the command sets " + k + " with a value written in it. Use an AWS profile (--profile or AWS_PROFILE) instead of keys on the command line."}
		}
	}
	switch name {
	case "git":
		return p.git(args)
	case "sudo", "doas":
		return p.sudo(args)
	case "su":
		return Decision{Deny, "`su` switches to another user, root by default. Never become root: give the user the exact command to run themselves, and wait for its output."}
	case "aws":
		return p.aws(env, args)
	case "ssh":
		return p.ssh(args)
	}
	if prof, ok := env["AWS_PROFILE"]; ok && contains(p.ProductionAWSProfiles, prof) {
		return Decision{Ask, "this runs with the production AWS profile " + prof + "; the user confirms first."}
	}
	return allow
}

// git blocks commit and push unless the repository is explicitly allowed, written with -C.
func (p Policy) git(args []string) Decision {
	dir := ""
	i := 0
	for ; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-C" && i+1 < len(args):
			dir = args[i+1]
			i++
		case a == "-c" && i+1 < len(args):
			i++
		case strings.HasPrefix(a, "-"):
		default:
			goto sub
		}
	}
sub:
	if i >= len(args) || (args[i] != "commit" && args[i] != "push") {
		return allow
	}
	if dir != "" {
		if d, err := resolve(dir); err == nil && contains(p.GitAllowedRepos, d) {
			return allow
		}
	}
	return Decision{Deny, "`git " + args[i] + "` is for the user. Leave the change ready and checked, show what would be committed, and tell the user it is ready. (To let the assistant commit in a repository, add it to git_allowed_repos in the configuration, and use `git -C <repo>`.)"}
}

var userOpt = regexp.MustCompile(`^-[A-Za-z]*u(.*)$`)

// sudo allows running as another user, but never as root, and checks the command it
// runs too (sudo -u app sudo id would end up as root).
func (p Policy) sudo(args []string) Decision {
	user := ""
	i := 0
	for ; i < len(args) && strings.HasPrefix(args[i], "-"); i++ {
		a := args[i]
		switch {
		case a == "--":
			i = len(args)
		case strings.HasPrefix(a, "--user="):
			user = strings.TrimPrefix(a, "--user=")
		case a == "--user" && i+1 < len(args):
			user = args[i+1]
			i++
		case !strings.HasPrefix(a, "--") && userOpt.MatchString(a):
			if v := userOpt.FindStringSubmatch(a)[1]; v != "" {
				user = v
			} else if i+1 < len(args) {
				user = args[i+1]
				i++
			}
		}
	}
	if user == "" || user == "root" || user == "0" || user == "#0" {
		return Decision{Deny, "this would run as root. Never become root: give the user the exact command to run themselves, and wait for its output. Running as another user (sudo -u <user>) is fine."}
	}
	if i < len(args) {
		return p.simple(args[i:])
	}
	return allow
}

func (p Policy) aws(env map[string]string, args []string) Decision {
	prof := env["AWS_PROFILE"]
	for i, a := range args {
		if a == "--profile" && i+1 < len(args) {
			prof = args[i+1]
		} else if strings.HasPrefix(a, "--profile=") {
			prof = strings.TrimPrefix(a, "--profile=")
		}
	}
	if prof != "" && contains(p.ProductionAWSProfiles, prof) {
		return Decision{Ask, "this AWS command uses the production profile " + prof + "; the user confirms first."}
	}
	return allow
}

func (p Policy) ssh(args []string) Decision {
	host, _ := sshTarget(args)
	if host != "" && contains(p.UntouchableHosts, host) {
		return Decision{Ask, "host " + host + " is marked as untouchable in this space; the user confirms first."}
	}
	return allow
}

// sshTarget returns the host of an ssh command and the remote command after it.
func sshTarget(args []string) (string, []string) {
	withValue := "bcDEeFIiJLlmOopQRSWw"
	for i := 0; i < len(args); i++ {
		a := args[i]
		if strings.HasPrefix(a, "-") && len(a) >= 2 {
			if len(a) == 2 && strings.ContainsRune(withValue, rune(a[1])) {
				i++
			}
			continue
		}
		host := a
		if at := strings.LastIndexByte(host, '@'); at >= 0 {
			host = host[at+1:]
		}
		return host, args[i+1:]
	}
	return "", nil
}

// Write checks text the assistant is about to write into a file.
func Write(text string) Decision {
	if what := FindSecret(text); what != "" {
		return Decision{Deny, "the text has " + what + " in it. Secrets never go into files: the code reads them from an environment variable, the system keychain or a secrets manager."}
	}
	return allow
}

func resolve(p string) (string, error) {
	if p == "~" || strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		p = filepath.Join(home, p[1:])
	}
	p, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	if r, err := filepath.EvalSymlinks(p); err == nil {
		p = r
	}
	return p, nil
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
