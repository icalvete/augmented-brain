package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/paths"
)

// IndexFile is the index loaded at the start of every session.
const IndexFile = "MEMORY.md"

// Index builds the text of MEMORY.md: one line per profile, rule and area, and the live
// tasks grouped by area. Closed and someday tasks stay out to keep it short.
func (m *Memory) Index(lang string) string {
	// Only the program writes the index, so its headings can follow the user's language
	// without the program ever reading them.
	l := func(key string) string { return i18n.T(lang, "index."+key) }
	var b strings.Builder
	line := func(n *Note, extra string) {
		fmt.Fprintf(&b, "- [%s](%s)", n.Header.Title, n.Path)
		if n.Header.Summary != "" {
			fmt.Fprintf(&b, " — %s", n.Header.Summary)
		}
		b.WriteString(extra + "\n")
	}
	section := func(title string, notes []*Note) {
		fmt.Fprintf(&b, "\n## %s\n", title)
		if len(notes) == 0 {
			fmt.Fprintf(&b, "- (%s)\n", l("none"))
		}
		for _, n := range notes {
			line(n, "")
		}
	}

	fmt.Fprintf(&b, "# %s\n\n%s\n", l("title"), l("generated"))
	if m.Shared {
		fmt.Fprintf(&b, "\n%s\n", l("shared"))
		section(l("rules"), m.OfType("rule"))
		section(l("knowledge"), append(m.OfType("reference"), m.OfType("lesson")...))
		return b.String()
	}
	if p := m.OfType("profile"); len(p) > 0 {
		section(l("profile"), p)
	}
	section(l("rules"), m.OfType("rule"))
	areas := m.OfType("area")
	section(l("areas"), areas)
	if people := m.OfType("person"); len(people) > 0 {
		section(l("people"), people)
	}

	fmt.Fprintf(&b, "\n## %s\n", l("tasks"))
	byArea := map[string][]*Note{}
	someday := 0
	for _, t := range m.OfType("task") {
		switch t.Header.Status {
		case "closed":
		case "someday":
			someday++
		default:
			// A subtask goes under its parent, when the parent is listed here.
			if m.Nested(t) {
				continue
			}
			byArea[t.Header.Area] = append(byArea[t.Header.Area], t)
		}
	}
	if len(byArea) == 0 {
		fmt.Fprintf(&b, "- (%s)\n", l("none"))
	}
	groups := make([]string, 0, len(byArea))
	for a := range byArea {
		groups = append(groups, a)
	}
	sort.Strings(groups) // "" (no area) sorts first
	taskLine := func(t *Note, indent string) {
		b.WriteString(indent)
		extra := " · " + t.Header.Status
		if done, total := StepCount(t.Body); total > 0 {
			extra += " · " + fmt.Sprintf(l("steps"), done, total)
		}
		line(t, fmt.Sprintf("%s · %s %s · %s: %s", extra, l("waits"), t.Header.WaitingOn, l("next"), t.Header.Next))
	}
	for _, a := range groups {
		title := l("noarea")
		if n, ok := m.Notes[a]; ok {
			title = n.Header.Title
		}
		fmt.Fprintf(&b, "\n### %s\n", title)
		for _, t := range byArea[a] {
			taskLine(t, "")
			for _, c := range m.Children(t.Header.ID) {
				if Live(c) {
					taskLine(c, "  ")
				}
			}
		}
	}
	if someday > 0 {
		fmt.Fprintf(&b, "\n"+l("someday")+"\n", someday)
	}
	return b.String()
}

// Live reports whether a task is listed in the session index: not closed, not someday.
func Live(n *Note) bool {
	return n.Header.Type == "task" && n.Header.Status != "closed" && n.Header.Status != "someday"
}

// Nested reports whether a task is listed under its parent rather than on its own: it
// is a subtask, and its parent is a valid, live task. A subtask whose parent is
// closed, someday or missing stands on its own, so it is never hidden.
func (m *Memory) Nested(n *Note) bool {
	p := m.ParentOf(n)
	return p != nil && Live(p)
}

// WriteIndex writes MEMORY.md and reports whether it changed.
func (m *Memory) WriteIndex(lang string) (bool, error) {
	path := filepath.Join(m.Dir, IndexFile)
	text := m.Index(lang)
	if old, err := os.ReadFile(path); err == nil && string(old) == text {
		return false, nil
	}
	return true, paths.WriteAtomic(path, []byte(text), 0o644)
}
