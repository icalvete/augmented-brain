package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/icalvete/augmented-brain/internal/brain"
	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/paths"
	"github.com/icalvete/augmented-brain/internal/remote"
	"github.com/icalvete/augmented-brain/internal/repostat"
)

func sessionFlags(errOut io.Writer) (*flag.FlagSet, *string) {
	fs := flag.NewFlagSet("session start", flag.ContinueOnError)
	fs.SetOutput(errOut)
	folder := fs.String("folder", "", "the folder to open the session in")
	return fs, folder
}

// cmdSession is `session start TASK | stop TASK | list`: a remote session of its own
// for one task, so a session reached from another computer can open one per task. The
// logic is in internal/remote (launch.go).
func cmdSession(args []string, out, errOut io.Writer) int {
	l := userLanguage()
	if len(args) == 0 || (args[0] != "start" && args[0] != "stop" && args[0] != "list") {
		fmt.Fprintln(errOut, i18n.T(l, "usage.session"))
		return 2
	}
	c, s, ok := currentSpace(errOut)
	if !ok {
		return 1
	}
	l = c.Language
	sys, err := remoteSystem()
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	host, _ := sys.Hostname()
	switch args[0] {
	case "list":
		if len(args) != 1 {
			fmt.Fprintln(errOut, i18n.T(l, "usage.session"))
			return 2
		}
		return sessionList(c, sys, out, errOut)
	case "stop":
		if len(args) != 2 {
			fmt.Fprintln(errOut, i18n.T(l, "usage.session"))
			return 2
		}
		if err := sys.Stop(args[1]); err != nil {
			return sessionErr(l, err, errOut)
		}
		fmt.Fprintln(out, i18n.T(l, "session.stopped", args[1]))
		return 0
	}
	// The task may come before or after --folder.
	fs, folder := sessionFlags(errOut)
	var id string
	rest := args[1:]
	if len(rest) > 0 && !strings.HasPrefix(rest[0], "-") {
		id, rest = rest[0], rest[1:]
	}
	if err := fs.Parse(rest); err != nil {
		return 2
	}
	if id == "" && fs.NArg() == 1 {
		id = fs.Arg(0)
	} else if id == "" || fs.NArg() > 0 {
		fmt.Fprintln(errOut, i18n.T(l, "usage.session"))
		return 2
	}
	b, err := brain.Open(c, s)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	task, err := b.Task(id)
	if err != nil {
		fmt.Fprintln(errOut, i18n.T(l, "session.notask", id, s.Name))
		return 1
	}
	set, err := remote.LoadSettings(c, host)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	dir := *folder
	if dir != "" {
		if dir, err = paths.Resolve(dir); err != nil {
			fmt.Fprintln(errOut, err)
			return 1
		}
	} else if dir = taskFolder(s, task.Header.Repos, sys.FolderTrusted); dir == "" {
		if len(task.Header.Repos) == 0 {
			first := ""
			if len(s.WorkFolders) > 0 {
				first = tilde(s.WorkFolders[0])
			}
			fmt.Fprintln(errOut, i18n.T(l, "session.nofolder.none", id, first))
		} else {
			fmt.Fprintln(errOut, i18n.T(l, "session.nofolder", id, strings.Join(task.Header.Repos, ", "), s.Name))
		}
		return 1
	}
	name := set.NamePrefix + " · " + task.Header.Title
	if err := sys.Start(c, set, remote.Launch{Task: id, Name: name, Folder: dir}); err != nil {
		switch {
		case errors.Is(err, remote.ErrUntrusted):
			fmt.Fprintln(out, i18n.T(l, "session.untrusted", tilde(dir), host, host))
			fmt.Fprintln(out, i18n.T(l, "session.untrusted.cmd", dir))
			fmt.Fprintln(out, i18n.T(l, "session.untrusted.after"))
			return 1
		case errors.Is(err, remote.ErrServed):
			fmt.Fprintln(errOut, i18n.T(l, "session.served", tilde(dir)))
			return 1
		case errors.Is(err, remote.ErrRunning):
			fmt.Fprintln(errOut, i18n.T(l, "session.running", id, id))
			return 1
		}
		return sessionErr(l, err, errOut)
	}
	fmt.Fprintln(out, i18n.T(l, "session.started", name, id, tilde(dir), id))
	return 0
}

func sessionErr(l string, err error, errOut io.Writer) int {
	if errors.Is(err, remote.ErrUnsupported) {
		fmt.Fprintln(errOut, i18n.T(l, "remote.unsupported"))
	} else {
		fmt.Fprintln(errOut, err)
	}
	return 1
}

func sessionList(c *config.Config, sys remote.System, out, errOut io.Writer) int {
	launched, err := sys.Launches()
	if err != nil {
		return sessionErr(c.Language, err, errOut)
	}
	if len(launched) == 0 {
		fmt.Fprintln(out, i18n.T(c.Language, "session.list.none"))
		return 0
	}
	for _, x := range launched {
		task := x.Task
		if task == "" {
			task = x.Unit
		}
		fmt.Fprintln(out, i18n.T(c.Language, "session.list.line", task, x.Active, tilde(x.Folder)))
	}
	return 0
}

// taskFolder is the folder of the first of a task's repositories found inside the
// space's work folders (by name, a few levels down), or "". A bare repository is never
// the folder: it holds no files to work on, and Claude does not trust it (on 9-nov a
// session was sent to one). One of its worktrees is, when it has one inside the
// space's work folders: one Claude trusts first (trusted), so the session can start
// without asking (on 9-nov the repository only existed as a bare one with worktrees
// named <repo>-<branch>, and no folder was found).
func taskFolder(s *config.Space, repos []string, trusted func(string) bool) string {
	pick := func(repo string) string {
		if !isBareRepo(repo) {
			return repo
		}
		return bareWorktree(s, repo, trusted)
	}
	for _, name := range repos {
		for _, wf := range s.WorkFolders {
			if direct := filepath.Join(wf, name); isGitRepo(direct) {
				if dir := pick(direct); dir != "" {
					return dir
				}
			}
			found, _ := repostat.Find(wf, 3, 500, nil)
			for _, f := range found {
				if filepath.Base(f) == name {
					if dir := pick(f); dir != "" {
						return dir
					}
				}
			}
		}
	}
	return ""
}

// bareWorktree is a worktree of a bare repository kept in <repo>/.git, read from git's
// own records (.git/worktrees/<name>/gitdir, the path of the worktree's .git file),
// without running git. Only worktrees that still exist and lie inside a work folder of
// the space count, so the session loads this space; one Claude trusts comes first,
// then by path. "" when there is none.
func bareWorktree(s *config.Space, repo string, trusted func(string) bool) string {
	entries, err := os.ReadDir(filepath.Join(repo, ".git", "worktrees"))
	if err != nil {
		return ""
	}
	var all []string
	for _, e := range entries {
		data, err := os.ReadFile(filepath.Join(repo, ".git", "worktrees", e.Name(), "gitdir"))
		if err != nil {
			continue
		}
		wt := filepath.Dir(filepath.Clean(strings.TrimSpace(string(data))))
		if !isGitRepo(wt) {
			continue // removed by hand, not pruned yet
		}
		for _, wf := range s.WorkFolders {
			if paths.Within(wt, wf) {
				all = append(all, wt)
				break
			}
		}
	}
	sort.Strings(all)
	for _, wt := range all {
		if trusted != nil && trusted(wt) {
			return wt
		}
	}
	if len(all) > 0 {
		return all[0]
	}
	return ""
}

func isGitRepo(p string) bool {
	_, err := os.Lstat(filepath.Join(p, ".git"))
	return err == nil
}

// isBareRepo says whether a folder's .git is a bare repository (core.bare = true), as
// a folder that keeps a bare repository in .git next to stale files does. It reads the
// file, without running git.
func isBareRepo(p string) bool {
	data, err := os.ReadFile(filepath.Join(p, ".git", "config"))
	if err != nil {
		return false
	}
	core := false
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.ToLower(strings.TrimSpace(line))
		if strings.HasPrefix(line, "[") {
			core = strings.HasPrefix(line, "[core")
			continue
		}
		if k, v, ok := strings.Cut(line, "="); core && ok && strings.TrimSpace(k) == "bare" {
			return strings.TrimSpace(v) == "true"
		}
	}
	return false
}
