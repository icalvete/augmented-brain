package brain

import (
	"regexp"
	"strings"

	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/memory"
)

// The sections of a task that an assistant writes as the work moves: what it is for
// (Definition), how things are (Analysis), the way there (Plan), the proof (Validation)
// and the wrap-up (Closing). Each is a level-2 heading of the task's body, numbered or
// not, in English or Spanish, as the task templates write them.
var taskSections = map[string]*regexp.Regexp{
	"definition": regexp.MustCompile(`(?i)^##\s+(\d+\.\s*)?(definition|definición)\s*$`),
	"analysis":   regexp.MustCompile(`(?i)^##\s+(\d+\.\s*)?(analysis|análisis)\s*$`),
	"plan":       regexp.MustCompile(`(?i)^##\s+(\d+\.\s*)?plan\s*$`),
	"validation": regexp.MustCompile(`(?i)^##\s+(\d+\.\s*)?(validation|validación)\s*$`),
	"closing":    regexp.MustCompile(`(?i)^##\s+(\d+\.\s*)?(closing|cierre)\s*$`),
}

// TaskSections are the names SetSection takes, in the order a task has them.
var TaskSections = []string{"definition", "analysis", "plan", "validation", "closing"}

var stepLine = regexp.MustCompile(`^\s*[-*] \[[ xX]\] `)

// SetSection replaces the text of one section of a task: definition, analysis, plan,
// validation or closing. The heading stays; the essentials and the journal are never
// touched. In the Plan only the prose is replaced: its steps stay as they are, after
// the new text, because steps are task_step's (ticking them, the panel's counts). A
// text with a level-2 heading, or with steps for the Plan, is refused rather than
// half-written. A short journal line says which section changed.
func (b *Brain) SetSection(id, section, text string) (*memory.Note, error) {
	section = strings.ToLower(strings.TrimSpace(section))
	re, ok := taskSections[section]
	if !ok {
		return nil, i18n.Errorf("err.section.unknown", section, strings.Join(TaskSections, ", "))
	}
	text = strings.TrimSpace(strings.ReplaceAll(text, "\r\n", "\n"))
	if text == "" {
		return nil, i18n.Errorf("err.section.empty")
	}
	inCode := false
	for _, l := range strings.Split(text, "\n") {
		if strings.HasPrefix(strings.TrimSpace(l), "```") {
			inCode = !inCode
			continue
		}
		if inCode {
			continue
		}
		if strings.HasPrefix(l, "## ") || strings.HasPrefix(l, "# ") {
			return nil, i18n.Errorf("err.section.heading")
		}
		if section == "plan" && stepLine.MatchString(l) {
			return nil, i18n.Errorf("err.section.steps")
		}
	}
	if inCode {
		return nil, i18n.Errorf("err.section.code")
	}
	n, err := b.Task(id)
	if err != nil {
		return nil, err
	}
	if err := closedTask(n); err != nil {
		return nil, err
	}
	lines := strings.SplitAfter(n.Body, "\n")
	heads := memory.Headings(lines)
	from, to := -1, len(lines)
	for k, i := range heads {
		if re.MatchString(strings.TrimRight(lines[i], "\r\n")) {
			from = i + 1
			if k+1 < len(heads) {
				to = heads[k+1]
			}
			break
		}
	}
	if from < 0 {
		return nil, i18n.Errorf("err.section.missing", id, section)
	}
	var kept []string
	if section == "plan" {
		for _, st := range memory.Steps(n.Body) {
			kept = append(kept, strings.TrimRight(lines[st.Line], "\r\n"))
		}
	}
	part := text + "\n"
	if len(kept) > 0 {
		part += "\n" + strings.Join(kept, "\n") + "\n"
	}
	head := strings.Join(lines[:from], "")
	if !strings.HasSuffix(head, "\n") {
		head += "\n"
	}
	rest := strings.Join(lines[to:], "")
	if rest != "" {
		part += "\n"
	}
	cp := *n
	cp.Body = b.journalLine(head+part+rest, i18n.T(b.Language, "note.section."+section))
	return &cp, b.save(&cp)
}
