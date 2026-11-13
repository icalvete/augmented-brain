package guard

import (
	"net/url"
	"path/filepath"
	"slices"
	"strings"

	"github.com/icalvete/augmented-brain/internal/paths"
)

// Outside every space, when augmented brain keeps to them (guard.toml's reach =
// "spaces"), the guard keeps only two kinds of rules:
//
//   - what protects its own files: the configuration, the seal, the panel's key, the
//     program and the plugin, and the home folder that holds the notes;
//   - what the user named in the lists: production profiles, untouchable hosts, and
//     the repositories of the three repository lists (or a clone of one, found by its
//     remote address).
//
// It also keeps asking when it cannot tell what a command runs (a program in a
// variable, a script it cannot read, too deep), since that could be either of the two.
// Every other rule (root, secrets, PowerShell, committing, pushing, merging and
// publishing in repositories nobody listed) is a general rule and lets the command
// through there. A new rule says which kind it is: a general one returns early when
// p.outside is set.
//
// A line is first read that way. If it reaches into a space (a cd there, git -C, a
// file it writes or a script it runs inside one) or goes somewhere the guard cannot
// follow, the whole line is read again with every rule: see Command.

// inSpace reports whether the resolved path is inside one of SpaceDirs.
func (p Policy) inSpace(path string) bool {
	return slices.ContainsFunc(p.SpaceDirs, func(d string) bool { return paths.Within(path, d) })
}

// touch notes, while reading a line outside, that it reaches into a space (or
// somewhere the guard cannot tell): the line is then read again with every rule.
func (p Policy) touch() {
	if p.outside && p.files != nil {
		p.files.inSpace = true
	}
}

// listedRepo reports whether the repository at dir (resolved) is one the lists name:
// in any of the three lists, or a clone of one (a remote address in common).
func (p Policy) listedRepo(dir string) bool {
	if repoListed(p.UntouchableRepos, dir) || repoListed(p.GitAskRepos, dir) || repoListed(p.GitAllowedRepos, dir) {
		return true
	}
	repo := findRepo(dir)
	if repo == "" {
		return false
	}
	return slices.ContainsFunc(remoteURLs(repo), func(r string) bool { return p.listedRemote(r, "") })
}

// listedRemote reports whether a repository written as an address or owner/name is
// the remote of a repository of the lists. host is the default host for owner/name.
func (p Policy) listedRemote(named, host string) bool {
	return slices.ContainsFunc(p.listedRemotes(), func(r string) bool {
		return sameRepo(named, r, host) || sameRepo(r, named, host)
	})
}

// listedRemotes are the remote addresses of every repository of the lists, read once
// per line.
func (p Policy) listedRemotes() []string {
	if p.files != nil && p.files.remotesRead {
		return p.files.remotes
	}
	var out []string
	for _, list := range [][]string{p.UntouchableRepos, p.GitAskRepos, p.GitAllowedRepos} {
		for _, entry := range list {
			dirs := []string{entry}
			if strings.Contains(filepath.Base(entry), "*") {
				dirs, _ = filepath.Glob(entry)
			}
			for _, d := range dirs {
				out = append(out, remoteURLs(d)...)
			}
		}
	}
	if p.files != nil {
		p.files.remotes, p.files.remotesRead = out, true
	}
	return out
}

// freeRepo reports whether, outside the spaces, the repository at dir is nobody's
// business: the guard knows the folder and no list names it.
func (p Policy) freeRepo(dir string) bool {
	if !p.outside || p.host != "" || dir == "" {
		return false
	}
	d, err := p.resolve(dir)
	if err != nil {
		return false
	}
	return !p.listedRepo(d)
}

// freePush reports whether a push from the free repository at dir to dest (as git
// push names it, "" for the default) stays clear of the lists: a remote of the
// repository, or an address or path no list names.
func (p Policy) freePush(dir, dest string) bool {
	if dest == "" || slices.Contains(remoteNames(dir), dest) {
		return true // its own remotes: listedRepo already compared their addresses
	}
	local := strings.TrimPrefix(dest, "file://")
	if !strings.Contains(local, "://") && !scpLike(local) {
		path := local
		if !filepath.IsAbs(path) && !strings.HasPrefix(path, "~") {
			path = filepath.Join(dir, path)
		}
		d, err := p.resolve(path)
		return err == nil && !p.listedRepo(d)
	}
	return !p.listedRemote(dest, "")
}

// freeGh reports whether, outside the spaces, a gh or glab command works on no
// repository of the lists: the folder it runs in is a free one, and every repository
// it names (-R, the pull request's address, GH_REPO, an API path) is no list's.
func (p Policy) freeGh(name string, env map[string]string, args []string, words []string) bool {
	if !p.outside || p.lost || p.Cwd == "" || !p.freeRepo(p.Cwd) {
		return false
	}
	for _, k := range []string{"GIT_DIR", "GIT_WORK_TREE"} {
		if env[k] != "" || p.Env[k] != "" {
			return false
		}
	}
	words2 := withoutText(name, args)
	if slices.ContainsFunc(words2, func(a string) bool { return strings.ContainsAny(a, "$`") || strings.ContainsRune(a, subst) }) {
		return false // a word only known when it runs may name a repository of the lists
	}
	named, ok := mergeTargets(name, env, p.Env, words2)
	if !ok {
		return false
	}
	if len(words) >= 2 && words[0] == "api" {
		repo, ok := apiRepo(words[1])
		if !ok {
			return false // an API path the guard cannot tie to a repository: as everywhere
		}
		if repo != "" {
			named = append(named, repo)
		}
	}
	host := defaultHost(name, env, p.Env)
	return !slices.ContainsFunc(named, func(n string) bool { return p.listedRemote(n, host) })
}

// apiRepo is the repository an API path names: repos/<owner>/<name>/… in GitHub's,
// projects/<group%2Fname>/… in GitLab's. A path that names none is ("", true); one
// whose repository cannot be told (a project by number, GraphQL) is ok false.
func apiRepo(path string) (repo string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	switch {
	case len(parts) >= 3 && strings.EqualFold(parts[0], "repos"):
		return parts[1] + "/" + parts[2], true
	case len(parts) >= 2 && strings.EqualFold(parts[0], "projects"):
		name, err := url.PathUnescape(parts[1])
		if err != nil || !strings.Contains(name, "/") {
			return "", false
		}
		return name, true
	case len(parts) >= 1 && (strings.EqualFold(parts[0], "graphql") || strings.EqualFold(parts[0], "repos") ||
		strings.EqualFold(parts[0], "projects")):
		return "", false
	}
	return "", true
}
