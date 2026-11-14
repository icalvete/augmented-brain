package brain

import (
	"fmt"
	"strings"

	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/memory"
)

// Steps are the checkboxes of a task's Plan ("- [ ] text"). They live in the body, so
// they need no field of their own; see memory.Steps.

// stepText is a journal line or error about steps, in the user's language.
func (b *Brain) stepText(key string) string { return i18n.T(b.Language, "step."+key) }

// StepInput ticks or unticks one step of a task, found by its number (1 for the first)
// or, when Number is 0, by its text. Expect, with a number, is the exact text the step
// must still have: a page shown before the plan changed never ticks another step.
type StepInput struct {
	ID     string
	Number int
	Text   string
	Expect string
	Done   bool
}

// findStep returns the step asked for: by number, or by text (the exact text, or else
// the only step that contains it, ignoring case).
func findStep(steps []memory.Step, number int, text string) (memory.Step, error) {
	if len(steps) == 0 {
		return memory.Step{}, i18n.Errorf("err.step.none")
	}
	if number != 0 {
		if number < 1 || number > len(steps) {
			return memory.Step{}, i18n.Errorf("err.step.number", number, len(steps))
		}
		return steps[number-1], nil
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return memory.Step{}, i18n.Errorf("err.step.which")
	}
	var found []memory.Step
	for _, s := range steps {
		if s.Text == text {
			return s, nil
		}
		if strings.Contains(strings.ToLower(s.Text), strings.ToLower(text)) {
			found = append(found, s)
		}
	}
	switch len(found) {
	case 1:
		return found[0], nil
	case 0:
		return memory.Step{}, i18n.Errorf("err.step.notext", text)
	}
	var n []string
	for _, s := range found {
		n = append(n, fmt.Sprintf("%d (%s)", s.N, s.Text))
	}
	return memory.Step{}, i18n.Errorf("err.step.several", text, strings.Join(n, ", "))
}

// Step ticks or unticks a step of a task's Plan. A change writes one short line in the
// journal, as any other thing done does; a step already as asked changes nothing.
func (b *Brain) Step(in StepInput) (*memory.Note, error) {
	n, err := b.Task(in.ID)
	if err != nil {
		return nil, err
	}
	if err := closedTask(n); err != nil {
		return nil, err
	}
	steps := memory.Steps(n.Body)
	if want := strings.TrimSpace(in.Expect); in.Number != 0 && want != "" &&
		(in.Number < 1 || in.Number > len(steps) || steps[in.Number-1].Text != want) {
		return nil, fmt.Errorf(b.stepText("moved"), in.Number, want)
	}
	s, err := findStep(steps, in.Number, in.Text)
	if err != nil {
		return nil, err
	}
	if s.Done == in.Done {
		return n, nil
	}
	cp := *n
	cp.Body = memory.SetStep(cp.Body, s.Line, in.Done)
	key := "undone"
	if in.Done {
		key = "done"
	}
	cp.Body = b.journalLine(cp.Body, fmt.Sprintf(b.stepText(key), s.N, s.Text))
	return &cp, b.save(&cp)
}

// AddStep adds an unticked step at the end of a task's Plan.
func (b *Brain) AddStep(id, text string) (*memory.Note, error) {
	text = memory.OneLine(text)
	if text == "" {
		return nil, i18n.Errorf("err.step.empty")
	}
	n, err := b.Task(id)
	if err != nil {
		return nil, err
	}
	if err := closedTask(n); err != nil {
		return nil, err
	}
	body, ok := memory.AddStep(n.Body, text)
	if !ok {
		return nil, fmt.Errorf(b.stepText("noplan"), id)
	}
	cp := *n
	cp.Body = b.journalLine(body, fmt.Sprintf(b.stepText("added"), text))
	return &cp, b.save(&cp)
}

// StepChange corrects, removes or moves one step of a task's Plan, found as Step finds
// it: by number (with Expect, the text it must still have) or by text. Exactly one of
// Edit (the new text), Remove or MoveTo (the place it goes, 1 for the first) is given.
type StepChange struct {
	ID     string
	Number int
	Text   string
	Expect string
	Edit   string
	Remove bool
	MoveTo int
}

// ChangeStep corrects, removes or moves a step. Each change writes one short line in
// the journal, as ticking one does; a change that leaves the step as it was changes
// nothing.
func (b *Brain) ChangeStep(in StepChange) (*memory.Note, error) {
	edit := memory.OneLine(in.Edit)
	asked := 0
	for _, on := range []bool{edit != "", in.Remove, in.MoveTo != 0} {
		if on {
			asked++
		}
	}
	if asked != 1 {
		return nil, i18n.Errorf("err.step.change")
	}
	n, err := b.Task(in.ID)
	if err != nil {
		return nil, err
	}
	if err := closedTask(n); err != nil {
		return nil, err
	}
	steps := memory.Steps(n.Body)
	if want := strings.TrimSpace(in.Expect); in.Number != 0 && want != "" &&
		(in.Number < 1 || in.Number > len(steps) || steps[in.Number-1].Text != want) {
		return nil, fmt.Errorf(b.stepText("moved"), in.Number, want)
	}
	s, err := findStep(steps, in.Number, in.Text)
	if err != nil {
		return nil, err
	}
	cp := *n
	var line string
	switch {
	case edit != "":
		if edit == s.Text {
			return n, nil
		}
		cp.Body = memory.SetStepText(cp.Body, s.Line, edit)
		line = fmt.Sprintf(b.stepText("edited"), s.N, s.Text, edit)
	case in.Remove:
		cp.Body = memory.RemoveStep(cp.Body, s.Line)
		line = fmt.Sprintf(b.stepText("removed"), s.N, s.Text)
	default:
		if in.MoveTo < 1 || in.MoveTo > len(steps) {
			return nil, i18n.Errorf("err.step.number", in.MoveTo, len(steps))
		}
		if in.MoveTo == s.N {
			return n, nil
		}
		cp.Body = memory.MoveStep(cp.Body, s.Line, steps[in.MoveTo-1].Line)
		line = fmt.Sprintf(b.stepText("reordered"), s.Text, s.N, in.MoveTo)
	}
	cp.Body = b.journalLine(cp.Body, line)
	return &cp, b.save(&cp)
}
