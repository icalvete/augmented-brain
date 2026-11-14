package brain

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/icalvete/augmented-brain/internal/i18n"

	"github.com/icalvete/augmented-brain/internal/memory"
)

// How much one field may hold, in characters. They are generous on purpose (the
// longest next step in years of real notes is under 800): they stop a page pasted by
// mistake, or text from outside, from filling what every session reads at the start,
// and each error says where the long text goes instead.
const (
	maxTitle     = 300
	maxSummary   = 1000
	maxNext      = 2000
	maxWaitingOn = 200
	maxWatchWhat = 1000
	maxName      = 200 // the user's name and the assistant's, in the profile
	maxCapture   = 4000
	maxJournal   = 32_000
	maxProfile   = 64_000
	maxBody      = 512_000 // the body of any note
)

// Where the text that does not fit goes, said in every error about a long field.
const (
	longOneLine i18n.Word = "err.toolong.oneline"
	longText    i18n.Word = "err.toolong.text"
)

// tooLong is the error for a field longer than max characters; nil when it fits.
func tooLong(what i18n.Word, text string, max int, where i18n.Word) error {
	if n := utf8.RuneCountInString(text); n > max {
		return i18n.Errorf("err.toolong", what, n, max, where)
	}
	return nil
}

// checkHeader makes every one-line field of n one line (memory.OneLine) and refuses
// one that is too long. A field is checked only when it changes from old (nil: a new
// note), so a note written long by hand can still be changed in other ways.
func checkHeader(n, old *memory.Note) error {
	n.Header.OneLine()
	var was memory.Header
	if old != nil {
		was = old.Header
	}
	h := &n.Header
	for _, f := range []struct {
		what     i18n.Word
		now, was string
		max      int
	}{
		{"err.what.title", h.Title, was.Title, maxTitle},
		{"err.what.summary", h.Summary, was.Summary, maxSummary},
		{"err.what.next", h.Next, was.Next, maxNext},
		{"err.what.waiting", h.WaitingOn, was.WaitingOn, maxWaitingOn},
	} {
		if f.now == f.was {
			continue
		}
		if err := tooLong(f.what, f.now, f.max, longOneLine); err != nil {
			return err
		}
	}
	for _, w := range h.Watch {
		if !hasWatch(was.Watch, w) {
			if err := tooLong("err.what.watch", w.What, maxWatchWhat, longOneLine); err != nil {
				return err
			}
		}
	}
	if old == nil || len(n.Body) > len(old.Body) {
		if err := tooLong("err.what.note", n.Body, maxBody, longText); err != nil {
			return err
		}
	}
	return nil
}

func hasWatch(list []memory.Watch, w memory.Watch) bool {
	for _, x := range list {
		if x.Date == w.Date && x.What == w.What {
			return true
		}
	}
	return false
}

// multiline keeps text of several lines inside the list item it is written in (a
// journal entry): every line after the first is indented, so none can start a heading,
// a step or an entry of its own. Blank lines go, which would end the item.
func multiline(text string) string {
	lines := strings.Split(strings.ReplaceAll(strings.TrimSpace(text), "\r\n", "\n"), "\n")
	out := lines[:0]
	for _, l := range lines {
		l = strings.TrimRight(strings.Map(func(r rune) rune {
			if (unicode.IsControl(r) && r != '\t') || r == '\u2028' || r == '\u2029' {
				return ' '
			}
			return r
		}, l), " \t")
		if strings.TrimSpace(l) != "" {
			out = append(out, l)
		}
	}
	return strings.Join(out, "\n  ")
}
