package remote

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/icalvete/augmented-brain/internal/paths"
	"github.com/icalvete/augmented-brain/internal/state"
)

// A session working on a task may need worktrees of its own in other repositories
// (task_worktree): it makes them through augmented brain, or notes one it made itself,
// so closing the task removes them. On 10-nov a session made two worktrees by hand,
// one with submodules, and closing its task left both behind.
//
// They are written down in state/task-worktrees/<task>, one per line: the folder, a tab,
// and the branch made for it ("" for a worktree noted, not made: its branch is not
// ours). A file of its own, apart from session-worktrees, which the two releases before
// this one read as a single path.

func taskWorktreeRecords(cfgDir string) string {
	return filepath.Join(cfgDir, state.Dir, "task-worktrees")
}

type noted struct{ folder, branch string }

func readNoted(cfgDir, task string) []noted {
	data, err := os.ReadFile(filepath.Join(taskWorktreeRecords(cfgDir), task))
	if err != nil {
		return nil
	}
	var res []noted
	for _, line := range strings.Split(string(data), "\n") {
		folder, branch, _ := strings.Cut(strings.TrimSpace(line), "\t")
		if folder != "" {
			res = append(res, noted{folder, branch})
		}
	}
	return res
}

func writeNoted(cfgDir, task string, list []noted) error {
	rec := filepath.Join(taskWorktreeRecords(cfgDir), task)
	if len(list) == 0 {
		err := os.Remove(rec)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	var b strings.Builder
	for _, n := range list {
		b.WriteString(n.folder + "\t" + n.branch + "\n")
	}
	if err := os.MkdirAll(filepath.Dir(rec), 0o700); err != nil {
		return err
	}
	return paths.WriteAtomic(rec, []byte(b.String()), 0o600)
}

func addNoted(cfgDir, task string, n noted) error {
	list := readNoted(cfgDir, task)
	for _, x := range list {
		if paths.Same(x.folder, n.folder) {
			return nil
		}
	}
	return writeNoted(cfgDir, task, append(list, n))
}

// ErrNotARepository: what task_worktree was given is not a git repository.
var ErrNotARepository = errors.New("not a git repository")

// ErrNotAWorktree: the folder to note is not a git worktree (a repository's own folder
// is never removed).
var ErrNotAWorktree = errors.New("not a git worktree of a repository")

// MakeTaskWorktree makes the task's worktree of repo (a repository with a .git folder,
// bare or not): <repo>-<task>, on a new branch named after the task from origin's
// default branch as it is now, and notes it for the task. Asked again, it returns the
// same one.
func MakeTaskWorktree(cfgDir, repo, task string) (folder, branch, base string, made, fetched bool, err error) {
	if fi, err := os.Stat(filepath.Join(repo, ".git")); err != nil || !fi.IsDir() {
		return "", "", "", false, false, fmt.Errorf("%s: %w", paths.Tilde(repo), ErrNotARepository)
	}
	folder, branch = TaskWorktreePath(repo, task)
	if _, statErr := os.Lstat(folder); statErr == nil {
		if r := WorktreeRoot(folder); r == "" || !paths.Same(r, repo) {
			return folder, branch, "", false, true, ErrNotOurWorktree
		}
		return folder, branch, "", false, true, addNoted(cfgDir, task, noted{folder, branch})
	}
	base, fetched, err = addWorktree(repo, folder, branch)
	if err != nil {
		return folder, branch, base, false, fetched, err
	}
	return folder, branch, base, true, fetched, addNoted(cfgDir, task, noted{folder, branch})
}

// NoteTaskWorktree notes a worktree made some other way, so closing the task removes it
// too (never its branch).
func NoteTaskWorktree(cfgDir, task, folder string) error {
	if WorktreeRoot(folder) == "" {
		return fmt.Errorf("%s: %w", paths.Tilde(folder), ErrNotAWorktree)
	}
	return addNoted(cfgDir, task, noted{folder, ""})
}

// RemoveNotedWorktrees removes the task's noted worktrees that hold nothing of their own
// (and the branch made for each), except those in skip (the folder a session still works
// in). It returns what it removed and what stays, with why.
func RemoveNotedWorktrees(cfgDir, task string, skip ...string) (removed []string, kept []Leftover) {
	var left []noted
	for _, n := range readNoted(cfgDir, task) {
		if slicesSame(skip, n.folder) {
			left = append(left, n)
			kept = append(kept, Leftover{task, n.folder, KeptInUse})
			continue
		}
		done, why := removeWorktree(n.folder, n.branch)
		switch {
		case done:
			removed = append(removed, n.folder)
		case why != "":
			left = append(left, n)
			kept = append(kept, Leftover{task, n.folder, why})
		}
	}
	_ = writeNoted(cfgDir, task, left)
	sort.Strings(removed)
	return removed, kept
}

// KeptInUse: a session still works in the worktree.
const KeptInUse = "in-use"

func slicesSame(list []string, folder string) bool {
	for _, s := range list {
		if s != "" && (paths.Same(s, folder) || paths.Within(s, folder)) {
			return true
		}
	}
	return false
}

// removeWorktree removes a worktree with branch (when not ""), if it holds no change and
// no commit that is not on the remote. done is false and why "" when it was gone
// already. --force only after those checks: git refuses a worktree with submodules
// without it.
func removeWorktree(folder, branch string) (done bool, why string) {
	repo := WorktreeRoot(folder)
	if repo == "" {
		return false, "" // removed by hand already
	}
	if out, err := GitRun(folder, "status", "--porcelain"); err != nil || strings.TrimSpace(out) != "" {
		return false, KeptChanges
	}
	if out, err := GitRun(folder, "rev-list", "HEAD", "--not", "--remotes"); err != nil || strings.TrimSpace(out) != "" {
		return false, KeptCommits
	}
	if _, err := GitRun(repo, "worktree", "remove", "--force", folder); err != nil {
		return false, KeptChanges
	}
	if branch != "" {
		_, _ = GitRun(repo, "branch", "-D", branch) // nothing of its own: checked above
	}
	return true, ""
}
