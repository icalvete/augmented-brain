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
	// Only start needs the space: stop and list work from any folder.
	var c *config.Config
	var s *config.Space
	ok := false
	if args[0] == "start" {
		c, s, ok = currentSpace(errOut)
	} else {
		c, ok = loadConfig(errOut)
	}
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
		if err := sys.Stop(c.Dir, args[1]); err != nil {
			return sessionErr(l, err, errOut)
		}
		fmt.Fprintln(out, i18n.T(l, "session.stopped", args[1]))
		switch wt, removed, why := remote.RemoveTaskWorktree(c.Dir, args[1]); {
		case removed:
			fmt.Fprintln(out, i18n.T(l, "session.worktree.removed", tilde(wt)))
		case why == remote.KeptChanges:
			fmt.Fprintln(out, i18n.T(l, "session.worktree.kept.changes", tilde(wt)))
		case why == remote.KeptCommits:
			fmt.Fprintln(out, i18n.T(l, "session.worktree.kept.commits", tilde(wt)))
		}
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
		// A bare repository named by hand: one of its worktrees.
		if isGitRepo(dir) && isBareRepo(dir) {
			if wt := bareWorktree(s, dir, sys.FolderTrusted); wt != "" {
				dir = wt
			}
		}
	} else if repo := taskRepo(s, task.Header.Repos); repo != "" && isBareRepo(repo) {
		// Leftovers of sessions that ended without session stop (a reboot, a session
		// closed from Claude Desktop) go first.
		removed, _ := sys.SweepWorktrees(c.Dir)
		for _, wt := range removed {
			fmt.Fprintln(out, i18n.T(l, "session.worktree.swept", tilde(wt)))
		}
		wt, base, made, fetched, err := remote.TaskWorktree(c.Dir, repo, id)
		switch {
		case errors.Is(err, remote.ErrNotOurWorktree):
			fmt.Fprintln(errOut, i18n.T(l, "session.worktree.taken", tilde(wt), tilde(repo)))
			return 1
		case err != nil:
			fmt.Fprintln(errOut, err)
			return 1
		}
		if made {
			if !fetched {
				fmt.Fprintln(out, i18n.T(l, "session.worktree.offline", base))
			}
			_, branch := remote.TaskWorktreePath(repo, id)
			fmt.Fprintln(out, i18n.T(l, "session.worktree.made", tilde(wt), branch, base))
		}
		dir = wt
	} else if dir = repo; dir == "" {
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
	if err := sys.Start(c, set, remote.Launch{Task: id, Name: name, Folder: dir}, i18n.T(l, "session.first", id)); err != nil {
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
	launched, err := sys.Launches(c.Dir)
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

// taskRepo is the first of a task's repositories found inside the space's work
// folders (by name, a few levels down), or "". A working copy wins over a bare
// repository of the same name; a bare one gets a worktree of its own for the task
// (taskWorktree).
func taskRepo(s *config.Space, repos []string) string {
	bare := ""
	for _, name := range repos {
		for _, wf := range s.WorkFolders {
			cands := []string{}
			if direct := filepath.Join(wf, name); isGitRepo(direct) {
				cands = append(cands, direct)
			}
			found, _ := repostat.Find(wf, 3, 500, nil)
			for _, f := range found {
				if filepath.Base(f) == name {
					cands = append(cands, f)
				}
			}
			for _, f := range cands {
				if !isBareRepo(f) {
					return f
				}
				if bare == "" {
					bare = f
				}
			}
		}
	}
	return bare
}

// bareWorktree, for --folder naming a bare repository, is a worktree of a bare
// repository kept in <repo>/.git, read from git's
// own records (.git/worktrees/<name>/gitdir, the path of the worktree's .git file),
// without running git. Only worktrees that still exist and lie inside a work folder of
// the space count, so the session loads this space. The one on the default branch
// comes first, then one named after it (<repo>-main, often left on a detached commit),
// then the rest; within each, one Claude trusts first, then by path. On 9-nov, with
// trust read through the main repository, every worktree was trusted and the session
// went to a leftover feature branch. "" when there is none.
func bareWorktree(s *config.Space, repo string, trusted func(string) bool) string {
	gitDir := filepath.Join(repo, ".git")
	entries, err := os.ReadDir(filepath.Join(gitDir, "worktrees"))
	if err != nil {
		return ""
	}
	def := remote.DefaultBranches(gitDir)
	type cand struct {
		path  string
		rank  int
		trust bool
	}
	var all []cand
	for _, e := range entries {
		rec := filepath.Join(gitDir, "worktrees", e.Name())
		data, err := os.ReadFile(filepath.Join(rec, "gitdir"))
		if err != nil {
			continue
		}
		wt := filepath.Dir(filepath.Clean(strings.TrimSpace(string(data))))
		if !isGitRepo(wt) {
			continue // removed by hand, not pruned yet
		}
		within := false
		for _, wf := range s.WorkFolders {
			if paths.Within(wt, wf) {
				within = true
				break
			}
		}
		if !within {
			continue
		}
		c := cand{path: wt, rank: 2, trust: trusted != nil && trusted(wt)}
		head, _ := os.ReadFile(filepath.Join(rec, "HEAD"))
		for _, d := range def {
			if strings.TrimSpace(string(head)) == "ref: refs/heads/"+d {
				c.rank = 0
				break
			}
			if strings.HasSuffix(filepath.Base(wt), "-"+d) {
				c.rank = 1
			}
		}
		all = append(all, c)
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].rank != all[j].rank {
			return all[i].rank < all[j].rank
		}
		if all[i].trust != all[j].trust {
			return all[i].trust
		}
		return all[i].path < all[j].path
	})
	if len(all) == 0 {
		return ""
	}
	return all[0].path
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
