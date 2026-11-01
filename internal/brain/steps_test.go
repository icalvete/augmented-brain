package brain

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/memory"
)

func TestSubtasks(t *testing.T) {
	b := open(t, "en")
	parent, err := b.NewTask(NewTaskInput{Title: "Move the shop online"})
	if err != nil {
		t.Fatal(err)
	}
	pid := parent.Header.ID

	// The parent must exist and be a task.
	if _, err := b.NewTask(NewTaskInput{Title: "Lost", Parent: "task-nowhere"}); err == nil || !strings.Contains(err.Error(), "no task") {
		t.Errorf("missing parent: %v", err)
	}
	if _, err := b.NewTask(NewTaskInput{Title: "In an area", Parent: "area-general"}); err == nil || !strings.Contains(err.Error(), "not a task") {
		t.Errorf("an area as parent: %v", err)
	}

	child, err := b.NewTask(NewTaskInput{Title: "Pick a payment provider", Parent: pid})
	if err != nil {
		t.Fatal(err)
	}
	if child.Header.Parent != pid || child.Header.Area != parent.Header.Area {
		t.Errorf("child header: %+v", child.Header)
	}
	if !strings.Contains(read(t, b, child.Path), "parent: "+pid) {
		t.Error("parent not written in the header")
	}

	// One level only.
	if _, err := b.NewTask(NewTaskInput{Title: "Grandchild", Parent: child.Header.ID}); err == nil || !strings.Contains(err.Error(), "one level") {
		t.Errorf("grandchild: %v", err)
	}

	// The index lists the child under its parent, indented, and not on its own.
	idx := read(t, b, "MEMORY.md")
	want := "- [Move the shop online](tasks/" + pid + ".md) · definition · waits on me · next: Agree on the definition\n" +
		"  - [Pick a payment provider](tasks/" + child.Header.ID + ".md) · definition"
	if !strings.Contains(idx, want) || strings.Count(idx, "Pick a payment provider") != 1 {
		t.Errorf("index:\n%s", idx)
	}

	// Pending puts the child right after its parent.
	items, _ := b.Pending(0)
	if len(items) != 2 || items[0].ID != pid || items[1].ID != child.Header.ID || !items[1].Sub || items[1].ParentTitle != "Move the shop online" {
		t.Errorf("pending: %+v", items)
	}

	// Parking the parent leaves the child alone.
	if _, err := b.Park(ParkInput{ID: pid, Next: "Wait for the provider"}); err != nil {
		t.Fatal(err)
	}
	if c, _ := b.Task(child.Header.ID); c.Header.Status != "definition" {
		t.Errorf("parking the parent changed the child: %s", c.Header.Status)
	}

	// The parent cannot close, by Close or by Update, while the child is open.
	if _, err := b.Close(pid, "Done"); err == nil || !strings.Contains(err.Error(), child.Header.ID) {
		t.Errorf("close with an open child: %v", err)
	}
	if _, err := b.Update(UpdateInput{ID: pid, Status: "closed"}); err == nil {
		t.Error("update to closed with an open child")
	}
	if _, err := b.Close(child.Header.ID, "Chose one"); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Close(pid, "Shop online"); err != nil {
		t.Errorf("close after the child: %v", err)
	}
	if _, err := b.NewTask(NewTaskInput{Title: "Late", Parent: pid}); err == nil || !strings.Contains(err.Error(), "closed") {
		t.Errorf("subtask of a closed task: %v", err)
	}
}

// A subtask whose parent is closed or missing stands on its own in the index, and a
// missing parent is a problem to fix that does not lock the note.
func TestOrphanSubtask(t *testing.T) {
	dir := t.TempDir()
	note := func(id, status, parent string) string {
		s := "---\nid: " + id + "\ntype: task\ntitle: " + id + "\nstatus: " + status + "\n"
		if parent != "" {
			s += "parent: " + parent + "\n"
		}
		if status != "closed" {
			s += "waiting_on: me\nnext: do it\n"
		}
		return s + "---\n"
	}
	files := map[string]string{
		"tasks/task-old.md":   note("task-old", "closed", ""),
		"tasks/task-a.md":     note("task-a", "plan", "task-old"),
		"tasks/task-b.md":     note("task-b", "plan", "task-gone"),
		"tasks/task-c.md":     note("task-c", "plan", "task-a"),
		"tasks/task-live.md":  note("task-live", "plan", ""),
		"tasks/task-child.md": note("task-child", "parked", "task-live"),
	}
	writeAll(t, dir, files)
	m, problems, err := memory.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, p := range problems {
		got = append(got, p.Error())
	}
	all := strings.Join(got, "\n")
	if len(problems) != 2 || !strings.Contains(all, `tasks/task-b.md: parent "task-gone" does not exist`) ||
		!strings.Contains(all, `tasks/task-c.md: parent "task-a" is itself a subtask`) {
		t.Errorf("problems:\n%s", all)
	}
	for _, id := range []string{"task-a", "task-b", "task-c"} {
		if _, ok := m.Notes[id]; !ok {
			t.Errorf("%s left out", id)
		}
	}
	idx := m.Index("en")
	for _, want := range []string{"\n- [task-a]", "\n- [task-b]", "\n- [task-c]", "\n- [task-live]", "\n  - [task-child]"} {
		if !strings.Contains(idx, want) {
			t.Errorf("index lacks %q:\n%s", want, idx)
		}
	}
}

func TestSteps(t *testing.T) {
	b := open(t, "en")
	n, err := b.NewTask(NewTaskInput{Title: "Paint the fence"})
	if err != nil {
		t.Fatal(err)
	}
	id := n.Header.ID
	if _, err := b.Step(StepInput{ID: id, Number: 1, Done: true}); err == nil || !strings.Contains(err.Error(), "no steps") {
		t.Errorf("tick with no steps: %v", err)
	}
	for _, s := range []string{"Buy paint", "Sand the boards", "Paint  two coats"} {
		if _, err := b.AddStep(id, s); err != nil {
			t.Fatal(err)
		}
	}
	body := read(t, b, n.Path)
	if !strings.Contains(body, "## 3. Plan\nSteps in order, each with how to undo it.\n\n- [ ] Buy paint\n- [ ] Sand the boards\n- [ ] Paint two coats\n\n## 4. Validation") {
		t.Fatalf("plan after adding steps:\n%s", body)
	}
	if !strings.Contains(read(t, b, "MEMORY.md"), "· definition · 0/3 steps · waits on me") {
		t.Errorf("index without the step count:\n%s", read(t, b, "MEMORY.md"))
	}

	if _, err := b.Step(StepInput{ID: id, Number: 2, Done: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Step(StepInput{ID: id, Text: "two coats", Done: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Step(StepInput{ID: id, Text: "a", Done: true}); err == nil || !strings.Contains(err.Error(), "several") {
		t.Errorf("ambiguous text: %v", err)
	}
	if _, err := b.Step(StepInput{ID: id, Number: 9, Done: true}); err == nil {
		t.Error("step 9 of 3")
	}
	body = read(t, b, n.Path)
	if !strings.Contains(body, "- [ ] Buy paint\n- [x] Sand the boards\n- [x] Paint two coats\n") ||
		!strings.Contains(body, "- 2026-10-15 — Step 2 done: Sand the boards\n") {
		t.Errorf("after ticking:\n%s", body)
	}
	items, _ := b.Pending(0)
	if items[0].StepsDone != 2 || items[0].StepsTotal != 3 {
		t.Errorf("pending steps: %+v", items[0])
	}
	// Asking for what is already so changes nothing, not even the journal.
	before := read(t, b, n.Path)
	if _, err := b.Step(StepInput{ID: id, Number: 2, Done: true}); err != nil || read(t, b, n.Path) != before {
		t.Errorf("a no-op tick wrote: %v", err)
	}
	if _, err := b.Step(StepInput{ID: id, Number: 2}); err != nil {
		t.Fatal(err)
	}
	if done, total := memory.StepCount(read(t, b, n.Path)); done != 1 || total != 3 {
		t.Errorf("after unticking: %d/%d", done, total)
	}
}

func TestStepsOnlyInThePlan(t *testing.T) {
	body := "## Essentials\n- [ ] not a step\n\n## Plan\nText.\n```\n- [ ] in code\n```\n* [X] one\n  - [ ] two\n\n## Journal\n- [ ] not either\n"
	steps := memory.Steps(body)
	if len(steps) != 2 || steps[0].Text != "one" || !steps[0].Done || steps[1].Text != "two" || steps[1].Done {
		t.Fatalf("steps: %+v", steps)
	}
	if got := memory.SetStep(body, steps[0].Line, false); !strings.Contains(got, "* [ ] one\n") {
		t.Errorf("untick:\n%s", got)
	}
	if _, ok := memory.AddStep("## Essentials\n\n## Journal\n", "x"); ok {
		t.Error("a step added with no Plan section")
	}
	if got, _ := memory.AddStep("## Plan\n\n## Journal\n- a\n", "first"); got != "## Plan\n- [ ] first\n\n## Journal\n- a\n" {
		t.Errorf("empty plan:\n%q", got)
	}
}

func writeAll(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for rel, body := range files {
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// With Expect, the step must still have that text: a stale page never ticks another.
func TestStepExpect(t *testing.T) {
	b := open(t, "es")
	n, err := b.NewTask(NewTaskInput{Title: "Pintar"})
	if err != nil {
		t.Fatal(err)
	}
	id := n.Header.ID
	for _, s := range []string{"Lijar", "Imprimar"} {
		if _, err := b.AddStep(id, s); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := b.Step(StepInput{ID: id, Number: 1, Expect: "Imprimar", Done: true}); err == nil || !strings.Contains(err.Error(), "ya no dice") {
		t.Errorf("wrong text: %v", err)
	}
	if _, err := b.Step(StepInput{ID: id, Number: 3, Expect: "Imprimar", Done: true}); err == nil || !strings.Contains(err.Error(), "ya no dice") {
		t.Errorf("a step that is gone: %v", err)
	}
	if n, _ := b.Task(id); strings.Contains(n.Body, "- [x]") {
		t.Fatal("ticked a step it should not")
	}
	if _, err := b.Step(StepInput{ID: id, Number: 2, Expect: "Imprimar", Done: true}); err != nil {
		t.Fatal(err)
	}
	// The MCP tool's way (number, and a text it ignores) works as before.
	if _, err := b.Step(StepInput{ID: id, Number: 1, Text: "otra cosa", Done: true}); err != nil {
		t.Fatal(err)
	}
	if n, _ := b.Task(id); !strings.Contains(n.Body, "- [x] Lijar") || !strings.Contains(n.Body, "- [x] Imprimar") {
		t.Errorf("body:\n%s", n.Body)
	}
}

// The panel can empty fields; the MCP tool's blank still keeps them.
func TestUpdateClear(t *testing.T) {
	b := open(t, "en")
	n, err := b.NewTask(NewTaskInput{Title: "Fence", Summary: "north side"})
	if err != nil {
		t.Fatal(err)
	}
	id := n.Header.ID
	if n, err = b.Update(UpdateInput{ID: id}); err != nil || n.Header.Summary != "north side" || n.Header.Area == "" {
		t.Fatalf("blank keeps: %+v %v", n.Header, err)
	}
	if n, err = b.Update(UpdateInput{ID: id, Clear: []string{"summary", "area"}}); err != nil || n.Header.Summary != "" || n.Header.Area != "" {
		t.Fatalf("clear: %+v %v", n.Header, err)
	}
	if _, err := b.Update(UpdateInput{ID: id, Clear: []string{"next"}}); err == nil {
		t.Error("a live task without a next step was saved")
	}
	if _, err := b.Update(UpdateInput{ID: id, Clear: []string{"title"}}); err == nil {
		t.Error("cleared the title")
	}
}

// The panel takes out one capture, by date and exact text; the MCP tool takes out
// every capture with that text, and a text with spaces around it still matches.
func TestInboxDoneOneAndAll(t *testing.T) {
	b := open(t, "en")
	path := filepath.Join(b.Mem.Dir, "inbox.md")
	start := "- 2026-10-01 — Call Ana\n- 2026-10-02 — Call Ana\n- 2026-10-03 —  spaced\n"
	if err := os.WriteFile(path, []byte(start), 0o644); err != nil {
		t.Fatal(err)
	}
	if ok, err := b.InboxDoneEntry("2026-10-02", "Call Ana"); !ok || err != nil {
		t.Fatal(ok, err)
	}
	if ok, _ := b.InboxDoneEntry("2026-10-02", "Call Ana"); ok {
		t.Error("removed twice")
	}
	if data, _ := os.ReadFile(path); string(data) != "- 2026-10-01 — Call Ana\n- 2026-10-03 —  spaced\n" {
		t.Errorf("inbox:\n%s", data)
	}
	if err := os.WriteFile(path, []byte(start), 0o644); err != nil {
		t.Fatal(err)
	}
	missing, err := b.InboxDone([]string{"Call Ana", " spaced"})
	if err != nil || len(missing) != 0 {
		t.Fatal(missing, err)
	}
	if data, _ := os.ReadFile(path); string(data) != "" {
		t.Errorf("inbox:\n%q", data)
	}
}
