package guard

import (
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/icalvete/augmented-brain/internal/i18n"
)

// The doors through which git runs a command, or takes settings, that the command line
// does not show as such: a setting given with -c, by environment variable or with git
// config; a variable that names a program; an option whose value is a command.

// gitSettingAsks says whether a git setting (its key, lower case) makes git run a
// command of its own, take an alias, or read settings or remotes from somewhere else:
// the user confirms it. runs says the value is itself a command, checked as one.
func gitSettingAsks(k string) (asks, runs bool) {
	sec, _, _ := strings.Cut(k, ".")
	last := k[strings.LastIndexByte(k, '.')+1:]
	switch {
	case slices.Contains([]string{"core.sshcommand", "core.pager", "core.editor", "sequence.editor",
		"diff.external", "core.gitproxy", "core.askpass", "gpg.program"}, k),
		sec == "alias",
		sec == "credential" && last == "helper",
		sec == "filter" && slices.Contains([]string{"clean", "smudge", "process"}, last),
		sec == "diff" && slices.Contains([]string{"command", "textconv"}, last),
		sec == "merge" && last == "driver",
		sec == "gpg" && last == "program":
		return true, true
	case slices.Contains([]string{"core.hookspath", "core.fsmonitor", "help.autocorrect", "include.path"}, k),
		sec == "includeif",
		sec == "protocol" && last == "allow",
		sec == "uploadpack",
		sec == "url" && (last == "insteadof" || last == "pushinsteadof"),
		sec == "remote" && slices.Contains([]string{"url", "pushurl", "uploadpack", "receivepack"}, last):
		return true, false
	}
	return false, false
}

// gitSetting is the answer to one setting given to git for this command: key=value as
// -c takes it (a key alone is true).
func (p Policy) gitSetting(setting string, env map[string]string, depth int) Decision {
	key, v, _ := strings.Cut(setting, "=")
	k := strings.ToLower(strings.TrimSpace(key))
	if k == "core.pager" && slices.Contains([]string{"", "cat", "less", "more"}, strings.TrimSpace(v)) {
		return allow // a pager that only shows the text: git --no-pager, said another way
	}
	asks, runs := gitSettingAsks(k)
	if !asks {
		return allow
	}
	d := Decision{Ask, i18n.T(p.Lang, "guard.git.config", setting)}
	if runs {
		if strings.HasPrefix(k, "alias.") && !strings.HasPrefix(v, "!") {
			return d // git's own words, checked when the alias is used
		}
		d = worse(d, p.inner(strings.TrimPrefix(v, "!"), env, depth))
	}
	return d
}

// gitEnvSettings are the settings given to git through its environment, as -c gives
// them: GIT_CONFIG_PARAMETERS ('key=value' 'key'='value', as git writes it) and
// GIT_CONFIG_COUNT with GIT_CONFIG_KEY_<n> and GIT_CONFIG_VALUE_<n>.
func gitEnvSettings(env map[string]string) []string {
	var out []string
	if s := env["GIT_CONFIG_PARAMETERS"]; s != "" {
		out = append(out, splitConfigParameters(s)...)
	}
	for k, key := range env {
		n, ok := strings.CutPrefix(k, "GIT_CONFIG_KEY_")
		if !ok {
			continue
		}
		out = append(out, key+"="+env["GIT_CONFIG_VALUE_"+n])
	}
	slices.Sort(out) // the map's order is random; the answer must not be
	return out
}

// splitConfigParameters reads GIT_CONFIG_PARAMETERS: settings separated by spaces, each
// in single quotes ('\” is a quote inside), the key and value quoted apart or together.
// What is not quoted is kept as it is, so nothing in it goes unread.
func splitConfigParameters(s string) []string {
	var out []string
	var cur strings.Builder
	in, has := false, false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '\'' && in:
			in = false
		case c == '\'':
			in, has = true, true
		case c == '\\' && !in && i+1 < len(s):
			i++
			cur.WriteByte(s[i])
			has = true
		case !in && (c == ' ' || c == '\t' || c == '\n'):
			if has {
				out = append(out, cur.String())
			}
			cur.Reset()
			has = false
		default:
			cur.WriteByte(c)
			has = true
		}
	}
	if has {
		out = append(out, cur.String())
	}
	return out
}

// gitEnv is the answer to git's variables in the command: those that name a program are
// checked as a command; those that give git a config file the guard does not read (and
// where an alias may hide) are confirmed by the user.
func (p Policy) gitEnv(env map[string]string, depth int) Decision {
	d := allow
	for _, k := range []string{"GIT_SSH_COMMAND", "GIT_SSH", "GIT_EXTERNAL_DIFF", "GIT_PROXY_COMMAND",
		"GIT_ASKPASS", "GIT_EDITOR", "GIT_SEQUENCE_EDITOR", "GIT_PAGER"} {
		if v := env[k]; v != "" {
			d = worse(d, p.inner(v, env, depth))
		}
	}
	for _, k := range []string{"GIT_CONFIG_GLOBAL", "GIT_CONFIG_SYSTEM", "GIT_CONFIG"} {
		if v, ok := env[k]; ok && v != "" && v != os.DevNull && v != "/dev/null" {
			d = worse(d, Decision{Ask, i18n.T(p.Lang, "guard.git.configfile", k+"="+v)})
		}
	}
	return d
}

// gitRunsOption are the options of a git command whose value is a command git runs.
var gitRunsOption = map[string][]string{
	"fetch":     {"--upload-pack"},
	"pull":      {"--upload-pack"},
	"clone":     {"--upload-pack", "-u"},
	"ls-remote": {"--upload-pack", "-u"},
	"push":      {"--receive-pack", "--exec"},
	"archive":   {"--exec"},
	"difftool":  {"-x", "--extcmd"},
}

// gitCommand is the answer to what a git command runs through its options, and to git
// config setting a key that -c would ask about.
func (p Policy) gitCommand(sub string, args []string, env map[string]string, depth int) Decision {
	d := allow
	opts := gitRunsOption[sub]
	for i := 0; i < len(args); i++ {
		a, run := args[i], ""
		for _, o := range opts {
			if a == o && i+1 < len(args) {
				i++
				run = args[i]
			} else if strings.HasPrefix(o, "--") && strings.HasPrefix(a, o+"=") {
				run = strings.TrimPrefix(a, o+"=")
			} else if len(o) == 2 && !strings.HasPrefix(a, "--") && strings.HasPrefix(a, o) && len(a) > 2 {
				run = a[2:]
			}
			if run != "" {
				break
			}
		}
		if run != "" {
			d = worse(d, p.inner(run, env, depth))
		}
	}
	if sub == "config" {
		d = worse(d, p.gitConfigWrite(args, env, depth))
	}
	return d
}

// gitConfigWrite is git config writing a setting that would ask with -c: the same
// setting, kept for every later command. Reading one (--get, a key alone) runs nothing.
func (p Policy) gitConfigWrite(args []string, env map[string]string, depth int) Decision {
	var ops []string
	for _, a := range args {
		if !strings.HasPrefix(a, "-") {
			ops = append(ops, a)
		}
	}
	if len(ops) > 0 && slices.Contains([]string{"set"}, ops[0]) {
		ops = ops[1:]
	}
	if len(ops) < 2 {
		return allow
	}
	if strings.HasPrefix(strings.ToLower(ops[0]), "alias.") {
		// kept where gitAlias reads it, and checked when used; a shell command is checked now
		if cmd, ok := strings.CutPrefix(ops[1], "!"); ok {
			return p.inner(cmd, env, depth)
		}
		return allow
	}
	return p.gitSetting(ops[0]+"="+ops[1], env, depth)
}

// gitCommits are git commands that make commits without saying commit, with the
// options that make them not do it.
var gitCommits = map[string][]string{
	"merge": {"--ff-only", "--abort", "--quit"}, "pull": {"--ff-only"},
	"cherry-pick": {"--abort", "--quit", "-n", "--no-commit"},
	"revert":      {"--abort", "--quit", "-n", "--no-commit"},
	"am":          {"--abort", "--quit", "--show-current-patch"}, "commit-tree": nil,
}

// commitsElsewhere says whether git sub with args makes a commit.
func commitsElsewhere(sub string, args []string) bool {
	not, ok := gitCommits[sub]
	if !ok || onlyAsks(args, "--help", "-h") {
		return false
	}
	return !slices.ContainsFunc(args, func(a string) bool { return slices.Contains(not, a) })
}

// pushDestination is the repository a push names, when it names one: the first word
// after the options (or the value of --repo). Empty when it pushes where it is set to.
func pushDestination(args []string) string {
	withValue := []string{"--repo", "-o", "--push-option", "--receive-pack", "--exec"}
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--":
			if i+1 < len(args) {
				return args[i+1]
			}
			return ""
		case a == "--repo" && i+1 < len(args):
			return args[i+1]
		case strings.HasPrefix(a, "--repo="):
			return strings.TrimPrefix(a, "--repo=")
		case slices.Contains(withValue, a):
			i++
		case strings.HasPrefix(a, "-"):
		default:
			return a
		}
	}
	return ""
}

// pushTo is the answer to where a push goes, from the repository at from (resolved): a
// remote of its own is what the repository lists say of from; the path of a repository
// on this machine is what they say of that one; any other address (a URL, a path the
// guard cannot read) the user confirms. named says the push names somewhere else.
func (p Policy) pushTo(from, dest string) (d Decision, named bool) {
	if dest == "" || slices.Contains(remoteNames(from), dest) {
		return allow, false
	}
	local := strings.TrimPrefix(dest, "file://")
	if !strings.Contains(local, "://") && (filepath.IsAbs(local) || !scpLike(local)) {
		path := local
		if !filepath.IsAbs(path) && !strings.HasPrefix(path, "~") {
			path = filepath.Join(from, path)
		}
		if r, err := p.resolve(path); err == nil {
			if info, err := os.Stat(r); err == nil && info.IsDir() {
				if repo := findRepo(r); repo != "" {
					r = repo
				}
				switch p.repoVerdict(r) {
				case Deny:
					return Decision{Deny, i18n.T(p.Lang, "guard.git.pushto", dest)}, true
				case Ask:
					return Decision{Ask, i18n.T(p.Lang, "guard.git.ask", "push", dest, p.facts(r))}, true
				}
				return allow, true
			}
		}
		if !strings.ContainsAny(local, `/\`) {
			return allow, false // a bare word: a remote set elsewhere, or git finds nothing
		}
	}
	return Decision{Ask, i18n.T(p.Lang, "guard.git.pushelsewhere", dest)}, true
}

// remoteNames are the names of the remotes of the repository at dir, read from its
// config without running git.
func remoteNames(dir string) []string {
	_, common, ok := gitDirs(dir)
	if !ok {
		common = dir // a bare repository holds its config itself
	}
	f, err := os.Open(filepath.Join(common, "config"))
	if err != nil {
		return nil
	}
	defer f.Close()
	var names []string
	readGitConfig(io.LimitReader(f, maxGitConfig), func(section, _, _ string) {
		sec, name, ok := strings.Cut(section, " ")
		if !ok {
			sec, name, ok = strings.Cut(section, ".") // the old form [remote.origin]
		}
		if ok && strings.EqualFold(sec, "remote") {
			name = strings.Trim(name, `"`)
			if !slices.Contains(names, name) {
				names = append(names, name)
			}
		}
	})
	return names
}
