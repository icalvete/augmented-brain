package memory

import (
	"io/fs"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// A note's body may name another note as [[id]]. Unlike related, area or parent, a
// body is free text and nothing checks it when it is written: the 22-nov audit found a
// live rule and a shared reference linking ids cut short, and older notes linking
// notes of the memory before augmented brain. BrokenLinks finds them for doctor.

// wikiLink is [[target]] or [[target|text]].
var wikiLink = regexp.MustCompile(`\[\[([^\]|\n]+)(\|[^\]\n]*)?\]\]`)

// linkID is what a note id looks like: a lowercase type prefix, a dash, the rest. Text
// such as [[id]] or [[…]] in a note explaining links is not a link.
var linkID = regexp.MustCompile(`^[a-z]+-[a-z0-9][a-z0-9-]*$`)

// BrokenLink is one [[id]] in the note at Path that names no note.
type BrokenLink struct {
	Path, ID string
}

// BrokenLinks are the [[id]] in the bodies of the notes that last (rules, areas,
// knowledge, people, the profile) and of the tasks not closed, whose id is no note of
// this memory, of also (the shared notes, other spaces) or of its archive/. Sorted by
// path, then id.
func (m *Memory) BrokenLinks(also ...*Memory) []BrokenLink {
	archived := map[string]bool{}
	_ = filepath.WalkDir(filepath.Join(m.Dir, "archive"), func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			archived[strings.TrimSuffix(d.Name(), ".md")] = true
		}
		return nil
	})
	known := func(id string) bool {
		// The profile lives next to the configuration, in no memory folder.
		if id == "profile" || m.Notes[id] != nil || archived[id] {
			return true
		}
		for _, o := range also {
			if o != nil && o.Notes[id] != nil {
				return true
			}
		}
		return false
	}
	var out []BrokenLink
	for _, n := range m.Notes {
		if n.Header.Type == "task" && n.Header.Status == "closed" {
			continue // its history: what it linked then is not worth a warning now
		}
		seen := map[string]bool{}
		for _, match := range wikiLink.FindAllStringSubmatch(n.Body, -1) {
			id := strings.TrimSpace(match[1])
			if !linkID.MatchString(id) || seen[id] || known(id) {
				continue
			}
			seen[id] = true
			out = append(out, BrokenLink{n.Path, id})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Path != out[j].Path {
			return out[i].Path < out[j].Path
		}
		return out[i].ID < out[j].ID
	})
	return out
}
