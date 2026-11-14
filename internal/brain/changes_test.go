package brain

import (
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/memory"
)

func stepTexts(n *memory.Note) []string {
	var out []string
	for _, s := range memory.Steps(n.Body) {
		out = append(out, s.Text)
	}
	return out
}

func TestChangeStep(t *testing.T) {
	b := open(t, "en")
	n, err := b.NewTask(NewTaskInput{Title: "Paint the fence"})
	if err != nil {
		t.Fatal(err)
	}
	id := n.Header.ID
	for _, s := range []string{"Buy paint", "Sand it", "Paint it"} {
		if _, err := b.AddStep(id, s); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := b.Step(StepInput{ID: id, Number: 1, Done: true}); err != nil {
		t.Fatal(err)
	}

	// One change at a time.
	if _, err := b.ChangeStep(StepChange{ID: id, Number: 1, Edit: "x", Remove: true}); err == nil {
		t.Error("two changes at once")
	}
	if _, err := b.ChangeStep(StepChange{ID: id, Number: 1}); err == nil {
		t.Error("no change")
	}
	// A page shown before the plan changed never touches another step.
	if _, err := b.ChangeStep(StepChange{ID: id, Number: 2, Expect: "Paint it", Remove: true}); err == nil {
		t.Error("stale page removed a step")
	}

	// Correct: the mark stays.
	n, err = b.ChangeStep(StepChange{ID: id, Number: 1, Expect: "Buy paint", Edit: "Buy  green paint"})
	if err != nil {
		t.Fatal(err)
	}
	if st := memory.Steps(n.Body); st[0].Text != "Buy green paint" || !st[0].Done {
		t.Errorf("edit: %+v", st[0])
	}
	if !strings.Contains(n.Body, "Step 1 corrected: Buy paint → Buy green paint") {
		t.Errorf("journal after edit:\n%s", n.Body)
	}

	// Move, by text, to the end and back up.
	n, err = b.ChangeStep(StepChange{ID: id, Text: "Sand", MoveTo: 3})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(stepTexts(n), "|"); got != "Buy green paint|Paint it|Sand it" {
		t.Errorf("move down: %s", got)
	}
	n, err = b.ChangeStep(StepChange{ID: id, Number: 3, MoveTo: 1})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(stepTexts(n), "|"); got != "Sand it|Buy green paint|Paint it" {
		t.Errorf("move up: %s", got)
	}
	if _, err := b.ChangeStep(StepChange{ID: id, Number: 1, MoveTo: 9}); err == nil {
		t.Error("move past the end")
	}

	// Remove.
	n, err = b.ChangeStep(StepChange{ID: id, Number: 3, Expect: "Paint it", Remove: true})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(stepTexts(n), "|"); got != "Sand it|Buy green paint" {
		t.Errorf("remove: %s", got)
	}
	if !strings.Contains(n.Body, "Step 3 taken out of the plan: Paint it") {
		t.Errorf("journal after remove:\n%s", n.Body)
	}

	// A closed task keeps its plan.
	if _, err := b.Close(id, "Painted"); err != nil {
		t.Fatal(err)
	}
	if _, err := b.ChangeStep(StepChange{ID: id, Number: 1, Remove: true}); err == nil {
		t.Error("changed a step of a closed task")
	}
}

func TestMoveStepKeepsOtherLines(t *testing.T) {
	body := "## Plan\n\nIntro.\n\n- [ ] a\n- [x] b\n- [ ] c"
	lines := memory.Steps(body)
	got := memory.MoveStep(body, lines[2].Line, lines[0].Line)
	if want := "## Plan\n\nIntro.\n\n- [ ] c\n- [ ] a\n- [x] b"; got != want {
		t.Errorf("move up from the last line:\n%q\nwant\n%q", got, want)
	}
	got = memory.MoveStep(body, lines[0].Line, lines[2].Line)
	if want := "## Plan\n\nIntro.\n\n- [x] b\n- [ ] c\n- [ ] a"; got != want {
		t.Errorf("move down to the last line:\n%q\nwant\n%q", got, want)
	}
}

func TestUpdateTitleAndWatchDrop(t *testing.T) {
	b := open(t, "en")
	n, err := b.NewTask(NewTaskInput{Title: "Old name"})
	if err != nil {
		t.Fatal(err)
	}
	id := n.Header.ID
	n, err = b.Update(UpdateInput{ID: id, Title: "New name"})
	if err != nil {
		t.Fatal(err)
	}
	if n.Header.Title != "New name" || n.Header.ID != id || !strings.Contains(n.Body, "Title changed: Old name → New name") {
		t.Errorf("title: %+v\n%s", n.Header, n.Body)
	}
	if idx := read(t, b, "MEMORY.md"); !strings.Contains(idx, "[New name](tasks/"+id+".md)") {
		t.Errorf("index after the title:\n%s", idx)
	}

	for _, w := range [][2]string{{"2026-12-01", "Call the bank"}, {"2026-12-01", "Check the post"}, {"2026-12-05", "Pay"}} {
		if _, err := b.AddWatch(id, w[0], w[1]); err != nil {
			t.Fatal(err)
		}
	}
	// Move one of two on the same day.
	n, err = b.Update(UpdateInput{ID: id, WatchDrop: "2026-12-01", WatchDropWhat: "Check the post", WatchDate: "2026-12-10", WatchWhat: "Check the post"})
	if err != nil {
		t.Fatal(err)
	}
	if len(n.Header.Watch) != 3 || n.Header.Watch[2].Date != "2026-12-10" || n.Header.Watch[0].What != "Call the bank" {
		t.Errorf("watch after the move: %+v", n.Header.Watch)
	}
	if !strings.Contains(n.Body, "Watch date moved from 2026-12-01 to 2026-12-10: Check the post") {
		t.Errorf("journal after the move:\n%s", n.Body)
	}
	// Take out a whole day.
	n, err = b.Update(UpdateInput{ID: id, WatchDrop: "2026-12-05"})
	if err != nil {
		t.Fatal(err)
	}
	if len(n.Header.Watch) != 2 || !strings.Contains(n.Body, "Watch date taken out: 2026-12-05 · Pay") {
		t.Errorf("drop: %+v\n%s", n.Header.Watch, n.Body)
	}
	// Nothing on that day: refused, nothing written.
	if _, err := b.Update(UpdateInput{ID: id, WatchDrop: "2026-12-05", Title: "Other"}); err == nil {
		t.Error("dropped a date that is not there")
	}
	if n, _ := b.Task(id); n.Header.Title != "New name" {
		t.Errorf("a refused change wrote the title: %s", n.Header.Title)
	}
}
