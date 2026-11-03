// Package repostat says, of a git repository on this computer, what is not yet safe
// elsewhere: files changed and not committed, and commits not pushed. It only reads:
// git runs with no optional locks (so a repository in use is never blocked), never
// waits for a password, and never starts a filesystem monitor a repository's own
// configuration may name.
package repostat

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Status is what one repository holds that is not yet elsewhere.
type Status struct {
	Path     string
	Branch   string // empty when detached
	Changed  int    // files changed, staged or not, and untracked files
	Ahead    int    // commits on the branch that its upstream does not have
	Upstream bool   // whether the branch follows a remote one
	Remote   bool   // whether the repository has any remote to push to
	NotRepo  bool   // the folder is not a git repository: nothing to say about it
	Err      string // git could not read it: it took too long, or failed
}

// Pending says whether there is anything to do: changes, or commits that are only here.
func (s Status) Pending() bool {
	return !s.NotRepo && s.Err == "" && (s.Changed > 0 || s.Ahead > 0)
}

// timeout is how long one repository may take; a slow disk or a huge repository is
// reported, not waited for.
const timeout = 4 * time.Second

// Read reads one repository.
func Read(path string) Status {
	st := Status{Path: path}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	git := func(args ...string) (string, error) {
		cmd := exec.CommandContext(ctx, "git", append([]string{"-c", "core.fsmonitor=false", "-C", path}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_OPTIONAL_LOCKS=0", "LC_ALL=C")
		out, err := cmd.Output()
		return string(out), err
	}
	// Only the top of a repository counts: a folder inside another one is not its own.
	top, err := git("rev-parse", "--show-toplevel")
	if ctx.Err() != nil {
		st.Err = "timeout"
		return st
	}
	if err != nil || !samePath(strings.TrimSpace(top), path) {
		st.NotRepo = true
		return st
	}
	remotes, _ := git("remote")
	st.Remote = strings.TrimSpace(remotes) != ""
	out, err := git("status", "--porcelain=v2", "--branch", "--untracked-files=normal")
	if ctx.Err() != nil {
		st.Err = "timeout"
		return st
	}
	if err != nil {
		st.Err = "failed"
		return st
	}
	parse(&st, out)
	return st
}

// samePath says whether git's top folder is the one asked for, links resolved.
func samePath(a, b string) bool {
	ra, err1 := filepath.EvalSymlinks(a)
	rb, err2 := filepath.EvalSymlinks(b)
	if err1 != nil || err2 != nil {
		return filepath.Clean(a) == filepath.Clean(b)
	}
	return filepath.Clean(ra) == filepath.Clean(rb)
}

func parse(st *Status, out string) {
	for _, l := range strings.Split(out, "\n") {
		switch {
		case strings.HasPrefix(l, "# branch.head "):
			if b := strings.TrimPrefix(l, "# branch.head "); b != "(detached)" {
				st.Branch = b
			}
		case strings.HasPrefix(l, "# branch.upstream "):
			st.Upstream = true
		case strings.HasPrefix(l, "# branch.ab "):
			f := strings.Fields(strings.TrimPrefix(l, "# branch.ab "))
			if len(f) > 0 {
				st.Ahead, _ = strconv.Atoi(strings.TrimPrefix(f[0], "+"))
			}
		case l == "" || strings.HasPrefix(l, "#"):
		default:
			st.Changed++ // "1 …", "2 …", "u …" (changed) and "? …" (untracked)
		}
	}
	if !st.Upstream && st.Branch != "" && st.Remote {
		// A branch that was never pushed to a remote the repository has: all its
		// commits are only here. Without an upstream git cannot count them; say one,
		// so it shows as pending. A repository with no remote has nowhere to push.
		st.Ahead = max(st.Ahead, 1)
	}
}

// ReadAll reads every repository at once, a few at a time, and returns them in the
// order given.
func ReadAll(paths []string) []Status {
	out := make([]Status, len(paths))
	sem := make(chan struct{}, 8)
	var wg sync.WaitGroup
	for i, p := range paths {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			out[i] = Read(p)
			<-sem
		}()
	}
	wg.Wait()
	return out
}
