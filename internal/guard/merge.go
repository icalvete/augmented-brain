package guard

import (
	"bufio"
	"cmp"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/paths"
)

// merge decides on gh pr merge, glab mr merge and glab mr accept as on a push: merging
// a pull request pushes to the branch it goes into. gh and glab have no -C: they work
// on the repository of the folder they run in. As a push counts only a repository
// written with git -C, a merge counts only one written in the command, with a cd
// before it (cd <repo> && gh pr merge 12), never the session's folder alone. That
// folder must be a repository in the lists, as written there (a worktree of one
// counts; a folder inside one does not). A repository named in the command (-R,
// --repo, GH_REPO, the address of the pull request) must be one of that repository's
// remotes, read from its .git/config. --admin, which skips the branch's protection,
// asks the user even where merging is allowed.
func (p Policy) merge(name, what string, env map[string]string, args []string) Decision {
	deny := p.publishDeny(name, what)
	if p.host != "" || p.lost || p.Cwd == "" {
		return deny // on another machine, or in a folder the guard cannot tell
	}
	if !p.cdNamed {
		if d, err := paths.Resolve(p.Cwd); err == nil && p.repoVerdict(d) != Deny {
			return Decision{Deny, i18n.T(p.Lang, "guard.merge.cd", name+" "+what, name+" "+what)}
		}
		return deny
	}
	for _, k := range []string{"GIT_DIR", "GIT_WORK_TREE"} {
		if env[k] != "" || p.Env[k] != "" {
			return deny // gh and glab find the repository through git, which would look there
		}
	}
	dir, err := paths.Resolve(p.Cwd)
	if err != nil {
		return deny
	}
	verdict := p.repoVerdict(dir)
	if verdict == Deny {
		return deny
	}
	// Only the command's own words count: text given to an option (a body, a subject, a
	// message) may say anything, a pull request's address or --admin included.
	args = withoutText(name, args)
	named, ok := mergeTargets(name, env, p.Env, args)
	if !ok {
		return Decision{Deny, i18n.T(p.Lang, "guard.merge.unread", name+" "+what)}
	}
	if len(named) > 0 {
		remotes := remoteURLs(dir)
		host := defaultHost(name, env, p.Env)
		for _, n := range named {
			if !slices.ContainsFunc(remotes, func(r string) bool { return sameRepo(n, r, host) }) {
				return Decision{Deny, i18n.T(p.Lang, "guard.merge.remote", name+" "+what, n, paths.Tilde(dir))}
			}
		}
	}
	if name == "gh" && slices.ContainsFunc(args, func(a string) bool { return a == "--admin" || a == "--admin=true" }) {
		return Decision{Ask, i18n.T(p.Lang, "guard.merge.admin", name+" "+what, paths.Tilde(dir))}
	}
	if verdict == Ask {
		return Decision{Ask, i18n.T(p.Lang, "guard.merge.ask", name+" "+what, paths.Tilde(dir))}
	}
	return allow
}

// mergeText are the options of gh pr merge and glab mr merge (and mr accept) whose
// value is free text or a file, which may hold an address that is not the pull
// request's, or words that look like options. gh's -m is --merge, with no value.
var mergeText = map[string][]string{
	"gh":   {"-b", "--body", "-t", "--subject", "-F", "--body-file", "-A", "--author-email", "--match-head-commit"},
	"glab": {"-m", "--message", "--squash-message", "--sha"},
}

// withoutText returns the arguments of a merge command without the options of
// mergeText and their values, written apart (--body x), with = (--body=x) or, for a
// short option, joined (-bx).
func withoutText(name string, args []string) []string {
	text := mergeText[name]
	var out []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if slices.Contains(text, a) {
			i++ // and its value
			continue
		}
		if slices.ContainsFunc(text, func(o string) bool {
			if strings.HasPrefix(o, "--") {
				return strings.HasPrefix(a, o+"=")
			}
			return strings.HasPrefix(a, o) && len(a) > len(o)
		}) {
			continue
		}
		out = append(out, a)
	}
	return out
}

// mergeTargets returns the repositories a merge command names: the values of -R and
// --repo, GH_REPO (gh) or GITLAB_REPO (glab) in the command or inherited, and the
// address of a pull request given as one. ok is false for an address it cannot read.
// args come without their free text (withoutText).
func mergeTargets(name string, env, inherited map[string]string, args []string) (named []string, ok bool) {
	vars := map[string]string{"gh": "GH_REPO", "glab": "GITLAB_REPO"}
	if v := cmp.Or(env[vars[name]], inherited[vars[name]]); v != "" {
		named = append(named, v)
	}
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-R" || a == "--repo":
			if i+1 < len(args) {
				named = append(named, args[i+1])
			}
			i++
		case strings.HasPrefix(a, "--repo="):
			named = append(named, strings.TrimPrefix(a, "--repo="))
		case strings.HasPrefix(a, "-R") && len(a) > 2:
			named = append(named, strings.TrimPrefix(strings.TrimPrefix(a, "-R"), "="))
		case strings.Contains(a, "://"):
			repo, found := pullRepo(name, a)
			if !found {
				return nil, false
			}
			named = append(named, repo)
		}
	}
	return named, true
}

// pullRepo is the repository of a pull request's address, as host/path:
// https://github.com/owner/name/pull/12 or https://gitlab.com/group/name/-/merge_requests/7.
func pullRepo(name, address string) (string, bool) {
	u, err := url.Parse(address)
	if err != nil || u.Host == "" {
		return "", false
	}
	marker := "/pull/"
	if name == "glab" {
		marker = "/-/merge_requests/"
	}
	repo, _, found := strings.Cut(u.Path, marker)
	if !found || strings.Trim(repo, "/") == "" {
		return "", false
	}
	return u.Hostname() + "/" + strings.Trim(repo, "/"), true
}

// defaultHost is the host gh or glab takes for a repository named without one: GH_HOST
// or GITLAB_HOST when set, github.com or gitlab.com otherwise.
func defaultHost(name string, env, inherited map[string]string) string {
	if name == "glab" {
		return cmp.Or(env["GITLAB_HOST"], inherited["GITLAB_HOST"], "gitlab.com")
	}
	return cmp.Or(env["GH_HOST"], inherited["GH_HOST"], "github.com")
}

// sameRepo reports whether named (OWNER/NAME, HOST/OWNER/NAME, a group path or an
// address) is the repository at the remote address remote. Named without a host, it is
// on host. GitHub and GitLab do not tell names apart by case.
func sameRepo(named, remote, host string) bool {
	rh, rp := splitRemote(remote)
	if rp == "" {
		return false
	}
	if strings.Contains(named, "://") || scpLike(named) {
		nh, np := splitRemote(named)
		return np != "" && strings.EqualFold(nh, rh) && strings.EqualFold(np, rp)
	}
	n := strings.Trim(strings.TrimSuffix(strings.Trim(named, "/"), ".git"), "/")
	return n != "" && rh != "" && (strings.EqualFold(n, rh+"/"+rp) || strings.EqualFold(rh, host) && strings.EqualFold(n, rp))
}

// scpLike reports whether a remote is written as git's [user@]host:path.
func scpLike(s string) bool {
	colon := strings.IndexByte(s, ':')
	return colon > 0 && !strings.Contains(s[:colon], "/")
}

// splitRemote returns the host (no user, no port) and the path (no .git, no slashes at
// either end) of a remote address; a local path has no host.
func splitRemote(remote string) (host, path string) {
	switch {
	case strings.Contains(remote, "://"):
		u, err := url.Parse(remote)
		if err != nil {
			return "", ""
		}
		host, path = u.Hostname(), u.Path
	case scpLike(remote):
		host, path, _ = strings.Cut(remote, ":")
		host = host[strings.LastIndexByte(host, '@')+1:]
	default:
		path = remote
	}
	path = strings.Trim(path, "/")
	path = strings.Trim(strings.TrimSuffix(path, ".git"), "/")
	return host, path
}

// maxGitConfig is the largest .git/config the guard reads.
const maxGitConfig = 1 << 20

// remoteURLs returns the url of each remote of the repository at dir, read from its
// .git/config (for a worktree, its repository's), without running git. Includes are
// not followed: a remote only there is not found, and the merge is denied.
func remoteURLs(dir string) []string {
	gitdir := filepath.Join(dir, ".git")
	info, err := os.Lstat(gitdir)
	if err != nil {
		return nil
	}
	if !info.IsDir() {
		data, err := os.ReadFile(gitdir)
		if err != nil {
			return nil
		}
		g, ok := strings.CutPrefix(strings.TrimSpace(string(data)), "gitdir:")
		if !ok {
			return nil
		}
		gitdir = strings.TrimSpace(g)
		if !filepath.IsAbs(gitdir) {
			gitdir = filepath.Join(dir, gitdir)
		}
		if common, err := os.ReadFile(filepath.Join(gitdir, "commondir")); err == nil {
			c := strings.TrimSpace(string(common))
			if !filepath.IsAbs(c) {
				c = filepath.Join(gitdir, c)
			}
			gitdir = c
		}
	}
	f, err := os.Open(filepath.Join(gitdir, "config"))
	if err != nil {
		return nil
	}
	defer f.Close()
	return parseRemotes(io.LimitReader(f, maxGitConfig))
}

// parseRemotes reads the url values of the [remote "<name>"] sections of a git config
// file. It is a small reader, not git's: what it does not understand is skipped.
func parseRemotes(r io.Reader) []string {
	var urls []string
	inRemote := false
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64<<10), maxGitConfig)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || line[0] == '#' || line[0] == ';' {
			continue
		}
		if line[0] == '[' {
			end := strings.IndexByte(line, ']')
			if end < 0 {
				inRemote = false
				continue
			}
			section := strings.TrimSpace(line[1:end])
			name, _, _ := strings.Cut(section, " ")
			name, _, _ = strings.Cut(name, ".") // the old form [remote.origin]
			inRemote = strings.EqualFold(name, "remote") && section != name
			continue
		}
		if !inRemote {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found || !strings.EqualFold(strings.TrimSpace(key), "url") {
			continue
		}
		if v := configValue(value); v != "" {
			urls = append(urls, v)
		}
	}
	return urls
}

// configValue is a git config value without its quotes and its comment.
func configValue(v string) string {
	var b strings.Builder
	quoted := false
	for i := 0; i < len(v); i++ {
		c := v[i]
		switch {
		case c == '"':
			quoted = !quoted
		case c == '\\' && i+1 < len(v):
			i++
			b.WriteByte(v[i])
		case !quoted && (c == '#' || c == ';'):
			return strings.TrimSpace(b.String())
		default:
			b.WriteByte(c)
		}
	}
	return strings.TrimSpace(b.String())
}
