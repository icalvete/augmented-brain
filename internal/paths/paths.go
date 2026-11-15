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
	"runtime"
	"strings"
)

// Home is the user's home folder. HOME wins where it is set, on Windows too: Git Bash,
// which runs the assistant's hooks there, sets it, and so do the tests. Otherwise it is
// the system's (USERPROFILE on Windows).
func Home() (string, error) {
	if h := os.Getenv("HOME"); h != "" {
		return h, nil
	}
	return os.UserHomeDir()
}

// Resolve expands a leading ~, makes the path absolute and resolves symlinks in the part
// of it that exists. The part that does not exist yet is kept as written.
func Resolve(p string) (string, error) {
	if p == "~" || strings.HasPrefix(p, "~/") || strings.HasPrefix(p, `~\`) {
		home, err := Home()
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

// Resolver resolves many paths as Resolve does, remembering every folder it has
// resolved, so paths that share folders read the disk for them once: the guard checks
// dozens of files in the same few folders on every call. It is for one short run (a
// hook): a link made or changed after a folder was resolved is not seen. The zero value
// is ready to use; it is not safe for concurrent use.
//
// The answer is Resolve's, except on Windows, where Resolve also writes every part in
// the disk's own case and this one keeps it as given: compare with Same or Within,
// which ignore case there. Short names (RUNNER~1) it does expand, as Resolve does: a
// short and a long name of one folder compare equal nowhere.
type Resolver struct{ seen map[string]string }

// Resolve is Resolve, remembering what it resolves.
func (r *Resolver) Resolve(p string) (string, error) {
	if p == "~" || strings.HasPrefix(p, "~/") || strings.HasPrefix(p, `~\`) {
		home, err := Home()
		if err != nil {
			return "", err
		}
		p = filepath.Join(home, p[1:])
	}
	p, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	if r.seen == nil {
		r.seen = map[string]string{}
	}
	return r.resolve(p), nil
}

// resolve resolves the absolute, clean path p: its folder (remembered), then its last
// part, followed only when it is a link or, on Windows, may be a short name. A last part
// that is missing or cannot be read stays as written, as Resolve keeps the part that
// does not exist.
func (r *Resolver) resolve(p string) string {
	if done, ok := r.seen[p]; ok {
		return done
	}
	out := p
	if parent := filepath.Dir(p); parent == p {
		if root, err := Resolve(p); err == nil { // a disk's root
			out = root
		}
	} else {
		base := filepath.Base(p)
		out = filepath.Join(r.resolve(parent), base)
		short := runtime.GOOS == "windows" && strings.Contains(base, "~")
		if info, err := os.Lstat(out); err == nil && (short || info.Mode()&os.ModeSymlink != 0) {
			if real, err := filepath.EvalSymlinks(out); err == nil {
				out = real
			}
		}
	}
	r.seen[p] = out
	return out
}

// foldCase says whether folders compare ignoring case: on macOS and Windows, whose
// disks usually do (/Users/Ana/Notes and /users/ana/notes are one folder). Tests set it.
var foldCase = runtime.GOOS == "darwin" || runtime.GOOS == "windows"

// rel is filepath.Rel, ignoring case where the system does.
func rel(base, target string) (string, error) {
	if foldCase {
		base, target = strings.ToLower(base), strings.ToLower(target)
	}
	return filepath.Rel(base, target)
}

// Within reports whether path is folder itself or inside it. Both must be resolved.
// It ignores case on macOS and Windows.
func Within(path, folder string) bool {
	r, err := rel(folder, path)
	if err != nil {
		return false
	}
	return r == "." || (r != ".." && !strings.HasPrefix(r, ".."+string(filepath.Separator)))
}

// Same reports whether two resolved paths are the same folder. Like Within, it ignores
// case on macOS and Windows.
func Same(a, b string) bool {
	r, err := rel(a, b)
	return err == nil && r == "."
}

// Tilde writes a path under the home folder as ~/…, for files people read.
func Tilde(p string) string {
	home, err := Home()
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
		r, _ := rel(home, p)
		if !foldCase || len(r) > len(p) || strings.ToLower(p[len(p)-len(r):]) != r {
			return "~/" + filepath.ToSlash(r)
		}
		// Keep the case the path was written in: only its length counts here.
		return "~/" + filepath.ToSlash(p[len(p)-len(r):])
	}
	return p
}
