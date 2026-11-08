package brain

import (
	"strings"
	"testing"
)

// Skipping phases is allowed and written down; a closed task is not closed again, and
// opens again only with a reason.
func TestPhases(t *testing.T) {
	b := open(t, "en")
	n, err := b.NewTask(NewTaskInput{Title: "Bake bread"})
	if err != nil {
		t.Fatal(err)
	}
	id := n.Header.ID
	n, err = b.Update(UpdateInput{ID: id, Status: "execution"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(n.Body, "skipping analysis, plan") {
		t.Errorf("the skip is not written:\n%s", n.Body)
	}
	n, err = b.Update(UpdateInput{ID: id, Status: "validation"})
	if err != nil || strings.Contains(n.Body, "skipping execution") {
		t.Fatalf("one step is no skip: %v", err)
	}
	if _, err := b.Close(id, "two loaves"); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Close(id, "something else"); err == nil || !strings.Contains(err.Error(), "already closed") {
		t.Errorf("closed twice: %v", err)
	}
	if n, _ := b.Task(id); !strings.Contains(n.Body, "two loaves") || strings.Contains(n.Body, "something else") {
		t.Errorf("the result was written over:\n%s", n.Body)
	}
	if _, err := b.Update(UpdateInput{ID: id, Status: "execution"}); err == nil || !strings.Contains(err.Error(), "reopen") {
		t.Errorf("reopened without a reason: %v", err)
	}
	n, err = b.Update(UpdateInput{ID: id, Status: "execution", WaitingOn: "nobody", Next: "bake again", Reopen: "the loaves burnt"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(n.Body, "Reopened (now execution): the loaves burnt") {
		t.Errorf("the reason is not in the journal:\n%s", n.Body)
	}
}
