package memory

import (
	"strings"
	"unicode"
)

// OneLine makes text a single line: line breaks, tabs, the Unicode line and paragraph
// separators and every other control character become spaces, runs of spaces become
// one, and none is left at the ends. A field the header calls "one line" (a title, a
// summary, the next step) goes through it on the way in and on the way out, so text
// with line breaks in it can never open a section of its own in the index or in what
// a session reads at the start ("## Rules").
func OneLine(text string) string {
	if !needsFlattening(text) {
		return text
	}
	return strings.Join(strings.Fields(strings.Map(breakToSpace, text)), " ")
}

// breakToSpace turns whatever could start a new line, or hide text, into a space.
func breakToSpace(r rune) rune {
	if unicode.IsControl(r) || r == '\u2028' || r == '\u2029' {
		return ' '
	}
	return r
}

// needsFlattening is the quick test that spares OneLine's copies for the common case:
// plain text with single spaces and none at the ends.
func needsFlattening(text string) bool {
	prev := ' ' // a space at the start needs trimming
	for _, r := range text {
		if unicode.IsSpace(r) || unicode.IsControl(r) || r == '\u2028' || r == '\u2029' {
			if r != ' ' || prev == ' ' {
				return true
			}
		}
		prev = r
	}
	return prev == ' ' && text != ""
}

// OneLine makes every field of the header that holds one line a single line (see the
// package function). Parse does it to every note it reads, so a note written by hand,
// or one that arrived through a backup, with a multi-line next step or title, still
// gives one line wherever it is shown. Only the id is left: it must match the file
// name, and the check says so.
func (h *Header) OneLine() {
	for _, f := range []*string{&h.Type, &h.Title, &h.Summary, &h.Area, &h.Created, &h.Updated,
		&h.Parent, &h.Status, &h.WaitingOn, &h.Next} {
		*f = OneLine(*f)
	}
	// A repository is a path, which may hold two spaces in a row: only what would
	// break the line goes.
	for _, list := range [][]string{h.Repos, h.Related} {
		for i := range list {
			list[i] = strings.TrimSpace(strings.Map(breakToSpace, list[i]))
		}
	}
	for i := range h.Watch {
		h.Watch[i].Date = OneLine(h.Watch[i].Date)
		h.Watch[i].What = OneLine(h.Watch[i].What)
	}
}
