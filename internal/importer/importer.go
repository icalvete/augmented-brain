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
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/guard"
	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/memory"
	"github.com/icalvete/augmented-brain/internal/paths"
)

// Options say how to import.
type Options struct {
	// Map sends notes to spaces: a note's own id, or else its area id → space name, or
	// "shared" for the shared notes (rules and knowledge for every space). The key "*" is
	// the space of a note that is not in the map by either. With one space and no map,
	// everything goes there.
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
	Copy   bool         // an archive or docs/ file, copied untouched into the same place in the space
	Inbox  []string     // inbox.md: its captures, added to the space's inbox (dated lines)
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
	NotShared      = "not-shared"       // sent to the shared notes, but a task, area or person: left out
	Taken          = "taken"            // the id is already a note of the space: left out
	Duplicate      = "duplicate"        // two source files with the same id: the second is left out
	CrossLink      = "cross-link"       // related to a note of another space or of none: dropped
	MissingArea    = "missing-area"     // its area is not a note of its space: dropped
	DroppedField   = "dropped-field"    // a header field augmented brain does not have
	TaskDefaults   = "task-defaults"    // a task with no status: parked, waiting on the user
	RepoOutside    = "repo-outside"     // a repository outside the space's work folders: kept as a full path
	UnknownType    = "unknown-type"     // no type, prefix or kind to go by: imported as a reference
	ArchiveNoSpace = "archive-no-space" // an archive file that belongs to no note: left out
	Secret         = "secret"           // a secret written in the file: imported, but the backup will refuse it
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
		switch {
		case it.Note != nil:
			out[it.Space][it.Note.Header.Type]++
		case it.Inbox != nil:
			out[it.Space]["inbox"] += len(it.Inbox)
		case strings.HasPrefix(it.Source, "docs/"):
			out[it.Space]["docs"]++
		default:
			out[it.Space]["archive"]++
		}
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

// kindOfID returns the type an id's prefix says, or "". A review is never imported by
// its prefix, and "reference-" is taken for a reference as well as "ref-".
func kindOfID(id string) string {
	if strings.HasPrefix(id, "reference-") {
		return "reference"
	}
	if t := memory.TypeOfID(id); t != "review" {
		return t
	}
	return ""
}

// claudeKinds are the four kinds of Claude Code's automatic memory.
var claudeKinds = map[string]string{"feedback": "rule", "reference": "reference", "project": "task"}

var linkText = regexp.MustCompile(`\[([^\]\n]+)\]\(([^)\s]+\.md)\)`)

// planner holds what MakePlan learns as it goes, for its steps to share.
type planner struct {
	c        *config.Config
	o        Options
	p        *Plan
	titles   map[string]string         // file base name → the text of a link to it
	notes    []*source                 // the source files with a header
	archived []string                  // the files under archive/ and docs/
	inbox    []string                  // the captures of inbox.md, as dated lines
	ids      map[string]string         // id → space, of the notes being imported
	existing map[string]*memory.Memory // the notes already in each space
}

// planned is a note on its way to a space.
type planned struct {
	src  *source
	note *memory.Note
	sp   string
}

// MakePlan reads the source folder and decides where each file goes.
func MakePlan(src string, c *config.Config, o Options) (*Plan, error) {
	info, err := os.Stat(src)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a folder", src)
	}
	if o.Map, err = spaceMap(c, o.Map); err != nil {
		return nil, err
	}
	pl := &planner{c: c, o: o, p: &Plan{Source: src}, titles: map[string]string{}, ids: map[string]string{}}
	if err := pl.scan(src); err != nil {
		return nil, err
	}
	pl.loadSpaces()
	for _, n := range pl.place() {
		pl.fixLinks(n)
	}
	pl.placeArchives()
	pl.placeInbox()
	pl.p.sort()
	return pl.p, nil
}

// spaceMap returns the map of areas to spaces, checked: with one space and no map,
// everything goes there; with several, a map is needed.
func spaceMap(c *config.Config, m map[string]string) (map[string]string, error) {
	if len(m) == 0 {
		names := c.SpaceNames()
		if len(names) != 1 {
			return nil, fmt.Errorf("there are %d spaces: say which space each area goes to (a map file, with a line \"* = <space>\" for the rest)", len(names))
		}
		m = map[string]string{"*": names[0]}
	}
	for area, sp := range m {
		if _, ok := c.Spaces[sp]; !ok && sp != config.SharedFolder {
			return nil, fmt.Errorf("the map sends %s to space %q, which does not exist", area, sp)
		}
	}
	return m, nil
}

// leave records a source file that is not imported, and why.
func (pl *planner) leave(rel, kind, detail string) {
	pl.p.Items = append(pl.p.Items, Item{Source: rel})
	pl.p.Issues = append(pl.p.Issues, Issue{rel, kind, detail})
}

// scan walks the source folder: it reads every note, sets the archive files aside,
// and collects the text of every link, which may be the only title a note has.
func (pl *planner) scan(src string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
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
		if strings.HasPrefix(rel, "docs/") && strings.Count(rel, "/") >= 2 {
			pl.archived = append(pl.archived, rel) // a task's documents, of any kind
			return nil
		}
		if !strings.HasSuffix(rel, ".md") {
			return nil
		}
		if strings.HasPrefix(rel, "archive/") {
			pl.archived = append(pl.archived, rel)
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if rel == "inbox.md" {
			pl.inbox = captures(data, pl.o.Today)
			return nil
		}
		pl.linkTitles(rel, data)
		if what := guard.FindSecret(string(data)); what != "" {
			pl.p.Issues = append(pl.p.Issues, Issue{rel, Secret, what})
		}
		s, err := read(rel, data)
		switch {
		case errors.Is(err, memory.ErrNoHeader):
			pl.leave(rel, NotANote, "")
		case err != nil:
			pl.leave(rel, Unreadable, err.Error())
		default:
			if fi, err := d.Info(); err == nil {
				s.modified = fi.ModTime()
			}
			pl.notes = append(pl.notes, s)
		}
		return nil
	})
}

// linkTitles keeps the text of each link in a file as the title of the file it points
// to: the first one found, unless the source's index has one, which wins.
func (pl *planner) linkTitles(rel string, data []byte) {
	for _, m := range linkText.FindAllStringSubmatch(string(data), -1) {
		base := strings.TrimSuffix(filepath.Base(m[2]), ".md")
		if _, seen := pl.titles[base]; !seen || filepath.Base(rel) == memory.IndexFile {
			pl.titles[base] = strings.TrimSpace(m[1])
		}
	}
}

// loadSpaces reads the notes each space already has, so an id is never taken twice.
func (pl *planner) loadSpaces() {
	pl.existing = map[string]*memory.Memory{}
	for name, sp := range pl.c.Spaces {
		if m, _, err := memory.Load(sp.Memory); err == nil {
			pl.existing[name] = m
		}
	}
	if m, _, err := memory.LoadShared(pl.c.SharedDir()); err == nil {
		pl.existing[config.SharedFolder] = m
	}
}

// place turns each source into its note and chooses its space. A note about the user,
// one with no space, a second one with the same id, or one whose id the space already
// has is left out.
func (pl *planner) place() []planned {
	var all []planned
	for _, s := range pl.notes {
		n, kind, issues := s.convert(pl.titles, pl.o)
		pl.p.Issues = append(pl.p.Issues, issues...)
		if kind == "user" {
			pl.leave(s.rel, ToProfile, "")
			continue
		}
		sp := spaceOf(n, pl.o.Map)
		if sp == "" {
			pl.leave(s.rel, NoSpace, n.Header.Area)
			continue
		}
		if sp == config.SharedFolder && !slices.Contains(memory.SharedTypes, n.Header.Type) {
			pl.leave(s.rel, NotShared, n.Header.Type)
			continue
		}
		if _, dup := pl.ids[n.Header.ID]; dup {
			pl.leave(s.rel, Duplicate, n.Header.ID)
			continue
		}
		if m := pl.existing[sp]; m != nil {
			if _, taken := m.Notes[n.Header.ID]; taken {
				pl.leave(s.rel, Taken, n.Header.ID)
				continue
			}
		}
		pl.ids[n.Header.ID] = sp
		all = append(all, planned{s, n, sp})
	}
	return all
}

// fixLinks drops the area and the related notes that are not in the note's space or in
// the shared notes (links only make sense inside one space), rewrites its repositories,
// and adds it to the plan. It runs once every note has its space.
func (pl *planner) fixLinks(n planned) {
	h := &n.note.Header
	in := func(sp, id string) bool {
		if pl.ids[id] == sp {
			return true
		}
		m := pl.existing[sp]
		if m == nil {
			return false
		}
		_, ok := m.Notes[id]
		return ok
	}
	inSpace := func(id string) bool { return in(n.sp, id) || in(config.SharedFolder, id) }
	if h.Area != "" && h.Type != "area" && (n.sp == config.SharedFolder || !in(n.sp, h.Area)) {
		pl.p.Issues = append(pl.p.Issues, Issue{n.src.rel, MissingArea, h.Area})
		h.Area = ""
	}
	var related []string
	for _, r := range h.Related {
		if inSpace(r) {
			related = append(related, r)
		} else {
			pl.p.Issues = append(pl.p.Issues, Issue{n.src.rel, CrossLink, r})
		}
	}
	h.Related = related
	if sp := pl.c.Spaces[n.sp]; sp != nil {
		h.Repos = repos(h.Repos, pl.o.ReposBase, sp, func(r string) {
			pl.p.Issues = append(pl.p.Issues, Issue{n.src.rel, RepoOutside, r})
		})
	} else { // shared: no work folder to be relative to, so full paths
		h.Repos = repos(h.Repos, pl.o.ReposBase, &config.Space{}, func(string) {})
	}
	pl.p.Items = append(pl.p.Items, Item{Source: n.src.rel, Space: n.sp, Note: n.note})
}

// placeArchives sends each archive file with the note it belongs to: archive/<id>-…
// goes with <id> (the longest id that fits), docs/<id>/… with <id>, or else to the
// space of the rest.
func (pl *planner) placeArchives() {
	for _, rel := range pl.archived {
		base := strings.TrimSuffix(filepath.Base(rel), ".md")
		if strings.HasPrefix(rel, "docs/") {
			base = strings.Split(rel, "/")[1]
		}
		sp, best := "", ""
		for id, s := range pl.ids {
			if (base == id || strings.HasPrefix(base, id+"-")) && len(id) > len(best) {
				sp, best = s, id
			}
		}
		if sp == "" {
			sp = pl.o.Map["*"]
		}
		if sp == "" {
			pl.leave(rel, ArchiveNoSpace, "")
			continue
		}
		pl.p.Items = append(pl.p.Items, Item{Source: rel, Space: sp, Copy: true})
	}
}

// placeInbox sends the captures of inbox.md to the space of the rest ("*"), or else to
// the default space: the inbox is the person's, not an area's.
func (pl *planner) placeInbox() {
	if len(pl.inbox) == 0 {
		return
	}
	sp := pl.o.Map["*"]
	if sp == "" || sp == config.SharedFolder {
		sp = pl.c.DefaultSpace
	}
	if sp == "" {
		pl.leave("inbox.md", NoSpace, "inbox")
		return
	}
	pl.p.Items = append(pl.p.Items, Item{Source: "inbox.md", Space: sp, Inbox: pl.inbox})
}

// captures returns the lines of an inbox that are captures ("- …" at the start of a
// line), as dated lines: one that has no date gets today's (or the file's day).
func captures(data []byte, today time.Time) []string {
	if today.IsZero() {
		today = time.Now()
	}
	var out []string
	for _, line := range strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
		if !strings.HasPrefix(line, "- ") || strings.TrimSpace(line[2:]) == "" {
			continue
		}
		date, text, ok := memory.ParseDatedLine(line)
		if !ok {
			date, text = today.Format(memory.DateLayout), strings.TrimSpace(line[2:])
		}
		out = append(out, strings.TrimSuffix(memory.DatedLine(date, text), "\n"))
	}
	return out
}

// sort orders the plan by source file, and each file's issues by kind.
func (p *Plan) sort() {
	sort.SliceStable(p.Items, func(i, j int) bool { return p.Items[i].Source < p.Items[j].Source })
	sort.SliceStable(p.Issues, func(i, j int) bool {
		if p.Issues[i].Source != p.Issues[j].Source {
			return p.Issues[i].Source < p.Issues[j].Source
		}
		return p.Issues[i].Kind < p.Issues[j].Kind
	})
}

func spaceOf(n *memory.Note, m map[string]string) string {
	if sp, ok := m[n.Header.ID]; ok {
		return sp
	}
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

// source is one file of the source memory: its header, loosely read, and its body.
type source struct {
	rel      string
	head     map[string]any
	body     string
	modified time.Time
}

// read splits a source file as memory does a note, but reads its header loosely: any
// field, of any shape. A file with no header gives memory.ErrNoHeader.
func read(rel string, data []byte) (*source, error) {
	header, body, err := memory.SplitHeader(data)
	if errors.Is(err, memory.ErrNotClosed) {
		return nil, errors.New("header not closed") // the shorter words of the import report
	}
	if err != nil {
		return nil, err
	}
	head := map[string]any{}
	if err := yaml.Unmarshal(header, &head); err != nil {
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
		return x.Format(memory.DateLayout)
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

// convert turns a source file into a note of augmented brain. It returns the kind
// "user" for a note about the user, which belongs in the profile.
func (s *source) convert(titles map[string]string, o Options) (*memory.Note, string, []Issue) {
	base := strings.TrimSuffix(filepath.Base(s.rel), ".md")
	id := s.id(base)
	kind := s.kind(id)
	if kind == "user" {
		return nil, "user", nil
	}
	var issues []Issue
	if kind == "" {
		kind = "reference"
		issues = append(issues, Issue{s.rel, UnknownType, s.claudeType()})
	}
	updated := cmp.Or(dateOf(s.str("updated")), dateOf(s.str("modified")), s.day(o.Today))
	h := memory.Header{
		ID: id, Type: kind, Title: s.title(id, kind, titles[base]),
		Summary: cmp.Or(s.str("summary"), firstLine(s.str("description"), 300)),
		Area:    s.str("area"), Repos: s.list("repos"), Related: s.list("related"),
		Created: cmp.Or(dateOf(s.str("created")), updated), Updated: updated,
	}
	if kind == "task" {
		issues = append(issues, s.taskFields(&h, o.Language)...)
	}
	issues = append(issues, s.droppedFields()...)
	return &memory.Note{Path: memory.PathFor(kind, id), Header: h, Body: strings.TrimLeft(s.body, "\n")}, kind, issues
}

// id is the note's id: the source's id or name, unless it cannot be a file name, and
// then the file's own name.
func (s *source) id(base string) string {
	id := cmp.Or(s.str("id"), s.str("name"))
	if id == "" || strings.ContainsAny(id, `/\ `) {
		return base
	}
	return id
}

// kind is the type of note the source becomes. A type of augmented brain at the top of
// the header wins (a step before importing put it there); then the id's prefix; Claude
// Code's own kind comes last, as it is often not what the note is (a lesson saved as
// "feedback"). It is "user" for a note about the user, and "" when nothing says.
func (s *source) kind(id string) string {
	if t, _ := s.head["type"].(string); t != "profile" && t != "review" {
		if _, ok := memory.Types[t]; ok {
			return t
		}
	}
	if k := kindOfID(id); k != "" {
		return k
	}
	claude := s.claudeType()
	if claude == "user" {
		return "user"
	}
	return claudeKinds[claude]
}

// claudeType is the kind Claude Code's memory gives the note: under metadata, or else
// at the top of the header.
func (s *source) claudeType() string {
	if m, ok := s.head["metadata"].(map[string]any); ok {
		if t, ok := m["type"].(string); ok {
			return t
		}
	}
	return s.str("type")
}

// title is the source's title, or else the text of a link to it, the first line of its
// description, or its id in words.
func (s *source) title(id, kind, linked string) string {
	return cmp.Or(s.str("title"), linked, firstLine(s.str("description"), 80),
		strings.ReplaceAll(strings.TrimPrefix(id, kind+"-"), "-", " "))
}

// day is the date of a note whose source says none: the import's, or else the file's.
func (s *source) day(today time.Time) string {
	if today.IsZero() {
		today = s.modified
	}
	return today.Format(memory.DateLayout)
}

// taskFields fills the fields only tasks have. A task with no valid status is parked,
// and a live one gets someone to wait on and a next step, so it is a valid note.
func (s *source) taskFields(h *memory.Header, lang string) []Issue {
	var issues []Issue
	h.Parent = s.str("parent")
	h.Status, h.WaitingOn, h.Next = s.str("status"), s.str("waiting_on"), s.str("next")
	if !slices.Contains(memory.Statuses, h.Status) {
		if h.Status != "" {
			issues = append(issues, Issue{s.rel, DroppedField, "status: " + h.Status})
		}
		h.Status = "parked"
		issues = append(issues, Issue{s.rel, TaskDefaults, ""})
	}
	if h.Status != "closed" {
		if h.WaitingOn == "" {
			h.WaitingOn = "me"
		}
		if h.Next == "" {
			h.Next = i18n.T(lang, "note.imported.next")
		}
	}
	h.Watch = s.watch()
	return issues
}

// droppedFields reports the header fields, at the top or under metadata, that
// augmented brain has no place for.
func (s *source) droppedFields() []Issue {
	var issues []Issue
	for _, k := range slices.Sorted(maps.Keys(s.head)) {
		if !known[k] {
			issues = append(issues, Issue{s.rel, DroppedField, k})
		}
	}
	if m, ok := s.head["metadata"].(map[string]any); ok {
		for _, k := range slices.Sorted(maps.Keys(m)) {
			if !known[k] && k != "node_type" && k != "originSessionId" {
				issues = append(issues, Issue{s.rel, DroppedField, "metadata." + k})
			}
		}
	}
	return issues
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
			w.Date = d.Format(memory.DateLayout)
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
