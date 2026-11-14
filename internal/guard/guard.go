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
	"path"
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
	// Absolute paths; writing one asks the user, and so does writing into the folder
	// that holds one (cp -t <folder>, tar -C <folder>), which may put a file there.
	Protected []string
	// ProtectedDirs are folders whose every file is protected that way: Claude Code's
	// copy of the plugin, whose version changes with each update.
	ProtectedDirs []string
	// Hidden are files the assistant must not read at all (the panel's token file), and
	// ConfigDirs the ways the configuration folder is written in a command (its path,
	// with ~, and the variable that moves it), to tell that a bare file name is theirs.
	Hidden     []string
	ConfigDirs []string
	// ConfigChanged says the guard's lists are not the ones the user last saw (at the
	// start of the session, or saved in the panel): something else changed them. Where
	// the lists would let a commit, a push or a merge through, it asks instead.
	ConfigChanged bool
	// NoPrompt says the session may not show the user a permission dialog
	// (bypassPermissions): what is asked only so the user decides, such as starting a
	// release workflow, is denied instead.
	NoPrompt bool
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
	// Outside says the session works outside every space while augmented brain keeps
	// to them (guard.toml's reach = "spaces"): only the rules that protect its own
	// files and what the lists name apply, unless the command reaches into one of
	// SpaceDirs, the folders of the spaces (absolute, resolved). See outside.go.
	Outside   bool
	SpaceDirs []string
	// outside says this reading of the line keeps to the rules that hold outside.
	outside bool
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
	// from is the file the command being checked reads as its standard input (< f, or
	// cat f | …), when its stdin is not text the line holds; blind says it reads the
	// output of a program the guard cannot know (curl … | sh). Set for each command.
	from  string
	blind bool
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
	if p.outside {
		return allow // a general rule
	}
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
			if n != "" && rootHost(n, e.Host) && p.ConfigChanged {
				// The exception may be what changed: it counts only once the user
				// has seen it (the panel's Guard page, or a session they start).
				return Decision{Deny, i18n.T(p.Lang, "guard.root.changed", reason)}
			}
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
	if p.Outside {
		// Outside the spaces first; a line that reaches into one is read again whole.
		q := p
		q.Outside, q.outside = false, true
		if d, inSpace := q.command(line); !inSpace {
			return d
		}
	}
	p.Outside, p.outside = false, false
	d, _ := p.command(line)
	return d
}

// command is one reading of a line; inSpace says it reached into a space while
// reading it outside them.
func (p Policy) command(line string) (Decision, bool) {
	if what := FindSecret(line); what != "" && !p.outside {
		return Decision{Deny, i18n.T(p.Lang, "guard.secret.command", p.secret(what))}, false
	}
	if p.panelAccess(line) {
		return p.panelDeny(), false
	}
	if len(line) > maxCommand {
		return Decision{Ask, i18n.T(p.Lang, "guard.toolong", len(line)>>10)}, false
	}
	p.files = &lineFiles{written: map[string]string{}, touched: map[string]bool{}}
	d := p.commands(parse(line), map[string]string{}, 0)
	return d, p.files.inSpace
}

// commands checks a list of commands in order. Exported variables carry over to the
// commands after them. A deny wins over an ask.
func (p Policy) commands(cmds []command, exported map[string]string, depth int) Decision {
	d := allow
	from := make([]string, len(cmds)) // what each command reads, when not its stdin text
	blind := make([]bool, len(cmds))
	for i, c := range cmds {
		if c.piped && c.stdin == "" && i > 0 {
			// echo 'sudo id' | bash: bash reads what echo writes. Kept in the list, so a
			// command further down the pipe (… | tee x | bash) gets it too.
			cmds[i].stdin, from[i], blind[i] = output(cmds[i-1], from[i-1], blind[i-1])
			c.stdin = cmds[i].stdin
		}
		if c.stdin == "" && len(c.inputs) > 0 {
			from[i], blind[i] = c.inputs[len(c.inputs)-1], false // bash < f: the last one wins
		}
		if len(c.words) > 0 && (base(c.words[0]) == "cd" || base(c.words[0]) == "pushd") {
			p = p.cd(c.words[1:]) // the commands after it run there
		}
		if len(c.words) > 0 && base(c.words[0]) == "popd" {
			p.Cwd, p.lost, p.cdNamed = "", true, false // back to a folder the guard did not keep
			p.touch()
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
		if _, name, args := program(c.words); writesArgs(name, args) {
			for _, a := range args {
				d = worse(d, p.protected(writtenArg(a)))
			}
		}
		if p.copiesHidden(c.words) {
			return p.panelDeny()
		}
		q := p
		q.from, q.blind = from[i], blind[i]
		if d = worse(d, q.simple(c.words, c.stdin, exported, depth)); d.Verdict == Deny {
			return d
		}
		p.recordWrites(c)
	}
	return d
}

// writers are the programs whose arguments name files they may write, read as such by
// the protected-file rule: a path among them, or the folder that holds one, asks.
var writers = []string{"sed", "perl", "tee", "cp", "mv", "ln", "rm", "truncate", "install", "dd",
	"rsync", "tar", "patch", "unzip"}

// writesArgs says whether the arguments of name are read for protected files: one of
// the writers, or awk changing files in place (gawk -i inplace).
func writesArgs(name string, args []string) bool {
	if slices.Contains(writers, name) {
		return true
	}
	if name == "awk" || name == "gawk" {
		return slices.ContainsFunc(args, func(a string) bool { return strings.Contains(a, "inplace") })
	}
	return false
}

// writtenArg is the path an argument names: dd's of=, tar's --directory=, cp's --target-directory=.
func writtenArg(a string) string {
	for _, prefix := range []string{"of=", "--directory=", "--target-directory="} {
		if strings.HasPrefix(a, prefix) {
			return strings.TrimPrefix(a, prefix)
		}
	}
	return a
}

// worse combines two decisions: a deny wins over an ask, and an ask over an allow. Of
// two asks, the first one's reason is kept.
func worse(a, b Decision) Decision {
	if b.Verdict > a.Verdict {
		return b
	}
	return a
}

// output returns what a command writes, for the command after it in a pipe: echo and
// printf of their arguments, cat and tee of what they read (their heredoc or
// here-string, or the file named in from, or cat's one file). A shell or ssh at the end
// of a pipe runs it, so the guard checks it as their stdin. Of any other program the
// output is not known: blind is true.
func output(c command, inFrom string, inBlind bool) (text, from string, blind bool) {
	_, name, args := program(c.words)
	if c.stdin == "" && len(c.inputs) > 0 {
		inFrom, inBlind = c.inputs[len(c.inputs)-1], false
	}
	switch name {
	case "tee":
		return c.stdin, inFrom, inBlind
	case "cat":
		files := slices.DeleteFunc(slices.Clone(args), func(a string) bool { return strings.HasPrefix(a, "-") && a != "-" })
		switch {
		case len(files) == 0:
			return c.stdin, inFrom, inBlind
		case len(files) == 1 && files[0] != "-":
			return "", files[0], false
		}
		return "", "", true
	case "echo":
		for len(args) > 0 && (args[0] == "-n" || args[0] == "-e" || args[0] == "-E" || args[0] == "-ne" || args[0] == "-en") {
			args = args[1:]
		}
		return strings.ReplaceAll(strings.Join(args, " "), `\n`, "\n") + "\n", "", false
	case "printf":
		return strings.ReplaceAll(strings.Join(args, " "), `\n`, "\n") + "\n", "", false
	}
	return "", "", true
}

// fromStdin checks the commands a program reads from its standard input (a shell with
// no -c and no script, at): the text the line holds, the file it is given (bash < f),
// or, when it reads the output of a program the guard cannot know, the user is asked.
func (p Policy) fromStdin(name, stdin string, env map[string]string, depth int) Decision {
	switch {
	case stdin != "":
		return p.inner(stdin, env, depth)
	case p.from != "":
		return p.script(p.from, env, depth)
	case p.blind:
		return Decision{Ask, i18n.T(p.Lang, "guard.stdin", name)}
	}
	return allow
}

// stdinPath says whether a script path is the program's own standard input.
func stdinPath(path string) bool {
	return path == "/dev/stdin" || path == "/dev/fd/0" || path == "/proc/self/fd/0"
}

// unknownWord says whether a word is only known when the command runs: it holds a
// variable or a command substitution.
func unknownWord(w string) bool {
	return strings.ContainsRune(w, '$') || strings.ContainsRune(w, subst) || strings.ContainsRune(w, '`')
}

// inner checks a command line run by another command (ssh, sh -c, a heredoc for bash).
func (p Policy) inner(line string, exported map[string]string, depth int) Decision {
	if strings.TrimSpace(line) == "" {
		return allow
	}
	if depth >= maxDepth {
		return Decision{Ask, i18n.T(p.Lang, "guard.depth")}
	}
	if what := FindSecret(line); what != "" && !p.outside {
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
		if v, ok := env[k]; ok && v != "" && !strings.HasPrefix(v, "$") && !p.outside {
			return c, Decision{Deny, i18n.T(p.Lang, "guard.awskey", k)}, true
		}
	}
	if (strings.HasPrefix(name, "$") || strings.ContainsRune(name, subst)) && !p.inScript {
		return c, Decision{Ask, i18n.T(p.Lang, "guard.unknownprogram", strings.ReplaceAll(c.raw, string(subst), "$(…)"))}, true
	}
	if sub, ok := subcommand(name, args); ok && unknownWord(sub) && !p.inScript {
		return c, Decision{Ask, i18n.T(p.Lang, "guard.unknownsub", name, strings.ReplaceAll(sub, string(subst), "$(…)"))}, true
	}
	if name != "" {
		if token, ok := xargsBefore(words[:len(words)-len(args)-1]); ok && xargsFills(name, args, token) {
			return c, Decision{Ask, i18n.T(p.Lang, "guard.xargs", name)}, true
		}
	}
	return c, allow, false
}

// gitValued are git's options before the subcommand that take the next word as value.
var gitValued = []string{"-C", "-c", "--git-dir", "--work-tree", "--namespace", "--super-prefix", "--config-env"}

// subcommand is the word that says what git, gh or glab does (push, pr), after the
// options before it; ok is false for other programs, or when there is no such word.
func subcommand(name string, args []string) (string, bool) {
	var valued []string
	switch name {
	case "git":
		valued = gitValued
	case "gh", "glab":
		valued = ghValue[""]
	default:
		return "", false
	}
	if i := firstOperand(args, oneOf(valued...)); i < len(args) {
		return args[i], true
	}
	return "", false
}

// xargsBefore says whether xargs is among the wrappers before a program (the words
// before it), and the word its -I (or -i, --replace) puts the input in place of.
func xargsBefore(prefix []string) (token string, ok bool) {
	for i, w := range prefix {
		if base(w) != "xargs" {
			continue
		}
		ok = true
		for j := i + 1; j < len(prefix); j++ {
			a := prefix[j]
			switch {
			case a == "-I" && j+1 < len(prefix):
				token = prefix[j+1]
			case strings.HasPrefix(a, "-I") && len(a) > 2:
				token = a[2:]
			case a == "-i" || a == "--replace":
				token = "{}"
			case strings.HasPrefix(a, "--replace="):
				token = strings.TrimPrefix(a, "--replace=")
			case strings.HasPrefix(a, "-i") && len(a) > 2:
				token = a[2:]
			}
		}
	}
	return token, ok
}

// xargsFills says whether the words xargs reads may become what a program does rather
// than what it works on: git, gh or glab given their subcommand, a shell given its
// commands or its script.
func xargsFills(name string, args []string, token string) bool {
	fills := func(w string) bool { return token != "" && w == token }
	if _, ok := map[string]bool{"git": true, "gh": true, "glab": true}[name]; ok {
		sub, ok := subcommand(name, args)
		return !ok || fills(sub)
	}
	if !isShell(name) {
		return false
	}
	for i, a := range args {
		switch {
		case strings.HasPrefix(a, "-") && !strings.HasPrefix(a, "--") && strings.ContainsRune(a[1:], 'c'):
			return i+1 >= len(args) || fills(args[i+1])
		case !strings.HasPrefix(a, "-") && !strings.HasPrefix(a, "+"):
			return fills(a) // the script
		}
	}
	return true // sh with nothing: the first word read is the script
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
		"mv":          Policy.mv,
		"tmux":        Policy.tmux,
		"sg":          Policy.sg,
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
	// sudo-rs and su-rs come with Ubuntu 25.10 and later, next to sudo and su.
	add(func(p Policy, c call) Decision { return p.sudo(c.name, c.args, c.stdin, c.env, c.depth) }, "sudo", "sudo-rs", "doas")
	add(Policy.su, "su", "su-rs", "run0", "pkexec")
	add(func(p Policy, c call) Decision { return p.awsTool(c.name, c.env, c.args) },
		"cdk", "sam", "serverless", "sls", "terraform", "tofu", "pulumi", "eksctl", "copilot")
	add(func(p Policy, c call) Decision { return p.publish(c.name, c.env, c.args, c.depth) }, "gh", "glab")
	add(Policy.remoteCopy, "scp", "rsync")
	add(Policy.sshfs, "sshfs")
	add(Policy.autossh, "autossh")
	add(func(p Policy, c call) Decision { return p.shell(c.name, c.args, c.stdin, c.env, c.depth) }, shells...)
	add(Policy.at, "at", "batch")
	add(Policy.container, "docker", "podman", "nerdctl", "kubectl", "docker-compose", "podman-compose")
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
		return p.shellScript(c.name, c.args[0], c.stdin, c.env, c.depth)
	}
	return allow
}

// shellScript checks the script a shell or source runs: a file, its own standard
// input (/dev/stdin), or the output of a command (<(…)), which the guard cannot see.
func (p Policy) shellScript(name, path, stdin string, env map[string]string, depth int) Decision {
	switch {
	case stdinPath(path):
		return p.fromStdin(name, stdin, env, depth)
	case strings.ContainsRune(path, subst):
		return Decision{Ask, i18n.T(p.Lang, "guard.procsubst", name)}
	}
	return p.script(path, env, depth)
}

// at and batch run later the commands they read: from their standard input, or from
// the file of -f.
func (p Policy) at(c call) Decision {
	if f := optValue(c.args, "-f"); f != "" {
		return p.script(f, c.env, c.depth)
	}
	if slices.ContainsFunc(c.args, func(a string) bool { return a == "-l" || a == "-d" || a == "-r" || a == "-c" }) {
		return allow // lists, removes or shows jobs
	}
	return p.fromStdin(c.name, c.stdin, c.env, c.depth)
}

// sg runs a command with another group: sg group -c 'command', or sg group command.
func (p Policy) sg(c call) Decision {
	args := c.args
	if len(args) > 0 && args[0] == "-" {
		args = args[1:]
	}
	if len(args) < 2 {
		return allow
	}
	if args[1] == "-c" {
		if len(args) < 3 {
			return allow
		}
		return p.inner(args[2], c.env, c.depth)
	}
	return p.inner(strings.Join(args[1:], " "), c.env, c.depth)
}

// tmuxRuns are tmux's commands whose last words are a shell command it runs, and
// send-keys, whose words are typed into a shell; tmuxValued the options that take a
// value, of those commands and of tmux itself.
var (
	tmuxRuns = map[string]bool{"new-session": true, "new": true, "new-window": true, "neww": true,
		"split-window": true, "splitw": true, "respawn-pane": true, "respawnp": true,
		"respawn-window": true, "respawnw": true, "run-shell": true, "run": true,
		"display-popup": true, "popup": true, "send-keys": true, "send": true}
	tmuxValued = "ceFfLlnSsTtxyw" // not -d: in new-session and new-window it takes no value
)

// tmux checks what tmux runs: the command of -c, and the shell command at the end of
// the commands that start one (new-session, new-window, split-window, run-shell…) or
// the keys send-keys types. Commands joined with \; are each read.
func (p Policy) tmux(c call) Decision {
	args := c.args
	for len(args) > 0 && strings.HasPrefix(args[0], "-") {
		a := args[0]
		args = args[1:]
		if a == "-c" && len(args) > 0 {
			return p.inner(args[0], c.env, c.depth) // tmux -c: a shell command, as sh -c
		}
		if len(a) == 2 && strings.ContainsRune("LSfT", rune(a[1])) && len(args) > 0 {
			args = args[1:]
		}
	}
	d := allow
	for len(args) > 0 {
		end := slices.Index(args, ";")
		if end < 0 {
			end = len(args)
		}
		cmd := args[:end]
		if len(cmd) > 0 && tmuxRuns[cmd[0]] {
			rest := cmd[1:]
			for len(rest) > 0 && strings.HasPrefix(rest[0], "-") && rest[0] != "-" {
				a := rest[0]
				rest = rest[1:]
				if a == "--" {
					break
				}
				if len(a) == 2 && strings.ContainsRune(tmuxValued, rune(a[1])) && len(rest) > 0 {
					rest = rest[1:]
				}
			}
			if cmd[0] == "send-keys" || cmd[0] == "send" {
				for _, k := range rest {
					d = worse(d, p.inner(k, c.env, c.depth))
				}
			} else if len(rest) > 0 {
				d = worse(d, p.inner(strings.Join(rest, " "), c.env, c.depth))
			}
		}
		if end == len(args) {
			break
		}
		args = args[end+1:]
	}
	return d
}

// su, run0 and pkexec switch to another user, root by default.
func (p Policy) su(c call) Decision {
	if onlyAsks(c.args, "--help", "--version", "-h", "-V") {
		return allow
	}
	return p.root(i18n.T(p.Lang, "guard.root.su", c.name, p.rootReason()))
}

// container is docker, podman, nerdctl and kubectl: one given the whole machine is
// root here, and what exec runs inside is checked (docker compose exec <service>
// <command> too). Global options before the subcommand (docker --context x run) are
// skipped, as the program skips them.
func (p Policy) container(c call) Decision {
	name := c.name
	args := afterGlobal(c.args, containerGlobal[name])
	if n, ok := strings.CutSuffix(name, "-compose"); ok { // docker-compose is docker compose
		name, args = n, append([]string{"compose"}, args...)
	}
	if len(args) > 1 && args[0] == "compose" {
		args = afterGlobal(args[1:], composeGlobal)
	}
	if name == "kubectl" {
		if node := kubectlDebugNode(args); node != "" {
			return p.root(i18n.T(p.Lang, "guard.root.container", "kubectl debug", node, p.rootReason()))
		}
	} else if how := hostAccess(args); how != "" {
		return p.root(i18n.T(p.Lang, "guard.root.container", name+" "+args[0], how, p.rootReason()))
	}
	return p.exec(name, args, c.env, c.depth)
}

// containerGlobal are, for each program, its options before the subcommand that take a
// value; composeGlobal are those of compose, before its own subcommand.
var (
	containerGlobal = map[string][]string{
		"docker": {"-c", "--context", "-H", "--host", "--config", "-l", "--log-level",
			"--tlscacert", "--tlscert", "--tlskey"},
		"podman": {"-c", "--connection", "--url", "--identity", "--root", "--runroot",
			"--storage-driver", "--storage-opt", "--log-level", "--cgroup-manager", "--conmon",
			"--events-backend", "--hooks-dir", "--imagestore", "--module", "--network-cmd-path",
			"--network-config-dir", "--runtime", "--runtime-flag", "--ssh", "--tmpdir",
			"--volumepath", "--registries-conf", "--cni-config-dir", "--db-backend",
			"--network-backend", "--out"},
		"nerdctl": {"-a", "--address", "-H", "--host", "-n", "--namespace", "--snapshotter",
			"--storage-driver", "--cni-path", "--cni-netconfpath", "--data-root",
			"--cgroup-manager", "--hosts-dir", "--host-gateway-ip", "--bridge-ip"},
		"kubectl": {"-n", "--namespace", "--context", "--kubeconfig", "-s", "--server",
			"--cluster", "--user", "--token", "--as", "--as-group", "--as-uid",
			"--certificate-authority", "--client-certificate", "--client-key",
			"--request-timeout", "--tls-server-name", "--cache-dir", "-v", "--v", "--vmodule",
			"--log-file", "--password", "--username", "--profile", "--profile-output"},
	}
	composeGlobal = []string{"-f", "--file", "-p", "--project-name", "--project-directory",
		"--env-file", "--profile", "--ansi", "--progress", "--parallel"}
)

func init() {
	containerGlobal["docker-compose"] = composeGlobal
	containerGlobal["podman-compose"] = composeGlobal
}

// afterGlobal returns args from the first word that is neither an option nor the value
// of one of valued (written apart: --context x; --context=x is one word).
func afterGlobal(args, valued []string) []string {
	return args[firstOperand(args, oneOf(valued...)):]
}

// kubectlDebugNode is the node of kubectl debug node/<name>: a pod on that node with
// its processes, network and whole disk (under /host). Empty for any other command.
func kubectlDebugNode(args []string) string {
	if len(args) == 0 || args[0] != "debug" {
		return ""
	}
	for _, a := range args[1:] {
		if strings.HasPrefix(a, "node/") || strings.HasPrefix(a, "nodes/") {
			return a
		}
	}
	return ""
}

// windowsShell is cmd, powershell or pwsh run from Bash: the guard reads neither.
func (p Policy) windowsShell(name string) Decision {
	if p.outside {
		return allow // a general rule
	}
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
func (p Policy) gitExpanded(env map[string]string, args []string, n, depth int) (d Decision) {
	dir := ""
	// Another repository, or another repository's refs, objects or index: never taken as listed.
	elsewhere := slices.ContainsFunc([]string{"GIT_DIR", "GIT_WORK_TREE", "GIT_COMMON_DIR", "GIT_INDEX_FILE",
		"GIT_OBJECT_DIRECTORY", "GIT_ALTERNATE_OBJECT_DIRECTORIES"}, func(k string) bool { return env[k] != "" })
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
	settings := gitEnvSettings(env)
	for j := 0; j < i; j++ {
		switch a := args[j]; {
		case (a == "-c" || a == "--config-env") && j+1 < i:
			settings = append(settings, args[j+1])
		case strings.HasPrefix(a, "--config-env="):
			settings = append(settings, strings.TrimPrefix(a, "--config-env="))
		}
	}
	runs := p.gitEnv(env, depth)
	for _, s := range settings {
		runs = worse(runs, p.gitSetting(s, env, depth))
	}
	if runs.Verdict == Deny {
		return runs
	}
	defer func() { d = worse(d, runs) }() // whatever git then does, the settings still ask
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
			if !elsewhere && p.freeRepo(cmp.Or(dir, p.cwdKnown())) {
				return allow // a general rule, in a repository no list names
			}
			return Decision{Deny, i18n.T(p.Lang, "guard.git.publish", sub)}
		}
	}
	if i < len(args) {
		if d := p.gitCommand(args[i], args[i+1:], env, depth); d.Verdict != Allow {
			return d
		}
		target := dir
		if target == "" && !p.lost {
			target = p.Cwd
		}
		if target != "" && !elsewhere && commitsElsewhere(args[i], args[i+1:]) {
			if d, err := p.resolve(target); err == nil && p.untouchableRepo(d) {
				return Decision{Deny, i18n.T(p.Lang, "guard.untouchable", paths.Tilde(d), "git "+args[i])}
			}
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
				if args[i] != "push" {
					return allow
				}
				if to, named := p.pushTo(d, pushDestination(args[i+1:])); named && to.Verdict != Allow {
					return to
				}
				if rewrites(args[i+1:]) {
					return Decision{Ask, i18n.T(p.Lang, "guard.git.rewrite")}
				}
				return allow
			}
		}
	}
	if !elsewhere {
		if repo := cmp.Or(dir, p.cwdKnown()); p.freeRepo(repo) &&
			(args[i] != "push" || p.freePush(repo, pushDestination(args[i+1:]))) {
			return allow // a general rule, in a repository no list names
		}
	}
	return Decision{Deny, i18n.T(p.Lang, "guard.git.deny", args[i])}
}

// cwdKnown is the folder the command runs in, or "" when a cd lost it.
func (p Policy) cwdKnown() string {
	if p.lost {
		return ""
	}
	return p.Cwd
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
		if strings.Trim(o, "/*") == string(subst) { // rm -r "$(…)": known only when it runs
			return Decision{Ask, i18n.T(p.Lang, "guard.rm.subst")}
		}
		if o = wholeFolder(o); o == "" {
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

// wholeFolder is the folder a pattern empties when its last part matches nearly every
// name in it: ~/*, ~/.*, ~/[a-z]*, ~/?* are ~/. A pattern with a fixed start (build*)
// is read without its last *, as the name it starts with.
func wholeFolder(o string) string {
	slash := strings.LastIndexAny(o, `/\`)
	last := o[slash+1:]
	if last != "" && (strings.ContainsRune("*?[", rune(last[0])) || strings.HasPrefix(last, ".*") || strings.HasPrefix(last, ".[") || strings.HasPrefix(last, ".?")) {
		return o[:slash+1]
	}
	return strings.TrimSuffix(o, "*")
}

// homeOrAbove says whether a path is the home folder, a folder that holds it, or a
// disk's root: what mv takes away and rsync --delete empties.
func (p Policy) homeOrAbove(o string) (string, bool) {
	o = homeVar.ReplaceAllString(o, "~")
	d, err := p.resolve(o)
	if err != nil {
		return "", false
	}
	if filepath.Dir(d) == d {
		return d, true
	}
	home, _ := paths.Home()
	if h, err := paths.Resolve(home); err == nil {
		home = h
	}
	return paths.Tilde(d), home != "" && paths.Within(home, d)
}

// mv never moves away the home folder, a folder that holds it or a disk's root.
func (p Policy) mv(c call) Decision {
	var operands []string
	target := false
	for i := 0; i < len(c.args); i++ {
		a := c.args[i]
		switch {
		case a == "--":
			operands = append(operands, c.args[i+1:]...)
			i = len(c.args)
		case a == "-t" || a == "--target-directory":
			target = true
			i++
		case strings.HasPrefix(a, "--target-directory=") || strings.HasPrefix(a, "-t") && len(a) > 2:
			target = true
		case a == "-S" || a == "--suffix":
			i++
		case strings.HasPrefix(a, "-") && len(a) > 1:
		default:
			operands = append(operands, a)
		}
	}
	if !target && len(operands) > 0 {
		operands = operands[:len(operands)-1] // the last one is where they go
	}
	for _, o := range operands {
		if d, ok := p.homeOrAbove(o); ok {
			return Decision{Deny, i18n.T(p.Lang, "guard.mv.home", d)}
		}
	}
	return allow
}

// syncDeletes is rsync --delete (or one of its kin) into the home folder, a folder that
// holds it or a disk's root: it deletes there whatever the source does not have.
func (p Policy) syncDeletes(args []string) Decision {
	deletes := slices.ContainsFunc(args, func(a string) bool {
		return a == "--del" || strings.HasPrefix(a, "--delete")
	})
	if !deletes {
		return allow
	}
	var last string
	for i := 0; i < len(args); i++ {
		switch a := args[i]; {
		case a == "--":
			if i+1 < len(args) {
				last = args[len(args)-1]
			}
			i = len(args)
		case strings.HasPrefix(a, "-"):
		default:
			last = a
		}
	}
	if last == "" || strings.Contains(last, ":") {
		return allow // another machine's folder: not this one's home
	}
	if d, ok := p.homeOrAbove(last); ok {
		return Decision{Deny, i18n.T(p.Lang, "guard.rsync.delete", d)}
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
		p.touch() // it may be a space
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
// When the lists are not the ones the user last saw (ConfigChanged), a profile or a
// host taken out of them may be what changed: the command asks too.
func (p Policy) listsUnknown(what string) (Decision, bool) {
	if p.ConfigProblem != "" {
		return Decision{Ask, i18n.T(p.Lang, "guard.config", what, p.ConfigProblem)}, true
	}
	if p.ConfigChanged {
		return Decision{Ask, i18n.T(p.Lang, "guard.config.changed", what)}, true
	}
	return allow, false
}

// ghValue are the options of gh and glab that take a value, by command, so the value is
// never taken for a word of the command: "" holds those every command takes. The same
// letter means different things in different commands (-b is --branch in repo sync and
// --body in pr merge), so each command has its own list. The free text of a merge is
// in mergeText.
var ghValue = map[string][]string{
	"": {"-R", "--repo", "--hostname"},
	"api": {"-X", "--method", "-H", "--header", "-f", "-F", "--field", "--raw-field", "--input",
		"-q", "--jq", "-t", "--template", "--cache", "-p", "--preview"},
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
func (p Policy) publish(name string, env map[string]string, args []string, depth int) Decision {
	return p.publishExpanded(name, env, args, 0, depth)
}

// publishExpanded is gh or glab after n aliases were expanded; depth is how deep it runs.
func (p Policy) publishExpanded(name string, env map[string]string, args []string, n, depth int) Decision {
	var words []string
	first := -1 // where the first word is in args
	for i := 0; i < len(args); i++ {
		a := args[i]
		if slices.Contains(ghValue[""], a) || slices.Contains(ghValue["api"], a) {
			i++ // the option's value
			continue
		}
		if !strings.HasPrefix(a, "-") {
			if first < 0 {
				first = i
			}
			words = append(words, a)
		}
	}
	if len(words) > 0 && n < 5 && !ghCommands[words[0]] {
		if v, ok := p.ghAlias(name, env, words[0]); ok {
			if shell, ok := strings.CutPrefix(v, "!"); ok { // a shell command, given the arguments
				return p.inner(strings.Join(append([]string{shell}, args[first+1:]...), " "), env, depth)
			}
			expanded := append(append(slices.Clone(args[:first]), strings.Fields(v)...), args[first+1:]...)
			return p.publishExpanded(name, env, expanded, n+1, depth)
		}
	}
	if p.freeGh(name, env, args, words) {
		return allow // general rules, on repositories no list names
	}
	if len(words) >= 2 && words[0] == "api" {
		return p.api(name, args, strings.ToLower(words[1]))
	}
	if name == "gh" && len(words) >= 2 && words[0] == "workflow" && words[1] == "run" {
		return p.dispatch(workflowArg(args))
	}
	if len(words) >= 2 && words[0] == "alias" && (words[1] == "set" || words[1] == "import") {
		return Decision{Ask, i18n.T(p.Lang, "guard.gh.alias", name+" alias "+words[1])}
	}
	if name == "gh" && len(words) >= 2 && words[0] == "run" && words[1] == "rerun" && !asksHelp(args) {
		return Decision{Ask, i18n.T(p.Lang, "guard.gh.rerun", "gh run rerun")}
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
// workflow publishes a version, which is for the user: the user is asked, so a release
// they asked for goes with one click, and without their yes nothing starts. Any other
// workflow (a deployment of content, say) goes on. A workflow named only by its number
// cannot be told apart, so the user is asked too.
func (p Policy) dispatch(workflow string) Decision {
	w := strings.ToLower(workflow)
	if w == "" {
		return allow // gh asks which one, and a session cannot answer
	}
	if strings.Contains(w, "release") {
		if p.NoPrompt {
			return Decision{Deny, i18n.T(p.Lang, "guard.dispatch.bypass", workflow)}
		}
		return Decision{Ask, i18n.T(p.Lang, "guard.dispatch", workflow)}
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
	o, rest := readSSH(args, sshValued, true)
	user, host := sshDest(o.dest)
	if o.hostName != "" {
		host = o.hostName // ssh -o HostName=db01 web02 goes to db01
	}
	if user == "" {
		user = o.user
	}
	around := p.sshAround("ssh", o, env, depth) // before p.host changes: a ProxyCommand runs here
	p.host = host                               // the remote command runs there
	if user == "root" {
		if d := p.root(i18n.T(p.Lang, "guard.root.ssh", p.host, p.rootReason())); d.Verdict == Deny {
			return d
		} else {
			around = worse(around, d)
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
	return worse(worse(p.checkHost("ssh", host), around), d)
}

// sshValued are ssh's single-letter options that take a value; scpValued are scp's.
const (
	sshValued = "BbcDEeFIiJLlmOoPpQRSWw"
	scpValued = "cDFiJlOoPSX"
)

// sshOpts is what an ssh command line says about where it goes: the destination as
// written, the user and the host name given as options, the hosts it jumps through
// and the commands it runs here to reach the host (ProxyCommand).
type sshOpts struct {
	dest, user, hostName string
	jumps, proxies       []string
}

// readSSH reads ssh's (or scp's) options as ssh does: -J host, -l user and -o in every
// form it takes (-o User=root, -oUser=root, -o 'User root', inside a cluster such as
// -vo). valued are the letters that take a value; withDest says the first operand is
// the destination, and then rest is the remote command after it. -l is the user only
// for ssh: scp's -l is a speed limit.
func readSSH(args []string, valued string, withDest bool) (o sshOpts, rest []string) {
	option := func(letter byte, v string) {
		switch letter {
		case 'J':
			o.jumps = append(o.jumps, v)
		case 'l':
			if withDest {
				o.user = v
			}
		case 'o':
			k, val := sshOption(v)
			switch strings.ToLower(k) {
			case "user":
				o.user = val
			case "hostname":
				o.hostName = val
			case "proxyjump":
				o.jumps = append(o.jumps, val)
			case "proxycommand":
				o.proxies = append(o.proxies, val)
			}
		}
	}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			if withDest && i+1 < len(args) {
				o.dest, rest = args[i+1], args[i+2:]
			}
			return o, rest
		}
		if len(a) < 2 || a[0] != '-' {
			if withDest {
				o.dest, rest = a, args[i+1:]
				return o, rest
			}
			continue
		}
		for j := 1; j < len(a); j++ { // a cluster: -vo User=root, -oUser=root
			if !strings.ContainsRune(valued, rune(a[j])) {
				continue
			}
			v := a[j+1:]
			if v == "" && i+1 < len(args) {
				i++
				v = args[i]
			}
			option(a[j], v)
			break
		}
	}
	return o, nil
}

// sshOption splits an -o value as ssh does: Key=value, Key value, Key = value.
func sshOption(s string) (key, value string) {
	s = strings.TrimSpace(s)
	i := strings.IndexAny(s, "= \t")
	if i < 0 {
		return s, ""
	}
	return s[:i], strings.Trim(s[i:], "= \t")
}

// sshDest splits an ssh destination ([user@]host, ssh://[user@]host[:port]) into its
// user and its host.
func sshDest(dest string) (user, host string) {
	if d, ok := strings.CutPrefix(dest, "ssh://"); ok {
		d, _, _ = strings.Cut(d, "/")
		if at := strings.LastIndexByte(d, '@'); at >= 0 {
			user, d = d[:at], d[at+1:]
		}
		if strings.HasPrefix(d, "[") { // ssh://[::1]:22
			if end := strings.IndexByte(d, ']'); end > 0 {
				return user, d[1:end]
			}
		}
		host, _, _ = strings.Cut(d, ":")
		return user, host
	}
	if at := strings.LastIndexByte(dest, '@'); at >= 0 {
		return dest[:at], dest[at+1:]
	}
	return "", dest
}

// sshAround checks what an ssh connection goes through before the host: the hosts it
// jumps through, a host given as HostName (scp and rsync -e have no destination of
// their own to put it in), and a ProxyCommand, which runs here.
func (p Policy) sshAround(name string, o sshOpts, env map[string]string, depth int) Decision {
	d := allow
	for _, list := range o.jumps {
		for _, h := range strings.Split(list, ",") {
			if _, h = sshDest(strings.TrimSpace(h)); h != "" {
				h, _, _ = strings.Cut(h, ":") // host:port
				d = worse(d, p.checkHost(name, h))
			}
		}
	}
	if o.hostName != "" {
		d = worse(d, p.checkHost(name, o.hostName))
	}
	for _, c := range o.proxies {
		if !strings.EqualFold(c, "none") {
			d = worse(d, p.inner(c, env, depth))
		}
	}
	return d
}

// vagrant ssh -c runs its command in the virtual machine: it is checked as ssh's is.
// What follows -- goes to ssh as it is, so its remote command is checked too.
func (p Policy) vagrant(args []string, env map[string]string, depth int) Decision {
	if len(args) == 0 || args[0] != "ssh" {
		return allow
	}
	here := p                    // where a ProxyCommand runs
	p.host = "(virtual machine)" // a root exception for this machine does not reach it
	d := allow
	for i := 1; i < len(args); i++ {
		switch a := args[i]; {
		case (a == "-c" || a == "--command") && i+1 < len(args):
			d = worse(d, p.inner(args[i+1], env, depth))
			i++
		case strings.HasPrefix(a, "--command="):
			d = worse(d, p.inner(strings.TrimPrefix(a, "--command="), env, depth))
		case a == "--":
			// vagrant puts the destination first, so the first word that is not an
			// option is where the remote command starts.
			o, rest := readSSH(args[i+1:], sshValued, true)
			if o.dest != "" {
				d = worse(d, p.inner(strings.Join(append([]string{o.dest}, rest...), " "), env, depth))
			}
			return worse(d, here.sshAround("vagrant", o, env, depth))
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
	i := firstOperand(args, oneOf("-e", "--env", "--env-file", "-u", "--user", "-w", "--workdir", "--detach-keys", "--index"))
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
func (p Policy) shell(name string, args []string, stdin string, env map[string]string, depth int) Decision {
	if noExec(args) {
		return allow // sh -n x.sh: reads the script for its syntax, runs nothing
	}
	wantsC := false
	d := allow
	for i := 0; i < len(args); i++ {
		a := args[i]
		if wantsC && !strings.HasPrefix(a, "-") && !strings.HasPrefix(a, "+") {
			return worse(d, p.inner(a, env, depth))
		}
		if name == "fish" { // fish -C, --command and --init-command run their value
			if v, ok := strings.CutPrefix(a, "--command="); ok {
				return worse(d, p.inner(v, env, depth))
			}
			if v, ok := strings.CutPrefix(a, "--init-command="); ok {
				d = worse(d, p.inner(v, env, depth))
				continue
			}
			if (a == "-C" || a == "--init-command" || a == "--command") && i+1 < len(args) {
				i++
				if a == "--command" {
					return worse(d, p.inner(args[i], env, depth))
				}
				d = worse(d, p.inner(args[i], env, depth))
				continue
			}
		}
		switch {
		case a == "-o" || a == "+o" || a == "-O" || a == "+O" || a == "--rcfile" || a == "--init-file":
			i++ // the option's value: pipefail, extglob, a file
		case strings.HasPrefix(a, "-") && !strings.HasPrefix(a, "--") && strings.ContainsRune(a[1:], 'c'):
			wantsC = true
		case !strings.HasPrefix(a, "-") && !strings.HasPrefix(a, "+"):
			return worse(d, p.shellScript(name, a, stdin, env, depth)) // runs a script file
		}
	}
	return worse(d, p.fromStdin(name, stdin, env, depth))
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

// remoteCopy is scp and rsync: the hosts they copy to or from, and how they get there:
// scp's -J and -o, the command rsync runs as its remote shell (-e, --rsh, RSYNC_RSH).
func (p Policy) remoteCopy(c call) Decision {
	if c.name == "scp" {
		o, _ := readSSH(c.args, scpValued, false)
		return worse(p.sshAround("scp", o, c.env, c.depth), p.copyHosts(c.name, c.args))
	}
	d := worse(p.copyHosts(c.name, c.args), p.syncDeletes(c.args))
	rsh := c.env["RSYNC_RSH"]
	for i := 0; i < len(c.args); i++ {
		a := c.args[i]
		switch {
		case a == "--":
			i = len(c.args)
		case a == "--rsh" && i+1 < len(c.args):
			rsh = c.args[i+1]
			i++
		case strings.HasPrefix(a, "--rsh="):
			rsh = strings.TrimPrefix(a, "--rsh=")
		case len(a) > 1 && a[0] == '-' && a[1] != '-':
			for j := 1; j < len(a); j++ { // a cluster: -ave 'ssh …'
				if strings.IndexByte("BfMT", a[j]) >= 0 {
					break // an option whose value is the rest
				}
				if a[j] == 'e' {
					if rsh = a[j+1:]; rsh == "" && i+1 < len(c.args) {
						i++
						rsh = c.args[i]
					}
					break
				}
			}
		}
	}
	if rsh == "" {
		return d
	}
	// rsync runs the remote shell here as "<rsh> [user@]host rsync --server …". The host
	// goes without its user: root@ in scp and rsync is not this rule's (it never was).
	host := ""
	if h, ok := remoteOperand(c.args); ok {
		_, h = sshDest(h)
		host = " " + h
	}
	return worse(d, p.inner(rsh+host, c.env, c.depth))
}

// remoteOperand is the [user@]host of the first host:path argument of scp or rsync.
func remoteOperand(args []string) (string, bool) {
	for _, a := range args {
		if strings.HasPrefix(a, "-") || strings.HasPrefix(a, "/") || strings.HasPrefix(a, ".") {
			continue
		}
		if h, _, ok := strings.Cut(a, ":"); ok && h != "" && !strings.Contains(h, "/") {
			return h, true
		}
	}
	return "", false
}

// sshfs mounts a folder of another machine over ssh: the host, its user and the ssh
// options it passes on in -o (comma-separated: ProxyJump, HostName, User, ssh_command).
func (p Policy) sshfs(c call) Decision {
	var o sshOpts
	dest := ""
	for i := 0; i < len(c.args); i++ {
		a := c.args[i]
		switch {
		case a == "-o" && i+1 < len(c.args), strings.HasPrefix(a, "-o") && len(a) > 2:
			v := strings.TrimPrefix(a, "-o")
			if v == "" {
				i++
				v = c.args[i]
			}
			for _, part := range strings.Split(v, ",") {
				k, val := sshOption(part)
				switch strings.ToLower(k) {
				case "user":
					o.user = val
				case "hostname":
					o.hostName = val
				case "proxyjump":
					o.jumps = append(o.jumps, val)
				case "proxycommand", "ssh_command":
					o.proxies = append(o.proxies, val)
				}
			}
		case a == "-p" && i+1 < len(c.args):
			i++
		case !strings.HasPrefix(a, "-") && dest == "" && strings.Contains(a, ":"):
			dest, _, _ = strings.Cut(a, ":")
		}
	}
	around := p.sshAround("sshfs", o, c.env, c.depth)
	user, host := sshDest(dest)
	if o.hostName != "" {
		host = o.hostName
	}
	if user == "" {
		user = o.user
	}
	if user == "root" && host != "" {
		q := p
		q.host = host
		if d := q.root(i18n.T(p.Lang, "guard.root.ssh", host, p.rootReason())); d.Verdict == Deny {
			return d
		} else {
			around = worse(around, d)
		}
	}
	return worse(p.checkHost("sshfs", host), around)
}

// autossh keeps an ssh connection open: its own options (-M port, -f, -V) go, and the
// rest is ssh's.
func (p Policy) autossh(c call) Decision {
	var args []string
	for i := 0; i < len(c.args); i++ {
		switch a := c.args[i]; {
		case a == "-M" && i+1 < len(c.args):
			i++
		case strings.HasPrefix(a, "-M"), a == "-f", a == "-V":
		default:
			args = append(args, c.args[i:]...)
			i = len(c.args)
		}
	}
	return p.ssh(args, c.stdin, c.env, c.depth)
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

// shells are the POSIX shells the guard reads: their -c string, their stdin, a script.
var shells = []string{"sh", "bash", "zsh", "dash", "ksh", "mksh", "fish", "tcsh", "csh"}

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
	general := true // a secret in the text: a general rule, so not outside the spaces
	if p.Outside {
		d, err := p.resolve(path)
		general = err != nil || p.inSpace(d)
	}
	if d := Write(p.Lang, text); general && d.Verdict == Deny {
		return d
	}
	return p.protected(path)
}

// protected asks before the assistant changes a file in Protected or in ProtectedDirs,
// or writes into the folder that holds one: a command given a folder (cp -t, tar -C,
// rsync, cp x <folder>/) may put any file there.
func (p Policy) protected(path string) Decision {
	if path == "" || len(p.Protected) == 0 && len(p.ProtectedDirs) == 0 {
		return allow
	}
	d, err := p.resolve(path)
	if err != nil {
		return allow
	}
	hit := false
	for _, f := range p.Protected {
		hit = hit || paths.Same(d, f) || paths.Same(d, filepath.Dir(f))
	}
	for _, f := range p.ProtectedDirs {
		hit = hit || paths.Within(d, f) || paths.Same(d, filepath.Dir(f))
	}
	if !hit {
		return allow
	}
	// Built only now: Tilde reads the disk, slow on Windows for a line of many writes.
	return Decision{Ask, i18n.T(p.Lang, "guard.protected", paths.Tilde(d))}
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
		p.touch()
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
	if p.outside && (r.err != nil || p.inSpace(r.path)) {
		p.touch()
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
// --privileged, the host's processes or namespaces, a capability that reaches the host,
// the host's /, /etc or the container engine's socket mounted inside it, or one of its
// disks as a device. Empty when it does not.
func hostAccess(args []string) string {
	if len(args) > 1 && args[0] == "container" {
		args = args[1:]
	}
	if len(args) == 0 || args[0] != "run" && args[0] != "create" {
		return ""
	}
	for i, a := range args[1:] {
		if a == "--" {
			break
		}
		next := ""
		if i+2 < len(args) {
			next = args[i+2]
		}
		key, value, hasValue := strings.Cut(a, "=")
		if !hasValue {
			value = next
		}
		if v, ok := strings.CutPrefix(a, "-v"); ok && v != "" && !strings.HasPrefix(v, "-") {
			key, value = "-v", strings.TrimPrefix(v, "=") // -v/:/h, -v=/:/h
		}
		switch {
		case a == "--privileged" || a == "--privileged=true":
			return "--privileged"
		case (key == "--pid" || key == "--userns" || key == "--ipc") && value == "host":
			return key + " host"
		case key == "--cap-add" && hostCapability(value):
			return "--cap-add " + value
		case key == "-v" || key == "--volume":
			if src, _, _ := strings.Cut(value, ":"); hostPath(src) {
				return "-v " + value
			}
		case key == "--mount":
			for _, part := range strings.Split(value, ",") {
				if k, v, _ := strings.Cut(part, "="); (k == "source" || k == "src") && hostPath(v) {
					return "--mount " + value
				}
			}
		case key == "--device":
			if dev, _, _ := strings.Cut(value, ":"); hostDisk(dev) {
				return "--device " + value
			}
		}
	}
	return ""
}

// hostCapability says whether a capability (ALL, SYS_ADMIN, CAP_SYS_ADMIN, sys_admin…)
// is one with which a container reaches the host: kernel modules, raw devices, any
// file of the host by its handle, or all of them. Several may come in one value.
func hostCapability(value string) bool {
	for _, c := range strings.Split(value, ",") {
		c = strings.ToUpper(strings.TrimSpace(c))
		c = strings.TrimPrefix(c, "CAP_")
		switch c {
		case "ALL", "SYS_ADMIN", "SYS_MODULE", "SYS_RAWIO", "DAC_READ_SEARCH":
			return true
		}
	}
	return false
}

// hostPath says whether mounting this folder of the host gives the container the
// machine: its /, a folder that holds what root runs or reads (/etc, /root, /boot,
// /proc, /sys, /dev), or the container engine's socket and the folders that hold it.
// The path is read as the engine reads it: //, /. and /x/.. are /.
func hostPath(src string) bool {
	if !strings.HasPrefix(src, "/") {
		return false // a named volume, or a relative path
	}
	src = path.Clean(src)
	switch src {
	case "/", "/etc", "/root", "/boot", "/proc", "/sys", "/dev", "/run", "/var/run":
		return true
	}
	switch path.Base(src) {
	case "docker.sock", "podman.sock", "containerd.sock", "crio.sock":
		return true
	}
	return false
}

// hostDisk says whether a device is one of the host's disks, or its memory: the
// container can then read and write the host's files below any permission.
func hostDisk(dev string) bool {
	dev = path.Clean(dev)
	if dev == "/dev/mem" || dev == "/dev/kmem" || dev == "/dev/port" {
		return true
	}
	for _, prefix := range []string{"/dev/sd", "/dev/hd", "/dev/vd", "/dev/xvd", "/dev/nvme",
		"/dev/mmcblk", "/dev/dm-", "/dev/mapper/", "/dev/disk/", "/dev/loop", "/dev/md"} {
		if strings.HasPrefix(dev, prefix) {
			return true
		}
	}
	return false
}
