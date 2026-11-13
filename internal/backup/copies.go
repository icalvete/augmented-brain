package backup

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/filelock"
	"github.com/icalvete/augmented-brain/internal/format"
	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/memory"
)

// The copies of a space kept in its own repository: which they are, what each one
// changed, and going back to one on this computer. Every copy is a commit whose message
// starts with "backup " (commit writes them); commits a person made are not copies.

// Copy is one copy of a space's notes.
type Copy struct {
	Rev  string // the commit
	Time time.Time
	Host string // the computer that made it
	Back string // when it went back to an earlier copy: that copy's date, as written
}

// Line is one line of what changed: Op is ' ' (the same), '-' (gone), '+' (new) or '~'
// (unchanged lines left out).
type Line struct {
	Op   byte
	Text string
}

// Kind names the line's Op for a page: "add", "del", "skip" or "same".
func (l Line) Kind() string {
	switch l.Op {
	case '+':
		return "add"
	case '-':
		return "del"
	case '~':
		return "skip"
	}
	return "same"
}

// FileChange is what changed in one note.
type FileChange struct {
	Name      string
	New, Gone bool
	Binary    bool // not text: no lines
	Lines     []Line
}

// Changes is what changed between two states of the notes: up to MaxChangedFiles notes,
// and how many more there were.
type Changes struct {
	Files []FileChange
	More  int
}

// MaxChangedFiles and maxChangedLines keep what a page shows readable.
const (
	MaxChangedFiles = 20
	maxChangedLines = 200
)

// backBy is how the message of a copy made by going back says so.
const backBy = " back to "

// notesOnly leaves out of a comparison what the program writes by itself: the index and
// the copy of the configuration.
var notesOnly = []string{"--", ".", ":(exclude)" + memory.IndexFile, ":(exclude)" + MetaDir}

// Copies are the last n copies of the notes in memory, newest first, from every computer
// that backs them up. Nothing when the folder is not a repository.
func Copies(mem string, n int) ([]Copy, error) {
	if _, err := os.Stat(filepath.Join(mem, ".git")); err != nil {
		return nil, nil
	}
	if _, err := git(mem, "rev-parse", "-q", "--verify", "HEAD"); err != nil {
		return nil, nil // no commit yet
	}
	out, err := git(mem, "log", "-n", "200", "--format=%H%x09%ct%x09%s", "HEAD")
	if err != nil {
		return nil, err
	}
	var cs []Copy
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if c, ok := parseCopy(line); ok {
			cs = append(cs, c)
			if len(cs) == n {
				break
			}
		}
	}
	return cs, nil
}

// parseCopy reads a line "<hash>\t<seconds>\t<message>"; ok only for a copy.
func parseCopy(line string) (Copy, bool) {
	f := strings.SplitN(line, "\t", 3)
	if len(f) != 3 {
		return Copy{}, false
	}
	rest, ok := strings.CutPrefix(f[2], "backup ")
	sec, err := strconv.ParseInt(f[1], 10, 64)
	if !ok || err != nil || sec <= 0 || f[0] == "" {
		return Copy{}, false
	}
	c := Copy{Rev: f[0], Time: time.Unix(sec, 0)}
	if _, back, ok := strings.Cut(rest, backBy); ok {
		c.Back = strings.TrimSpace(back)
	}
	if w := strings.Fields(rest); len(w) > 0 {
		c.Host = w[0]
	}
	return c, true
}

// ErrNotACopy is a commit that is not one of the space's copies.
var ErrNotACopy = errors.New("not a copy of these notes")

// copyOf checks that rev is a copy in the history of the notes, and returns it.
func copyOf(mem, rev string) (Copy, error) {
	if len(rev) < 7 || strings.Trim(strings.ToLower(rev), "0123456789abcdef") != "" {
		return Copy{}, ErrNotACopy
	}
	out, err := git(mem, "log", "-n", "1", "--format=%H%x09%ct%x09%s", rev+"^{commit}", "--")
	if err != nil {
		return Copy{}, ErrNotACopy
	}
	c, ok := parseCopy(strings.TrimSpace(out))
	if !ok {
		return Copy{}, ErrNotACopy
	}
	if _, err := git(mem, "merge-base", "--is-ancestor", c.Rev, "HEAD"); err != nil {
		return Copy{}, ErrNotACopy
	}
	return c, nil
}

// emptyTree is git's tree with nothing in it: what the first copy is compared with.
const emptyTree = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"

// ChangesOf is what a copy changed from the copy before it (its first parent).
func ChangesOf(mem, rev string) (Changes, error) {
	c, err := copyOf(mem, rev)
	if err != nil {
		return Changes{}, err
	}
	parent := emptyTree
	if p, err := git(mem, "rev-parse", "-q", "--verify", c.Rev+"^1"); err == nil {
		parent = strings.TrimSpace(p)
	}
	out, err := git(mem, append([]string{"-c", "core.quotePath=false", "diff", "--no-color", "--no-renames", "--no-ext-diff", "--no-prefix", "-U2", parent, c.Rev}, notesOnly...)...)
	if err != nil {
		return Changes{}, err
	}
	return parseDiff(out), nil
}

// ChangesBack is what going back to a copy would change in the notes as they are now,
// with the notes no copy holds yet: the comparison runs on a temporary index of git, so
// nothing of the repository changes.
func ChangesBack(mem, rev string) (Changes, error) {
	c, err := copyOf(mem, rev)
	if err != nil {
		return Changes{}, err
	}
	tmp, err := os.MkdirTemp("", "ab-index-")
	if err != nil {
		return Changes{}, err
	}
	defer os.RemoveAll(tmp)
	env := []string{"GIT_INDEX_FILE=" + filepath.Join(tmp, "index")}
	if _, err := gitEnv(mem, env, "read-tree", "HEAD"); err != nil {
		return Changes{}, err
	}
	if _, err := gitEnv(mem, env, "add", "-A", "--", "."); err != nil {
		return Changes{}, err
	}
	out, err := gitEnv(mem, env, append([]string{"-c", "core.quotePath=false", "diff", "--no-color", "--no-renames", "--no-ext-diff", "--no-prefix", "-U2", "-R", "--cached", c.Rev}, notesOnly...)...)
	if err != nil {
		return Changes{}, err
	}
	return parseDiff(out), nil
}

// gitEnv is git with more variables in its environment.
func gitEnv(dir string, env []string, args ...string) (string, error) {
	out, err := runGitEnv(env, append([]string{"-C", dir}, args...)...)
	if err != nil {
		return out, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(out))
	}
	return out, nil
}

// parseDiff reads git's unified diff into what changed, note by note.
func parseDiff(out string) Changes {
	var ch Changes
	var cur *FileChange
	lines := 0
	flush := func() {
		if cur == nil {
			return
		}
		if len(ch.Files) < MaxChangedFiles {
			ch.Files = append(ch.Files, *cur)
		} else {
			ch.More++
		}
		cur = nil
	}
	inHunk := false
	for _, l := range strings.Split(out, "\n") {
		switch {
		case strings.HasPrefix(l, "diff --git "):
			flush()
			cur, lines, inHunk = &FileChange{}, 0, false
		case cur == nil:
		case !inHunk && strings.HasPrefix(l, "new file mode"):
			cur.New = true
		case !inHunk && strings.HasPrefix(l, "deleted file mode"):
			cur.Gone = true
		case !inHunk && strings.HasPrefix(l, "Binary files "):
			cur.Binary = true
			if cur.Name == "" {
				cur.Name = binaryName(l)
			}
		case !inHunk && (strings.HasPrefix(l, "--- ") || strings.HasPrefix(l, "+++ ")):
			if n := l[4:]; n != "/dev/null" {
				cur.Name = n
			}
		case strings.HasPrefix(l, "@@"):
			if inHunk {
				cur.Lines = append(cur.Lines, Line{Op: '~'})
			}
			inHunk = true
		case inHunk && l != "" && strings.ContainsRune(" +-", rune(l[0])):
			lines++
			if lines <= maxChangedLines {
				cur.Lines = append(cur.Lines, Line{Op: l[0], Text: l[1:]})
			} else if lines == maxChangedLines+1 {
				cur.Lines = append(cur.Lines, Line{Op: '~'})
			}
		}
	}
	flush()
	return ch
}

// binaryName takes the note's name from "Binary files x and x differ" (git runs with
// --no-prefix; one side may be /dev/null).
func binaryName(l string) string {
	l = strings.TrimSuffix(strings.TrimPrefix(l, "Binary files "), " differ")
	a, b, _ := strings.Cut(l, " and ")
	if b != "/dev/null" {
		return b
	}
	return a
}

// ErrChangedMeanwhile is a note written while going back was getting ready.
var ErrChangedMeanwhile = errors.New("the notes changed meanwhile")

// NotValidError is a copy whose notes this program would not load: going back was undone.
type NotValidError struct{ Problems []string }

func (e *NotValidError) Error() string {
	return "the notes of that copy are not valid: " + strings.Join(e.Problems, "; ")
}

// GoBack puts the notes of a space back as they were in one of its copies, on this
// computer. First what there is now becomes a copy (so going back is undone by going
// back to that one); then, with the notes locked, the notes come from the copy, are
// checked, and are recorded as a new copy whose message says which one it went back to.
// The copy of the configuration in MetaDir stays this computer's; only the version of
// the notes' format comes with them, and an older one is moved forward as at the start
// of a session. Nothing is sent: the next backup does that.
func GoBack(c *config.Config, s *config.Space, rev, host string) (*Result, error) {
	res, err := goBack(c, s, rev, host)
	return res, Localize(err, c.Language)
}

// goBack is GoBack; GoBack gives its errors the user's language.
func goBack(c *config.Config, s *config.Space, rev, host string) (*Result, error) {
	if s.Backup == "" {
		return nil, errors.New(i18n.T(c.Language, "backup.none", s.Name))
	}
	unlock, err := filelock.LockWithin(filepath.Join(c.Dir, backupLock), LockWait)
	if errors.Is(err, filelock.ErrBusy) {
		return nil, errors.New(i18n.T(c.Language, "backup.busy", LockWait))
	}
	if err != nil {
		return nil, err
	}
	defer unlock()
	if err := ownRepo(s); err != nil {
		return nil, err
	}
	to, err := copyOf(s.Memory, rev)
	if err != nil {
		return nil, err
	}
	res := &Result{}
	if err := commit(c, s, host, res); err != nil {
		return res, err
	}
	unlockNotes, err := lockNotes(c.Dir)
	if err != nil {
		return res, err
	}
	defer unlockNotes()
	if out, _ := git(s.Memory, "status", "--porcelain"); strings.TrimSpace(out) != "" {
		return res, ErrChangedMeanwhile
	}
	head, err := git(s.Memory, "rev-parse", "HEAD")
	if err != nil {
		return res, err
	}
	head = strings.TrimSpace(head)
	undo := func() { _, _ = git(s.Memory, "reset", "-q", "--hard", head) }

	before := map[string]bool{}
	if _, problems, err := memory.LoadSpace(s.Memory, c.SharedDir()); err == nil {
		for _, p := range problems {
			before[p.Error()] = true
		}
	}
	if _, err := git(s.Memory, append([]string{"restore", "--source=" + to.Rev, "--staged", "--worktree"}, notesOnly...)...); err != nil {
		undo()
		return res, err
	}
	// The version of the notes' format travels with them.
	fv := MetaDir + "/" + format.File
	if _, err := git(s.Memory, "cat-file", "-e", to.Rev+":"+fv); err == nil {
		_, err = git(s.Memory, "restore", "--source="+to.Rev, "--staged", "--worktree", "--", fv)
		if err != nil {
			undo()
			return res, err
		}
	} else if err := os.Remove(filepath.Join(s.Memory, fv)); err != nil && !errors.Is(err, os.ErrNotExist) {
		undo()
		return res, err
	}
	if _, err := format.EnsureNotes(s.Memory, c.Dir, s.Name, c.Language, time.Now()); err != nil {
		undo()
		return res, err
	}
	m, problems, err := memory.LoadSpace(s.Memory, c.SharedDir())
	if err != nil {
		undo()
		return res, err
	}
	var added []string
	for _, p := range problems {
		if !before[p.Error()] {
			added = append(added, p.Error())
		}
	}
	if len(added) > 0 {
		undo()
		return res, &NotValidError{Problems: added}
	}
	if _, err := m.WriteIndex(c.Language); err != nil {
		undo()
		return res, err
	}
	if _, err := git(s.Memory, "add", "-A"); err != nil {
		undo()
		return res, err
	}
	if out, _ := git(s.Memory, "status", "--porcelain"); strings.TrimSpace(out) != "" {
		msg := fmt.Sprintf("backup %s %s%s%s", host, time.Now().Format("2006-01-02 15:04"), backBy, to.Time.Local().Format("2006-01-02 15:04"))
		if _, err := git(s.Memory, append(ident, "commit", "-q", "-m", msg)...); err != nil {
			undo()
			return res, err
		}
		res.Committed = true
	}
	return res, nil
}
