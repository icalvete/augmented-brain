package config

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/icalvete/augmented-brain/internal/paths"
)

// A list setting is edited where it is, one value at a time, so the user's layout and
// comments survive: a list that takes many lines, several values on a line, a comment
// after a value or between values. A value taken out goes with its comma, and with its
// line when nothing else was on it; a value added goes right before the closing "]".
// What cannot be read that way (a nested array, a multi-line string, a number) is
// written again on one line, as before.

// listToken is one string of a list, where the file has it: line, and its bytes
// [start, end) on that line, quotes included.
type listToken struct {
	line, start, end int
	value            string
}

// arrayTokens reads the strings of the array set on line i. It returns them, the line
// and byte of its closing "]", and ok false when the array holds anything else, or
// does not close.
func arrayTokens(lines []string, i int) (toks []listToken, closeLine, closeCol int, ok bool) {
	eq := strings.Index(lines[i], "=")
	if eq < 0 {
		return nil, 0, 0, false
	}
	opened := false
	for n := i; n < len(lines); n++ {
		text := lines[n]
		k := 0
		if n == i {
			k = eq + 1
		}
		for ; k < len(text); k++ {
			c := text[k]
			switch {
			case c == ' ' || c == '\t' || c == '\r':
			case c == '#':
				k = len(text)
			case !opened:
				if c != '[' {
					return nil, 0, 0, false
				}
				opened = true
			case c == ',':
			case c == ']':
				return toks, n, k, true
			case c == '"' || c == '\'':
				if strings.HasPrefix(text[k:], `"""`) || strings.HasPrefix(text[k:], "'''") {
					return nil, 0, 0, false
				}
				end := stringEnd(text, k)
				if end < 0 {
					return nil, 0, 0, false
				}
				var v struct{ V string }
				if _, err := toml.Decode("V = "+text[k:end], &v); err != nil {
					return nil, 0, 0, false
				}
				toks = append(toks, listToken{line: n, start: k, end: end, value: v.V})
				k = end - 1
			default:
				return nil, 0, 0, false
			}
		}
		if !opened {
			return nil, 0, 0, false
		}
	}
	return nil, 0, 0, false
}

// stringEnd is the byte after the string that opens at text[k], or -1 when it does not
// end on the line.
func stringEnd(text string, k int) int {
	q := text[k]
	for n := k + 1; n < len(text); n++ {
		switch {
		case q == '"' && text[n] == '\\':
			n++
		case text[n] == q:
			return n + 1
		}
	}
	return -1
}

// sameValue says whether two values of a list are the same: equal, or two paths (one
// written with "~", the other in full) of the same folder.
func sameValue(a, b string) bool {
	if a == b {
		return true
	}
	isPath := func(s string) bool { return strings.HasPrefix(s, "~") || filepath.IsAbs(s) }
	return isPath(a) && isPath(b) && paths.Tilde(a) == paths.Tilde(b)
}

// whyComment is the comment that goes with a value added for a reason: the day and
// the reason, on one line.
func whyComment(why string) string {
	why = strings.Join(strings.Fields(why), " ")
	if why == "" {
		return ""
	}
	return fmt.Sprintf("# %s, %s", time.Now().Format("2006-01-02"), why)
}

// newList writes a list that is not in the file yet: on one line, or, with a reason,
// one value a line and the reason after each.
func newList(key string, values []string, comment string) []string {
	if comment == "" || len(values) == 0 {
		return []string{listLine(key, values)}
	}
	out := []string{key + " = ["}
	for _, v := range values {
		out = append(out, fmt.Sprintf("  %q,  %s", v, comment))
	}
	return append(out, "]")
}

// setList makes the list set on line i hold values, editing it in place; comment, when
// not empty, goes after each value added.
func (f *lineEditor) setList(i int, key string, values []string, comment string) {
	toks, _, _, ok := arrayTokens(f.lines, i)
	if !ok {
		j := valueEnd(f.lines, i)
		f.lines = append(f.lines[:i], append(newList(key, values, comment), f.lines[j:]...)...)
		return
	}
	// Out: every string no longer wanted, from the bottom up so what is still to come
	// keeps its line and its bytes.
	_, closeLine, _, _ := arrayTokens(f.lines, i)
	for k := len(toks) - 1; k >= 0; k-- {
		t := toks[k]
		if slices.ContainsFunc(values, func(v string) bool { return sameValue(v, t.value) }) {
			continue
		}
		f.lines[t.line] = dropToken(f.lines[t.line], t.start, t.end)
		if t.line == i || t.line == closeLine {
			continue
		}
		if _, _, kept := lineTokens(f.lines[t.line]); !kept {
			f.lines = append(f.lines[:t.line], f.lines[t.line+1:]...)
			closeLine--
		}
	}
	// In: every value not there yet, in the order given.
	toks, closeLine, closeCol, _ := arrayTokens(f.lines, i)
	var add []string
	for _, v := range values {
		if !slices.ContainsFunc(toks, func(t listToken) bool { return sameValue(v, t.value) }) &&
			!slices.ContainsFunc(add, func(a string) bool { return sameValue(v, a) }) {
			add = append(add, v)
		}
	}
	if len(add) == 0 {
		return
	}
	quoted := make([]string, len(add))
	for k, v := range add {
		quoted[k] = fmt.Sprintf("%q", v)
	}
	line := f.lines[closeLine]
	if closeLine == i && comment == "" {
		before := strings.TrimRight(line[:closeCol], " \t")
		sep := ", "
		switch {
		case len(toks) == 0:
			sep = ""
		case strings.HasSuffix(before, ","):
			sep = " "
		}
		f.lines[i] = before + sep + strings.Join(quoted, ", ") + line[closeCol:]
		return
	}
	// One value a line: the last value there needs its comma first.
	indent := "  "
	if n := len(toks); n > 0 {
		last := toks[n-1]
		if last.line != i {
			l := f.lines[last.line]
			indent = l[:len(l)-len(strings.TrimLeft(l, " \t"))]
		}
		if rest := strings.TrimLeft(f.lines[last.line][last.end:], " \t"); !strings.HasPrefix(rest, ",") {
			l := f.lines[last.line]
			f.lines[last.line] = l[:last.end] + "," + l[last.end:]
			if last.line == closeLine {
				closeCol++
			}
		}
	}
	added := make([]string, len(quoted))
	for k, q := range quoted {
		added[k] = indent + q + ","
		if comment != "" {
			added[k] += "  " + comment
		}
	}
	line = f.lines[closeLine]
	before, after := line[:closeCol], line[closeCol:]
	if strings.TrimSpace(before) == "" {
		f.insert(closeLine, added...)
		return
	}
	closing := ""
	if closeLine != i {
		closing = before[:len(before)-len(strings.TrimLeft(before, " \t"))]
	}
	f.lines[closeLine] = strings.TrimRight(before, " \t")
	f.insert(closeLine+1, append(added, closing+after)...)
}

// lineTokens says whether a line inside an array still holds a value (anything before
// its comment but commas and blanks).
func lineTokens(line string) (string, string, bool) {
	code, comment := line, ""
	for k := 0; k < len(line); k++ {
		if c := line[k]; c == '"' || c == '\'' {
			if e := stringEnd(line, k); e > 0 {
				k = e - 1
				continue
			}
		} else if c == '#' {
			code, comment = line[:k], line[k:]
			break
		}
	}
	return code, comment, strings.Trim(code, " \t,") != ""
}

// dropToken takes the string at [start, end) out of line, with the comma after it, or
// the one before it when it is the last on the line.
func dropToken(line string, start, end int) string {
	rest := strings.TrimLeft(line[end:], " \t")
	if after, ok := strings.CutPrefix(rest, ","); ok {
		after = strings.TrimLeft(after, " \t")
		switch {
		case after == "":
			return strings.TrimRight(line[:start], " \t")
		case strings.HasPrefix(after, "#"):
			if head := strings.TrimRight(line[:start], " \t"); head != "" {
				return head + "  " + after
			}
			return line[:start] + after
		case strings.HasPrefix(after, "]"):
			return strings.TrimRight(line[:start], " \t") + after
		}
		return line[:start] + after
	}
	head := strings.TrimRight(line[:start], " \t")
	if strings.HasSuffix(head, ",") {
		return strings.TrimSuffix(head, ",") + line[end:]
	}
	return line[:start] + strings.TrimLeft(line[end:], " \t")
}
