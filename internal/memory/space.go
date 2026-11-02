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
}

// Load reads every note of the memory in dir, except archive/. It returns what it could
// read, and one Problem per file that is not a valid note. A note may link (related) to
// the notes of also: the shared notes, which every space can see.
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
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, path)
		if d.IsDir() {
			if rel == "archive" || (strings.HasPrefix(d.Name(), ".") && rel != ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".md") || skipped[rel] {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
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
	problems = append(problems, m.checkLinks(also)...)
	sort.Slice(problems, func(i, j int) bool { return problems[i].Path < problems[j].Path })
	return m, problems, nil
}

// SharedTypes are the types of note the shared folder holds: what applies in every
// space. Tasks, areas and people always belong to one space.
var SharedTypes = []string{"rule", "reference", "lesson"}

// LoadShared reads the shared notes in dir, which may not exist yet (then there are
// none). A note of a type that is not shared, or with an area, is a problem.
func LoadShared(dir string) (*Memory, []Problem, error) {
	if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
		return &Memory{Dir: dir, Notes: map[string]*Note{}}, nil, nil
	}
	m, problems, err := Load(dir)
	if err != nil {
		return nil, nil, err
	}
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
