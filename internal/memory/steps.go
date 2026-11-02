package memory

import (
	"regexp"
	"strings"
)

// Steps are the small parts of a task: Markdown checkboxes ("- [ ] text", "- [x] text")
// in its Plan section. They need no storage of their own; the program counts them and
// ticks them, and never reads the rest of the prose.

// Step is one checkbox of a task's Plan.
type Step struct {
	N    int // 1 for the first step of the Plan
	Text string
	Done bool
	Line int // index of its line in the body, split after every "\n"
}

var (
	planHeading = regexp.MustCompile(`(?i)^##\s+(\d+\.\s*)?plan\s*$`)
	checkbox    = regexp.MustCompile(`^(\s*[-*] \[)([ xX])(\] )(.*)$`)
)

// Headings returns the index of every level-2 heading ("## " at the start of a line)
// among lines, outside code blocks. Sections run from one heading to the next, so this
// is how the program finds the few it relies on: the essentials, the Plan.
func Headings(lines []string) []int {
	var out []int
	inCode := false
	for i, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), "```") {
			inCode = !inCode
			continue
		}
		if !inCode && strings.HasPrefix(l, "## ") {
			out = append(out, i)
		}
	}
	return out
}

// PlanSection returns the lines of body (split after every "\n") and the range of the
// Plan section: from the line after its heading ("## Plan" or "## 3. Plan", any case)
// to the next level-2 heading or the end. ok is false when there is no Plan section.
// Lines inside code blocks are never taken for headings.
func PlanSection(body string) (lines []string, from, to int, ok bool) {
	lines = strings.SplitAfter(body, "\n")
	heads := Headings(lines)
	for k, i := range heads {
		if !planHeading.MatchString(strings.TrimRight(lines[i], "\r\n")) {
			continue
		}
		to = len(lines)
		if k+1 < len(heads) {
			to = heads[k+1]
		}
		return lines, i + 1, to, true
	}
	return lines, 0, 0, false
}

// Steps returns the checkboxes of the Plan section of a task's body, in order.
func Steps(body string) []Step {
	lines, from, to, ok := PlanSection(body)
	if !ok {
		return nil
	}
	var out []Step
	inCode := false
	for i := from; i < to; i++ {
		t := strings.TrimRight(lines[i], "\r\n")
		if strings.HasPrefix(strings.TrimSpace(t), "```") {
			inCode = !inCode
			continue
		}
		if inCode {
			continue
		}
		if m := checkbox.FindStringSubmatch(t); m != nil && strings.TrimSpace(m[4]) != "" {
			out = append(out, Step{N: len(out) + 1, Text: strings.TrimSpace(m[4]), Done: m[2] != " ", Line: i})
		}
	}
	return out
}

// StepCount returns how many steps of the Plan are done, and how many there are.
func StepCount(body string) (done, total int) {
	for _, s := range Steps(body) {
		total++
		if s.Done {
			done++
		}
	}
	return done, total
}

// SetStep returns body with the checkbox on line i ticked or unticked; the rest of the
// line, and of the body, stays as it was.
func SetStep(body string, i int, done bool) string {
	lines := strings.SplitAfter(body, "\n")
	if i < 0 || i >= len(lines) {
		return body
	}
	end := lines[i][len(strings.TrimRight(lines[i], "\r\n")):]
	m := checkbox.FindStringSubmatch(strings.TrimRight(lines[i], "\r\n"))
	if m == nil {
		return body
	}
	mark := " "
	if done {
		mark = "x"
	}
	lines[i] = m[1] + mark + m[3] + m[4] + end
	return strings.Join(lines, "")
}

// AddStep returns body with a new, unticked step at the end of the Plan section, after
// its last line of text. ok is false when the body has no Plan section.
func AddStep(body, text string) (string, bool) {
	lines, from, to, ok := PlanSection(body)
	if !ok {
		return body, false
	}
	at := to
	for at > from && strings.TrimSpace(lines[at-1]) == "" {
		at--
	}
	var b strings.Builder
	b.WriteString(strings.Join(lines[:at], ""))
	if at > 0 && !strings.HasSuffix(lines[at-1], "\n") {
		b.WriteString("\n")
	}
	if at > from && !checkbox.MatchString(strings.TrimRight(lines[at-1], "\r\n")) {
		b.WriteString("\n") // a blank line between the Plan's prose and its list
	}
	b.WriteString("- [ ] " + strings.Join(strings.Fields(text), " ") + "\n")
	if at == to && to < len(lines) {
		b.WriteString("\n") // a blank line before the next heading
	}
	b.WriteString(strings.Join(lines[at:], ""))
	return b.String(), true
}
