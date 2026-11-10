package remote

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/icalvete/augmented-brain/internal/backup"
	"github.com/icalvete/augmented-brain/internal/paths"
	"github.com/icalvete/augmented-brain/internal/state"
)

// A task whose repository is bare gets a worktree of its own: <repo>-<task>, next to
// the repository, on a new branch named after the task, from the default branch of
// origin as it is now. Two tasks of one repository never share a folder (Claude serves
// each folder once, and they would edit each other's files), and a session never starts
// on old code. Decided on 9-nov, after a session opened on a leftover worktree.
//
// The worktrees session start makes are written down in the configuration folder
// (state/session-worktrees/<task>, holding the worktree's path): only those are ever
// removed, by session stop, by the sweep that session start and doctor run, and when
// the task is closed; and only when nothing would be lost: no change not committed, no
// commit that is not on the remote. One with work of its own stays, and doctor names it.

// GitRun runs git in a folder; tests may swap it.
var GitRun = backup.Git

// ErrNotOurWorktree: the folder for a task's worktree is taken by something else.
var ErrNotOurWorktree = errors.New("the folder for the task's worktree is taken")

// TaskWorktreePath is where a task's worktree of repo goes, and its branch.
func TaskWorktreePath(repo, task string) (folder, branch string) {
	slug := strings.TrimPrefix(task, "task-")
	return filepath.Join(filepath.Dir(repo), filepath.Base(repo)+"-"+slug), task
}

func worktreeRecords(cfgDir string) string {
	return filepath.Join(cfgDir, state.Dir, "session-worktrees")
}

// DefaultBranches are the names the repository's default branch may have: the one
// origin's HEAD points to, when git recorded it, else main and master.
func DefaultBranches(gitDir string) []string {
	const prefix = "ref: refs/remotes/origin/"
	if data, err := os.ReadFile(filepath.Join(gitDir, "refs", "remotes", "origin", "HEAD")); err == nil {
		if b, ok := strings.CutPrefix(strings.TrimSpace(string(data)), prefix); ok && b != "" {
			return []string{b}
		}
	}
	return []string{"main", "master"}
}

// TaskWorktree returns the task's worktree of the bare repository repo (a folder whose
// .git is bare), made now if it is not there yet. fetched is false when origin could
// not be reached: the worktree then starts from what this computer last knew of it.
func TaskWorktree(cfgDir, repo, task string) (folder, base string, made, fetched bool, err error) {
	folder, branch := TaskWorktreePath(repo, task)
	if _, statErr := os.Lstat(folder); statErr == nil {
		if r := WorktreeRoot(folder); r != "" && paths.Same(r, repo) {
			return folder, "", false, true, nil
		}
		return folder, "", false, true, ErrNotOurWorktree
	}
	base, fetched, err = addWorktree(repo, folder, branch)
	if err != nil {
		return folder, base, false, fetched, err
	}
	rec := filepath.Join(worktreeRecords(cfgDir), task)
	if err := os.MkdirAll(filepath.Dir(rec), 0o700); err != nil {
		return folder, base, true, fetched, err
	}
	return folder, base, true, fetched, paths.WriteAtomic(rec, []byte(folder+"\n"), 0o600)
}

// addWorktree makes folder, a worktree of repo on branch: the branch an earlier session
// of the task left, or a new one from origin's default branch as it is now (fetched
// first; fetched is false when origin could not be reached).
func addWorktree(repo, folder, branch string) (base string, fetched bool, err error) {
	_, fetchErr := GitRun(repo, "fetch", "--quiet", "origin")
	fetched = fetchErr == nil
	for _, d := range DefaultBranches(filepath.Join(repo, ".git")) {
		if _, err := GitRun(repo, "rev-parse", "--verify", "--quiet", "refs/remotes/origin/"+d); err == nil {
			base = "origin/" + d
			break
		}
	}
	if base == "" {
		return "", fetched, fmt.Errorf("%s has no default branch from origin (main or master) to start the worktree from", paths.Tilde(repo))
	}
	args := []string{"worktree", "add", "--quiet"}
	if _, err := GitRun(repo, "rev-parse", "--verify", "--quiet", "refs/heads/"+branch); err == nil {
		args = append(args, folder, branch) // a branch left by an earlier session of the task
	} else {
		// --no-track: the task's branch must not push to the default branch by default.
		args = append(args, "--no-track", "-b", branch, folder, base)
	}
	_, err = GitRun(repo, args...)
	return base, fetched, err
}

// Why a task's worktree stays.
const (
	KeptChanges = "changes" // changes not committed
	KeptCommits = "commits" // commits not on the remote
)

// RemoveTaskWorktree removes the worktree session start made for a task, with its
// branch, when it holds no change and no commit that is not on the remote. The caller
// makes sure no session runs in it. It says whether it removed it, and why not
// otherwise ("" when there was nothing of ours to remove).
func RemoveTaskWorktree(cfgDir, task string) (folder string, removed bool, why string) {
	rec := filepath.Join(worktreeRecords(cfgDir), task)
	data, err := os.ReadFile(rec)
	if err != nil {
		return "", false, ""
	}
	folder = strings.TrimSpace(string(data))
	repo := WorktreeRoot(folder)
	if repo == "" {
		_ = os.Remove(rec) // removed by hand already
		return "", false, ""
	}
	_, branch := TaskWorktreePath(repo, task)
	done, why := removeWorktree(folder, branch)
	if !done {
		return folder, false, why
	}
	_ = os.Remove(rec)
	return folder, true, ""
}

// Leftover is a task's worktree that stays because it holds work of its own.
type Leftover struct {
	Task, Folder, Why string
}

// SweepWorktrees goes through the worktrees session start made whose session no longer
// runs (a reboot ends launched sessions without session stop, and a session closed from
// Claude Desktop or a phone never calls it): it removes those with nothing of their own
// and returns the ones that stay. It does nothing when it cannot tell which sessions
// run.
func (s System) SweepWorktrees(cfgDir string) (removed []string, kept []Leftover) {
	entries, err := os.ReadDir(worktreeRecords(cfgDir))
	if err != nil || len(entries) == 0 {
		return nil, nil
	}
	running, ok := s.runningTasks(cfgDir)
	if !ok {
		return nil, nil
	}
	for _, e := range entries {
		task := e.Name()
		if running[task] {
			continue
		}
		folder, done, why := RemoveTaskWorktree(cfgDir, task)
		switch {
		case done:
			removed = append(removed, folder)
		case why != "":
			kept = append(kept, Leftover{task, folder, why})
		}
	}
	sort.Strings(removed)
	return removed, kept
}

// CloseTaskWorktree removes a closed task's worktree, unless its session still runs
// (often the one closing the task, working in that folder: the next sweep takes it).
func (s System) CloseTaskWorktree(cfgDir, task string) (folder string, removed bool, why string) {
	if running, ok := s.runningTasks(cfgDir); !ok || running[task] {
		return "", false, ""
	}
	return RemoveTaskWorktree(cfgDir, task)
}

// runningTasks are the tasks whose launched session is up; ok is false when that
// cannot be known.
func (s System) runningTasks(cfgDir string) (map[string]bool, bool) {
	launched, err := s.Launches(cfgDir)
	if err != nil {
		return nil, false
	}
	running := map[string]bool{}
	for _, x := range launched {
		if x.Active == "active" {
			running[x.Task] = true
		}
	}
	return running, true
}
