// Package fresh tells a session which tasks changed outside it since it last looked.
//
// The index a session reads at its start is a snapshot: another session, the panel or
// a hand edit may change a task a minute later, and the session would go on trusting
// what it read. So each Claude process keeps a "seen" file: for every task of its
// space, the version it last saw (modification time, size, a hash of the content, and
// a fingerprint of what the index says about it).
//
//   - SessionStart takes the first one (Baseline), before the index is written into the
//     session, so a change in between is at worst reported twice, never missed.
//   - The MCP server, which holds the notes lock while a tool runs, compares before the
//     tool (Check: changes from outside) and after it (Absorb: what the tool itself
//     wrote, which no one else could write meanwhile). Its tools are this session's own
//     writes, so they are never reported back to it.
//   - UserPromptSubmit compares before each answer (Check).
//
// The file is named after the Claude process, not the session: after /clear Claude
// Code gives the hooks a new session id but keeps the same MCP server, whose
// environment still holds the old one. The hooks get the process in CLAUDE_PID; the
// MCP server is a direct child of it. Without CLAUDE_PID nothing is tracked.
//
// What is reported: a change of what the index says (status, who it waits on, next
// step, title, summary, area, parent) of any task; any change of a task this session
// read or wrote (it is the one it works with); a new task; a task that is gone. A
// journal entry of another session in a task this session never touched is not news.
//
// Every operation reads one small JSON file and stats the tasks: well under a
// millisecond for a hundred tasks; only files whose time or size moved are read.
package fresh

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/icalvete/augmented-brain/internal/format"
	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/memory"
	"github.com/icalvete/augmented-brain/internal/paths"
)

// Dir is the folder of the seen files, inside the configuration folder's state.
const Dir = "state/seen"

// Racy is how close to the time a version was recorded a file's modification time may
// be and still hide a change: some file systems keep whole seconds (or two), so a
// write right after the one recorded can leave time and size alike. Such files are
// hashed every time.
const Racy = 2 * time.Second

// keep is how long a seen file of a process that never ended is kept.
const keep = 30 * 24 * time.Hour

// Version is what a session last saw of one task.
type Version struct {
	ModTime int64  `json:"mtime"` // UnixNano
	Size    int64  `json:"size"`
	Hash    string `json:"hash"`
	Index   string `json:"index"`             // what the index says, see indexOf
	Touched bool   `json:"touched,omitempty"` // read or written by this session
}

// Seen is a process's seen file.
type Seen struct {
	Session string             `json:"session,omitempty"` // the session id when it was taken
	Space   string             `json:"space"`
	Format  int                `json:"format"` // the notes' format version
	Taken   time.Time          `json:"taken"`  // when the versions were last brought up to date
	Tasks   map[string]Version `json:"tasks"`
}

// Change is a task that changed outside the session.
type Change struct {
	ID        string
	Kind      string // "new", "gone" or "changed"
	Title     string
	Status    string
	WaitingOn string
	Next      string
}

// Report is what Check found.
type Report struct {
	Changes []Change
	// Format is true when the notes moved to another format version meanwhile: every
	// task may have been rewritten, so the list is not worth giving.
	Format bool
}

// Empty reports whether there is nothing to tell.
func (r Report) Empty() bool { return len(r.Changes) == 0 && !r.Format }

// Key names the seen file of a Claude process; "" when pid is not one.
func Key(pid string) string {
	n, err := strconv.Atoi(strings.TrimSpace(pid))
	if err != nil || n <= 1 {
		return ""
	}
	return "pid-" + strconv.Itoa(n)
}

// HookKey is the key of the process a hook runs under: CLAUDE_PID, which Claude Code
// gives its hooks. "" when it is missing.
func HookKey(getenv func(string) string) string { return Key(getenv("CLAUDE_PID")) }

// ServerKey is the key of the process an MCP server runs under: its parent, when that
// parent is Claude Code (which sets CLAUDECODE in the server's environment).
func ServerKey(getenv func(string) string, ppid int) string {
	if getenv("CLAUDECODE") == "" {
		return ""
	}
	return Key(strconv.Itoa(ppid))
}

func file(cfgDir, key string) string {
	return filepath.Join(cfgDir, filepath.FromSlash(Dir), key+".json")
}

// Load reads a seen file; ok is false when there is none or it cannot be read.
func Load(cfgDir, key string) (Seen, bool) {
	var s Seen
	if key == "" {
		return s, false
	}
	data, err := os.ReadFile(file(cfgDir, key))
	if err != nil || json.Unmarshal(data, &s) != nil || s.Tasks == nil {
		return Seen{}, false
	}
	return s, true
}

func save(cfgDir, key string, s Seen) error {
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(file(cfgDir, key)), 0o700); err != nil {
		return err
	}
	return paths.WriteAtomic(file(cfgDir, key), data, 0o600)
}

// Remove deletes a process's seen file (SessionEnd). Nothing there is fine.
func Remove(cfgDir, key string) {
	if key != "" {
		_ = os.Remove(file(cfgDir, key))
	}
}

// Prune deletes the seen files not brought up to date for 30 days: processes that
// ended without SessionEnd.
func Prune(cfgDir string, now time.Time) {
	dir := filepath.Join(cfgDir, filepath.FromSlash(Dir))
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if fi, err := e.Info(); err == nil && now.Sub(fi.ModTime()) > keep {
			_ = os.Remove(filepath.Join(dir, e.Name()))
		}
	}
}

// Space is where the tasks of a space are and which format its notes are in.
type Space struct {
	Name   string
	Tasks  string // the tasks folder
	Format int
}

// Baseline records every task as seen now: what the session has just been given at
// its start. When the same process already had a seen file of the same space and
// keepTouched is set (resume, compact), the tasks it had touched stay touched.
func Baseline(cfgDir, key, session string, sp Space, now time.Time, keepTouched bool) error {
	if key == "" {
		return nil
	}
	old, had := Load(cfgDir, key)
	cur, _ := scan(sp.Tasks, nil, time.Time{})
	if had && keepTouched && old.Space == sp.Name {
		for id, v := range old.Tasks {
			if c, ok := cur[id]; ok && v.Touched {
				c.Touched = true
				cur[id] = c
			}
		}
	}
	return save(cfgDir, key, Seen{Session: session, Space: sp.Name, Format: sp.Format, Taken: now, Tasks: cur})
}

// Check compares the tasks with what the process last saw, returns what changed
// outside it, and records the tasks as seen. With no seen file, or one of another
// space (the session moved), it only takes a new one, silently.
func Check(cfgDir, key string, sp Space, now time.Time) (Report, error) {
	if key == "" {
		return Report{}, nil
	}
	old, ok := Load(cfgDir, key)
	if !ok || old.Space != sp.Name {
		return Report{}, Baseline(cfgDir, key, old.Session, sp, now, false)
	}
	cur, err := scan(sp.Tasks, old.Tasks, old.Taken)
	if err != nil {
		return Report{}, err
	}
	var r Report
	if old.Format != 0 && sp.Format != 0 && old.Format != sp.Format {
		r.Format = true
	}
	for id, c := range cur {
		o, had := old.Tasks[id]
		switch {
		case !had:
			r.Changes = append(r.Changes, change(id, "new", c))
		case o.Hash == c.Hash:
			// unchanged
		case o.Touched || o.Index != c.Index:
			r.Changes = append(r.Changes, change(id, "changed", c))
		}
		c.Touched = had && o.Touched
		cur[id] = c
	}
	for id := range old.Tasks {
		if _, still := cur[id]; !still {
			r.Changes = append(r.Changes, Change{ID: id, Kind: "gone"})
		}
	}
	if r.Format {
		r.Changes = nil
	}
	sort.Slice(r.Changes, func(i, j int) bool { return r.Changes[i].ID < r.Changes[j].ID })
	old.Tasks, old.Format, old.Taken = cur, sp.Format, now
	return r, save(cfgDir, key, old)
}

// Absorb records as seen, and touched, every task that changed since the last Check or
// Absorb, plus the ones named in touched (read by the session). The MCP server calls
// it after a tool, still holding the notes lock: whatever changed meanwhile was that
// tool's own doing.
func Absorb(cfgDir, key string, sp Space, now time.Time, touched []string) error {
	if key == "" {
		return nil
	}
	old, ok := Load(cfgDir, key)
	if !ok || old.Space != sp.Name {
		return nil // no baseline to keep up to date: the next Check takes one
	}
	cur, err := scan(sp.Tasks, old.Tasks, old.Taken)
	if err != nil {
		return err
	}
	read := map[string]bool{}
	for _, id := range touched {
		read[id] = true
	}
	for id, c := range cur {
		o, had := old.Tasks[id]
		c.Touched = (had && o.Touched) || read[id] || !had || o.Hash != c.Hash
		cur[id] = c
	}
	old.Tasks, old.Format, old.Taken = cur, sp.Format, now
	return save(cfgDir, key, old)
}

func change(id, kind string, v Version) Change {
	c := Change{ID: id, Kind: kind}
	var h memory.Header
	if yaml.Unmarshal([]byte(v.Index), &h) == nil {
		c.Title, c.Status, c.WaitingOn, c.Next = h.Title, h.Status, h.WaitingOn, h.Next
	}
	return c
}

// scan reads the version of every task in dir. A file whose time and size match prev,
// and whose time is not within Racy of when prev was taken, is not read again.
func scan(dir string, prev map[string]Version, taken time.Time) (map[string]Version, error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]Version{}, nil
	}
	if err != nil {
		return nil, err
	}
	out := make(map[string]Version, len(entries))
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".md") || strings.HasPrefix(name, ".") {
			continue
		}
		id := strings.TrimSuffix(name, ".md")
		fi, err := e.Info()
		if err != nil {
			if p, ok := prev[id]; ok {
				out[id] = p // gone for a moment (a rename on Windows): unchanged
			}
			continue
		}
		mt, size := fi.ModTime().UnixNano(), fi.Size()
		if p, ok := prev[id]; ok && p.ModTime == mt && p.Size == size && !taken.IsZero() &&
			time.Duration(taken.UnixNano()-mt) > Racy {
			out[id] = p
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			if p, ok := prev[id]; ok {
				out[id] = p
			}
			continue
		}
		sum := sha256.Sum256(data)
		out[id] = Version{ModTime: mt, Size: size, Hash: hex.EncodeToString(sum[:]), Index: indexOf(data)}
	}
	return out, nil
}

// indexOf is what the index says about a task, as a small YAML document: the fields a
// session reads at its start and could act on. "" when the header cannot be read.
func indexOf(data []byte) string {
	head, _, err := memory.SplitHeader(data)
	if err != nil {
		return ""
	}
	var h memory.Header
	if yaml.Unmarshal(head, &h) != nil {
		return ""
	}
	out, err := yaml.Marshal(struct {
		Title     string `yaml:"title,omitempty"`
		Summary   string `yaml:"summary,omitempty"`
		Area      string `yaml:"area,omitempty"`
		Parent    string `yaml:"parent,omitempty"`
		Status    string `yaml:"status,omitempty"`
		WaitingOn string `yaml:"waiting_on,omitempty"`
		Next      string `yaml:"next,omitempty"`
	}{h.Title, h.Summary, h.Area, h.Parent, h.Status, h.WaitingOn, h.Next})
	if err != nil {
		return ""
	}
	return string(out)
}

// MaxLines is the most changes a report lists; the rest are counted.
const MaxLines = 8

// Text is the note for the assistant, in the configuration's language; "" when there
// is nothing to tell.
func Text(lang string, r Report) string {
	if r.Format {
		return i18n.T(lang, "fresh.format")
	}
	if len(r.Changes) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(i18n.T(lang, "fresh.head"))
	for i, c := range r.Changes {
		if i == MaxLines {
			b.WriteString(i18n.T(lang, "fresh.more", len(r.Changes)-MaxLines))
			break
		}
		switch c.Kind {
		case "gone":
			b.WriteString(i18n.T(lang, "fresh.gone", oneLine(c.ID)))
		case "new":
			b.WriteString(i18n.T(lang, "fresh.new", oneLine(c.ID), oneLine(c.Title), oneLine(c.Status)))
		default:
			b.WriteString(i18n.T(lang, "fresh.changed", oneLine(c.ID), oneLine(c.Title), oneLine(c.Status), oneLine(c.WaitingOn), oneLine(c.Next)))
		}
	}
	return b.String()
}

// oneLine keeps a field on one line and short: it comes from a note anyone may have
// written.
func oneLine(s string) string {
	s = memory.OneLine(s)
	if r := []rune(s); len(r) > 160 {
		s = string(r[:157]) + "..."
	}
	return s
}

// SpaceOf is the Space of a memory folder.
func SpaceOf(name, memDir string) Space {
	v, _ := format.Of(memDir)
	return Space{Name: name, Tasks: filepath.Join(memDir, memory.Types["task"].Folder), Format: v}
}
