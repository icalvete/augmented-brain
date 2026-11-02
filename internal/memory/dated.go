package memory

import (
	"regexp"
	"time"
)

// DateLayout is how every date in the notes is written: YYYY-MM-DD.
const DateLayout = "2006-01-02"

// ParseDate reads a date written as YYYY-MM-DD.
func ParseDate(s string) (time.Time, error) { return time.Parse(DateLayout, s) }

// DaysBetween returns how many days to is after from, both YYYY-MM-DD; negative when
// it is before. ok is false when either is not a date.
func DaysBetween(from, to string) (days int, ok bool) {
	f, err := ParseDate(from)
	if err != nil {
		return 0, false
	}
	t, err := ParseDate(to)
	if err != nil {
		return 0, false
	}
	return int(t.Sub(f).Hours() / 24), true
}

// datedLine is a line of a task's journal or of the inbox: "- 2026-10-17 — text".
var datedLine = regexp.MustCompile(`^- (\d{4}-\d{2}-\d{2}) — (.*)$`)

// DatedLine returns the line for text on date, with its line end. The journal and the
// inbox are both written this way.
func DatedLine(date, text string) string { return "- " + date + " — " + text + "\n" }

// ParseDatedLine reads a line written by DatedLine (without its "\n"): its date and the
// text after the dash. ok is false for any other line.
func ParseDatedLine(line string) (date, text string, ok bool) {
	m := datedLine.FindStringSubmatch(line)
	if m == nil {
		return "", "", false
	}
	return m[1], m[2], true
}
