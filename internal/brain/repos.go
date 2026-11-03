package brain

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/icalvete/augmented-brain/internal/memory"
	"github.com/icalvete/augmented-brain/internal/paths"
)

// RepoNoteID is the reference that says what each repository of the space is: one line
// per repository, "- `path` — what it is". It is an ordinary reference, so it is
// backed up with the notes, and releases that know nothing of it read it as one.
const RepoNoteID = "ref-repositories"

// maxRepoText is how long a repository's description may be: one line.
const maxRepoText = 300

// RepoDescription is one line of the repositories' note.
type RepoDescription struct {
	Key  string // the repository as the note writes it: ~/…, an absolute path, or relative to a work folder
	Path string // where it is on this computer, resolved; empty when it cannot be told
	Text string
}

// RepoDescriptions returns what the repositories' note says, in its order. Lines that
// do not have the form "- `path` — text" are left out (and left alone in the note).
func (b *Brain) RepoDescriptions() []RepoDescription {
	n, ok := b.Mem.Notes[RepoNoteID]
	if !ok || n.Header.Type != "reference" {
		return nil
	}
	var out []RepoDescription
	for _, l := range strings.Split(n.Body, "\n") {
		key, text, ok := parseRepoLine(l)
		if !ok {
			continue
		}
		out = append(out, RepoDescription{Key: key, Path: b.repoKeyPath(key), Text: text})
	}
	return out
}

// RepoDescriptionOf returns the description of the repository at path (resolved), or "".
func (b *Brain) RepoDescriptionOf(path string) string {
	for _, d := range b.RepoDescriptions() {
		if d.Path != "" && paths.Same(d.Path, path) {
			return d.Text
		}
	}
	return ""
}

// repoKeyPath resolves a key of the note: ~ and absolute paths as they are, a relative
// one against the work folders (the first that has it, or else the first).
func (b *Brain) repoKeyPath(key string) string {
	if p, ok := b.RepoPath(key); ok {
		return p
	}
	if strings.HasPrefix(key, "~") || filepath.IsAbs(key) {
		p, _ := paths.Resolve(key)
		return p
	}
	return ""
}

// parseRepoLine reads "- `path` — text" (also with " - " for the dash).
func parseRepoLine(l string) (key, text string, ok bool) {
	l = strings.TrimRight(l, " \r")
	rest, found := strings.CutPrefix(l, "- `")
	if !found {
		return "", "", false
	}
	end := strings.Index(rest, "`")
	if end <= 0 {
		return "", "", false
	}
	key, rest = rest[:end], rest[end+1:]
	for _, dash := range []string{" — ", " - "} {
		if t, found := strings.CutPrefix(rest, dash); found {
			return key, strings.TrimSpace(t), true
		}
	}
	return "", "", false
}

// DescribeRepo writes what a repository is, in one line, in the repositories' note
// (made the first time); an empty text takes its line out. repo is a path (~ or
// absolute) or a name as a task's repos: writes it, relative to a work folder, and it
// must be a folder inside one of the space's work folders. It returns where the
// repository is, written with ~.
func (b *Brain) DescribeRepo(repo, text string) (string, error) {
	repo = strings.TrimSpace(repo)
	if repo == "" {
		return "", errors.New("say which repository: its path, or its name as a task's repos: writes it")
	}
	p, ok := b.RepoPath(repo)
	if !ok {
		return "", fmt.Errorf("%q is not a folder in this space's work folders on this computer", repo)
	}
	if !slices.ContainsFunc(b.Space.WorkFolders, func(wf string) bool { return paths.Within(p, wf) }) {
		return "", fmt.Errorf("%s is outside this space's work folders (%s): only repositories inside them are described here", paths.Tilde(p), strings.Join(tildeAll(b.Space.WorkFolders), ", "))
	}
	key := paths.Tilde(p)
	if strings.ContainsAny(key, "`\n\r") {
		return "", fmt.Errorf("%q cannot be written in the note: its path has a backquote or a line break", key)
	}
	text = oneLine(text)
	if len([]rune(text)) > maxRepoText {
		return "", fmt.Errorf("the description is %d characters long; keep it to one line of %d at most", len([]rune(text)), maxRepoText)
	}

	var cp memory.Note
	if old, ok := b.Mem.Notes[RepoNoteID]; ok {
		if old.Header.Type != "reference" {
			return "", fmt.Errorf("%s is a %s, not the reference of the repositories; rename it first", RepoNoteID, old.Header.Type)
		}
		cp = *old // the note in the index stays as it was until saved
	} else {
		if text == "" {
			return key, nil // nothing to take out of a note that does not exist
		}
		cp = memory.Note{
			Path: memory.PathFor("reference", RepoNoteID),
			Header: memory.Header{ID: RepoNoteID, Type: "reference", Title: b.text("repos.title"),
				Summary: b.text("repos.summary"), Created: b.today()},
			Body: b.text("repos.intro") + "\n\n",
		}
	}
	cp.Body = setRepoLine(cp.Body, func(k string) bool {
		kp := b.repoKeyPath(k)
		return k == key || (kp != "" && paths.Same(kp, p))
	}, key, text)
	if old, ok := b.Mem.Notes[RepoNoteID]; ok && old.Body == cp.Body {
		return key, nil // nothing changes
	}
	return key, b.save(&cp)
}

// setRepoLine puts "- `key` — text" in place of the first line whose key matches, or
// at the end; an empty text takes out every line that matches. Other lines stay.
func setRepoLine(body string, match func(string) bool, key, text string) string {
	lines := strings.Split(strings.TrimRight(body, "\n"), "\n")
	line := "- `" + key + "` — " + text
	var out []string
	done := false
	for _, l := range lines {
		if k, _, ok := parseRepoLine(l); ok && match(k) {
			if text != "" && !done {
				out = append(out, line)
			}
			done = true
			continue
		}
		out = append(out, l)
	}
	if !done && text != "" {
		if last := out[len(out)-1]; last != "" {
			if _, _, ok := parseRepoLine(last); !ok {
				out = append(out, "") // the list starts after a paragraph: a blank line between
			}
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n") + "\n"
}

func tildeAll(list []string) []string {
	out := make([]string, len(list))
	for i, p := range list {
		out[i] = paths.Tilde(p)
	}
	return out
}
