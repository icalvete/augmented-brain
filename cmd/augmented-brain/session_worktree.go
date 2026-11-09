package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/icalvete/augmented-brain/internal/backup"
	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/paths"
	"github.com/icalvete/augmented-brain/internal/remote"
	"github.com/icalvete/augmented-brain/internal/state"
)

// A task whose repository is bare gets a worktree of its own: <repo>-<task>, next to
// the repository, on a new branch named after the task, from the default branch of
// origin as it is now. Two tasks of one repository never share a folder (Claude serves
// each folder once, and they would edit each other's files), and a session never starts
// on old code. Decided on 9-nov, after a session opened on a leftover worktree.
//
// The worktrees session start made are written down in the configuration folder
// (state/session-worktrees/<task>), so session stop removes only those, and only when
// nothing would be lost.

// gitRun runs git in a folder; tests swap it.
var gitRun = backup.Git

// sessionWorktree is where a task's worktree of repo goes, and its branch.
func sessionWorktree(repo, task string) (folder, branch string) {
	slug := strings.TrimPrefix(task, "task-")
	return filepath.Join(filepath.Dir(repo), filepath.Base(repo)+"-"+slug), task
}

// worktreeRecord is the file that says session start made the task's worktree.
func worktreeRecord(c *config.Config, task string) string {
	return filepath.Join(c.Dir, state.Dir, "session-worktrees", task)
}

// errNotOurWorktree: the folder for the task's worktree is taken by something else.
var errNotOurWorktree = errors.New("taken")

// taskWorktree returns the task's worktree of the bare repository repo, made now if it
// is not there yet. fetched is false when origin could not be reached: the worktree
// then starts from what this computer last knew of it.
func taskWorktree(c *config.Config, repo, task string) (folder, base string, made, fetched bool, err error) {
	folder, branch := sessionWorktree(repo, task)
	if _, statErr := os.Lstat(folder); statErr == nil {
		if r := remote.WorktreeRoot(folder); r != "" && paths.Same(r, repo) {
			return folder, "", false, true, nil
		}
		return folder, "", false, true, errNotOurWorktree
	}
	_, fetchErr := gitRun(repo, "fetch", "--quiet", "origin")
	fetched = fetchErr == nil
	for _, d := range defaultBranches(filepath.Join(repo, ".git")) {
		if _, err := gitRun(repo, "rev-parse", "--verify", "--quiet", "refs/remotes/origin/"+d); err == nil {
			base = "origin/" + d
			break
		}
	}
	if base == "" {
		return folder, "", false, fetched, fmt.Errorf("%s has no default branch from origin (main or master) to start the worktree from", paths.Tilde(repo))
	}
	args := []string{"worktree", "add", "--quiet"}
	if _, err := gitRun(repo, "rev-parse", "--verify", "--quiet", "refs/heads/"+branch); err == nil {
		args = append(args, folder, branch) // a branch left by an earlier session of the task
	} else {
		// --no-track: the new branch must not push to the default branch by default.
		args = append(args, "--no-track", "-b", branch, folder, base)
	}
	if _, err := gitRun(repo, args...); err != nil {
		return folder, base, false, fetched, err
	}
	rec := worktreeRecord(c, task)
	if err := os.MkdirAll(filepath.Dir(rec), 0o700); err != nil {
		return folder, base, true, fetched, err
	}
	return folder, base, true, fetched, paths.WriteAtomic(rec, []byte(folder+"\n"), 0o600)
}

// removeTaskWorktree removes the worktree session start made for a task, with its
// branch, when it holds no change and no commit that is not on the remote. It says
// whether it removed it, and why not otherwise ("" when there was nothing to remove).
func removeTaskWorktree(c *config.Config, task string) (folder string, removed bool, why string) {
	rec := worktreeRecord(c, task)
	data, err := os.ReadFile(rec)
	if err != nil {
		return "", false, ""
	}
	folder = strings.TrimSpace(string(data))
	repo := remote.WorktreeRoot(folder)
	if repo == "" {
		_ = os.Remove(rec) // removed by hand already
		return "", false, ""
	}
	if out, err := gitRun(folder, "status", "--porcelain"); err != nil || strings.TrimSpace(out) != "" {
		return folder, false, "changes"
	}
	if out, err := gitRun(folder, "rev-list", "HEAD", "--not", "--remotes"); err != nil || strings.TrimSpace(out) != "" {
		return folder, false, "commits"
	}
	if _, err := gitRun(repo, "worktree", "remove", folder); err != nil {
		return folder, false, "changes"
	}
	_, branch := sessionWorktree(repo, task)
	_, _ = gitRun(repo, "branch", "-D", branch) // no commit of its own: checked above
	_ = os.Remove(rec)
	return folder, true, ""
}
