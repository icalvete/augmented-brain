// Package guard decides, before the assistant runs a command or writes a file, whether
// to let it through, block it, or ask the user first.
//
// It is a safety net against mistakes, not a security boundary: it reads the command as
// text, so a determined program could get around it.
package guard

import (
	"cmp"
	"errors"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/icalvete/augmented-brain/internal/paths"
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

// Policy is what the guard protects. The production profiles and untouchable hosts are
// machine-wide; the repositories the assistant may commit in come from the space of the
// folder it works in. With no configuration, only the general rules apply.
type Policy struct {
	// GitAllowedRepos are the repositories where the assistant may commit and push, and
	// GitAskRepos those where it may once the user confirms. Absolute paths; the last
	// part may hold a * (one folder: ~/code/app-* is every app-<name> next to app), and
	// a git worktree of a listed repository counts as that repository.
	GitAllowedRepos       []string
	GitAskRepos           []string
	ProductionAWSProfiles []string // AWS profiles that need the user's confirmation
	UntouchableHosts      []string // hosts that need the user's confirmation for ssh, scp, rsync
	Cwd                   string   // the folder the command runs in, for relative paths
	// Env is the environment the command inherits (AWS_PROFILE, AWS_DEFAULT_PROFILE), so
	// an aws command with no profile of its own is checked against the one it will use.
	// The hook passes it in; the guard never reads the environment itself.
	Env map[string]string
	// ConfigProblem, when not empty, says why the configuration could not be read: the
	// lists above are unknown, so a command they would cover asks the user every time.
	ConfigProblem string
	// lost says a cd earlier in the command went somewhere the guard cannot tell
	// (cd -, cd "$DIR"): relative paths after it are not resolved.
	lost bool
}

var allow = Decision{Allow, ""}

// PowerShell is the answer to any command for Claude Code's PowerShell tool, the main
// one on Windows. The guard reads POSIX shell, not PowerShell, so it cannot tell a safe
// PowerShell command from a push or an elevation: it blocks them all and sends the
// assistant to Bash (Git Bash on Windows), where every rule applies.
func PowerShell() Decision {
	return Decision{Deny, "PowerShell commands are not checked by augmented brain yet, so they are blocked. Run the command with the Bash tool instead (Git Bash on Windows), in POSIX syntax. If it can only be done in PowerShell, give the user the exact command to run themselves, and wait for its output."}
}

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
	d := allow
	for i, c := range cmds {
		if c.piped && c.stdin == "" && i > 0 {
			// echo 'sudo id' | bash: bash reads what echo writes. Kept in the list, so a
			// command further down the pipe (… | tee x | bash) gets it too.
			cmds[i].stdin = output(cmds[i-1])
			c.stdin = cmds[i].stdin
		}
		if len(c.words) > 0 && (base(c.words[0]) == "cd" || base(c.words[0]) == "pushd") {
			p = p.cd(c.words[1:]) // the commands after it run there
		}
		if len(c.words) > 0 && base(c.words[0]) == "export" {
			for _, a := range c.words[1:] {
				if eq := strings.IndexByte(a, '='); eq > 0 && isName(a[:eq]) {
					exported[a[:eq]] = a[eq+1:]
				}
			}
		}
		if d = worse(d, p.simple(c.words, c.stdin, exported, depth)); d.Verdict == Deny {
			return d
		}
	}
	return d
}

// worse combines two decisions: a deny wins over an ask, and an ask over an allow. Of
// two asks, the first one's reason is kept.
func worse(a, b Decision) Decision {
	if b.Verdict > a.Verdict {
		return b
	}
	return a
}

// output returns what a command writes when that is known from the command line alone:
// cat or tee of its heredoc or here-string, echo and printf of their arguments. A shell
// or ssh at the end of a pipe runs it, so the guard checks it as their stdin.
func output(c command) string {
	_, name, args := program(c.words)
	switch name {
	case "cat", "tee":
		return c.stdin
	case "echo":
		for len(args) > 0 && (args[0] == "-n" || args[0] == "-e" || args[0] == "-E" || args[0] == "-ne" || args[0] == "-en") {
			args = args[1:]
		}
		return strings.ReplaceAll(strings.Join(args, " "), `\n`, "\n") + "\n"
	case "printf":
		return strings.ReplaceAll(strings.Join(args, " "), `\n`, "\n") + "\n"
	}
	return ""
}

// inner checks a command line run by another command (ssh, sh -c, a heredoc for bash).
func (p Policy) inner(line string, exported map[string]string, depth int) Decision {
	if depth >= maxDepth || strings.TrimSpace(line) == "" {
		return allow
	}
	if what := FindSecret(line); what != "" {
		return Decision{Deny, "the command has " + what + " written in it."}
	}
	return p.commands(parse(line), maps.Clone(exported), depth+1)
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
		return p.git(env, args)
	case "sudo", "doas":
		return p.sudo(name, args, stdin, env, depth)
	case "su", "run0", "pkexec":
		return Decision{Deny, "`" + name + "` switches to another user, root by default. " + rootReason}
	case "aws":
		return p.aws(env, args)
	case "cdk", "sam", "serverless", "sls", "terraform", "tofu", "pulumi", "eksctl", "copilot":
		return p.awsTool(name, env, args)
	case "aws-vault":
		return p.awsVault(args, stdin, env, depth)
	case "gh", "glab":
		return publish(name, args)
	case "ssh":
		return p.ssh(args, stdin, env, depth)
	case "mosh":
		return p.mosh(args, env, depth)
	case "sftp":
		return p.sftp(args)
	case "scp", "rsync":
		return p.copyHosts(name, args)
	case "sh", "bash", "zsh", "dash", "ksh":
		return p.shell(args, stdin, env, depth)
	case "eval":
		return p.inner(strings.Join(args, " "), env, depth)
	case "find":
		return p.find(args, env, depth)
	case "rm":
		return p.rm(args)
	case "vagrant":
		return p.vagrant(args, env, depth)
	case "docker", "podman", "kubectl":
		return p.exec(name, args, env, depth)
	}
	// Any other program set to a production profile in the command itself.
	for _, k := range []string{"AWS_PROFILE", "AWS_DEFAULT_PROFILE"} {
		if prof, ok := env[k]; ok && slices.Contains(p.ProductionAWSProfiles, prof) {
			return Decision{Ask, "this runs with the production AWS profile " + prof + "; the user confirms first."}
		}
	}
	return allow
}

// checkHost asks before name reaches target (host or user@host) when the host is
// untouchable, or when the configuration could not be read and nobody can tell.
func (p Policy) checkHost(name, target string) Decision {
	host := target[strings.LastIndexByte(target, '@')+1:]
	if host == "" {
		return allow
	}
	if d, unknown := p.listsUnknown(name + " to " + host); unknown {
		return d
	}
	if p.untouchable(host) {
		return Decision{Ask, name + " reaches " + host + ", marked as untouchable; the user confirms first."}
	}
	return allow
}

// firstOperand returns the index of the first argument that is neither an option nor
// the value of one, or len(args) when there is none. takesValue says which options
// take the next argument as their value.
func firstOperand(args []string, takesValue func(string) bool) int {
	for i := 0; i < len(args); i++ {
		if !strings.HasPrefix(args[i], "-") {
			return i
		}
		if takesValue(args[i]) {
			i++
		}
	}
	return len(args)
}

// oneOf is a takesValue for firstOperand: options written exactly as one of opts.
func oneOf(opts ...string) func(string) bool {
	return func(a string) bool { return slices.Contains(opts, a) }
}

// shortOf is a takesValue for firstOperand: a single-letter option (-p) among letters.
func shortOf(letters string) func(string) bool {
	return func(a string) bool { return len(a) == 2 && strings.ContainsRune(letters, rune(a[1])) }
}

// git blocks commit and push unless the repository is explicitly allowed, written with -C.
// Each -C is taken from the one before it, as git does. A repository given another way
// (--git-dir, --work-tree, GIT_DIR, GIT_WORK_TREE) is never taken as listed.
func (p Policy) git(env map[string]string, args []string) Decision {
	dir := ""
	elsewhere := env["GIT_DIR"] != "" || env["GIT_WORK_TREE"] != ""
	withValue := map[string]bool{"-C": true, "-c": true, "--git-dir": true, "--work-tree": true,
		"--namespace": true, "--super-prefix": true, "--config-env": true}
	i := 0
	for ; i < len(args) && strings.HasPrefix(args[i], "-"); i++ {
		key, _, _ := strings.Cut(args[i], "=")
		if key == "--git-dir" || key == "--work-tree" {
			elsewhere = true
		}
		if withValue[args[i]] && i+1 < len(args) {
			if args[i] == "-C" {
				dir = gitDir(dir, args[i+1])
			}
			i++
		}
	}
	if i >= len(args) || (args[i] != "commit" && args[i] != "push") {
		return allow
	}
	if dir != "" && !elsewhere {
		if d, err := p.resolve(dir); err == nil {
			// Asking wins over allowing: a repository may be in both lists, from two spaces.
			if repoListed(p.GitAskRepos, d) {
				return Decision{Ask, "`git " + args[i] + "` in " + paths.Tilde(d) + " needs the user's go-ahead each time. Before asking, show them what would be committed or pushed."}
			}
			if repoListed(p.GitAllowedRepos, d) {
				if args[i] == "push" && rewrites(args[i+1:]) {
					return Decision{Ask, "this push rewrites or deletes what is on the remote (force, delete or mirror). The user confirms it each time, even in a repository the assistant may push to."}
				}
				return allow
			}
		}
	}
	return Decision{Deny, "`git " + args[i] + "` is for the user. Leave the change ready and checked, show what would be committed, and tell the user it is ready. (To let the assistant commit in a repository, add it to git_allowed_repos in the configuration, or to git_ask_repos to be asked each time, and use `git -C <repo>`.)"}
}

// rewrites reports whether the arguments of a push change history on the remote rather
// than add to it: a force, a delete or a mirror, or a refspec that starts with + or :.
func rewrites(args []string) bool {
	for _, a := range args {
		switch {
		case a == "--":
			return false
		case strings.HasPrefix(a, "--force"), a == "--delete", a == "--mirror", a == "--prune":
			return true
		case strings.HasPrefix(a, "--"):
		case strings.HasPrefix(a, "-"):
			if strings.ContainsAny(a[1:], "fd") {
				return true
			}
		case strings.HasPrefix(a, "+"), strings.HasPrefix(a, ":"):
			return true
		}
	}
	return false
}

// rm never deletes, recursively, the whole disk or the user's home: nothing the
// assistant does needs it, and one wrong variable makes it happen.
func (p Policy) rm(args []string) Decision {
	recursive := false
	var operands []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--":
			operands = append(operands, args[i+1:]...)
			i = len(args)
		case a == "--recursive":
			recursive = true
		case strings.HasPrefix(a, "--"):
		case strings.HasPrefix(a, "-") && len(a) > 1:
			recursive = recursive || strings.ContainsAny(a, "rR")
		default:
			operands = append(operands, a)
		}
	}
	if !recursive {
		return allow
	}
	home, _ := paths.Home()
	if h, err := paths.Resolve(home); err == nil {
		home = h
	}
	for _, o := range operands {
		for _, v := range []string{"$HOME", "${HOME}", "$USERPROFILE", "${USERPROFILE}"} {
			if o == v || strings.HasPrefix(o, v+"/") {
				o = "~" + strings.TrimPrefix(o, v)
			}
		}
		if o = strings.TrimSuffix(o, "*"); o == "" {
			o = "." // rm -r * empties the folder it runs in
		}
		if o = strings.TrimRight(o, "/"); o == "" {
			return Decision{Deny, "`rm -r` of the whole disk is never run."}
		}
		if p.lost && !filepath.IsAbs(o) && !strings.HasPrefix(o, "~") && strings.Trim(o, "./") == "" {
			return Decision{Ask, "`rm -r " + o + "` after a cd the guard cannot follow: it cannot tell which folder this deletes. The user confirms it."}
		}
		if d, err := p.resolve(o); err == nil && filepath.Dir(d) == d {
			return Decision{Deny, "`rm -r` of a whole disk (" + d + ") is never run."}
		} else if err == nil && home != "" && d == home {
			return Decision{Deny, "`rm -r` of the user's whole home folder (" + paths.Tilde(d) + ") is never run. Delete the files or folders inside it that need to go, one by one."}
		}
	}
	return allow
}

// cd returns the policy for the commands after a cd (or pushd) with args: no folder is
// the home folder, and one the guard cannot tell (-, a variable) leaves it lost.
func (p Policy) cd(args []string) Policy {
	dir := "~"
	for _, a := range args {
		if a == "-" || !strings.HasPrefix(a, "-") {
			dir = a
			break
		}
	}
	if dir == "-" || strings.ContainsAny(dir, "$`") {
		p.Cwd, p.lost = "", true
		return p
	}
	if d, err := p.resolve(dir); err == nil {
		p.Cwd, p.lost = d, false
	} else {
		p.Cwd, p.lost = "", true // not there: the cd fails, and where the rest runs depends on how it is joined
	}
	return p
}

// gitDir is where git -C next goes after the -C options before it, which are in dir.
func gitDir(dir, next string) string {
	if dir == "" || filepath.IsAbs(next) || strings.HasPrefix(next, "~") {
		return next
	}
	return filepath.Join(dir, next)
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

// aws asks before a command that may use a production profile. Which one it uses
// depends on the AWS CLI's version and on what is set, so every candidate counts: the
// one given with --profile, AWS_PROFILE and AWS_DEFAULT_PROFILE (set in the command or
// inherited), and "default" when nothing is set.
func (p Policy) aws(env map[string]string, args []string) Decision {
	if d, unknown := p.listsUnknown("this AWS command"); unknown {
		return d
	}
	if prof := p.productionProfile(env, profileFlag(args, "--profile")); prof != "" {
		return Decision{Ask, "this AWS command may use the production profile " + prof + "; the user confirms first."}
	}
	return allow
}

// awsTool does the same for the tools that call AWS on their own: cdk, sam, serverless
// (whose --aws-profile counts too), terraform, tofu, pulumi, eksctl and copilot.
func (p Policy) awsTool(name string, env map[string]string, args []string) Decision {
	if d, unknown := p.listsUnknown(name); unknown {
		return d
	}
	given := cmp.Or(profileFlag(args, "--profile"), profileFlag(args, "--aws-profile"))
	if prof := p.productionProfile(env, given); prof != "" {
		return Decision{Ask, name + " may run with the production AWS profile " + prof + "; the user confirms first."}
	}
	return allow
}

// productionProfile returns the first candidate profile that is production, or "".
func (p Policy) productionProfile(env map[string]string, given string) string {
	candidates := []string{given, env["AWS_PROFILE"], env["AWS_DEFAULT_PROFILE"], p.Env["AWS_PROFILE"], p.Env["AWS_DEFAULT_PROFILE"]}
	if given == "" && cmp.Or(candidates...) == "" {
		candidates = append(candidates, "default")
	}
	for _, c := range candidates {
		if c != "" && slices.Contains(p.ProductionAWSProfiles, c) {
			return c
		}
	}
	return ""
}

// awsVault asks before aws-vault exec with a production profile, and checks the
// command it runs.
func (p Policy) awsVault(args []string, stdin string, env map[string]string, depth int) Decision {
	if len(args) == 0 || args[0] != "exec" {
		return allow
	}
	i := 1 + firstOperand(args[1:], oneOf("-d", "--duration", "--region", "-t", "--mfa-token", "--prompt", "--backend"))
	if i >= len(args) {
		return allow
	}
	prof, rest := args[i], args[i+1:]
	if len(rest) > 0 && rest[0] == "--" {
		rest = rest[1:]
	}
	inner := allow
	if len(rest) > 0 {
		sub := maps.Clone(env)
		sub["AWS_PROFILE"] = prof
		inner = p.simple(rest, stdin, sub, depth)
	}
	d := allow
	if u, unknown := p.listsUnknown("aws-vault"); unknown {
		d = u
	} else if slices.Contains(p.ProductionAWSProfiles, prof) {
		d = Decision{Ask, "aws-vault runs this with the production AWS profile " + prof + "; the user confirms first."}
	}
	return worse(d, inner)
}

// profileFlag returns the value of a --flag value or --flag=value argument.
func profileFlag(args []string, flag string) string {
	prof := ""
	for i, a := range args {
		if a == flag && i+1 < len(args) {
			prof = args[i+1]
		} else if strings.HasPrefix(a, flag+"=") {
			prof = strings.TrimPrefix(a, flag+"=")
		}
	}
	return prof
}

// listsUnknown asks when the configuration could not be read: then the production
// profiles and untouchable hosts are unknown, and what would check them cannot.
func (p Policy) listsUnknown(what string) (Decision, bool) {
	if p.ConfigProblem == "" {
		return allow, false
	}
	return Decision{Ask, what + " may reach production or an untouchable host, and augmented brain cannot tell: it cannot read its configuration (" +
		p.ConfigProblem + "). The user confirms first; `augmented-brain doctor` shows how to fix the configuration."}, true
}

// syncDestination reports whether gh repo sync names a destination repository (any
// argument besides "repo", "sync" and the values of --branch and --source).
func syncDestination(args []string) bool {
	n := 0
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-b" || a == "--branch" || a == "-s" || a == "--source":
			i++
		case strings.HasPrefix(a, "-"):
		default:
			if n++; n > 2 {
				return true
			}
		}
	}
	return false
}

// publish blocks the gh and glab commands that publish or merge, as it blocks a push.
func publish(name string, args []string) Decision {
	var words []string
	for _, a := range args {
		if !strings.HasPrefix(a, "-") {
			words = append(words, a)
		}
	}
	if len(words) < 2 {
		return allow
	}
	blocked := map[string]map[string]bool{
		"gh":   {"pr merge": true, "release create": true, "release upload": true, "release edit": true},
		"glab": {"mr merge": true, "mr accept": true, "release create": true, "release upload": true},
	}[name]
	what := words[0] + " " + words[1]
	// gh repo sync with no repository only updates the local clone; with one, it updates
	// that repository on GitHub.
	if name == "gh" && what == "repo sync" && syncDestination(args) {
		blocked = map[string]bool{what: true}
	}
	if !blocked[what] {
		return allow
	}
	return Decision{Deny, "`" + name + " " + what + "` publishes or merges, like a git push, so it is for the user. Leave the change ready and checked, and give the user the exact command to run themselves."}
}

// ssh asks before reaching an untouchable host, and checks the remote command, or what
// it sends to the remote shell through stdin.
func (p Policy) ssh(args []string, stdin string, env map[string]string, depth int) Decision {
	host, rest := sshTarget(dropRedirections(args))
	var d Decision
	switch {
	case len(rest) > 0 && isShell(base(rest[0])):
		d = p.simple(rest, stdin, env, depth)
	case len(rest) > 0:
		d = p.inner(strings.Join(rest, " "), env, depth)
	default:
		d = p.inner(stdin, env, depth) // the remote shell reads stdin
	}
	return worse(p.checkHost("ssh", host), d)
}

// vagrant ssh -c runs its command in the virtual machine: it is checked as ssh's is.
func (p Policy) vagrant(args []string, env map[string]string, depth int) Decision {
	if len(args) == 0 || args[0] != "ssh" {
		return allow
	}
	d := allow
	for i := 1; i < len(args); i++ {
		switch a := args[i]; {
		case (a == "-c" || a == "--command") && i+1 < len(args):
			d = worse(d, p.inner(args[i+1], env, depth))
			i++
		case strings.HasPrefix(a, "--command="):
			d = worse(d, p.inner(strings.TrimPrefix(a, "--command="), env, depth))
		}
	}
	return d
}

// exec checks the command that docker exec, podman exec and kubectl exec run in a
// container, as if it ran here: becoming root there is still for the user.
func (p Policy) exec(name string, args []string, env map[string]string, depth int) Decision {
	if len(args) == 0 || args[0] != "exec" || depth >= maxDepth {
		return allow
	}
	args = args[1:]
	if name == "kubectl" {
		// kubectl exec <pod> [options] -- <command>
		if i := slices.Index(args, "--"); i >= 0 && i+1 < len(args) {
			return p.simple(args[i+1:], "", env, depth+1)
		}
		return allow
	}
	// docker exec [options] <container> <command>
	i := firstOperand(args, oneOf("-e", "--env", "--env-file", "-u", "--user", "-w", "--workdir", "--detach-keys"))
	if i+1 >= len(args) {
		return allow
	}
	return p.simple(args[i+1:], "", env, depth+1)
}

// mosh is ssh for unsteady connections: the same checks on its host and its command.
func (p Policy) mosh(args []string, env map[string]string, depth int) Decision {
	i := firstOperand(args, oneOf("-p", "--port", "--ssh", "--server", "--client", "--predict", "--family"))
	if i >= len(args) {
		return allow
	}
	host, rest := args[i], args[i+1:]
	if len(rest) > 0 && rest[0] == "--" {
		rest = rest[1:]
	}
	d := allow
	if len(rest) > 0 {
		d = p.simple(rest, "", env, depth)
	}
	return worse(p.checkHost("mosh", host), d)
}

// sftp reaches the host given as its first argument that is not an option, with or
// without a :path after it.
func (p Policy) sftp(args []string) Decision {
	i := firstOperand(args, shortOf("BbcDFiJloPRSs"))
	if i >= len(args) {
		return allow
	}
	host, _, _ := strings.Cut(args[i], ":")
	return p.checkHost("sftp", host)
}

// dropRedirections takes the redirections out of a command's words (< file, >file,
// 2>&1…): they are not arguments, and the guard does not know what a file holds.
func dropRedirections(args []string) []string {
	var out []string
	for i := 0; i < len(args); i++ {
		a := strings.TrimLeft(args[i], "0123456789")
		if strings.HasPrefix(a, "<") || strings.HasPrefix(a, ">") || strings.HasPrefix(a, "&>") {
			if strings.TrimLeft(a, "<>&") == "" && i+1 < len(args) {
				i++ // the operator alone: the file is the next word
			}
			continue
		}
		out = append(out, args[i])
	}
	return out
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
	d := allow
	for i, a := range args {
		if a == "-exec" || a == "-execdir" || a == "-ok" || a == "-okdir" {
			var cmd []string
			for _, w := range args[i+1:] {
				if w == ";" || w == "+" {
					break
				}
				cmd = append(cmd, w)
			}
			if d = worse(d, p.simple(cmd, "", env, depth)); d.Verdict == Deny {
				return d
			}
		}
	}
	return d
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
		if d := p.checkHost(name, hostPart); d.Verdict != Allow {
			return d
		}
	}
	return allow
}

// untouchable matches a host by its name, ignoring case, or by its short name either
// way: a saved db01 matches db01.example.com, and a saved db01.example.com matches db01.
func (p Policy) untouchable(host string) bool {
	host = strings.TrimSuffix(strings.ToLower(host), ".")
	if host == "" {
		return false
	}
	for _, h := range p.UntouchableHosts {
		h = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(h)), ".")
		if h == "" {
			continue
		}
		if host == h || strings.HasPrefix(host, h+".") || strings.HasPrefix(h, host+".") {
			return true
		}
	}
	return false
}

// sshTarget returns the destination of an ssh command ([user@]host) and the remote
// command after it. Unlike firstOperand, it takes a lone - as the destination, as ssh
// does, so the command after it is still checked.
func sshTarget(args []string) (string, []string) {
	takesValue := shortOf("BbcDEeFIiJLlmOoPpQRSWw")
	for i := 0; i < len(args); i++ {
		a := args[i]
		if len(a) < 2 || a[0] != '-' {
			return a, args[i+1:]
		}
		if takesValue(a) {
			i++
		}
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

// bashPath turns a path as Git Bash writes it on Windows (/c/Users/ana) into the one
// Windows knows (C:/Users/ana). On other systems, and for any other path, it is unchanged.
func bashPath(path, goos string) string {
	if goos != "windows" || len(path) < 2 || path[0] != '/' || !(path[1] >= 'a' && path[1] <= 'z' || path[1] >= 'A' && path[1] <= 'Z') {
		return path
	}
	if len(path) > 2 && path[2] != '/' {
		return path // /cygdrive, /usr: not a drive
	}
	return strings.ToUpper(path[1:2]) + ":/" + strings.TrimPrefix(path[2:], "/")
}

// resolve makes a path absolute against the command's folder and resolves symlinks.
// Claude runs commands in Git Bash on Windows, so /c/… is the C: drive.
func (p Policy) resolve(path string) (string, error) {
	path = bashPath(path, runtime.GOOS)
	if p.lost && !filepath.IsAbs(path) && path != "~" && !strings.HasPrefix(path, "~/") {
		return "", errors.New("after a cd the guard cannot follow")
	}
	if !filepath.IsAbs(path) && path != "~" && !strings.HasPrefix(path, "~/") && p.Cwd != "" {
		path = filepath.Join(p.Cwd, path)
	}
	return paths.Resolve(path)
}

// repoListed reports whether the repository at dir (resolved) is in list: the same
// path, a match of an entry with a * in its last part, or a worktree of one of them.
func repoListed(list []string, dir string) bool {
	main := mainRepo(dir)
	for _, x := range list {
		for _, d := range []string{dir, main} {
			if d == "" {
				continue
			}
			if x == d {
				return true
			}
			if strings.Contains(filepath.Base(x), "*") && filepath.Dir(x) == filepath.Dir(d) {
				if ok, err := filepath.Match(filepath.Base(x), filepath.Base(d)); err == nil && ok {
					return true
				}
			}
		}
	}
	return false
}

// mainRepo is the repository a git worktree belongs to, read from its .git file
// ("gitdir: <repo>/.git/worktrees/<name>", whose commondir leads to <repo>/.git); empty
// when dir is not a linked worktree. A submodule has a .git file too, but its folder
// is not a worktree's, so it never counts as its parent repository.
func mainRepo(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, ".git"))
	if err != nil {
		return "" // a folder (a repository of its own) or nothing
	}
	gitdir, ok := strings.CutPrefix(strings.TrimSpace(string(data)), "gitdir:")
	if !ok {
		return ""
	}
	gitdir = strings.TrimSpace(gitdir)
	if !filepath.IsAbs(gitdir) {
		gitdir = filepath.Join(dir, gitdir)
	}
	common, err := os.ReadFile(filepath.Join(gitdir, "commondir"))
	if err != nil {
		return ""
	}
	c := strings.TrimSpace(string(common))
	if !filepath.IsAbs(c) {
		c = filepath.Join(gitdir, c)
	}
	c = filepath.Clean(c)
	if filepath.Base(c) != ".git" {
		return ""
	}
	r, err := paths.Resolve(filepath.Dir(c))
	if err != nil {
		return ""
	}
	return r
}
