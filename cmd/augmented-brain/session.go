package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

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

func sessionStopFlags(errOut io.Writer) (*flag.FlagSet, *bool) {
	fs := flag.NewFlagSet("session stop", flag.ContinueOnError)
	fs.SetOutput(errOut)
	whenIdle := fs.Bool("when-idle", false, "wait until the session has finished its answer (what closing the task uses)")
	return fs, whenIdle
}

// cmdSession is `session start TASK | stop TASK | restart SESSION | list`: a remote session of its own
// for one task, so a session reached from another computer can open one per task. The
// logic is in internal/remote (launch.go).
func cmdSession(args []string, out, errOut io.Writer) int {
	l := userLanguage()
	if len(args) == 0 || (args[0] != "start" && args[0] != "stop" && args[0] != "restart" && args[0] != "list") {
		fmt.Fprintln(errOut, i18n.T(l, "usage.session"))
		return 2
	}
	// Only start needs the space: stop, restart and list work from any folder.
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
	case "restart":
		return sessionRestart(c, sys, args[1:], out, errOut)
	case "list":
		if len(args) != 1 {
			fmt.Fprintln(errOut, i18n.T(l, "usage.session"))
			return 2
		}
		return sessionList(c, sys, out, errOut)
	case "stop":
		fs, whenIdle := sessionStopFlags(errOut)
		var task string
		rest := args[1:]
		if len(rest) > 0 && !strings.HasPrefix(rest[0], "-") {
			task, rest = rest[0], rest[1:]
		}
		if err := fs.Parse(rest); err != nil {
			return 2
		}
		if task == "" && fs.NArg() == 1 {
			task = fs.Arg(0)
		} else if task == "" || fs.NArg() > 0 {
			fmt.Fprintln(errOut, i18n.T(l, "usage.session"))
			return 2
		}
		if *whenIdle {
			// What closing a task starts, detached: its output goes to a log in the
			// configuration's state folder (task_close sets that up), each line dated.
			err := sys.StopWhenIdle(c.Dir, task, 30*time.Minute)
			stamp := time.Now().Format("2006-01-02 15:04:05") + " "
			code, _ := sessionStopped(l, task, err, stamp, out, errOut)
			return code
		}
		if code, ok := sessionStopped(l, task, sys.Stop(c.Dir, task), "", out, errOut); !ok {
			return code
		}
		args = []string{"stop", task}
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
	dir, code := sessionFolder(c, s, sys, id, task.Header.Repos, *folder, out, errOut)
	if code != 0 {
		return code
	}
	name := set.NamePrefix + " · " + task.Header.Title
	shared := len(s.WorkFolders) > 0 && *folder == "" && len(task.Header.Repos) == 0
	launch := remote.Launch{Task: id, Name: name, Folder: dir, Shared: shared}
	// A session Claude stopped comes back with its conversation; only when it cannot,
	// a new one opens and replaces it.
	woke, err := sys.Wake(c.Dir, launch)
	if err == nil && woke {
		fmt.Fprintln(out, i18n.T(l, "session.woken", name, id, tilde(dir), id))
		return 0
	}
	if err == nil {
		err = sys.Start(c, set, launch, firstMessage(l, id, os.Getenv(messagingSocket)))
	}
	if err != nil {
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
		case errors.Is(err, remote.ErrLoggedOut):
			fmt.Fprintln(errOut, i18n.T(l, "session.loggedout", id))
			return 1
		}
		var failed *remote.FirstTurnFailed
		if errors.As(err, &failed) {
			key := "session.firstturn"
			if failed.Login {
				key = "session.firstturn.login"
			}
			fmt.Fprintln(errOut, i18n.T(l, key, id, failed.Cause, id))
			return 1
		}
		return sessionErr(l, err, errOut)
	}
	fmt.Fprintln(out, i18n.T(l, "session.started", name, id, tilde(dir), id))
	return 0
}

// sessionFolder is the folder a task's session opens in: the one --folder gives, or
// the first of the task's repositories in the space's work folders. A bare repository,
// named either way, has no files to work on: the task gets a worktree of its own, made
// now if it is not there (on 10-nov, --folder on a bare repository reused one of its
// worktrees, and the session opened on another branch's code). code is the exit code
// when there is no folder.
func sessionFolder(c *config.Config, s *config.Space, sys remote.System, id string, repos []string, folder string, out, errOut io.Writer) (string, int) {
	l := c.Language
	repo := ""
	if folder != "" {
		dir, err := paths.Resolve(folder)
		if err != nil {
			fmt.Fprintln(errOut, err)
			return "", 1
		}
		if !isGitRepo(dir) || !isBareRepo(dir) {
			return dir, 0
		}
		repo = dir
	} else if len(repos) == 0 {
		// A task with no repository works in the space's own folder: Claude lets a
		// background session share it with the space's remote service (measured on
		// 10-nov). Before, the user had to find a trusted folder of its own.
		if len(s.WorkFolders) == 0 {
			fmt.Fprintln(errOut, i18n.T(l, "session.nofolder.none", id))
			return "", 1
		}
		return s.WorkFolders[0], 0
	} else if repo = taskRepo(s, repos); repo == "" {
		fmt.Fprintln(errOut, i18n.T(l, "session.nofolder", id, strings.Join(repos, ", "), s.Name))
		return "", 1
	} else if !isBareRepo(repo) {
		return repo, 0
	}
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
		return "", 1
	case err != nil:
		fmt.Fprintln(errOut, err)
		return "", 1
	}
	if made {
		if !fetched {
			fmt.Fprintln(out, i18n.T(l, "session.worktree.offline", base))
		}
		_, branch := remote.TaskWorktreePath(repo, id)
		fmt.Fprintln(out, i18n.T(l, "session.worktree.made", tilde(wt), branch, base))
	}
	return wt, 0
}

// messagingSocket is where Claude Code tells a session's commands the address other
// sessions reach it at: the session asking for a task's session, when there is one.
const messagingSocket = "CLAUDE_CODE_MESSAGING_SOCKET"

// firstMessage is what a launched session starts with: its task, and, when another
// session asked for it, to tell that one when it is done.
func firstMessage(l, task, socket string) string {
	msg := i18n.T(l, "session.first", task)
	if socket = strings.TrimSpace(socket); socket != "" {
		msg += " " + i18n.T(l, "session.first.report", "uds:"+socket)
	}
	return msg
}

// sessionStopped says how a stop went, every line after prefix; ok is true when the
// session was closed (err nil, or Claude only kept showing it in its lists).
func sessionStopped(l, task string, err error, prefix string, out, errOut io.Writer) (code int, ok bool) {
	var left *remote.LeftInClaude
	switch {
	case err == nil:
		fmt.Fprintln(out, prefix+i18n.T(l, "session.stopped", task))
		return 0, true
	case errors.As(err, &left):
		fmt.Fprintln(out, prefix+i18n.T(l, "session.stopped", task))
		fmt.Fprintln(out, prefix+i18n.T(l, "session.left", task, left.ID))
		return 0, true
	case errors.Is(err, remote.ErrNoSession):
		fmt.Fprintln(errOut, prefix+i18n.T(l, "session.none", task))
	case errors.Is(err, remote.ErrStillBusy):
		fmt.Fprintln(errOut, prefix+i18n.T(l, "session.busy", task))
	default:
		fmt.Fprint(errOut, prefix)
		return sessionErr(l, err, errOut), false
	}
	return 1, false
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
		state := x.Active
		switch state {
		case remote.Retired:
			state = i18n.T(c.Language, "session.list.retired", task)
		case remote.Failed:
			key := "session.list.failed"
			if remote.LoginProblem(x.Problem) {
				key = "session.list.failed.login"
			}
			state = i18n.T(c.Language, key, x.Problem, task)
		}
		fmt.Fprintln(out, i18n.T(c.Language, "session.list.line", task, state, tilde(x.Folder)))
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
