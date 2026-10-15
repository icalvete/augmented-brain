// Package guard decides, before the assistant runs a command or writes a file, whether
// to let it through, block it, or ask the user first.
//
// It is a safety net against mistakes, not a security boundary: it reads the command as
// text, so a determined program could get around it.
package guard

import (
	"os"
	"path/filepath"
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
	UntouchableHosts      []string // hosts that need the user's confirmation for ssh, scp, rsync
	Cwd                   string   // the folder the command runs in, for relative paths
}

var allow = Decision{Allow, ""}

const (
	rootReason = "this would run as root. Never become root: give the user the exact command to run themselves, and wait for its output. Running as another user (sudo -u <user>) is fine."
	maxDepth   = 4
)

// Command checks a shell command.
func (p Policy) Command(line string) Decision {
	if what := FindSecret(line); what != "" {
		return Decision{Deny, "the command has " + what + " written in it. Read secrets from the system keychain, an environment variable or a secrets manager, never from the command line, and never ask the user to paste one in the chat."}
	}
	return p.commands(parse(line), map[string]string{}, 0)
}

// commands checks a list of commands in order. Exported variables carry over to the
// commands after them. A deny wins over an ask.
func (p Policy) commands(cmds []command, exported map[string]string, depth int) Decision {
	var ask *Decision
	for _, c := range cmds {
		if len(c.words) > 0 && base(c.words[0]) == "export" {
			for _, a := range c.words[1:] {
				if eq := strings.IndexByte(a, '='); eq > 0 && isName(a[:eq]) {
					exported[a[:eq]] = a[eq+1:]
				}
			}
		}
		d := p.simple(c.words, c.stdin, exported, depth)
		if d.Verdict == Deny {
			return d
		}
		if d.Verdict == Ask && ask == nil {
			ask = &d
		}
	}
	if ask != nil {
		return *ask
	}
	return allow
}

// inner checks a command line run by another command (ssh, sh -c, a heredoc for bash).
func (p Policy) inner(line string, exported map[string]string, depth int) Decision {
	if depth >= maxDepth || strings.TrimSpace(line) == "" {
		return allow
	}
	if what := FindSecret(line); what != "" {
		return Decision{Deny, "the command has " + what + " written in it."}
	}
	env := map[string]string{}
	for k, v := range exported {
		env[k] = v
	}
	return p.commands(parse(line), env, depth+1)
}

func (p Policy) simple(words []string, stdin string, exported map[string]string, depth int) Decision {
	env, name, args := program(words)
	for k, v := range exported {
		if _, set := env[k]; !set {
			env[k] = v
		}
	}
	for _, k := range []string{"AWS_SECRET_ACCESS_KEY", "AWS_ACCESS_KEY_ID", "AWS_SESSION_TOKEN"} {
		if v, ok := env[k]; ok && v != "" && !strings.HasPrefix(v, "$") {
			return Decision{Deny, "the command sets " + k + " with a value written in it. Use an AWS profile (--profile or AWS_PROFILE) instead of keys on the command line."}
		}
	}
	switch name {
	case "git":
		return p.git(args)
	case "sudo", "doas":
		return p.sudo(name, args, stdin, env, depth)
	case "su", "run0", "pkexec":
		return Decision{Deny, "`" + name + "` switches to another user, root by default. " + rootReason}
	case "aws":
		return p.aws(env, args)
	case "ssh":
		return p.ssh(args, stdin, env, depth)
	case "scp", "rsync", "sftp":
		return p.copyHosts(name, args)
	case "sh", "bash", "zsh", "dash", "ksh":
		return p.shell(args, stdin, env, depth)
	case "eval":
		return p.inner(strings.Join(args, " "), env, depth)
	case "find":
		return p.find(args, env, depth)
	}
	if prof, ok := env["AWS_PROFILE"]; ok && contains(p.ProductionAWSProfiles, prof) {
		return Decision{Ask, "this runs with the production AWS profile " + prof + "; the user confirms first."}
	}
	return allow
}

// git blocks commit and push unless the repository is explicitly allowed, written with -C.
func (p Policy) git(args []string) Decision {
	dir := ""
	withValue := map[string]bool{"-C": true, "-c": true, "--git-dir": true, "--work-tree": true,
		"--namespace": true, "--super-prefix": true, "--config-env": true}
	i := 0
	for ; i < len(args) && strings.HasPrefix(args[i], "-"); i++ {
		if withValue[args[i]] && i+1 < len(args) {
			if args[i] == "-C" {
				dir = args[i+1]
			}
			i++
		}
	}
	if i >= len(args) || (args[i] != "commit" && args[i] != "push") {
		return allow
	}
	if dir != "" {
		if d, err := p.resolve(dir); err == nil && contains(p.GitAllowedRepos, d) {
			return allow
		}
	}
	return Decision{Deny, "`git " + args[i] + "` is for the user. Leave the change ready and checked, show what would be committed, and tell the user it is ready. (To let the assistant commit in a repository, add it to git_allowed_repos in the configuration, and use `git -C <repo>`.)"}
}

// sudo allows running as another user, but never as root, and checks the command it
// runs too (sudo -u app sudo id would end up as root).
func (p Policy) sudo(name string, args []string, stdin string, env map[string]string, depth int) Decision {
	withValue := "ugpCDhrtTU" // options that take a value
	user := ""
	listOnly := false
	i := 0
	for ; i < len(args) && strings.HasPrefix(args[i], "-"); i++ {
		a := args[i]
		if a == "--" {
			i++
			break
		}
		if strings.HasPrefix(a, "--") {
			key, val, hasVal := strings.Cut(a, "=")
			switch key {
			case "--user":
				if hasVal {
					user = val
				} else if i+1 < len(args) {
					user = args[i+1]
					i++
				}
			case "--list", "--validate", "--reset-timestamp", "--remove-timestamp", "--version", "--help":
				listOnly = true
			case "--prompt", "--group", "--close-from", "--chdir", "--host", "--role", "--type", "--command-timeout", "--other-user":
				if !hasVal {
					i++
				}
			}
			continue
		}
		// a cluster of short options, the last one may take a value: -iu app, -uapp
		for j := 1; j < len(a); j++ {
			o := a[j]
			if strings.ContainsRune("lvkKV", rune(o)) {
				listOnly = true
			}
			if strings.ContainsRune(withValue, rune(o)) {
				val := a[j+1:]
				if val == "" && i+1 < len(args) {
					val = args[i+1]
					i++
				}
				if o == 'u' {
					user = val
				}
				break
			}
		}
	}
	rest := args[i:] // doas, like sudo, is root without -u
	if listOnly && len(rest) == 0 {
		return allow // sudo -l, sudo -v: nothing runs
	}
	if user == "" || user == "root" || user == "0" || user == "#0" {
		return Decision{Deny, rootReason}
	}
	if len(rest) == 0 {
		return allow
	}
	return p.simple(rest, stdin, env, depth)
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

// ssh asks before reaching an untouchable host, and checks the remote command, or what
// it sends to the remote shell through stdin.
func (p Policy) ssh(args []string, stdin string, env map[string]string, depth int) Decision {
	host, rest := sshTarget(args)
	var d Decision
	switch {
	case len(rest) > 0 && isShell(base(rest[0])):
		d = p.simple(rest, stdin, env, depth)
	case len(rest) > 0:
		d = p.inner(strings.Join(rest, " "), env, depth)
	default:
		d = p.inner(stdin, env, depth) // the remote shell reads stdin
	}
	if d.Verdict == Deny {
		return d
	}
	if host != "" && p.untouchable(host) {
		return Decision{Ask, "host " + host + " is marked as untouchable in this space; the user confirms first."}
	}
	return d
}

// shell checks the commands a shell would run: the string after -c (also -lc, -ec…),
// or what it reads from stdin when it runs no script.
func (p Policy) shell(args []string, stdin string, env map[string]string, depth int) Decision {
	wantsC := false
	for _, a := range args {
		if wantsC && !strings.HasPrefix(a, "-") {
			return p.inner(a, env, depth)
		}
		if strings.HasPrefix(a, "-") && !strings.HasPrefix(a, "--") && strings.ContainsRune(a[1:], 'c') {
			wantsC = true
			continue
		}
		if !strings.HasPrefix(a, "-") {
			return allow // runs a script file: its content is not known here
		}
	}
	return p.inner(stdin, env, depth)
}

// find checks the command of -exec, -execdir, -ok and -okdir.
func (p Policy) find(args []string, env map[string]string, depth int) Decision {
	for i, a := range args {
		if a == "-exec" || a == "-execdir" || a == "-ok" || a == "-okdir" {
			var cmd []string
			for _, w := range args[i+1:] {
				if w == ";" || w == "+" {
					break
				}
				cmd = append(cmd, w)
			}
			if d := p.simple(cmd, "", env, depth); d.Verdict != Allow {
				return d
			}
		}
	}
	return allow
}

// copyHosts asks before copying to or from an untouchable host (host:path arguments).
func (p Policy) copyHosts(name string, args []string) Decision {
	for _, a := range args {
		if strings.HasPrefix(a, "-") || strings.HasPrefix(a, "/") || strings.HasPrefix(a, ".") {
			continue
		}
		hostPart, _, isRemote := strings.Cut(strings.TrimPrefix(a, "rsync://"), ":")
		if !isRemote {
			continue
		}
		if at := strings.LastIndexByte(hostPart, '@'); at >= 0 {
			hostPart = hostPart[at+1:]
		}
		if p.untouchable(hostPart) {
			return Decision{Ask, name + " reaches " + hostPart + ", marked as untouchable in this space; the user confirms first."}
		}
	}
	return allow
}

// untouchable matches a host by its name or by its short name (db01 matches
// db01.example.com).
func (p Policy) untouchable(host string) bool {
	for _, h := range p.UntouchableHosts {
		if host == h || strings.HasPrefix(host, h+".") {
			return true
		}
	}
	return false
}

// sshTarget returns the host of an ssh command and the remote command after it.
func sshTarget(args []string) (string, []string) {
	withValue := "BbcDEeFIiJLlmOoPpQRSWw"
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			continue
		}
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

func isShell(name string) bool {
	switch name {
	case "sh", "bash", "zsh", "dash", "ksh":
		return true
	}
	return false
}

// Write checks text the assistant is about to write into a file.
func Write(text string) Decision {
	if what := FindSecret(text); what != "" {
		return Decision{Deny, "the text has " + what + " in it. Secrets never go into files: the code reads them from an environment variable, the system keychain or a secrets manager."}
	}
	return allow
}

// resolve makes a path absolute against the command's folder and resolves symlinks.
func (p Policy) resolve(path string) (string, error) {
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = filepath.Join(home, path[1:])
	}
	if !filepath.IsAbs(path) && p.Cwd != "" {
		path = filepath.Join(p.Cwd, path)
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if r, err := filepath.EvalSymlinks(path); err == nil {
		path = r
	}
	return path, nil
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
