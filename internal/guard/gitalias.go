package guard

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/icalvete/augmented-brain/internal/paths"
)

// gitBuiltin are git's own commands the guard looks at: an alias with one of these
// names is never used by git, so it is not looked up.
var gitBuiltin = map[string]bool{"commit": true, "push": true, "submodule": true, "rebase": true,
	"bisect": true, "send-pack": true, "subtree": true, "svn": true, "status": true, "log": true,
	"diff": true, "add": true, "fetch": true, "pull": true, "checkout": true, "switch": true,
	"branch": true, "show": true, "merge": true, "reset": true, "stash": true, "tag": true, "clone": true,
	"init": true, "remote": true, "config": true, "worktree": true, "restore": true, "rm": true, "mv": true}

// gitAlias returns what git's alias name stands for, read from the repository's
// .git/config (the one at dir, or the folder the command runs in) and the user's
// ~/.gitconfig and ~/.config/git/config, without running git. Includes are not
// followed: an alias only there is not found, and the command is checked as written.
func (p Policy) gitAlias(dir string, elsewhere bool, name string) (string, bool) {
	var files []string
	if !elsewhere {
		start := p.Cwd
		if dir != "" {
			if d, err := p.resolve(dir); err == nil {
				start = d
			} else {
				start = ""
			}
		} else if p.lost {
			start = ""
		}
		if repo := findRepo(start); repo != "" {
			files = append(files, filepath.Join(gitDirOf(repo), "config"))
		}
	}
	if home, err := paths.Home(); err == nil {
		files = append(files, filepath.Join(home, ".gitconfig"), filepath.Join(home, ".config", "git", "config"))
	}
	for _, f := range files { // the repository's first: it wins over the user's
		fh, err := os.Open(f)
		if err != nil {
			continue
		}
		v, ok := parseAlias(io.LimitReader(fh, maxGitConfig), name)
		fh.Close()
		if ok {
			return v, true
		}
	}
	return "", false
}

// findRepo is the folder holding .git at or above dir, or "".
func findRepo(dir string) string {
	for dir != "" {
		if _, err := os.Lstat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
	return ""
}

// gitDirOf is the git folder of the repository or worktree at dir: for a worktree, its
// repository's, where the shared config is.
func gitDirOf(dir string) string {
	gitdir := filepath.Join(dir, ".git")
	info, err := os.Lstat(gitdir)
	if err != nil || info.IsDir() {
		return gitdir
	}
	data, err := os.ReadFile(gitdir)
	if err != nil {
		return gitdir
	}
	g, ok := strings.CutPrefix(strings.TrimSpace(string(data)), "gitdir:")
	if !ok {
		return gitdir
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
	return gitdir
}

// parseAlias reads the value of name in the [alias] section of a git config file; the
// last one wins, as in git. A small reader, like parseRemotes.
func parseAlias(r io.Reader, name string) (string, bool) {
	value, found, inAlias := "", false, false
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64<<10), maxGitConfig)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || line[0] == '#' || line[0] == ';' {
			continue
		}
		if line[0] == '[' {
			end := strings.IndexByte(line, ']')
			inAlias = end > 0 && strings.EqualFold(strings.TrimSpace(line[1:end]), "alias")
			if end < 0 || !inAlias {
				continue
			}
			line = strings.TrimSpace(line[end+1:]) // [alias] ci = commit, on one line
			if line == "" {
				continue
			}
		}
		if !inAlias {
			continue
		}
		key, v, ok := strings.Cut(line, "=")
		if ok && strings.EqualFold(strings.TrimSpace(key), name) {
			value, found = configValue(v), true
		}
	}
	return value, found
}
