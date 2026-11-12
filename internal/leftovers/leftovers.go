// Package leftovers finds what working with Claude Code leaves behind that nobody
// cleans up: the worktrees Claude Code makes (.claude/worktrees inside a repository)
// once their session is over and they hold nothing of their own, git's records of
// worktrees whose folder is gone, and the scratch folders of sessions that ended
// (/tmp/claude-<uid>). Find only looks; Clean removes what Find found, checking each
// one again first. Decided on 12-nov: doctor says it, `augmented-brain clean` shows the
// list and removes it only when the user says yes.
package leftovers

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/icalvete/augmented-brain/internal/paths"
	"github.com/icalvete/augmented-brain/internal/remote"
	"github.com/icalvete/augmented-brain/internal/state"
)

// ScratchAge is how long a scratch folder of a session that is not running must have
// gone untouched before it counts as left over.
const ScratchAge = 24 * time.Hour

// Env is what Find and Clean look at, so tests can point them at temp folders.
type Env struct {
	// ClaudeJSON is Claude Code's own file (~/.claude.json): its projects are the folders
	// Claude Code was opened in. Read, never written.
	ClaudeJSON string
	// ClaudeDir is Claude Code's folder: sessions/<pid>.json are the sessions running.
	ClaudeDir string
	// TempRoot is Claude Code's scratch folder (/tmp/claude-<uid>); "" for none.
	TempRoot string
	// Alive says whether a process runs; nil is state.Alive.
	Alive func(pid int) bool
	// Now is the time to measure ages from; zero is time.Now().
	Now time.Time
}

// Local is the Env of this computer, given Claude Code's file and folder.
func Local(claudeJSON, claudeDir string) Env {
	e := Env{ClaudeJSON: claudeJSON, ClaudeDir: claudeDir}
	if uid := os.Getuid(); uid >= 0 { // -1 on Windows, where Claude Code keeps none there
		e.TempRoot = filepath.Join(os.TempDir(), "claude-"+strconv.Itoa(uid))
	}
	return e
}

// Worktree is a worktree Claude Code made whose session is over.
type Worktree struct {
	Repo, Folder string
	// Branch is the branch to remove with it: Claude Code's own (worktree-<name>), or ""
	// when the worktree is on a branch of the user's, which stays.
	Branch string
	// Work is what it holds of its own (remote.KeptChanges, remote.KeptCommits): such a
	// worktree is only named, never removed. "" when nothing would be lost.
	Work   string
	locked bool
}

// Scratch is a session's scratch folder, or a file of it, that is left over.
type Scratch struct {
	Path string
	Size int64
}

// Report is what Find found.
type Report struct {
	Worktrees []Worktree
	// Prunable are the repositories that keep a record of a worktree whose folder is gone
	// (`git worktree prune` forgets them; their branches stay).
	Prunable []string
	Scratch  []Scratch
	// ScratchOK is false when the running sessions could not be read: then no scratch
	// folder is called left over.
	ScratchOK bool
}

// Removable are the worktrees Clean would remove.
func (r Report) Removable() []Worktree {
	var out []Worktree
	for _, w := range r.Worktrees {
		if w.Work == "" {
			out = append(out, w)
		}
	}
	return out
}

// ScratchSize is the size of every left-over scratch folder.
func (r Report) ScratchSize() int64 {
	var n int64
	for _, s := range r.Scratch {
		n += s.Size
	}
	return n
}

// Empty says there is nothing to clean (worktrees with work of their own are not).
func (r Report) Empty() bool {
	return len(r.Removable()) == 0 && len(r.Prunable) == 0 && len(r.Scratch) == 0
}

func (e Env) alive(pid int) bool {
	if e.Alive != nil {
		return e.Alive(pid)
	}
	return state.Alive(pid)
}

func (e Env) now() time.Time {
	if e.Now.IsZero() {
		return time.Now()
	}
	return e.Now
}

// Find looks for leftovers. It never writes anything.
func Find(e Env) Report {
	sessions, ok := e.running()
	r := Report{ScratchOK: ok}
	r.Worktrees, r.Prunable = e.worktrees(sessions)
	if ok {
		r.Scratch = e.scratch(sessions)
	}
	return r
}

// session is a running Claude Code session: its id and folder.
type session struct{ id, cwd string }

// running reads sessions/<pid>.json in Claude Code's folder and keeps those whose
// process still runs. ok is false when the folder cannot be read.
func (e Env) running() ([]session, bool) {
	if e.ClaudeDir == "" {
		return nil, false
	}
	dir := filepath.Join(e.ClaudeDir, "sessions")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, false
	}
	var out []session
	for _, en := range entries {
		name, isJSON := strings.CutSuffix(en.Name(), ".json")
		pid, err := strconv.Atoi(name)
		if !isJSON || err != nil || !e.alive(pid) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, en.Name()))
		if err != nil {
			continue
		}
		var s struct {
			SessionID string `json:"sessionId"`
			Cwd       string `json:"cwd"`
		}
		if json.Unmarshal(data, &s) == nil {
			out = append(out, session{s.SessionID, s.Cwd})
		}
	}
	return out, true
}

// projects are the folders Claude Code was opened in, from its own file.
func (e Env) projects() []string {
	data, err := os.ReadFile(e.ClaudeJSON)
	if err != nil {
		return nil
	}
	var c struct {
		Projects map[string]json.RawMessage `json:"projects"`
	}
	if json.Unmarshal(data, &c) != nil {
		return nil
	}
	var out []string
	for p := range c.Projects {
		if filepath.IsAbs(p) {
			out = append(out, filepath.Clean(p))
		}
	}
	slices.Sort(out)
	return out
}

// claudeWorktrees is the folder inside a repository where Claude Code makes worktrees.
var claudeWorktrees = filepath.Join(".claude", "worktrees")

// lockedBy reads the reason Claude Code gives when it locks a worktree for a session:
// "claude session <name> (pid <n> start <t>)".
var lockedBy = regexp.MustCompile(`^claude session .*\(pid (\d+)\b`)

// worktrees goes through every repository Claude Code was opened in (or in one of its
// worktrees), once each.
func (e Env) worktrees(sessions []session) (found []Worktree, prunable []string) {
	seen := map[string]bool{}
	for _, p := range e.projects() {
		repo := p
		if before, _, ok := strings.Cut(p, string(filepath.Separator)+claudeWorktrees+string(filepath.Separator)); ok {
			repo = before
		}
		if seen[repo] {
			continue
		}
		seen[repo] = true
		if _, err := os.Stat(filepath.Join(repo, ".git")); err != nil {
			continue // not a repository's top, or gone
		}
		out, err := remote.GitRun(repo, "worktree", "list", "--porcelain")
		if err != nil {
			continue
		}
		main := ""
		for _, w := range parseList(out) {
			if main == "" {
				main = w.path
				if seen[main+"\x00"] { // the same repository, reached from another folder
					break
				}
				seen[main+"\x00"] = true
				continue
			}
			if w.prunable {
				if !slices.Contains(prunable, main) {
					prunable = append(prunable, main)
				}
				continue
			}
			if !paths.Within(w.path, filepath.Join(main, claudeWorktrees)) {
				continue // not one Claude Code made
			}
			if w.locked {
				m := lockedBy.FindStringSubmatch(w.reason)
				if m == nil {
					continue // locked by someone else, for a reason of their own
				}
				if pid, _ := strconv.Atoi(m[1]); e.alive(pid) {
					continue // its session runs
				}
			}
			if inUse(w.path, sessions) {
				continue
			}
			branch := ""
			if b := strings.TrimPrefix(w.branch, "refs/heads/"); b == "worktree-"+filepath.Base(w.path) {
				branch = b
			}
			found = append(found, Worktree{Repo: main, Folder: w.path, Branch: branch,
				Work: remote.WorktreeWork(w.path), locked: w.locked})
		}
	}
	return found, prunable
}

// inUse says whether a running session works in the folder or inside it.
func inUse(folder string, sessions []session) bool {
	for _, s := range sessions {
		if s.cwd != "" && paths.Within(s.cwd, folder) {
			return true
		}
	}
	return false
}

type listed struct {
	path, branch, reason string
	locked, prunable     bool
}

// parseList reads `git worktree list --porcelain`: blocks of lines, one per worktree,
// the main one first.
func parseList(out string) []listed {
	var all []listed
	for _, block := range strings.Split(strings.ReplaceAll(out, "\r\n", "\n"), "\n\n") {
		var w listed
		for _, line := range strings.Split(block, "\n") {
			key, val, _ := strings.Cut(line, " ")
			switch key {
			case "worktree":
				w.path = filepath.Clean(val)
			case "branch":
				w.branch = val
			case "locked":
				w.locked, w.reason = true, val
			case "prunable":
				w.prunable = true
			}
		}
		if w.path != "" {
			all = append(all, w)
		}
	}
	return all
}

// scratch are the scratch folders, under TempRoot/<working folder>/<session id>, of sessions not
// running and untouched for ScratchAge, and the files cache-break-state-<id>.json of
// those sessions. Anything else there is left alone.
func (e Env) scratch(sessions []session) []Scratch {
	if e.TempRoot == "" {
		return nil
	}
	live := map[string]bool{}
	for _, s := range sessions {
		live[s.id] = true
	}
	old := e.now().Add(-ScratchAge)
	tops, err := os.ReadDir(e.TempRoot)
	if err != nil {
		return nil
	}
	var out []Scratch
	for _, top := range tops {
		p := filepath.Join(e.TempRoot, top.Name())
		if !top.IsDir() {
			id, ok := strings.CutPrefix(top.Name(), "cache-break-state-")
			id, isJSON := strings.CutSuffix(id, ".json")
			if ok && isJSON && isID(id) && !live[id] && top.Type().IsRegular() {
				if size, newest := measure(p); newest.Before(old) {
					out = append(out, Scratch{p, size})
				}
			}
			continue
		}
		if !strings.HasPrefix(top.Name(), "-") {
			continue // not a working folder's (/home/ana/x is -home-ana-x): Claude Code's own
		}
		kids, err := os.ReadDir(p)
		if err != nil {
			continue
		}
		for _, k := range kids {
			if !k.IsDir() || !isID(k.Name()) || live[k.Name()] {
				continue
			}
			kp := filepath.Join(p, k.Name())
			if size, newest := measure(kp); newest.Before(old) {
				out = append(out, Scratch{kp, size})
			}
		}
	}
	return out
}

// isID says whether a name is a session id: 8-4-4-4-12 hexadecimal digits.
func isID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		switch i {
		case 8, 13, 18, 23:
			if c != '-' {
				return false
			}
		default:
			if !strings.ContainsRune("0123456789abcdef", c) {
				return false
			}
		}
	}
	return true
}

// measure is the size of everything under p and when anything there last changed.
// Links are counted as themselves, never followed.
func measure(p string) (size int64, newest time.Time) {
	_ = filepath.WalkDir(p, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if info, err := d.Info(); err == nil {
			if !d.IsDir() {
				size += info.Size()
			}
			if info.ModTime().After(newest) {
				newest = info.ModTime()
			}
		}
		return nil
	})
	return size, newest
}

// Done is what Clean did.
type Done struct {
	Worktrees []string // folders removed
	Kept      []Worktree
	Pruned    []string // repositories
	Scratch   int      // scratch folders and files removed
	Freed     int64
	Errors    []error
}

// Clean removes what r found, looking at each worktree and scratch folder again first:
// a worktree with work of its own by now, or a session that started again since, stays.
func Clean(e Env, r Report) Done {
	var d Done
	sessions, ok := e.running()
	for _, w := range r.Removable() {
		if inUse(w.Folder, sessions) {
			continue
		}
		if w.locked {
			if _, err := remote.GitRun(w.Repo, "worktree", "unlock", w.Folder); err != nil {
				d.Errors = append(d.Errors, err)
				continue
			}
		}
		done, why := remote.RemoveWorktree(w.Folder, w.Branch)
		switch {
		case done:
			d.Worktrees = append(d.Worktrees, w.Folder)
		case why != "":
			w.Work = why
			d.Kept = append(d.Kept, w)
		}
	}
	for _, repo := range r.Prunable {
		if _, err := remote.GitRun(repo, "worktree", "prune"); err != nil {
			d.Errors = append(d.Errors, err)
			continue
		}
		d.Pruned = append(d.Pruned, repo)
	}
	if !ok {
		return d
	}
	still := map[string]bool{}
	for _, s := range e.scratch(sessions) {
		still[s.Path] = true
	}
	for _, s := range r.Scratch {
		if !still[s.Path] || !paths.Within(s.Path, e.TempRoot) {
			continue // in use again, or not ours
		}
		if err := os.RemoveAll(s.Path); err != nil {
			d.Errors = append(d.Errors, err)
			continue
		}
		d.Scratch++
		d.Freed += s.Size
		// The folder of its working folder goes when nothing else is left in it.
		if parent := filepath.Dir(s.Path); parent != e.TempRoot {
			_ = os.Remove(parent)
		}
	}
	return d
}

// HumanSize writes a size for people: "154 MB", "3 KB", "2.1 GB".
func HumanSize(n int64) string {
	switch {
	case n >= 1e9:
		return strconv.FormatFloat(float64(n)/1e9, 'f', 1, 64) + " GB"
	case n >= 1e6:
		return strconv.FormatInt((n+5e5)/1e6, 10) + " MB"
	default:
		return strconv.FormatInt((n+500)/1e3, 10) + " KB"
	}
}

// LocalEnv is Local for this computer, from what remote access already knows of Claude
// Code; ok is false when that cannot be read.
func LocalEnv() (Env, bool) {
	s, err := remote.NewSystem()
	if err != nil {
		return Env{}, false
	}
	return Local(s.ClaudeJSON, s.ClaudeDir), true
}
