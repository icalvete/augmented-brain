package guard

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// gitDirs reads where the repository or worktree at dir keeps its git data, without
// running git. own is its own git folder (where a worktree's HEAD is); common is the
// one it shares with its repository (where the config is), the same as own outside a
// worktree. ok is false when dir has no .git, or a .git file that does not say where
// (a link to a folder included: it is not read through).
func gitDirs(dir string) (own, common string, ok bool) {
	own = filepath.Join(dir, ".git")
	info, err := os.Lstat(own)
	if err != nil {
		return "", "", false
	}
	if info.IsDir() {
		return own, own, true
	}
	data, err := os.ReadFile(own)
	if err != nil {
		return "", "", false
	}
	g, found := strings.CutPrefix(strings.TrimSpace(string(data)), "gitdir:")
	if !found {
		return "", "", false
	}
	own = strings.TrimSpace(g)
	if !filepath.IsAbs(own) {
		own = filepath.Join(dir, own)
	}
	common = own
	if c, err := os.ReadFile(filepath.Join(own, "commondir")); err == nil {
		common = strings.TrimSpace(string(c))
		if !filepath.IsAbs(common) {
			common = filepath.Join(own, common)
		}
	}
	return own, common, true
}

// gitDirOf is the git folder of the repository or worktree at dir: for a worktree, its
// repository's, where the shared config is.
func gitDirOf(dir string) string {
	if _, common, ok := gitDirs(dir); ok {
		return common
	}
	return filepath.Join(dir, ".git")
}

// readGitConfig calls visit with the section, key and value of each setting of a git
// config file, in order: the section as written between the brackets ("alias",
// `remote "origin"`), the value without its quotes and comment. A small reader, not
// git's: what it does not understand is skipped, and includes are not followed.
func readGitConfig(r io.Reader, visit func(section, key, value string)) {
	section := ""
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
				section = ""
				continue
			}
			section = strings.TrimSpace(line[1:end])
			if line = strings.TrimSpace(line[end+1:]); line == "" {
				continue // [alias] ci = commit may be on one line
			}
		}
		if key, value, ok := strings.Cut(line, "="); ok {
			visit(section, strings.TrimSpace(key), configValue(value))
		}
	}
}
