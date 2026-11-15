package remote

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/icalvete/augmented-brain/internal/filelock"
	"github.com/icalvete/augmented-brain/internal/i18n"
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

// changeNoted changes a task's record under a lock of its own: task_worktree and
// closing a task run git without the notes lock (it may take minutes with the network
// down), so two of them may meet here.
func changeNoted(cfgDir, task string, change func([]noted) ([]noted, error)) error {
	unlock, err := filelock.Lock(taskWorktreeRecords(cfgDir) + ".lock")
	if err != nil {
		return err
	}
	defer unlock()
	list, err := change(readNoted(cfgDir, task))
	if err != nil {
		return err
	}
	return writeNoted(cfgDir, task, list)
}

// recordName is a task's id as the name of its record files: ids are file names of
// notes already, and this keeps one that is not from reaching another folder (R-16).
func recordName(task string) string { return unitSafe.ReplaceAllString(task, "-") }

func readNoted(cfgDir, task string) []noted {
	data, err := os.ReadFile(filepath.Join(taskWorktreeRecords(cfgDir), recordName(task)))
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
	rec := filepath.Join(taskWorktreeRecords(cfgDir), recordName(task))
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

// ErrOddPath: a folder whose path holds a tab or a line break, which the record could
// not tell apart from its next field (R-16).
var ErrOddPath = errors.New("the folder's path has a tab or a line break")

func addNoted(cfgDir, task string, n noted) error {
	if strings.ContainsAny(n.folder, "\t\r\n") {
		return fmt.Errorf("%s: %w", paths.Tilde(n.folder), ErrOddPath)
	}
	return changeNoted(cfgDir, task, func(list []noted) ([]noted, error) {
		for _, x := range list {
			if paths.Same(x.folder, n.folder) {
				return list, nil
			}
		}
		return append(list, n), nil
	})
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
	for _, n := range readNoted(cfgDir, task) {
		if slicesSame(skip, n.folder) {
			kept = append(kept, Leftover{task, n.folder, KeptInUse})
			continue
		}
		done, why := RemoveWorktree(n.folder, n.branch)
		switch {
		case done:
			removed = append(removed, n.folder)
		case why != "":
			kept = append(kept, Leftover{task, n.folder, why})
		}
	}
	// Git ran without the record's lock: what was noted meanwhile stays.
	_ = changeNoted(cfgDir, task, func(list []noted) ([]noted, error) {
		var left []noted
		for _, n := range list {
			gone := false
			for _, r := range removed {
				gone = gone || paths.Same(r, n.folder)
			}
			if !gone {
				left = append(left, n)
			}
		}
		return left, nil
	})
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

// WorktreeWork says what a worktree holds of its own: KeptChanges, KeptCommits,
// KeptIgnored, or "" when nothing would be lost by removing it. Commits count on HEAD
// and on branch (when not ""): a session may leave the worktree on another commit, and
// the branch is what RemoveWorktree deletes (R-02). Files not committed count whatever
// the user's git settings hide (status.showUntrackedFiles, an fsmonitor), and so do the
// files git ignores, but those in folders a build or an install makes again (N-06, R-04).
// What git cannot answer counts as work.
func WorktreeWork(folder, branch string) string {
	out, err := GitRun(folder, "-c", "core.fsmonitor=false", "status", "--porcelain",
		"--untracked-files=all", "--ignored=matching")
	if err != nil {
		return KeptChanges
	}
	ignored := false
	for _, line := range strings.Split(out, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		path, isIgnored := strings.CutPrefix(strings.TrimSpace(line), "!! ")
		if !isIgnored {
			return KeptChanges
		}
		if !regenerable(folder, path) {
			ignored = true
		}
	}
	if !branchPushed(folder, "HEAD") {
		return KeptCommits
	}
	if branch != "" {
		if _, err := GitRun(folder, "rev-parse", "--verify", "--quiet", "refs/heads/"+branch); err == nil && !branchPushed(folder, "refs/heads/"+branch) {
			return KeptCommits
		}
	}
	if ignored {
		return KeptIgnored
	}
	return ""
}

// branchPushed says whether every commit of ref is on a remote; false when git cannot
// tell.
func branchPushed(dir, ref string) bool {
	out, err := GitRun(dir, "rev-list", ref, "--not", "--remotes")
	return err == nil && strings.TrimSpace(out) == ""
}

// regenerableFolders are folders git ignores whose contents a build, an install or a
// tool's cache makes again: what is ignored inside them is not work of the user's.
var regenerableFolders = map[string]bool{
	"node_modules": true, "vendor": true, "target": true, "dist": true, "build": true,
	".venv": true, "venv": true, "__pycache__": true, ".pytest_cache": true, ".cache": true,
	"tmp": true, "log": true, "coverage": true, ".next": true, ".nuxt": true,
	".gradle": true, ".mypy_cache": true, ".ruff_cache": true, ".tox": true, ".turbo": true,
	".parcel-cache": true,
}

// regenerableExts are the endings of files a build makes: objects, libraries, bytecode,
// test binaries and logs.
var regenerableExts = map[string]bool{
	".o": true, ".a": true, ".so": true, ".dylib": true, ".dll": true, ".exe": true,
	".class": true, ".pyc": true, ".pyo": true, ".test": true, ".log": true,
}

// regenerable says whether an ignored path of the worktree folder, as git status prints
// it, is something a build makes again: it lies in one of regenerableFolders, ends in
// one of regenerableExts, or is a program a compiler wrote (an ELF, Mach-O or PE file,
// such as a Go build's binary at the repository's root).
func regenerable(folder, path string) bool {
	if p, err := strconv.Unquote(path); err == nil {
		path = p
	}
	isFolder := strings.HasSuffix(path, "/")
	parts := strings.Split(strings.TrimSuffix(path, "/"), "/")
	// The last part is a folder only when git printed it with its slash.
	if !isFolder {
		parts = parts[:len(parts)-1]
	}
	for _, p := range parts {
		if regenerableFolders[p] {
			return true
		}
	}
	if isFolder {
		return false
	}
	return regenerableExts[strings.ToLower(filepath.Ext(path))] || compiled(filepath.Join(folder, filepath.FromSlash(path)))
}

// compiled says whether file starts as a program a compiler writes: ELF, Mach-O (32 or
// 64 bits, either byte order, or universal) or PE.
func compiled(file string) bool {
	f, err := os.Open(file)
	if err != nil {
		return false
	}
	defer f.Close()
	var b [4]byte
	if n, _ := io.ReadFull(f, b[:]); n < 2 {
		return false
	}
	switch string(b[:]) {
	case "\x7fELF", "\xfe\xed\xfa\xce", "\xfe\xed\xfa\xcf", "\xce\xfa\xed\xfe", "\xcf\xfa\xed\xfe", "\xca\xfe\xba\xbe":
		return true
	}
	return string(b[:2]) == "MZ"
}

// IgnoredWork are the files git ignores in a worktree that no build makes again, as
// WorktreeWork counts them: what the user would lose with it. Sorted; nil when git
// cannot answer.
func IgnoredWork(folder string) []string {
	out, err := GitRun(folder, "-c", "core.fsmonitor=false", "status", "--porcelain",
		"--untracked-files=all", "--ignored=matching")
	if err != nil {
		return nil
	}
	var list []string
	for _, line := range strings.Split(out, "\n") {
		path, isIgnored := strings.CutPrefix(strings.TrimSpace(line), "!! ")
		if !isIgnored || regenerable(folder, path) {
			continue
		}
		if p, err := strconv.Unquote(path); err == nil {
			path = p
		}
		list = append(list, path)
	}
	sort.Strings(list)
	return list
}

// IgnoredNames are the first files of IgnoredWork, for a message: at most three, and
// how many more ("" when git cannot tell which).
func IgnoredNames(lang, folder string) string {
	list := IgnoredWork(folder)
	if len(list) <= 3 {
		return strings.Join(list, ", ")
	}
	return i18n.T(lang, "session.worktree.ignored.more", strings.Join(list[:3], ", "), len(list)-3)
}

// WhyKept is why a worktree stays, for a person to read: with KeptIgnored, it names the
// files.
func WhyKept(lang, folder, why string) string {
	if why == KeptIgnored {
		return i18n.T(lang, "session.worktree.why.ignored", IgnoredNames(lang, folder))
	}
	return i18n.T(lang, "session.worktree.why."+why)
}

// registered says whether folder is a worktree git knows of in repo: a .git file that
// points to a repository is not enough to remove a folder (N-06).
func registered(repo, folder string) bool {
	out, err := GitRun(repo, "worktree", "list", "--porcelain")
	if err != nil {
		return false
	}
	for _, line := range strings.Split(out, "\n") {
		if p, ok := strings.CutPrefix(line, "worktree "); ok && paths.Same(strings.TrimSpace(p), folder) {
			return true
		}
	}
	return false
}

// RemoveWorktree removes a worktree (a task's, or one Claude Code made) with branch
// (when not ""), if it holds nothing of its own (WorktreeWork). done is false and why ""
// when it was gone already, or is not a worktree git knows of: nothing of ours to
// remove. git removes it without --force first, which refuses a file written since the
// check; --force only when it still holds nothing of its own (git refuses a worktree
// with submodules without it). The branch goes only when every commit of it is on a
// remote, checked again after the worktree went.
func RemoveWorktree(folder, branch string) (done bool, why string) {
	repo := WorktreeRoot(folder)
	if repo == "" || !registered(repo, folder) {
		return false, "" // removed by hand already, or not a worktree of ours
	}
	if why := WorktreeWork(folder, branch); why != "" {
		return false, why
	}
	if _, err := GitRun(repo, "worktree", "remove", folder); err != nil {
		if why := WorktreeWork(folder, branch); why != "" {
			return false, why
		}
		if _, err := GitRun(repo, "worktree", "remove", "--force", folder); err != nil {
			return false, KeptChanges
		}
	}
	if branch != "" && branchPushed(repo, "refs/heads/"+branch) {
		_, _ = GitRun(repo, "branch", "-D", branch)
	}
	return true, ""
}
