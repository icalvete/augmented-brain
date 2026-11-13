package guard

import (
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

// parseAlias reads the value of name in the [alias] section of a git config file; the
// last one wins, as in git.
func parseAlias(r io.Reader, name string) (string, bool) {
	value, found := "", false
	readGitConfig(r, func(section, key, v string) {
		if strings.EqualFold(section, "alias") && strings.EqualFold(key, name) {
			value, found = v, true
		}
	})
	return value, found
}
