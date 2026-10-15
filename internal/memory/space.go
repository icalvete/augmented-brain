package memory

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
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
// read, and one Problem per file that is not a valid note.
func Load(dir string) (*Memory, []Problem, error) {
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
	problems = append(problems, m.checkLinks()...)
	sort.Slice(problems, func(i, j int) bool { return problems[i].Path < problems[j].Path })
	return m, problems, nil
}

// checkLinks reports areas and related notes that do not exist.
func (m *Memory) checkLinks() []Problem {
	var problems []Problem
	for _, id := range m.IDs() {
		n := m.Notes[id]
		if a := n.Header.Area; a != "" {
			if target, ok := m.Notes[a]; !ok || target.Header.Type != "area" {
				problems = append(problems, Problem{n.Path, fmt.Errorf("area %q does not exist", a)})
			}
		}
		for _, r := range n.Header.Related {
			if _, ok := m.Notes[r]; !ok {
				problems = append(problems, Problem{n.Path, fmt.Errorf("related %q does not exist", r)})
			}
		}
	}
	return problems
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
