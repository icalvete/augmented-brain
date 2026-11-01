package backup

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/icalvete/augmented-brain/internal/filelock"
	"github.com/icalvete/augmented-brain/internal/guard"
)

// Lock files in the configuration folder. NotesLock is the one every writer of notes
// takes (the MCP server, the end-of-session journal line, the merge of a backup);
// backupLock keeps two backups from sharing a repository. When both are needed,
// backupLock is taken first, and NotesLock only for short steps that touch no network.
const (
	NotesLock  = ".lock"
	backupLock = ".backup.lock"
)

// lockNotes takes NotesLock of the configuration folder cfgDir.
func lockNotes(cfgDir string) (func(), error) {
	return filelock.Lock(filepath.Join(cfgDir, NotesLock))
}

// repoFiles are the lines the notes repository needs in its own git files: the inbox
// merges line by line (two machines that each captured something keep both lines), and
// leftovers of a crash and the files operating systems drop in folders never travel.
var repoFiles = map[string][]string{
	".gitattributes": {"inbox.md merge=union"},
	".gitignore":     {"*.tmp", ".DS_Store", "Thumbs.db", "desktop.ini"},
}

// ensureRepoFiles adds the lines of repoFiles that are missing, keeping anything else
// the user wrote in those files.
func ensureRepoFiles(dir string) error {
	names := make([]string, 0, len(repoFiles))
	for n := range repoFiles {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, name := range names {
		path := filepath.Join(dir, name)
		old, err := os.ReadFile(path)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		have := map[string]bool{}
		for _, l := range strings.Split(string(old), "\n") {
			have[strings.TrimSpace(l)] = true
		}
		text := string(old)
		for _, l := range repoFiles[name] {
			if have[l] {
				continue
			}
			if text != "" && !strings.HasSuffix(text, "\n") {
				text += "\n"
			}
			text += l + "\n"
		}
		if text != string(old) {
			if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
				return err
			}
		}
	}
	return nil
}

// MaxFileSize is the largest file a backup takes. GitHub refuses a push with a file of
// more than 100 MB, which would stop every backup after it; files over this size stay
// out, and the user is told.
var MaxFileSize int64 = 50 << 20

const (
	excludeStart = "# augmented brain: files too big for the backup (rewritten by every backup)"
	excludeEnd   = "# augmented brain: end"
)

// leaveOutLarge keeps the files over MaxFileSize out of the backup: it lists them in
// .git/info/exclude (between its own markers, rewritten each time, so a file that shrinks
// comes back) and stops tracking any that were already in it. It returns them as
// "path (size)", for the warning.
func leaveOutLarge(dir string) ([]string, error) {
	var big []string
	var shown []string
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.Size() > MaxFileSize {
			rel, _ := filepath.Rel(dir, p)
			rel = filepath.ToSlash(rel)
			big = append(big, rel)
			shown = append(shown, fmt.Sprintf("%s (%d MB)", rel, info.Size()>>20))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	exclude := filepath.Join(dir, ".git", "info", "exclude")
	old, _ := os.ReadFile(exclude)
	var keep []string
	inside := false
	for _, l := range strings.Split(strings.TrimRight(string(old), "\n"), "\n") {
		switch {
		case l == excludeStart:
			inside = true
		case l == excludeEnd:
			inside = false
		case !inside:
			keep = append(keep, l)
		}
	}
	text := strings.TrimRight(strings.Join(keep, "\n"), "\n")
	if len(big) > 0 {
		if text != "" {
			text += "\n"
		}
		text += excludeStart + "\n"
		for _, b := range big {
			text += "/" + escapePattern(b) + "\n"
		}
		text += excludeEnd
	}
	if text != "" {
		text += "\n"
	}
	if text != string(old) {
		if err := os.MkdirAll(filepath.Dir(exclude), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(exclude, []byte(text), 0o644); err != nil {
			return nil, err
		}
	}
	for _, b := range big {
		if _, err := git(dir, "ls-files", "--error-unmatch", "--", b); err == nil {
			if _, err := git(dir, "rm", "-q", "--cached", "--", b); err != nil {
				return nil, err
			}
		}
	}
	return shown, nil
}

// escapePattern writes a path as a gitignore pattern that matches only that path: the
// characters patterns give a meaning to get a backslash, and so does a trailing space.
func escapePattern(p string) string {
	var b strings.Builder
	for i, r := range p {
		switch {
		case strings.ContainsRune(`\[]*?!#`, r):
			b.WriteByte('\\')
		case r == ' ' && strings.TrimRight(p[i:], " ") == "":
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// checkStaged looks at what `git add` staged: a file over MaxFileSize is unstaged and
// returned for the warning, and a secret stops the backup. Only changed files are read.
func checkStaged(dir string) ([]string, error) {
	out, err := git(dir, "diff", "--cached", "--name-only", "-z", "--diff-filter=ACMR")
	if err != nil {
		return nil, err
	}
	var big, found []string
	for _, rel := range strings.Split(out, "\x00") {
		if rel == "" {
			continue
		}
		p := filepath.Join(dir, filepath.FromSlash(rel))
		info, err := os.Stat(p)
		if err != nil {
			continue
		}
		if info.Size() > MaxFileSize {
			if _, err := git(dir, "rm", "-q", "--cached", "--", rel); err != nil {
				return nil, err
			}
			big = append(big, fmt.Sprintf("%s (%d MB)", rel, info.Size()>>20))
			continue
		}
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		if what := guard.FindSecret(string(data)); what != "" {
			found = append(found, fmt.Sprintf("%s (%s)", rel, what))
		}
	}
	if len(found) > 0 {
		return nil, fmt.Errorf("nothing was backed up: these files hold secrets, remove them first: %s", strings.Join(found, ", "))
	}
	return big, nil
}
