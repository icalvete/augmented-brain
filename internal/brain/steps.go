package brain

import (
	"errors"
	"fmt"
	"strings"

	"github.com/icalvete/augmented-brain/internal/memory"
)

// Steps are the checkboxes of a task's Plan ("- [ ] text"). They live in the body, so
// they need no field of their own; see memory.Steps.

// stepTexts are the journal lines a change of step writes, per language.
var stepTexts = map[string]map[string]string{
	"en": {"done": "Step %d done: %s", "undone": "Step %d not done after all: %s", "added": "Step added to the plan: %s",
		"noplan": "task %q has no Plan section (a line \"## Plan\") to keep its steps in",
		"moved":  "step %d no longer says %q: the plan changed since. Look at it again and choose the step once more"},
	"es": {"done": "Paso %d hecho: %s", "undone": "Paso %d sin hacer, al final: %s", "added": "Paso añadido al plan: %s",
		"noplan": "la tarea %q no tiene sección Plan (una línea \"## Plan\") donde guardar sus pasos",
		"moved":  "el paso %d ya no dice %q: el plan ha cambiado desde entonces. Míralo otra vez y vuelve a elegir el paso"},
}

func (b *Brain) stepText(key string) string { return stepTexts[b.lang()][key] }

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
		return memory.Step{}, errors.New("this task has no steps: they are the checkboxes (- [ ] text) of its Plan")
	}
	if number != 0 {
		if number < 1 || number > len(steps) {
			return memory.Step{}, fmt.Errorf("there is no step %d: the plan has %d", number, len(steps))
		}
		return steps[number-1], nil
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return memory.Step{}, errors.New("say which step: its number or its text")
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
		return memory.Step{}, fmt.Errorf("no step says %q", text)
	}
	var n []string
	for _, s := range found {
		n = append(n, fmt.Sprintf("%d (%s)", s.N, s.Text))
	}
	return memory.Step{}, fmt.Errorf("%q matches several steps: %s; give its number", text, strings.Join(n, ", "))
}

// Step ticks or unticks a step of a task's Plan. A change writes one short line in the
// journal, as any other thing done does; a step already as asked changes nothing.
func (b *Brain) Step(in StepInput) (*memory.Note, error) {
	n, err := b.Task(in.ID)
	if err != nil {
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
	text = strings.Join(strings.Fields(text), " ")
	if text == "" {
		return nil, errors.New("the step is empty")
	}
	n, err := b.Task(id)
	if err != nil {
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
