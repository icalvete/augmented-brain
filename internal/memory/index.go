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
func (m *Memory) Index(lang string) string { return m.index(lang, false) }

// SessionIndex is the index as the session start gives it: like Index, but each live
// task without its summary, and with its next step only when it waits on the user, and
// without the list of rules, which the session start gives whole. In real sessions the
// assistant reads a task (task_get) before working on it, so the rest only took room at
// every start.
func (m *Memory) SessionIndex(lang string) string { return m.index(lang, true) }

func (m *Memory) index(lang string, compact bool) string {
	// Only the program writes the index, so its headings can follow the user's language
	// without the program ever reading them.
	l := func(key string) string { return i18n.T(lang, "index."+key) }
	var b strings.Builder
	// Every field goes through OneLine again, even though Parse already did it: a note
	// made in memory reaches the index too, and a line break here would open a section
	// of its own ("## Rules") in what every session reads.
	line := func(n *Note, extra string) {
		fmt.Fprintf(&b, "- [%s](%s)", OneLine(n.Header.Title), OneLine(n.Path))
		if s := OneLine(n.Header.Summary); s != "" && !(compact && n.Header.Type == "task") {
			fmt.Fprintf(&b, " — %s", s)
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
	if !compact { // the session start gives the rules whole, in parts of their own
		section(l("rules"), m.OfType("rule"))
	}
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
		extra := " · " + OneLine(t.Header.Status)
		if done, total := StepCount(t.Body); total > 0 {
			extra += " · " + fmt.Sprintf(l("steps"), done, total)
		}
		extra += fmt.Sprintf(" · %s %s", l("waits"), OneLine(t.Header.WaitingOn))
		switch {
		case !compact:
			extra += fmt.Sprintf(" · %s: %s", l("next"), OneLine(t.Header.Next))
		case t.Header.WaitingOn == "me":
			extra += fmt.Sprintf(" · %s: %s", l("next"), Clip(OneLine(t.Header.Next), maxSessionNext))
		}
		line(t, extra)
	}
	for _, a := range groups {
		title := l("noarea")
		if n, ok := m.Notes[a]; ok {
			title = OneLine(n.Header.Title)
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
	return true, paths.WriteAtomicNoLink(path, []byte(text), 0o600)
}

// maxSessionNext is the most of a task's next step the session start shows, in
// characters: enough to say what is pending; the whole step is in the task.
const maxSessionNext = 120

// Clip cuts s to max characters, ending it with "…" when it was longer.
func Clip(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return strings.TrimSpace(string(r[:max-1])) + "…"
}
