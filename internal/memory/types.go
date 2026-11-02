package memory

import "strings"

// TypeInfo is what the program knows of one type of note: the folder it lives in and
// how its ids start.
type TypeInfo struct {
	Folder string // relative to the memory folder; empty for the top
	Prefix string // the start of every id the program makes for it; empty for the profile
}

// Types are the types of note. Every path and id the program builds comes from here, so
// the layout of a space is written down once.
var Types = map[string]TypeInfo{
	"area":      {"areas", "area-"},
	"task":      {"tasks", "task-"},
	"reference": {"knowledge", "ref-"},
	"lesson":    {"knowledge", "lesson-"},
	"rule":      {"rules", "rule-"},
	"review":    {"reviews", "review-"},
	"person":    {"people", "person-"},
	"profile":   {"", ""},
}

// PathFor returns where a note of that type and id lives, relative to the memory
// folder and with forward slashes: "tasks/task-x.md".
func PathFor(typ, id string) string {
	if f := Types[typ].Folder; f != "" {
		return f + "/" + id + ".md"
	}
	return id + ".md"
}

// TypeOfID returns the type whose id prefix id starts with, or "" when none does.
// No prefix is the start of another, so at most one type matches.
func TypeOfID(id string) string {
	for typ, t := range Types {
		if t.Prefix != "" && strings.HasPrefix(id, t.Prefix) {
			return typ
		}
	}
	return ""
}
