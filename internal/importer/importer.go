// Package importer brings an existing memory of Claude Code into augmented brain's
// spaces: the Markdown files Claude Code's automatic memory writes (a header with name,
// description and metadata, then free text), or anything shaped like them.
//
// It works in two steps. Plan reads the source and decides, for every file, where it
// goes and how its header is rewritten, without writing anything; its report says what
// could not be decided (a note with no space, a link that would cross spaces, an id
// already taken) instead of guessing. Apply writes the plan. The source is only read.
//
// What the importer understands is the header. A field augmented brain knows (status,
// waiting_on, next, watch, area, repos, related…) is kept when the source has it, at
// the top of the header or under metadata, so a person's own conventions can be turned
// into those fields by a step of their own before importing. The body is kept as it is.
package importer

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/memory"
	"github.com/icalvete/augmented-brain/internal/paths"
)

// Options say how to import.
type Options struct {
	// Map sends notes to spaces by area: area id → space name. The key "*" is the space
	// of a note whose area is not in the map, or that has none. With one space and no
	// map, everything goes there.
	Map map[string]string
	// ReposBase is the folder the source's repos: paths are relative to. Each one is
	// rewritten relative to a work folder of the note's space when it falls inside one.
	ReposBase string
	// Language of the texts the importer writes into notes (en, es).
	Language string
	// Today is the date for notes whose source says none; zero means the file's date.
	Today time.Time
}

// Item is one source file and what becomes of it.
type Item struct {
	Source string       // path relative to the source folder, with forward slashes
	Space  string       // where it goes; empty when it is left out
	Note   *memory.Note // the note to write (nil for an archive file, copied as it is)
	Copy   bool         // an archive file, copied untouched into the space's archive/
}

// Issue is something the importer could not decide, or changed and says so.
type Issue struct {
	Source string // the file, relative to the source folder
	Kind   string // see the Kind constants
	Detail string // the field, link or path concerned
}

// Kinds of issue.
const (
	NotANote       = "not-a-note"       // no header: left out (an index, a stray file)
	Unreadable     = "unreadable"       // a header that cannot be read: left out
	ToProfile      = "to-profile"       // about the user: goes in the profile, by hand
	NoSpace        = "no-space"         // its area maps to no space: left out
	Taken          = "taken"            // the id is already a note of the space: left out
	Duplicate      = "duplicate"        // two source files with the same id: the second is left out
	CrossLink      = "cross-link"       // related to a note of another space or of none: dropped
	MissingArea    = "missing-area"     // its area is not a note of its space: dropped
	DroppedField   = "dropped-field"    // a header field augmented brain does not have
	TaskDefaults   = "task-defaults"    // a task with no status: parked, waiting on the user
	RepoOutside    = "repo-outside"     // a repository outside the space's work folders: kept as a full path
	UnknownType    = "unknown-type"     // no type, prefix or kind to go by: imported as a reference
	ArchiveNoSpace = "archive-no-space" // an archive file that belongs to no note: left out
)

// Plan is what Apply would write, and what it could not decide.
type Plan struct {
	Source string
	Items  []Item
	Issues []Issue
}

// Count returns, per space, how many notes of each type go there.
func (p *Plan) Count() map[string]map[string]int {
	out := map[string]map[string]int{}
	for _, it := range p.Items {
		if it.Space == "" {
			continue
		}
		if out[it.Space] == nil {
			out[it.Space] = map[string]int{}
		}
		kind := "archive"
		if it.Note != nil {
			kind = it.Note.Header.Type
		}
		out[it.Space][kind]++
	}
	return out
}

// Left returns the source files that are not imported.
func (p *Plan) Left() int {
	n := 0
	for _, it := range p.Items {
		if it.Space == "" {
			n++
		}
	}
	return n
}

var prefixes = []struct{ prefix, kind string }{
	{"task-", "task"}, {"area-", "area"}, {"ref-", "reference"}, {"reference-", "reference"},
	{"lesson-", "lesson"}, {"rule-", "rule"}, {"person-", "person"},
}

// claudeKinds are the four kinds of Claude Code's automatic memory.
var claudeKinds = map[string]string{"feedback": "rule", "reference": "reference", "project": "task"}

var linkText = regexp.MustCompile(`\[([^\]\n]+)\]\(([^)\s]+\.md)\)`)

// MakePlan reads the source folder and decides where each file goes.
func MakePlan(src string, c *config.Config, o Options) (*Plan, error) {
	info, err := os.Stat(src)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a folder", src)
	}
	if len(o.Map) == 0 {
		names := c.SpaceNames()
		if len(names) != 1 {
			return nil, fmt.Errorf("there are %d spaces: say which space each area goes to (a map file, with a line \"* = <space>\" for the rest)", len(names))
		}
		o.Map = map[string]string{"*": names[0]}
	}
	for area, sp := range o.Map {
		if _, ok := c.Spaces[sp]; !ok {
			return nil, fmt.Errorf("the map sends %s to space %q, which does not exist", area, sp)
		}
	}
	p := &Plan{Source: src}
	titles := map[string]string{} // file base name → the text of a link to it
	var notes []*source
	var archived []string
	err = filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel := filepath.ToSlash(mustRel(src, path))
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") && rel != "." {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(rel, ".md") {
			return nil
		}
		if strings.HasPrefix(rel, "archive/") {
			archived = append(archived, rel)
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, m := range linkText.FindAllStringSubmatch(string(data), -1) {
			base := strings.TrimSuffix(filepath.Base(m[2]), ".md")
			if _, seen := titles[base]; !seen || filepath.Base(rel) == memory.IndexFile {
				titles[base] = strings.TrimSpace(m[1])
			}
		}
		s, err := read(rel, data)
		switch {
		case errors.Is(err, errNoHeader):
			p.Items = append(p.Items, Item{Source: rel})
			p.Issues = append(p.Issues, Issue{rel, NotANote, ""})
		case err != nil:
			p.Items = append(p.Items, Item{Source: rel})
			p.Issues = append(p.Issues, Issue{rel, Unreadable, err.Error()})
		default:
			if fi, err := d.Info(); err == nil {
				s.modified = fi.ModTime()
			}
			notes = append(notes, s)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// First pass: the note each source becomes, and its space.
	type planned struct {
		src  *source
		note *memory.Note
		sp   string
	}
	var all []planned
	ids := map[string]string{} // id → space, of the notes being imported
	existing := map[string]*memory.Memory{}
	for name, sp := range c.Spaces {
		if m, _, err := memory.Load(sp.Memory); err == nil {
			existing[name] = m
		}
	}
	for _, s := range notes {
		n, kind, issues := s.convert(titles, o)
		p.Issues = append(p.Issues, issues...)
		if kind == "user" {
			p.Items = append(p.Items, Item{Source: s.rel})
			p.Issues = append(p.Issues, Issue{s.rel, ToProfile, ""})
			continue
		}
		sp := spaceOf(n, o.Map)
		if sp == "" {
			p.Items = append(p.Items, Item{Source: s.rel})
			p.Issues = append(p.Issues, Issue{s.rel, NoSpace, n.Header.Area})
			continue
		}
		if _, dup := ids[n.Header.ID]; dup {
			p.Items = append(p.Items, Item{Source: s.rel})
			p.Issues = append(p.Issues, Issue{s.rel, Duplicate, n.Header.ID})
			continue
		}
		if m := existing[sp]; m != nil {
			if _, taken := m.Notes[n.Header.ID]; taken {
				p.Items = append(p.Items, Item{Source: s.rel})
				p.Issues = append(p.Issues, Issue{s.rel, Taken, n.Header.ID})
				continue
			}
		}
		ids[n.Header.ID] = sp
		all = append(all, planned{s, n, sp})
	}

	// Second pass: links that only make sense inside one space, and repositories.
	for _, pl := range all {
		h := &pl.note.Header
		inSpace := func(id string) bool {
			if ids[id] == pl.sp {
				return true
			}
			if m := existing[pl.sp]; m != nil {
				_, ok := m.Notes[id]
				return ok
			}
			return false
		}
		if h.Area != "" && h.Type != "area" && !inSpace(h.Area) {
			p.Issues = append(p.Issues, Issue{pl.src.rel, MissingArea, h.Area})
			h.Area = ""
		}
		var related []string
		for _, r := range h.Related {
			if inSpace(r) {
				related = append(related, r)
			} else {
				p.Issues = append(p.Issues, Issue{pl.src.rel, CrossLink, r})
			}
		}
		h.Related = related
		h.Repos = repos(h.Repos, o.ReposBase, c.Spaces[pl.sp], func(r string) {
			p.Issues = append(p.Issues, Issue{pl.src.rel, RepoOutside, r})
		})
		p.Items = append(p.Items, Item{Source: pl.src.rel, Space: pl.sp, Note: pl.note})
	}

	// Archive files follow the note they belong to: archive/<id>-… goes with <id>.
	for _, rel := range archived {
		base := strings.TrimSuffix(filepath.Base(rel), ".md")
		sp, best := "", ""
		for id, s := range ids {
			if (base == id || strings.HasPrefix(base, id+"-")) && len(id) > len(best) {
				sp, best = s, id
			}
		}
		if sp == "" {
			sp = o.Map["*"]
		}
		if sp == "" {
			p.Items = append(p.Items, Item{Source: rel})
			p.Issues = append(p.Issues, Issue{rel, ArchiveNoSpace, ""})
			continue
		}
		p.Items = append(p.Items, Item{Source: rel, Space: sp, Copy: true})
	}
	sort.SliceStable(p.Items, func(i, j int) bool { return p.Items[i].Source < p.Items[j].Source })
	sort.SliceStable(p.Issues, func(i, j int) bool {
		if p.Issues[i].Source != p.Issues[j].Source {
			return p.Issues[i].Source < p.Issues[j].Source
		}
		return p.Issues[i].Kind < p.Issues[j].Kind
	})
	return p, nil
}

func spaceOf(n *memory.Note, m map[string]string) string {
	area := n.Header.Area
	if n.Header.Type == "area" {
		area = n.Header.ID
	}
	if sp, ok := m[area]; ok && area != "" {
		return sp
	}
	return m["*"]
}

// repos rewrites each repository relative to a work folder of the space, when it falls
// inside one; otherwise it keeps the full path (with ~) and says so.
func repos(list []string, base string, s *config.Space, outside func(string)) []string {
	var out []string
	for _, r := range list {
		r = strings.TrimSpace(r)
		if r == "" {
			continue
		}
		abs := r
		if !filepath.IsAbs(r) && !strings.HasPrefix(r, "~") {
			if base == "" {
				out = append(out, r) // nothing to rewrite against: as it was
				continue
			}
			abs = filepath.Join(base, filepath.FromSlash(r))
		}
		if resolved, err := paths.Resolve(abs); err == nil {
			abs = resolved
		}
		rel := ""
		for _, wf := range s.WorkFolders {
			if paths.Within(abs, wf) {
				if x, err := filepath.Rel(wf, abs); err == nil {
					rel = filepath.ToSlash(x)
					break
				}
			}
		}
		if rel == "" || rel == "." {
			outside(r)
			rel = paths.Tilde(abs)
		}
		out = append(out, rel)
	}
	return out
}

func mustRel(base, path string) string {
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return path
	}
	return rel
}

// ---- reading a source file ----

var errNoHeader = errors.New("no header")

// source is one file of the source memory: its header, loosely read, and its body.
type source struct {
	rel      string
	head     map[string]any
	body     string
	modified time.Time
}

func read(rel string, data []byte) (*source, error) {
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	if !bytes.HasPrefix(data, []byte("---\n")) {
		return nil, errNoHeader
	}
	rest := data[4:]
	end := bytes.Index(rest, []byte("\n---\n"))
	body := []byte{}
	switch {
	case end >= 0:
		body = rest[end+5:]
		rest = rest[:end]
	case bytes.HasSuffix(rest, []byte("\n---")):
		rest = rest[:len(rest)-4]
	default:
		return nil, errors.New("header not closed")
	}
	head := map[string]any{}
	if err := yaml.Unmarshal(rest, &head); err != nil {
		return nil, err
	}
	return &source{rel: rel, head: head, body: string(body)}, nil
}

// field reads a header field from the top of the header, or else from metadata.
func (s *source) field(key string) (any, bool) {
	if v, ok := s.head[key]; ok {
		return v, true
	}
	if m, ok := s.head["metadata"].(map[string]any); ok {
		v, ok := m[key]
		return v, ok
	}
	return nil, false
}

func (s *source) str(key string) string {
	v, _ := s.field(key)
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x)
	case time.Time:
		return x.Format("2006-01-02")
	case nil:
		return ""
	default:
		return strings.TrimSpace(fmt.Sprint(x))
	}
}

func (s *source) list(key string) []string {
	v, _ := s.field(key)
	var out []string
	switch x := v.(type) {
	case []any:
		for _, e := range x {
			if t := strings.TrimSpace(fmt.Sprint(e)); t != "" {
				out = append(out, t)
			}
		}
	case string:
		if t := strings.TrimSpace(x); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// known are the header fields augmented brain has, plus those of Claude Code's memory
// that the importer turns into them; anything else is dropped and reported.
var known = map[string]bool{
	"id": true, "type": true, "title": true, "summary": true, "area": true, "repos": true,
	"related": true, "created": true, "updated": true, "parent": true, "status": true,
	"waiting_on": true, "next": true, "watch": true,
	"name": true, "description": true, "metadata": true, "modified": true,
}

var defaults = map[string]map[string]string{
	"en": {"next": "Review this imported task and set its next step"},
	"es": {"next": "Revisar esta tarea importada y poner su siguiente paso"},
}

// convert turns a source file into a note of augmented brain. It returns the kind
// "user" for a note about the user, which belongs in the profile.
func (s *source) convert(titles map[string]string, o Options) (*memory.Note, string, []Issue) {
	var issues []Issue
	base := strings.TrimSuffix(filepath.Base(s.rel), ".md")
	id := s.str("id")
	if id == "" {
		id = s.str("name")
	}
	if id == "" || strings.ContainsAny(id, `/\ `) {
		id = base
	}
	claude := s.str("type")
	if m, ok := s.head["metadata"].(map[string]any); ok {
		if t, ok := m["type"].(string); ok {
			claude = t
		}
	}
	// A type of augmented brain at the top of the header wins (a step before importing
	// put it there); then the file name's prefix; Claude Code's own kind comes last, as
	// it is often not what the note is (a lesson saved as "feedback").
	kind := ""
	if t, _ := s.head["type"].(string); t != "profile" && t != "review" {
		if _, ok := memory.Folders[t]; ok {
			kind = t
		}
	}
	if kind == "" {
		for _, px := range prefixes {
			if strings.HasPrefix(id, px.prefix) {
				kind = px.kind
				break
			}
		}
	}
	if kind == "" {
		if claude == "user" {
			return nil, "user", nil
		}
		kind = claudeKinds[claude]
	}
	if kind == "" {
		kind = "reference"
		issues = append(issues, Issue{s.rel, UnknownType, claude})
	}

	title := s.str("title")
	if title == "" {
		title = titles[base]
	}
	if title == "" {
		title = firstLine(s.str("description"), 80)
	}
	if title == "" {
		title = strings.ReplaceAll(strings.TrimPrefix(id, kind+"-"), "-", " ")
	}
	day := o.Today
	if day.IsZero() {
		day = s.modified
	}
	updated := firstNonEmpty(dateOf(s.str("updated")), dateOf(s.str("modified")), day.Format("2006-01-02"))
	h := memory.Header{
		ID: id, Type: kind, Title: title,
		Summary: firstNonEmpty(s.str("summary"), firstLine(s.str("description"), 300)),
		Area:    s.str("area"), Repos: s.list("repos"), Related: s.list("related"),
		Created: firstNonEmpty(dateOf(s.str("created")), updated), Updated: updated,
	}
	if kind == "task" {
		h.Parent = s.str("parent")
		h.Status, h.WaitingOn, h.Next = s.str("status"), s.str("waiting_on"), s.str("next")
		if !contains(memory.Statuses, h.Status) {
			if h.Status != "" {
				issues = append(issues, Issue{s.rel, DroppedField, "status: " + h.Status})
			}
			h.Status = "parked"
			issues = append(issues, Issue{s.rel, TaskDefaults, ""})
		}
		lang := o.Language
		if defaults[lang] == nil {
			lang = "en"
		}
		if h.Status != "closed" {
			if h.WaitingOn == "" {
				h.WaitingOn = "me"
			}
			if h.Next == "" {
				h.Next = defaults[lang]["next"]
			}
		}
		h.Watch = s.watch()
	}
	for _, k := range sortedKeys(s.head) {
		if !known[k] {
			issues = append(issues, Issue{s.rel, DroppedField, k})
		}
	}
	if m, ok := s.head["metadata"].(map[string]any); ok {
		for _, k := range sortedKeys(m) {
			if !known[k] && k != "node_type" && k != "originSessionId" {
				issues = append(issues, Issue{s.rel, DroppedField, "metadata." + k})
			}
		}
	}
	folder := memory.Folders[kind]
	path := id + ".md"
	if folder != "" {
		path = folder + "/" + path
	}
	return &memory.Note{Path: path, Header: h, Body: strings.TrimLeft(s.body, "\n")}, kind, issues
}

func (s *source) watch() []memory.Watch {
	v, _ := s.field("watch")
	items, _ := v.([]any)
	var out []memory.Watch
	for _, it := range items {
		m, ok := it.(map[string]any)
		if !ok {
			continue
		}
		w := memory.Watch{What: strings.TrimSpace(fmt.Sprint(m["what"]))}
		switch d := m["date"].(type) {
		case time.Time:
			w.Date = d.Format("2006-01-02")
		default:
			w.Date = strings.TrimSpace(fmt.Sprint(d))
		}
		w.Done, _ = m["done"].(bool)
		if w.Date != "" && w.What != "" {
			out = append(out, w)
		}
	}
	return out
}

var dateRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}`)

func dateOf(s string) string { return dateRe.FindString(s) }

func firstLine(s string, max int) string {
	sc := bufio.NewScanner(strings.NewReader(s))
	if !sc.Scan() {
		return ""
	}
	line := strings.TrimSpace(sc.Text())
	if r := []rune(line); len(r) > max {
		line = strings.TrimSpace(string(r[:max-1])) + "…"
	}
	return line
}

func firstNonEmpty(xs ...string) string {
	for _, x := range xs {
		if x != "" {
			return x
		}
	}
	return ""
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
