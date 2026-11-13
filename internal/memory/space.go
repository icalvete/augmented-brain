package memory

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

// Generated files and free-form files: they have no header and are not notes.
var skipped = map[string]bool{"MEMORY.md": true, "inbox.md": true}

// Problem is a file that could not be read as a note.
type Problem struct {
	Path string
	Err  error
}

func (p Problem) Error() string { return fmt.Sprintf("%s: %v", p.Path, p.Err) }

// Memory is the content of one space's memory folder.
type Memory struct {
	Dir   string
	Notes map[string]*Note // by id
	// Shared is set for the shared notes, which have no areas, people or tasks: their
	// index lists rules and knowledge instead.
	Shared bool
}

// Load reads every note of the memory in dir, except archive/ and docs/ (a task's long
// documents, which are not notes) and hidden files (an editor's lock file, such as
// Emacs's .#note.md). It returns what it could read, and one Problem per file that is
// not a valid note, cannot be read, or sits in a folder that cannot be opened: one bad
// file never keeps the rest from loading. Only a memory folder that cannot be read at
// all is an error. A note may link (related) to the notes of also: the shared notes,
// which every space can see.
func Load(dir string, also ...*Memory) (*Memory, []Problem, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return nil, nil, err
	}
	if !info.IsDir() {
		return nil, nil, fmt.Errorf("%s is not a folder", dir)
	}
	m := &Memory{Dir: dir, Notes: map[string]*Note{}}
	var problems []Problem
	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		rel, _ := filepath.Rel(dir, path)
		if err != nil {
			if rel == "." {
				return err
			}
			// A subfolder that cannot be opened: what it holds is left out, and said.
			problems = append(problems, Problem{filepath.ToSlash(rel), err})
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		hidden := strings.HasPrefix(d.Name(), ".") && rel != "."
		if d.IsDir() {
			if rel == "archive" || rel == "docs" || hidden {
				return filepath.SkipDir
			}
			return nil
		}
		if hidden || !strings.HasSuffix(path, ".md") || skipped[rel] {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			problems = append(problems, Problem{filepath.ToSlash(rel), unreadable(err)})
			return nil
		}
		n, perr := Parse(rel, data)
		if perr != nil {
			problems = append(problems, Problem{filepath.ToSlash(rel), perr})
			return nil
		}
		if prev, dup := m.Notes[n.Header.ID]; dup {
			problems = append(problems, Problem{n.Path, fmt.Errorf("id %q is also used by %s", n.Header.ID, prev.Path)})
			return nil
		}
		m.Notes[n.Header.ID] = n
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	problems = append(problems, linkedFolders(dir)...)
	problems = append(problems, m.checkLinks(also)...)
	problems = append(problems, m.checkDates()...)
	sort.Slice(problems, func(i, j int) bool { return problems[i].Path < problems[j].Path })
	return m, problems, nil
}

// unreadable says why a note's file could not be read, in words that point at the fix.
func unreadable(err error) error {
	return fmt.Errorf("cannot be read (%w); fix its permissions, or move it out if it is a broken link", err)
}

// linkedFolders reports the folders of a type of note (tasks, knowledge...) that are a
// link to another place: the walk does not follow links, so their notes would vanish
// without a word.
func linkedFolders(dir string) []Problem {
	var problems []Problem
	seen := map[string]bool{}
	for _, t := range Types {
		if t.Folder == "" || seen[t.Folder] {
			continue
		}
		seen[t.Folder] = true
		if fi, err := os.Lstat(filepath.Join(dir, t.Folder)); err == nil && fi.Mode()&fs.ModeSymlink != 0 {
			problems = append(problems, Problem{t.Folder, errors.New("is a link to another folder; its notes are not read: put the folder itself here")})
		}
	}
	return problems
}

// checkDates reports dates in a header that are not YYYY-MM-DD (a date written by hand
// as 2026-11-3 would never come up). Soft problems: the note is kept.
func (m *Memory) checkDates() []Problem {
	var problems []Problem
	bad := func(n *Note, field, value string) {
		if value == "" {
			return
		}
		if _, err := ParseDate(value); err != nil {
			problems = append(problems, Problem{n.Path, fmt.Errorf("%s %q is not a date: use YYYY-MM-DD", field, value)})
		}
	}
	for _, id := range m.IDs() {
		n := m.Notes[id]
		bad(n, "created", n.Header.Created)
		bad(n, "updated", n.Header.Updated)
		for _, w := range n.Header.Watch {
			if w.Date == "" {
				problems = append(problems, Problem{n.Path, errors.New("a watch date is missing its date: use YYYY-MM-DD")})
				continue
			}
			bad(n, "watch date", w.Date)
		}
	}
	return problems
}

// SharedTypes are the types of note the shared folder holds: what applies in every
// space. Tasks, areas and people always belong to one space.
var SharedTypes = []string{"rule", "reference", "lesson"}

// LoadShared reads the shared notes in dir, which may not exist yet (then there are
// none). A note of a type that is not shared, or with an area, is a problem.
func LoadShared(dir string) (*Memory, []Problem, error) {
	if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
		return &Memory{Dir: dir, Notes: map[string]*Note{}, Shared: true}, nil, nil
	}
	m, problems, err := Load(dir)
	if err != nil {
		return nil, nil, err
	}
	m.Shared = true
	for _, id := range m.IDs() {
		n := m.Notes[id]
		var err error
		switch {
		case !slices.Contains(SharedTypes, n.Header.Type):
			err = fmt.Errorf("a %s belongs to one space; the shared notes are rules, references and lessons", n.Header.Type)
		case n.Header.Area != "":
			err = fmt.Errorf("area %q: areas belong to one space, so a shared note has none", n.Header.Area)
		default:
			continue
		}
		problems = append(problems, Problem{n.Path, err})
		delete(m.Notes, id)
	}
	sort.Slice(problems, func(i, j int) bool { return problems[i].Path < problems[j].Path })
	return m, problems, nil
}

// LoadSpace loads a space's memory in dir, where notes may link to the shared notes in
// sharedDir. Shared notes that cannot be read only count as missing here; reading them
// is checked where they are loaded on their own.
func LoadSpace(dir, sharedDir string) (*Memory, []Problem, error) {
	shared, _, err := LoadShared(sharedDir)
	if err != nil {
		shared = &Memory{Dir: sharedDir, Notes: map[string]*Note{}}
	}
	return Load(dir, shared)
}

// checkLinks reports areas, parents and related notes that do not exist. These are
// soft problems: the note itself is valid, is kept and can be written.
func (m *Memory) checkLinks(also []*Memory) []Problem {
	var problems []Problem
	for _, id := range m.IDs() {
		n := m.Notes[id]
		if a := n.Header.Area; a != "" {
			if target, ok := m.Notes[a]; !ok || target.Header.Type != "area" {
				problems = append(problems, Problem{n.Path, fmt.Errorf("area %q does not exist", a)})
			}
		}
		if p := n.Header.Parent; p != "" {
			var err error
			switch {
			case n.Header.Type != "task":
				err = fmt.Errorf("parent %q: only a task can be part of another task", p)
			case p == id:
				err = fmt.Errorf("parent %q is the task itself", p)
			default:
				err = m.parentProblem(p)
			}
			if err != nil {
				problems = append(problems, Problem{n.Path, err})
			}
		}
		for _, r := range n.Header.Related {
			if _, ok := m.Notes[r]; !ok && !inAny(also, r) {
				problems = append(problems, Problem{n.Path, fmt.Errorf("related %q does not exist", r)})
			}
		}
	}
	return problems
}

func inAny(ms []*Memory, id string) bool {
	for _, m := range ms {
		if _, ok := m.Notes[id]; ok {
			return true
		}
	}
	return false
}

// parentProblem says why id cannot be a subtask's parent, or nil when it can.
func (m *Memory) parentProblem(id string) error {
	p, ok := m.Notes[id]
	switch {
	case !ok:
		return fmt.Errorf("parent %q does not exist", id)
	case p.Header.Type != "task":
		return fmt.Errorf("parent %q is a %s, not a task", id, p.Header.Type)
	case p.Header.Parent != "":
		return fmt.Errorf("parent %q is itself a subtask: subtasks go one level deep only", id)
	}
	return nil
}

// ParentOf returns the parent task of a subtask, when it exists and can be one: a task
// that is not itself a subtask (so a loop of parents never counts). Otherwise nil.
func (m *Memory) ParentOf(n *Note) *Note {
	h := n.Header
	if h.Type != "task" || h.Parent == "" || h.Parent == h.ID || m.parentProblem(h.Parent) != nil {
		return nil
	}
	return m.Notes[h.Parent]
}

// Children returns the subtasks of a task, sorted by id: those whose parent is this
// task and is a valid one (see ParentOf). A task that cannot be a parent has none, so
// a subtask is never listed twice, under its parent and on its own.
func (m *Memory) Children(id string) []*Note {
	var out []*Note
	for _, n := range m.OfType("task") {
		if n.Header.Parent == id && m.ParentOf(n) != nil {
			out = append(out, n)
		}
	}
	return out
}

// IDs returns every note id, sorted.
func (m *Memory) IDs() []string {
	ids := make([]string, 0, len(m.Notes))
	for id := range m.Notes {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// OfType returns the notes of one type, sorted by id.
func (m *Memory) OfType(t string) []*Note {
	var out []*Note
	for _, id := range m.IDs() {
		if n := m.Notes[id]; n.Header.Type == t {
			out = append(out, n)
		}
	}
	return out
}
