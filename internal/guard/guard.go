// Package guard decides, before the assistant runs a command or writes a file, whether
// to let it through, block it, or ask the user first.
//
// It is a safety net against mistakes, not a security boundary: it reads the command as
// text, so a determined program could get around it.
package guard

import (
	"bytes"
	"cmp"
	"errors"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/icalvete/augmented-brain/internal/i18n"
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
	GitAllowedRepos []string
	GitAskRepos     []string
	// UntouchableRepos are repositories where the assistant never commits, pushes or
	// merges, whatever the two lists above say. Same form as them.
	UntouchableRepos      []string
	ProductionAWSProfiles []string // AWS profiles that need the user's confirmation
	UntouchableHosts      []string // hosts that need the user's confirmation for ssh, scp, rsync
	Cwd                   string   // the folder the command runs in, for relative paths
	// Env is the environment the command inherits (AWS_PROFILE, AWS_DEFAULT_PROFILE), so
	// an aws command with no profile of its own is checked against the one it will use,
	// and gh or glab against the repository it will merge in (GH_REPO, GIT_DIR). The hook passes it in; the guard never reads the environment itself.
	Env map[string]string
	// Protected are files the assistant must not change on its own: the guard's
	// configuration and Claude Code's settings, which could switch the guard off.
	// Absolute paths; writing one asks the user.
	Protected []string
	// Hidden are files the assistant must not read at all (the panel's token file), and
	// ConfigDirs the ways the configuration folder is written in a command (its path,
	// with ~, and the variable that moves it), to tell that a bare file name is theirs.
	Hidden     []string
	ConfigDirs []string
	// ConfigChanged says the guard's lists are not the ones the user last saw (at the
	// start of the session, or saved in the panel): something else changed them. Where
	// the lists would let a commit, a push or a merge through, it asks instead.
	ConfigChanged bool
	// SealProblem, when not empty, says why the record of the lists the user last saw
	// could not be read: ConfigChanged is set too, since the guard cannot tell.
	SealProblem string
	// ConfigProblem, when not empty, says why the configuration could not be read: the
	// lists above are unknown, so a command they would cover asks the user every time.
	ConfigProblem string
	// GuardProblem, when not empty, says why guard.toml could not be read: the
	// untouchable repositories are unknown, so a commit, push or merge the other lists
	// would let through asks the user instead, until the file is fixed. The hook sets
	// ConfigProblem too then, on purpose: a broken guard.toml may mean the configuration
	// is not as the user left it, so every command the lists cover asks as well.
	GuardProblem string
	// RootAsk are the machines where becoming root asks the user instead of being denied,
	// each until a time; Hostname is this machine's name, for an exception for it. Now
	// is the time to compare with (zero: the clock).
	RootAsk  []RootException
	Hostname string
	Now      time.Time
	// Lang is the user's language, for the reasons the user reads when asked (en, es).
	Lang string
	// host is the machine the command runs on, as ssh names it; "" is this one.
	host string
	// inScript says the commands come from a script file, where "$@" or "$1" as the
	// program is how scripts are written: those are not asked about there.
	inScript bool
	// lost says a cd earlier in the command went somewhere the guard cannot tell
	// (cd -, cd "$DIR"): relative paths after it are not resolved.
	lost bool
	// cdNamed says Cwd is a folder a cd earlier in the command named, not the session's:
	// what gh and glab, which have no -C, take as the repository to merge in.
	cdNamed bool
	// files are the files the command line writes before it may run one as a script.
	files *lineFiles
}

var allow = Decision{Allow, ""}

// RootException lets a command run as root on Host ask the user, until Until.
type RootException struct {
	Host  string
	Until time.Time
}

// root is the answer to a command that would run as root: denied, unless the machine
// it runs on has a root exception in force, and then the user confirms it.
func (p Policy) root(reason string) Decision {
	now := p.Now
	if now.IsZero() {
		now = time.Now()
	}
	where, names := p.host, []string{p.host}
	if where == "" {
		where, names = i18n.T(p.Lang, "guard.root.thismachine"), []string{p.Hostname, "localhost"}
	}
	for _, e := range p.RootAsk {
		if !now.Before(e.Until) {
			continue
		}
		for _, n := range names {
			if n != "" && rootHost(n, e.Host) {
				return Decision{Ask, i18n.T(p.Lang, "guard.root.exception", where, e.Until.UTC().Format("2006-01-02 15:04"))}
			}
		}
	}
	return Decision{Deny, reason}
}

// PowerShell is the answer to any command for Claude Code's PowerShell tool, the main
// one on Windows. The guard reads POSIX shell, not PowerShell, so it cannot tell a safe
// PowerShell command from a push or an elevation: it blocks them all and sends the
// assistant to Bash (Git Bash on Windows), where every rule applies. lang is the
// user's language.
func PowerShell(lang string) Decision {
	return Decision{Deny, i18n.T(lang, "guard.powershell")}
}

// maxDepth is how deep the guard follows a command run by another one (sudo, ssh,
// bash -c, find -exec, a git alias…); past it the user is asked. Every level counts,
// so no input can keep the guard busy for long.
const maxDepth = 4

// maxCommand is the longest command line the guard reads; a longer one asks the user.
const maxCommand = 256 << 10

// rootReason is why becoming root is denied, and what to do instead.
func (p Policy) rootReason() string { return i18n.T(p.Lang, "guard.root") }

// secret is the kind of secret FindSecret found, in the user's language.
func (p Policy) secret(what string) string { return i18n.T(p.Lang, "secret."+what) }

// Command checks a shell command.
func (p Policy) Command(line string) Decision {
	if what := FindSecret(line); what != "" {
		return Decision{Deny, i18n.T(p.Lang, "guard.secret.command", p.secret(what))}
	}
	if p.panelAccess(line) {
		return p.panelDeny()
	}
	if len(line) > maxCommand {
		return Decision{Ask, i18n.T(p.Lang, "guard.toolong", len(line)>>10)}
	}
	p.files = &lineFiles{written: map[string]string{}, touched: map[string]bool{}}
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
		if len(c.words) > 0 && base(c.words[0]) == "popd" {
			p.Cwd, p.lost, p.cdNamed = "", true, false // back to a folder the guard did not keep
		}
		if len(c.words) > 0 && base(c.words[0]) == "export" {
			for _, a := range c.words[1:] {
				if eq := strings.IndexByte(a, '='); eq > 0 && isName(a[:eq]) && guardVar(a[:eq]) {
					exported[a[:eq]] = a[eq+1:]
				}
			}
		}
		if c.tooDeep {
			d = worse(d, Decision{Ask, i18n.T(p.Lang, "guard.depth")})
		}
		for _, r := range c.redirects {
			d = worse(d, p.protected(r))
		}
		if _, name, args := program(c.words); slices.Contains([]string{"sed", "perl", "tee", "cp", "mv", "ln", "rm", "truncate", "install", "dd"}, name) {
			for _, a := range args {
				d = worse(d, p.protected(strings.TrimPrefix(a, "of=")))
			}
		}
		if d = worse(d, p.simple(c.words, c.stdin, exported, depth)); d.Verdict == Deny {
			return d
		}
		p.recordWrites(c)
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
	if strings.TrimSpace(line) == "" {
		return allow
	}
	if depth >= maxDepth {
		return Decision{Ask, i18n.T(p.Lang, "guard.depth")}
	}
	if what := FindSecret(line); what != "" {
		return Decision{Deny, i18n.T(p.Lang, "guard.secret.inner", p.secret(what))}
	}
	return p.commands(parse(line), maps.Clone(exported), depth+1)
}

// nested checks a command that another one runs with its arguments (sudo id, ssh host
// ls, find -exec rm): one level deeper, and past maxDepth the user is asked.
func (p Policy) nested(words []string, stdin string, env map[string]string, depth int) Decision {
	if len(words) == 0 {
		return allow
	}
	if depth >= maxDepth {
		return Decision{Ask, i18n.T(p.Lang, "guard.depth")}
	}
	return p.simple(words, stdin, env, depth+1)
}

// guardVar says whether the guard reads an exported variable (the AWS profile and
// keys, where git and gh look): only those are carried to the commands after export.
func guardVar(name string) bool {
	for _, prefix := range []string{"AWS_", "GIT_", "GH_", "GITLAB_"} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// call is one program as the guard checks it: its name (no path), as written (with its
// path), its arguments, what it reads on stdin when that is known, its environment (set
// in the command or exported earlier) and how deep it runs inside other commands.
type call struct {
	name, raw string
	args      []string
	stdin     string
	env       map[string]string
	depth     int
}

// simple checks one command in three steps: prepare reads what holds for any program
// (keys written in the command, a program only known when it runs); the program's own
// rule, if it has one; and the rules for every program, applied to whatever the rule
// said, so no rule can skip them.
func (p Policy) simple(words []string, stdin string, exported map[string]string, depth int) Decision {
	c, d, done := p.prepare(words, stdin, exported, depth)
	if done {
		return d
	}
	if rule, ok := rules[c.name]; ok {
		d = rule(p, c)
	} else {
		d = p.other(c)
	}
	return worse(d, p.everyProgram(c))
}

// prepare finds the program and its environment, and answers on its own when the
// command can be judged without knowing the program: AWS keys written in it, or a
// program that is only known when it runs.
func (p Policy) prepare(words []string, stdin string, exported map[string]string, depth int) (call, Decision, bool) {
	env, name, args := program(words)
	c := call{name: name, args: args, stdin: stdin, env: env, depth: depth}
	if name != "" {
		c.raw = words[len(words)-len(args)-1] // the program as written, with its path
	}
	for k, v := range exported {
		if _, set := env[k]; !set {
			env[k] = v
		}
	}
	for _, k := range []string{"AWS_SECRET_ACCESS_KEY", "AWS_ACCESS_KEY_ID", "AWS_SESSION_TOKEN"} {
		if v, ok := env[k]; ok && v != "" && !strings.HasPrefix(v, "$") {
			return c, Decision{Deny, i18n.T(p.Lang, "guard.awskey", k)}, true
		}
	}
	if (strings.HasPrefix(name, "$") || strings.ContainsRune(name, subst)) && !p.inScript {
		return c, Decision{Ask, i18n.T(p.Lang, "guard.unknownprogram", strings.ReplaceAll(c.raw, string(subst), "$(…)"))}, true
	}
	return c, allow, false
}

// everyProgram is what holds whatever the program: one set to a production profile in
// the command itself (AWS_PROFILE=prod bash deploy.sh) asks, since whatever it runs may
// reach AWS with it.
func (p Policy) everyProgram(c call) Decision {
	for _, k := range []string{"AWS_PROFILE", "AWS_DEFAULT_PROFILE"} {
		if prof, ok := c.env[k]; ok && slices.Contains(p.ProductionAWSProfiles, prof) {
			return Decision{Ask, i18n.T(p.Lang, "guard.aws.env", prof)}
		}
	}
	return allow
}

// rules are the programs with a rule of their own. A name here must not be a wrapper:
// program strips wrappers before the rule is looked up (TestWrappersHaveNoRule).
var rules map[string]func(Policy, call) Decision

func init() { // in init: the rules call simple, which reads this table
	rules = map[string]func(Policy, call) Decision{
		"git":         func(p Policy, c call) Decision { return p.git(c.env, c.args, c.depth) },
		"runuser":     Policy.runuser,
		"systemd-run": Policy.systemdRun,
		"machinectl":  Policy.machinectl,
		"script":      Policy.scriptCommand,
		"aws":         func(p Policy, c call) Decision { return p.aws(c.env, c.args) },
		"aws-vault":   func(p Policy, c call) Decision { return p.awsVault(c.args, c.stdin, c.env, c.depth) },
		"ssh":         func(p Policy, c call) Decision { return p.ssh(c.args, c.stdin, c.env, c.depth) },
		"mosh":        func(p Policy, c call) Decision { return p.mosh(c.args, c.env, c.depth) },
		"sftp":        func(p Policy, c call) Decision { return p.sftp(c.args) },
		"eval":        func(p Policy, c call) Decision { return p.inner(strings.Join(c.args, " "), c.env, c.depth) },
		"find":        func(p Policy, c call) Decision { return p.find(c.args, c.env, c.depth) },
		"rm":          func(p Policy, c call) Decision { return p.rm(c.args) },
		"vagrant":     func(p Policy, c call) Decision { return p.vagrant(c.args, c.env, c.depth) },
	}
	add := func(rule func(Policy, call) Decision, names ...string) {
		for _, n := range names {
			rules[n] = rule
		}
	}
	add(func(p Policy, c call) Decision { return p.windowsShell(c.name) }, "powershell", "pwsh", "cmd")
	add(func(p Policy, c call) Decision {
		return p.root(i18n.T(p.Lang, "guard.root.admin", c.name, p.rootReason()))
	}, "gsudo", "runas", "sudoedit")
	add(Policy.source, "source", ".")
	add(func(p Policy, c call) Decision { return p.sudo(c.name, c.args, c.stdin, c.env, c.depth) }, "sudo", "doas")
	add(Policy.su, "su", "run0", "pkexec")
	add(func(p Policy, c call) Decision { return p.awsTool(c.name, c.env, c.args) },
		"cdk", "sam", "serverless", "sls", "terraform", "tofu", "pulumi", "eksctl", "copilot")
	add(func(p Policy, c call) Decision { return p.publish(c.name, c.env, c.args) }, "gh", "glab")
	add(func(p Policy, c call) Decision { return p.copyHosts(c.name, c.args) }, "scp", "rsync")
	add(func(p Policy, c call) Decision { return p.shell(c.args, c.stdin, c.env, c.depth) }, shells...)
	add(Policy.container, "docker", "podman", "kubectl")
}

// other is a program with no rule of its own: one run by its path (./deploy.sh) is
// read, when it is a script, for what it runs.
func (p Policy) other(c call) Decision {
	if strings.Contains(c.raw, "/") {
		return p.runScript(c.raw, c.env, c.depth, shellScript(c.raw))
	}
	return allow
}

// onlyAsks says whether a program's arguments are just one of opts (--help, --version):
// it prints and runs nothing.
func onlyAsks(args []string, opts ...string) bool {
	return len(args) == 1 && slices.Contains(opts, args[0])
}

// runuser runs a command as another user, root unless -u names one.
func (p Policy) runuser(c call) Decision {
	if onlyAsks(c.args, "--help", "--version", "-h", "-V") {
		return allow
	}
	if u := optValue(c.args, "-u", "--user"); u != "" && u != "root" && u != "0" {
		return p.nested(afterOptions(c.args, "u"), c.stdin, c.env, c.depth)
	}
	return p.root(i18n.T(p.Lang, "guard.root.runuser", p.rootReason()))
}

// systemdRun starts the command as a system service, as root unless --user.
func (p Policy) systemdRun(c call) Decision {
	if onlyAsks(c.args, "--version", "--help", "-h") {
		return allow
	}
	if !slices.Contains(c.args, "--user") {
		return p.root(i18n.T(p.Lang, "guard.root.systemdrun", p.rootReason()))
	}
	return allow
}

// machinectl shell and login open a shell as root unless given another user@.
func (p Policy) machinectl(c call) Decision {
	args := c.args
	if len(args) > 0 && (args[0] == "shell" || args[0] == "login") {
		if len(args) < 2 || !strings.Contains(args[1], "@") || strings.HasPrefix(args[1], "root@") || strings.HasPrefix(args[1], "@") {
			return p.root(i18n.T(p.Lang, "guard.root.machinectl", args[0], p.rootReason()))
		}
	}
	return allow
}

// scriptCommand is script -qc 'command' [file]: -c may come last in a cluster.
func (p Policy) scriptCommand(c call) Decision {
	for i, a := range c.args {
		if (a == "--command" || len(a) > 1 && a[0] == '-' && a[1] != '-' && strings.HasSuffix(a, "c")) && i+1 < len(c.args) {
			return p.inner(c.args[i+1], c.env, c.depth)
		}
		if cmd, ok := strings.CutPrefix(a, "--command="); ok {
			return p.inner(cmd, c.env, c.depth)
		}
	}
	return allow
}

// source (and .) runs a script in this shell.
func (p Policy) source(c call) Decision {
	if len(c.args) > 0 {
		return p.script(c.args[0], c.env, c.depth)
	}
	return allow
}

// su, run0 and pkexec switch to another user, root by default.
func (p Policy) su(c call) Decision {
	if onlyAsks(c.args, "--help", "--version", "-h", "-V") {
		return allow
	}
	return p.root(i18n.T(p.Lang, "guard.root.su", c.name, p.rootReason()))
}

// container is docker, podman and kubectl: one given the whole machine is root here,
// and what exec runs inside is checked (docker compose exec <service> <command> too).
func (p Policy) container(c call) Decision {
	args := c.args
	if len(args) > 1 && args[0] == "compose" {
		args = args[1:]
	}
	if how := hostAccess(args); how != "" && c.name != "kubectl" {
		return p.root(i18n.T(p.Lang, "guard.root.container", c.name+" "+args[0], how, p.rootReason()))
	}
	return p.exec(c.name, args, c.env, c.depth)
}

// windowsShell is cmd, powershell or pwsh run from Bash: the guard reads neither.
func (p Policy) windowsShell(name string) Decision {
	return Decision{Deny, i18n.T(p.Lang, "guard.powershell.bash", name)}
}

// checkHost asks before name reaches target (host or user@host) when the host is
// untouchable, or when the configuration could not be read and nobody can tell.
func (p Policy) checkHost(name, target string) Decision {
	host := target[strings.LastIndexByte(target, '@')+1:]
	if host == "" {
		return allow
	}
	if d, unknown := p.listsUnknown(i18n.T(p.Lang, "guard.config.to", name, host)); unknown {
		return d
	}
	if p.untouchable(host) {
		return Decision{Ask, i18n.T(p.Lang, "guard.host", name, host)}
	}
	return allow
}

// subst stands, in a word, for the text of a $(…) or `…`: the word is only known when
// the command runs.
const subst = '\uFFFC'

// optValue returns the value of the first of opts in args (-u app, --user=app, -uapp).
func optValue(args []string, opts ...string) string {
	for i, a := range args {
		for _, o := range opts {
			switch {
			case a == o && i+1 < len(args):
				return args[i+1]
			case strings.HasPrefix(o, "--") && strings.HasPrefix(a, o+"="):
				return strings.TrimPrefix(a, o+"=")
			case len(o) == 2 && !strings.HasPrefix(a, "--") && strings.HasPrefix(a, o) && len(a) > 2:
				return a[2:]
			}
		}
	}
	return ""
}

// afterOptions returns the command after the options of a wrapper whose short options
// in withValue take a value.
func afterOptions(args []string, withValue string) []string {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--":
			return args[i+1:]
		case !strings.HasPrefix(a, "-"):
			return args[i:]
		case len(a) == 2 && strings.ContainsRune(withValue, rune(a[1])):
			i++
		}
	}
	return nil
}

// maxScript is the largest script the guard reads to check what it runs.
const maxScript = 64 << 10

// script checks what a shell script would run (bash deploy.sh, ./deploy.sh, source
// env.sh) when it is a small text file already on disk; one it cannot read, or too
// large to, asks the user.
func (p Policy) script(path string, env map[string]string, depth int) Decision {
	return p.runScript(path, env, depth, true)
}

// shellScript says whether a program run by its path is named as a shell script
// (./x.sh), not a program a build may have just made (./bin/tool).
func shellScript(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext != "" && isShell(ext[1:])
}

// runScript is script; askUnknown says whether it is surely a script (bash x, source
// x, ./x.sh), so that one the guard cannot read, or this command line changed in a way
// it cannot follow, asks the user.
func (p Policy) runScript(path string, env map[string]string, depth int, askUnknown bool) Decision {
	if p.files != nil {
		k := p.fileKey(path)
		if text, ok := p.files.written[k]; ok {
			q := p
			q.inScript = true
			inner := q.inner(text, env, depth)
			if inner.Verdict != Allow {
				inner.Reason = i18n.T(p.Lang, "guard.script", path, inner.Reason)
			}
			return inner
		}
		if askUnknown && p.isTouched(k) {
			return Decision{Ask, i18n.T(p.Lang, "guard.script.written", path)}
		}
	}
	d, err := p.resolve(path)
	if err != nil {
		return allow
	}
	info, err := os.Stat(d)
	if err != nil || !info.Mode().IsRegular() {
		return allow // nothing there to run, or not a file
	}
	if info.Size() > maxScript {
		if !askUnknown {
			return allow // ./bin/tool: a program a build made
		}
		return Decision{Ask, i18n.T(p.Lang, "guard.script.unread", paths.Tilde(d), i18n.T(p.Lang, "guard.script.big", maxScript>>10))}
	}
	data, err := os.ReadFile(d)
	if err != nil {
		if !askUnknown {
			return allow
		}
		return Decision{Ask, i18n.T(p.Lang, "guard.script.unread", paths.Tilde(d), err.Error())}
	}
	if bytes.IndexByte(data, 0) >= 0 {
		return allow // a program, not a script
	}
	p.inScript = true
	inner := p.inner(string(data), env, depth)
	if inner.Verdict != Allow {
		inner.Reason = i18n.T(p.Lang, "guard.script", paths.Tilde(d), inner.Reason)
	}
	return inner
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
func (p Policy) git(env map[string]string, args []string, depth int) Decision {
	return p.gitExpanded(env, args, 0, depth)
}

// gitExpanded is git after n aliases were expanded; depth is how deep it runs.
func (p Policy) gitExpanded(env map[string]string, args []string, n, depth int) Decision {
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
	for j := 1; j < i; j++ {
		if args[j-1] != "-c" {
			continue
		}
		k, v, _ := strings.Cut(strings.ToLower(args[j]), "=")
		if k == "core.pager" && slices.Contains([]string{"", "cat", "less", "more"}, strings.TrimSpace(v)) {
			continue // a pager that only shows the text: git --no-pager, said another way
		}
		if strings.HasPrefix(k, "alias.") ||
			slices.Contains([]string{"core.sshcommand", "core.pager", "core.editor", "core.hookspath", "core.fsmonitor", "sequence.editor", "credential.helper"}, k) {
			return Decision{Ask, i18n.T(p.Lang, "guard.git.config", args[j])}
		}
	}
	if i < len(args) && n < 5 && !gitBuiltin[args[i]] {
		if v, ok := p.gitAlias(dir, elsewhere, args[i]); ok {
			if shell, ok := strings.CutPrefix(v, "!"); ok { // a shell command, given the arguments
				return p.inner(strings.Join(append([]string{shell}, args[i+1:]...), " "), env, depth)
			}
			expanded := append(append(slices.Clone(args[:i]), strings.Fields(v)...), args[i+1:]...)
			return p.gitExpanded(env, expanded, n+1, depth)
		}
	}
	if i < len(args) {
		switch sub := args[i]; {
		case sub == "submodule" && i+2 < len(args) && args[i+1] == "foreach":
			cmd := args[i+2:]
			for len(cmd) > 0 && strings.HasPrefix(cmd[0], "-") {
				cmd = cmd[1:]
			}
			return p.inner(strings.Join(cmd, " "), env, depth)
		case sub == "rebase" || sub == "bisect":
			if c := optValue(args[i+1:], "-x", "--exec"); c != "" {
				return p.inner(c, env, depth)
			}
			if sub == "bisect" && i+2 < len(args) && args[i+1] == "run" {
				return p.nested(args[i+2:], "", env, depth)
			}
		case sub == "send-pack" || sub == "subtree" && i+1 < len(args) && args[i+1] == "push" ||
			sub == "svn" && i+1 < len(args) && args[i+1] == "dcommit":
			return Decision{Deny, i18n.T(p.Lang, "guard.git.publish", sub)}
		}
	}
	if i >= len(args) || (args[i] != "commit" && args[i] != "push") || onlyAsks(args[i+1:], "--help", "-h") {
		return allow // git commit --help shows the manual
	}
	if dir != "" && !elsewhere {
		if d, err := p.resolve(dir); err == nil {
			if p.untouchableRepo(d) {
				return Decision{Deny, i18n.T(p.Lang, "guard.untouchable", paths.Tilde(d), "git "+args[i])}
			}
			verdict := p.repoVerdict(d)
			if verdict != Deny && p.GuardProblem != "" {
				return Decision{Ask, i18n.T(p.Lang, "guard.guardfile", "git "+args[i], paths.Tilde(d), p.GuardProblem)}
			}
			switch verdict {
			case Ask:
				return Decision{Ask, i18n.T(p.Lang, "guard.git.ask", args[i], paths.Tilde(d), p.facts(d))}
			case Allow:
				if p.ConfigChanged {
					return p.changedAsk("git "+args[i], d)
				}
				if args[i] == "push" && rewrites(args[i+1:]) {
					return Decision{Ask, i18n.T(p.Lang, "guard.git.rewrite")}
				}
				return allow
			}
		}
	}
	return Decision{Deny, i18n.T(p.Lang, "guard.git.deny", args[i])}
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
		o = homeVar.ReplaceAllString(o, "~")
		if v := leadingVar.FindString(o); v != "" && !strings.HasPrefix(o, "~") {
			rest := strings.Trim(strings.TrimPrefix(o, v), "/*")
			if rest == "" {
				return Decision{Ask, i18n.T(p.Lang, "guard.rm.var", o, v)}
			}
		}
		if o = strings.TrimSuffix(o, "*"); o == "" {
			o = "." // rm -r * empties the folder it runs in
		}
		if o = strings.TrimRight(o, "/"); o == "" {
			return Decision{Deny, i18n.T(p.Lang, "guard.rm.all")}
		}
		if p.lost && !filepath.IsAbs(o) && !strings.HasPrefix(o, "~") && strings.Trim(o, "./") == "" {
			return Decision{Ask, i18n.T(p.Lang, "guard.rm.lost", o)}
		}
		if d, err := p.resolve(o); err == nil && filepath.Dir(d) == d {
			return Decision{Deny, i18n.T(p.Lang, "guard.rm.disk", d)}
		} else if err == nil && home != "" && d != home && paths.Within(home, d) {
			return Decision{Deny, i18n.T(p.Lang, "guard.rm.abovehome", d)}
		} else if err == nil && home != "" && d == home {
			return Decision{Deny, i18n.T(p.Lang, "guard.rm.home", paths.Tilde(d))}
		}
	}
	return allow
}

// homeVar is the home folder written as a variable: $HOME, ${HOME}, ${HOME:?…},
// ${HOME:-…}, and the same with USERPROFILE (Windows).
var homeVar = regexp.MustCompile(`^(\$(HOME|USERPROFILE)\b|\$\{(HOME|USERPROFILE)(:?[-?=+][^}]*)?\})`)

// leadingVar is a variable at the start of a path: $DIR, ${DIR}, ${DIR:-x}.
var leadingVar = regexp.MustCompile(`^(\$[A-Za-z_][A-Za-z0-9_]*|\$\{[^}]*\})`)

// cd returns the policy for the commands after a cd (or pushd) with args: no folder is
// the home folder, and one the guard cannot tell (-, a variable) leaves it lost.
func (p Policy) cd(args []string) Policy {
	dir, named := "~", false
	for _, a := range args {
		if a == "-" || !strings.HasPrefix(a, "-") {
			dir, named = a, true
			break
		}
	}
	p.cdNamed = false
	if dir == "-" || strings.ContainsAny(dir, "$`") || strings.ContainsRune(dir, subst) {
		p.Cwd, p.lost = "", true
		return p
	}
	if d, err := p.resolve(dir); err == nil {
		p.Cwd, p.lost, p.cdNamed = d, false, named
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
			if strings.ContainsRune("lvkKV", rune(o)) || name == "doas" && o == 'L' {
				listOnly = true // doas -L only forgets the remembered password
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
		d := p.root(p.rootReason())
		if d.Verdict == Deny || len(rest) == 0 {
			return d
		}
		return worse(d, p.nested(rest, stdin, env, depth)) // what runs as root is checked too
	}
	if len(rest) == 0 {
		return allow
	}
	return p.nested(rest, stdin, env, depth)
}

// unknownProfile asks when a profile is a variable and production profiles exist: the
// guard cannot tell which one it will be.
func (p Policy) unknownProfile(prof string) (Decision, bool) {
	if len(p.ProductionAWSProfiles) > 0 && (strings.HasPrefix(prof, "$") || strings.ContainsRune(prof, subst)) {
		return Decision{Ask, i18n.T(p.Lang, "guard.aws.unknown", prof)}, true
	}
	return Decision{}, false
}

// aws asks before a command that may use a production profile. Which one it uses
// depends on the AWS CLI's version and on what is set, so every candidate counts: the
// one given with --profile, AWS_PROFILE and AWS_DEFAULT_PROFILE (set in the command or
// inherited), and "default" when nothing is set.
func (p Policy) aws(env map[string]string, args []string) Decision {
	if d, unknown := p.listsUnknown(i18n.T(p.Lang, "guard.config.aws")); unknown {
		return d
	}
	if d, unknown := p.unknownProfile(profileFlag(args, "--profile")); unknown {
		return d
	}
	if prof := p.productionProfile(env, profileFlag(args, "--profile")); prof != "" {
		return Decision{Ask, i18n.T(p.Lang, "guard.aws.cli", prof)}
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
	if d, unknown := p.unknownProfile(given); unknown {
		return d
	}
	if prof := p.productionProfile(env, given); prof != "" {
		return Decision{Ask, i18n.T(p.Lang, "guard.aws.tool", name, prof)}
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
		inner = p.nested(rest, stdin, sub, depth)
	}
	d := allow
	if u, unknown := p.listsUnknown("aws-vault"); unknown {
		d = u
	} else if slices.Contains(p.ProductionAWSProfiles, prof) {
		d = Decision{Ask, i18n.T(p.Lang, "guard.aws.vault", prof)}
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
	return Decision{Ask, i18n.T(p.Lang, "guard.config", what, p.ConfigProblem)}, true
}

// ghValue are the options of gh and glab that take a value, by command, so the value is
// never taken for a word of the command: "" holds those every command takes. The same
// letter means different things in different commands (-b is --branch in repo sync and
// --body in pr merge), so each command has its own list. The free text of a merge is
// in mergeText.
var ghValue = map[string][]string{
	"":             {"-R", "--repo", "--hostname"},
	"api":          {"-X", "--method", "-H", "--header", "-f", "-F", "--field", "--raw-field"},
	"workflow run": {"-r", "--ref", "-f", "--raw-field", "-F", "--field"},
	"repo sync":    {"-b", "--branch", "-s", "--source"},
}

// syncDestination reports whether gh repo sync names a destination repository (any
// argument besides "repo", "sync" and the values of --branch and --source).
func syncDestination(args []string) bool {
	n := 0
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case slices.Contains(ghValue["repo sync"], a):
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

// publish blocks the gh and glab commands that publish, as it blocks a push. Merging a
// pull request is a push to the branch it goes into, so it goes by the repository lists
// as a push does (merge).
func (p Policy) publish(name string, env map[string]string, args []string) Decision {
	var words []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if slices.Contains(ghValue[""], a) || slices.Contains(ghValue["api"], a) {
			i++ // the option's value
			continue
		}
		if !strings.HasPrefix(a, "-") {
			words = append(words, a)
		}
	}
	if len(words) >= 2 && words[0] == "api" {
		method := strings.ToUpper(optValue(args, "-X", "--method"))
		path := strings.ToLower(words[1])
		writes := method != "" && method != "GET" || slices.ContainsFunc(args, func(a string) bool {
			return a == "-f" || a == "-F" || a == "--field" || a == "--raw-field" || a == "--input"
		})
		if writes && (strings.Contains(path, "/merge") || strings.Contains(path, "/releases") || strings.Contains(path, "/git/refs")) {
			return Decision{Deny, i18n.T(p.Lang, "guard.api", name)}
		}
		if _, rest, ok := strings.Cut(path, "/actions/workflows/"); writes && name == "gh" && ok {
			if wf, ok := strings.CutSuffix(strings.TrimRight(rest, "/"), "/dispatches"); ok {
				return p.dispatch(wf)
			}
		}
		return allow
	}
	if name == "gh" && len(words) >= 2 && words[0] == "workflow" && words[1] == "run" {
		return p.dispatch(workflowArg(args))
	}
	if len(words) < 2 {
		return allow
	}
	blocked := map[string]map[string]bool{
		"gh":   {"pr merge": true, "release create": true, "release upload": true, "release edit": true},
		"glab": {"mr merge": true, "mr accept": true, "release create": true, "release upload": true, "release update": true, "release edit": true},
	}[name]
	what := words[0] + " " + words[1]
	// gh repo sync with no repository only updates the local clone; with one, it updates
	// that repository on GitHub.
	if name == "gh" && what == "repo sync" && syncDestination(args) {
		blocked = map[string]bool{what: true}
	}
	if !blocked[what] || asksHelp(args) {
		return allow // gh pr merge --help shows the help and does nothing else
	}
	if what == "pr merge" || what == "mr merge" || what == "mr accept" {
		return p.merge(name, what, env, args)
	}
	return p.publishDeny(name, what)
}

// asksHelp says whether gh or glab is only asked for help: the two words of the
// command (pr merge) and --help or -h, nothing else. With anything more the option may
// be another one's value (--body -h), and the command would run.
func asksHelp(args []string) bool {
	return len(args) == 3 && !strings.HasPrefix(args[0], "-") && !strings.HasPrefix(args[1], "-") &&
		(args[2] == "--help" || args[2] == "-h")
}

// dispatch is the answer to starting a GitHub Actions workflow by hand. A release
// workflow publishes a version, which is for the user, as gh release create is; any
// other workflow (a deployment of content, say) goes on. A workflow named only by its
// number cannot be told apart, so the user is asked.
func (p Policy) dispatch(workflow string) Decision {
	w := strings.ToLower(workflow)
	if w == "" {
		return allow // gh asks which one, and a session cannot answer
	}
	if strings.Contains(w, "release") {
		return Decision{Deny, i18n.T(p.Lang, "guard.dispatch", workflow)}
	}
	if strings.Trim(w, "0123456789") == "" {
		return Decision{Ask, i18n.T(p.Lang, "guard.dispatch.id", workflow)}
	}
	return allow
}

// workflowArg is the workflow gh workflow run names: the first word after run that is
// not an option or an option's value.
func workflowArg(args []string) string {
	run := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case slices.Contains(ghValue[""], a) || slices.Contains(ghValue["workflow run"], a):
			i++
		case strings.HasPrefix(a, "-"):
		case !run:
			run = a == "run"
		default:
			return a
		}
	}
	return ""
}

// publishDeny is the answer to a gh or glab command that publishes or merges where the
// assistant may not.
func (p Policy) publishDeny(name, what string) Decision {
	return Decision{Deny, i18n.T(p.Lang, "guard.publish", name+" "+what)}
}

// ssh asks before reaching an untouchable host, and checks the remote command, or what
// it sends to the remote shell through stdin.
func (p Policy) ssh(args []string, stdin string, env map[string]string, depth int) Decision {
	host, rest := sshTarget(args)
	host = strings.TrimPrefix(host, "ssh://")
	p.host = host[strings.LastIndexByte(host, '@')+1:] // the remote command runs there
	jumps := allow
	for _, j := range sshJumps(args) {
		jumps = worse(jumps, p.checkHost("ssh", j))
	}
	if sshUser(args, host) == "root" {
		if d := p.root(i18n.T(p.Lang, "guard.root.ssh", p.host, p.rootReason())); d.Verdict == Deny {
			return d
		} else {
			jumps = worse(jumps, d)
		}
	}
	var d Decision
	switch {
	case len(rest) > 0 && isShell(base(rest[0])):
		d = p.nested(rest, stdin, env, depth)
	case len(rest) > 0:
		d = p.inner(strings.Join(rest, " "), env, depth)
	default:
		d = p.inner(stdin, env, depth) // the remote shell reads stdin
	}
	return worse(worse(p.checkHost("ssh", host), jumps), d)
}

// sshUser is the user ssh logs in as: user@host, -l user or -o User=user; "" when the
// command does not say.
func sshUser(args []string, host string) string {
	if at := strings.LastIndexByte(host, '@'); at >= 0 {
		return host[:at]
	}
	if u := optValue(args, "-l"); u != "" {
		return u
	}
	for i, a := range args {
		if a == "-o" && i+1 < len(args) {
			if k, v, ok := strings.Cut(args[i+1], "="); ok && strings.EqualFold(strings.TrimSpace(k), "User") {
				return strings.TrimSpace(v)
			}
		}
	}
	return ""
}

// sshJumps are the hosts an ssh command goes through first (-J, ProxyJump).
func sshJumps(args []string) []string {
	var out []string
	add := func(list string) {
		for _, h := range strings.Split(list, ",") {
			if h = strings.TrimPrefix(strings.TrimSpace(h), "ssh://"); h != "" {
				h, _, _ = strings.Cut(h, ":") // host:port
				out = append(out, h)
			}
		}
	}
	for i, a := range args {
		switch {
		case a == "-J" && i+1 < len(args):
			add(args[i+1])
		case a == "-o" && i+1 < len(args):
			if k, v, ok := strings.Cut(args[i+1], "="); ok && strings.EqualFold(strings.TrimSpace(k), "ProxyJump") {
				add(v)
			}
		}
	}
	return out
}

// vagrant ssh -c runs its command in the virtual machine: it is checked as ssh's is.
func (p Policy) vagrant(args []string, env map[string]string, depth int) Decision {
	if len(args) == 0 || args[0] != "ssh" {
		return allow
	}
	p.host = "(virtual machine)" // a root exception for this machine does not reach it
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
	if len(args) == 0 || args[0] != "exec" {
		return allow
	}
	p.host = "(container)" // a root exception for this machine does not reach it
	args = args[1:]
	if name == "kubectl" {
		// kubectl exec <pod> [options] -- <command>
		if i := slices.Index(args, "--"); i >= 0 && i+1 < len(args) {
			return p.nested(args[i+1:], "", env, depth)
		}
		return allow
	}
	// docker exec [options] <container> <command>
	i := firstOperand(args, oneOf("-e", "--env", "--env-file", "-u", "--user", "-w", "--workdir", "--detach-keys"))
	if i+1 >= len(args) {
		return allow
	}
	return p.nested(args[i+1:], "", env, depth)
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
		q := p
		q.host = host[strings.LastIndexByte(host, '@')+1:]
		d = q.nested(rest, "", env, depth)
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
	host, _, _ := strings.Cut(strings.TrimPrefix(args[i], "sftp://"), ":")
	host, _, _ = strings.Cut(host, "/")
	return p.checkHost("sftp", host)
}

// shell checks the commands a shell would run: the string after -c (also -lc, -ec…),
// or what it reads from stdin when it runs no script.
func (p Policy) shell(args []string, stdin string, env map[string]string, depth int) Decision {
	if noExec(args) {
		return allow // sh -n x.sh: reads the script for its syntax, runs nothing
	}
	wantsC := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		if wantsC && !strings.HasPrefix(a, "-") && !strings.HasPrefix(a, "+") {
			return p.inner(a, env, depth)
		}
		switch {
		case a == "-o" || a == "+o" || a == "-O" || a == "+O" || a == "--rcfile" || a == "--init-file":
			i++ // the option's value: pipefail, extglob, a file
		case strings.HasPrefix(a, "-") && !strings.HasPrefix(a, "--") && strings.ContainsRune(a[1:], 'c'):
			wantsC = true
		case !strings.HasPrefix(a, "-") && !strings.HasPrefix(a, "+"):
			return p.script(a, env, depth) // runs a script file
		}
	}
	return p.inner(stdin, env, depth)
}

// noExec says whether a shell's options turn on noexec (-n, -en, -o noexec), which
// reads commands without running any: a syntax check. The last of -n and +n wins, as
// in the shell; an interactive shell (-i) ignores it. Only the options before the
// script or the -c string count: what follows them is the script's arguments.
func noExec(args []string) bool {
	on, interactive := false, false
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--rcfile" || a == "--init-file":
			i++
			continue
		case strings.HasPrefix(a, "--") && a != "--":
			continue // a long option: --norc, --login…
		case a == "--" || len(a) < 2 || a[0] != '-' && a[0] != '+':
			return on && !interactive // the script, the -c string or --
		}
		set, flags := a[0] == '-', a[1:]
		if strings.ContainsRune(flags, 'n') {
			on = set
		}
		if set && strings.ContainsRune(flags, 'i') {
			interactive = true
		}
		if last := flags[len(flags)-1]; (last == 'o' || last == 'O') && i+1 < len(args) {
			i++ // the option's value: pipefail, noexec, extglob
			if last == 'o' && args[i] == "noexec" {
				on = set
			}
		}
	}
	return on && !interactive
}

// find checks the command of -exec, -execdir, -ok and -okdir.
func (p Policy) find(args []string, env map[string]string, depth int) Decision {
	d := allow
	var roots []string
	for _, a := range args {
		if strings.HasPrefix(a, "-") || a == "(" || a == "!" {
			break
		}
		roots = append(roots, a)
	}
	if len(roots) == 0 {
		roots = []string{"."}
	}
	// Deleting everything under a root, with no test to pick what (find ~ -delete), is
	// rm -r of that root; with -name, -type or another test it deletes only some.
	deletes, picks := slices.Contains(args, "-delete"), false
	for i, a := range args {
		if (a == "-exec" || a == "-execdir") && i+1 < len(args) && base(args[i+1]) == "rm" {
			deletes = true
		}
		if strings.HasPrefix(a, "-") && !slices.Contains([]string{"-delete", "-exec", "-execdir", "-depth", "-mindepth", "-maxdepth", "-xdev", "-mount", "-follow", "-print", "-print0", "-L", "-P", "-H"}, a) {
			picks = true
		}
		if a == "-exec" || a == "-execdir" {
			break // what follows is the command, not tests
		}
	}
	if deletes && !picks {
		if r := p.rm(append([]string{"-r"}, roots...)); r.Verdict != Allow {
			r.Reason = i18n.T(p.Lang, "guard.find", r.Reason)
			return r
		}
	}
	for i := 0; i < len(args); i++ {
		if a := args[i]; a == "-exec" || a == "-execdir" || a == "-ok" || a == "-okdir" {
			var cmd []string
			for _, w := range args[i+1:] {
				if w == ";" || w == "+" {
					break
				}
				cmd = append(cmd, w)
			}
			// The command is checked once, and the search goes on after it: a -exec
			// inside it is its own (find -exec find -exec …), never read twice.
			i += len(cmd)
			if d = worse(d, p.nested(cmd, "", env, depth)); d.Verdict == Deny {
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
		if u, ok := strings.CutPrefix(a, "scp://"); ok {
			a = strings.Replace(u, "/", ":", 1) // scp://host/path is host:path
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
	for _, h := range p.UntouchableHosts {
		if sameHost(host, h) {
			return true
		}
	}
	return false
}

// rootHost matches a host to a root exception: the same name, or a short name and a
// full one that starts with it; two full names must be equal (web01 matches
// web01.example.com, but web01.example.com never matches web01.other.net).
func rootHost(host, saved string) bool {
	norm := func(h string) string {
		h = strings.TrimSpace(h)
		return strings.TrimSuffix(strings.ToLower(h[strings.LastIndexByte(h, '@')+1:]), ".")
	}
	host, saved = norm(host), norm(saved)
	if host == "" || saved == "" {
		return false
	}
	if strings.Contains(host, ".") && strings.Contains(saved, ".") {
		return host == saved
	}
	return sameHost(host, saved)
}

// sameHost matches two host names ignoring case, or by the short name either way. A
// user@ in the saved one is left out: the exception is for the machine.
func sameHost(host, saved string) bool {
	host = strings.TrimSuffix(strings.ToLower(host), ".")
	saved = strings.TrimSpace(saved)
	saved = strings.TrimSuffix(strings.ToLower(saved[strings.LastIndexByte(saved, '@')+1:]), ".")
	if host == "" || saved == "" {
		return false
	}
	return host == saved || strings.HasPrefix(host, saved+".") || strings.HasPrefix(saved, host+".")
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

// shells are the POSIX shells the guard reads: their -c string, their stdin, a script.
var shells = []string{"sh", "bash", "zsh", "dash", "ksh"}

func isShell(name string) bool { return slices.Contains(shells, name) }

// Write checks text the assistant is about to write into a file; lang is the user's
// language, for the reason.
func Write(lang, text string) Decision {
	if what := FindSecret(text); what != "" {
		return Decision{Deny, i18n.T(lang, "guard.secret.write", i18n.T(lang, "secret."+what))}
	}
	return allow
}

// WriteFile checks a write of text into the file at path: Write, and a protected file
// asks the user.
func (p Policy) WriteFile(path, text string) Decision {
	if d := Write(p.Lang, text); d.Verdict == Deny {
		return d
	}
	return p.protected(path)
}

// protected asks before the assistant changes a file in Protected.
func (p Policy) protected(path string) Decision {
	if path == "" || len(p.Protected) == 0 {
		return allow
	}
	d, err := p.resolve(path)
	if err != nil {
		return allow
	}
	for _, f := range p.Protected {
		if paths.Same(d, f) {
			return Decision{Ask, i18n.T(p.Lang, "guard.protected", paths.Tilde(d))}
		}
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
	if p.files == nil {
		return paths.Resolve(path)
	}
	if p.files.resolved == nil {
		p.files.resolved = map[string]resolved{}
	}
	r, ok := p.files.resolved[path]
	if !ok {
		r.path, r.err = paths.Resolve(path)
		p.files.resolved[path] = r
	}
	return r.path, r.err
}

// repoVerdict is what the repository lists say of the repository at dir (resolved):
// Ask in git_ask_repos, Allow in git_allowed_repos, Deny in neither. Asking wins over
// allowing: a repository may be in both lists, from two spaces. A push and the merge
// of a pull request (gh pr merge, glab mr merge) go by the same answer.
func (p Policy) repoVerdict(dir string) Verdict {
	if p.untouchableRepo(dir) {
		return Deny // the one list that wins over the others
	}
	if repoListed(p.GitAskRepos, dir) {
		return Ask
	}
	if repoListed(p.GitAllowedRepos, dir) {
		return Allow
	}
	return Deny
}

// untouchableRepo reports whether the repository at dir (resolved) is in UntouchableRepos.
func (p Policy) untouchableRepo(dir string) bool { return repoListed(p.UntouchableRepos, dir) }

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
	own, common, ok := gitDirs(dir)
	if !ok || common == own {
		return "" // a repository of its own, or nothing
	}
	if c := filepath.Clean(common); filepath.Base(c) == ".git" {
		if r, err := paths.Resolve(filepath.Dir(c)); err == nil {
			return r
		}
	}
	return ""
}

// hostAccess says how docker run (or create) gives a container the whole machine:
// --privileged, the host's processes or network namespace, or the host's / mounted
// inside it. Empty when it does not.
func hostAccess(args []string) string {
	if len(args) > 1 && args[0] == "container" {
		args = args[1:]
	}
	if len(args) == 0 || args[0] != "run" && args[0] != "create" {
		return ""
	}
	for i, a := range args[1:] {
		next := ""
		if i+2 < len(args) {
			next = args[i+2]
		}
		key, value, hasValue := strings.Cut(a, "=")
		if !hasValue {
			value = next
		}
		switch {
		case a == "--privileged" || a == "--privileged=true":
			return "--privileged"
		case (key == "--pid" || key == "--userns" || key == "--ipc") && value == "host":
			return key + " host"
		case key == "--cap-add" && (strings.EqualFold(value, "ALL") || strings.EqualFold(value, "SYS_ADMIN")):
			return "--cap-add " + value
		case key == "-v" || key == "--volume":
			if src, _, _ := strings.Cut(value, ":"); src == "/" {
				return "-v " + value
			}
		case strings.HasPrefix(a, "-v/:"):
			return a
		case key == "--mount":
			for _, part := range strings.Split(value, ",") {
				if k, v, _ := strings.Cut(part, "="); (k == "source" || k == "src") && v == "/" {
					return "--mount " + value
				}
			}
		}
	}
	return ""
}
