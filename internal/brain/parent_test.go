package brain

import (
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/memory"
)

// TestUpdateParent: task_update gives a task a parent, moves it to another, and takes
// it away, with the checks of a new subtask and one level only.
func TestUpdateParent(t *testing.T) {
	b := open(t, "en")
	mk := func(in NewTaskInput) string {
		t.Helper()
		n, err := b.NewTask(in)
		if err != nil {
			t.Fatal(err)
		}
		return n.Header.ID
	}
	garden, err := b.NewArea("Garden", "the garden")
	if err != nil {
		t.Fatal(err)
	}
	shop := mk(NewTaskInput{Title: "Move the shop online"})
	roses := mk(NewTaskInput{Title: "Plant roses", Area: garden.Header.ID})
	loose := mk(NewTaskInput{Title: "Pick a payment provider"})
	sub := mk(NewTaskInput{Title: "Pick a domain", Parent: shop})

	// Given a parent: the header, the journal, and the parent's area.
	n, err := b.Update(UpdateInput{ID: loose, Parent: roses})
	if err != nil {
		t.Fatal(err)
	}
	if n.Header.Parent != roses || n.Header.Area != garden.Header.ID {
		t.Errorf("header after a parent: %+v", n.Header)
	}
	if text := read(t, b, n.Path); !strings.Contains(text, "parent: "+roses) ||
		!strings.Contains(text, "Now a subtask of "+roses) || !strings.Contains(text, "Area changed to its parent's") {
		t.Errorf("note after a parent:\n%s", text)
	}
	if kids := b.Mem.Children(roses); len(kids) != 1 || kids[0].Header.ID != loose {
		t.Errorf("children of %s: %v", roses, kids)
	}

	// Moved to another parent, keeping the area it is given.
	n, err = b.Update(UpdateInput{ID: loose, Parent: shop, Area: garden.Header.ID})
	if err != nil {
		t.Fatal(err)
	}
	if n.Header.Parent != shop || n.Header.Area != garden.Header.ID {
		t.Errorf("header after a move: %+v", n.Header)
	}
	if !strings.Contains(read(t, b, n.Path), "Parent changed: "+roses+" → "+shop) {
		t.Error("the move is not in the journal")
	}

	// The same parent again changes nothing.
	before := read(t, b, n.Path)
	if _, err := b.Update(UpdateInput{ID: loose, Parent: shop}); err != nil {
		t.Fatal(err)
	}
	if read(t, b, n.Path) != before {
		t.Error("the same parent changed the note")
	}

	// Refused: a missing parent, an area, itself, a subtask (one level), a task with
	// subtasks of its own, and a closed task.
	for _, c := range []struct{ id, parent, want string }{
		{loose, "task-nowhere", "no task"},
		{loose, "area-general", "not a task"},
		{roses, roses, "its own parent"},
		{roses, sub, "one level"},
		{shop, roses, "subtasks of its own"},
	} {
		if _, err := b.Update(UpdateInput{ID: c.id, Parent: c.parent}); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s under %s: %v, want %q", c.id, c.parent, err, c.want)
		}
	}
	if _, err := b.Close(roses, "done"); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Update(UpdateInput{ID: sub, Parent: roses}); err == nil || !strings.Contains(err.Error(), "closed") {
		t.Errorf("a closed parent: %v", err)
	}

	// Taken away: none, and Clear "parent" from the panel.
	n, err = b.Update(UpdateInput{ID: loose, Parent: NoParent})
	if err != nil {
		t.Fatal(err)
	}
	if n.Header.Parent != "" || !strings.Contains(read(t, b, n.Path), "No longer a subtask of "+shop) {
		t.Errorf("parent not taken away: %+v", n.Header)
	}
	n, err = b.Update(UpdateInput{ID: sub, Clear: []string{"parent"}})
	if err != nil {
		t.Fatal(err)
	}
	if n.Header.Parent != "" {
		t.Errorf("Clear parent: %+v", n.Header)
	}
	if _, problems, _ := memory.Load(b.Mem.Dir); len(problems) > 0 {
		t.Errorf("check: %v", problems)
	}
}
