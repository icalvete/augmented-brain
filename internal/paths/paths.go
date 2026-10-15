// Package paths turns any way of writing a folder into one canonical form, so the same
// folder always compares equal.
//
// It matters on macOS, where /var and /tmp are symlinks to /private/var and
// /private/tmp: a path that exists resolves through the link, and one that does not
// exist yet must resolve the same way, or the two never match.
package paths

import (
	"os"
	"path/filepath"
	"strings"
)

// Resolve expands a leading ~, makes the path absolute and resolves symlinks in the part
// of it that exists. The part that does not exist yet is kept as written.
func Resolve(p string) (string, error) {
	if p == "~" || strings.HasPrefix(p, "~/") || strings.HasPrefix(p, `~\`) {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		p = filepath.Join(home, p[1:])
	}
	p, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	// Walk up to the deepest part that exists, resolve it, and put the rest back.
	rest := ""
	for cur := p; ; {
		if real, err := filepath.EvalSymlinks(cur); err == nil {
			return filepath.Join(real, rest), nil
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return p, nil
		}
		rest = filepath.Join(filepath.Base(cur), rest)
		cur = parent
	}
}

// Within reports whether path is folder itself or inside it. Both must be resolved.
func Within(path, folder string) bool {
	rel, err := filepath.Rel(folder, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

// Tilde writes a path under the home folder as ~/…, for files people read.
func Tilde(p string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return p
	}
	if h, err := Resolve(home); err == nil {
		home = h
	}
	if r, err := Resolve(p); err == nil {
		p = r
	}
	if p == home {
		return "~"
	}
	if Within(p, home) {
		rel, _ := filepath.Rel(home, p)
		return "~/" + filepath.ToSlash(rel)
	}
	return p
}
