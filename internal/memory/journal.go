package memory

import (
	"fmt"
	"regexp"
	"strings"
)

// A task's journal grows with every entry. When the note passes JournalArchiveAt bytes,
// its oldest entries move to a file of their own in JournalArchiveDir, and one dated line
// takes their place, naming that file. Nothing is deleted: the lines are moved as they
// were. Reading a task in one go then always fits what an assistant can take.
const (
	JournalArchiveAt  = 24 * 1024          // a task note longer than this moves old entries out
	JournalKeep       = 8 * 1024           // about how much of the journal stays in the note
	JournalMinMove    = 4 * 1024           // less than this is not worth a file
	JournalArchiveDir = "archive/journals" // where they go, from the memory folder
)

// archiveLink finds the file a journal line points to: its path, in any language.
var archiveLink = regexp.MustCompile("archive/journals/[^\\s)`\\]]+\\.md")

// ArchiveLink returns the archive file a journal line names (from the memory folder),
// when it is a dated line that names one.
func ArchiveLink(line string) (string, bool) {
	if _, _, ok := ParseDatedLine(strings.TrimRight(line, "\r\n")); !ok {
		return "", false
	}
	p := archiveLink.FindString(line)
	return p, p != ""
}

// journalEntries returns the lines of body (split after every "\n") and where each
// journal entry starts: a dated line of the last section, with the lines that follow it
// up to the next dated line. What the section holds before its first dated line is not
// an entry. Lines inside code blocks never start one.
func journalEntries(body string) (lines []string, starts []int) {
	lines = strings.SplitAfter(body, "\n")
	from := 0
	if heads := Headings(lines); len(heads) > 0 {
		from = heads[len(heads)-1] + 1
	}
	inCode := false
	for i := from; i < len(lines); i++ {
		l := strings.TrimRight(lines[i], "\r\n")
		if strings.HasPrefix(strings.TrimSpace(l), "```") {
			inCode = !inCode
			continue
		}
		if _, _, ok := ParseDatedLine(l); ok && !inCode {
			starts = append(starts, i)
		}
	}
	return lines, starts
}

// JournalSplit is a journal cut in two: Before + Moved + After is the body it came from.
type JournalSplit struct {
	Before, Moved, After string
	First, Last          string // the dates of the first and the last entry moved
	Entries              int    // how many entries were moved
}

// SplitJournal finds the oldest entries of body's journal to move out, so that about
// keep bytes of entries stay (always the last one). ok is false when there is nothing to
// move.
func SplitJournal(body string, keep int) (JournalSplit, bool) {
	lines, starts := journalEntries(body)
	if len(starts) < 2 {
		return JournalSplit{}, false
	}
	size := func(from, to int) int {
		n := 0
		for _, l := range lines[from:to] {
			n += len(l)
		}
		return n
	}
	// The first entry that stays: from the end back, while the kept part fits.
	cut := len(starts) - 1
	for cut > 0 && size(starts[cut-1], len(lines)) <= keep {
		cut--
	}
	// Too little to move: a note long for other reasons would otherwise move a line or
	// two on every write.
	if cut == 0 || size(starts[0], starts[cut]) < JournalMinMove {
		return JournalSplit{}, false
	}
	first, _, _ := ParseDatedLine(strings.TrimRight(lines[starts[0]], "\r\n"))
	last, _, _ := ParseDatedLine(strings.TrimRight(lines[starts[cut-1]], "\r\n"))
	join := func(from, to int) string { return strings.Join(lines[from:to], "") }
	s := JournalSplit{
		Before: join(0, starts[0]), Moved: join(starts[0], starts[cut]), After: join(starts[cut], len(lines)),
		First: first, Last: last, Entries: cut,
	}
	if !strings.HasSuffix(s.Moved, "\n") { // only when nothing follows, which cannot be
		s.Moved += "\n"
	}
	return s, true
}

// ArchivedEntries returns the entries an archive file holds: everything from its first
// dated line on, as written. The title and the line on where they came from are not.
func ArchivedEntries(file string) string {
	lines := strings.SplitAfter(file, "\n")
	for i, l := range lines {
		if _, _, ok := ParseDatedLine(strings.TrimRight(l, "\r\n")); ok {
			return strings.Join(lines[i:], "")
		}
	}
	return ""
}

// ExpandJournal puts back, in place of each line that names an archive file, the entries
// of that file, as read by read (from the memory folder). Lines whose file cannot be read
// stay. It returns the text and, for each line put back, the entries it stood for.
func ExpandJournal(text string, read func(rel string) (string, bool)) (string, map[string]string) {
	lines := strings.SplitAfter(text, "\n")
	links := map[string]string{}
	var out strings.Builder
	for _, l := range lines {
		rel, ok := ArchiveLink(l)
		if !ok {
			out.WriteString(l)
			continue
		}
		file, found := read(rel)
		entries := ArchivedEntries(file)
		if !found || entries == "" {
			out.WriteString(l)
			continue
		}
		line := l
		if !strings.HasSuffix(line, "\n") {
			line += "\n"
		}
		links[line] = entries
		out.WriteString(entries)
	}
	return out.String(), links
}

// CollapseJournal undoes ExpandJournal: each block of entries goes back to the line that
// named its file. ok is false when a block is not in text as it was.
func CollapseJournal(text string, links map[string]string) (string, bool) {
	for line, entries := range links {
		i := strings.Index(text, entries)
		if i < 0 || (i > 0 && text[i-1] != '\n') {
			return text, false
		}
		text = text[:i] + line + text[i+len(entries):]
	}
	return text, true
}

// ShortenJournal leaves out the oldest entries of body's journal until body fits in max
// bytes, in their place one line saying how many were left out and where they are (in
// note, from the memory folder). It is for reading only: the note keeps them. The last
// entry always stays. omitted is how many entries were left out.
func ShortenJournal(body string, max int, note string) (string, int) {
	if len(body) <= max {
		return body, 0
	}
	lines, starts := journalEntries(body)
	if len(starts) < 2 {
		return body, 0
	}
	head := strings.Join(lines[:starts[0]], "")
	for k := 1; k < len(starts); k++ {
		rest := strings.Join(lines[starts[k]:], "")
		say := fmt.Sprintf("- (%d older journal entries left out here so the task fits in one read; they are in %s, read that file for them)\n", k, note)
		if len(head)+len(say)+len(rest) <= max || k == len(starts)-1 {
			return head + say + rest, k
		}
	}
	return body, 0
}

// Oversized returns the tasks longer than JournalArchiveAt whose journal has old entries
// to move, sorted by id. The program moves them when it next writes the task, and
// `check` does it for all of them.
func (m *Memory) Oversized() []string {
	var out []string
	for _, n := range m.OfType("task") {
		data, err := n.Render()
		if err != nil || len(data) <= JournalArchiveAt {
			continue
		}
		if _, ok := SplitJournal(n.Body, JournalKeep); ok {
			out = append(out, n.Header.ID)
		}
	}
	return out
}
