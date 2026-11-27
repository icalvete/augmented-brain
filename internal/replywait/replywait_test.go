package replywait

import (
	"testing"
	"time"
)

func ids(ms []Message) []int {
	var out []int
	for _, m := range ms {
		out = append(out, m.ID)
	}
	return out
}

func TestRepliesOneToOne(t *testing.T) {
	w := Wait{Chat: "user:7", After: 10}
	msgs := []Message{
		{ID: 9, Text: "older"},
		{ID: 10, Out: true, Text: "the question"},
		{ID: 11, Out: true, Text: "a second line of ours"},
		{ID: 12, Text: "sí"},
		{ID: 13, Text: "y otra cosa"},
	}
	if got := ids(w.Replies(msgs)); len(got) != 2 || got[0] != 12 || got[1] != 13 {
		t.Fatalf("replies %v, want [12 13]", got)
	}
}

func TestRepliesInAGroupOnlyAnswers(t *testing.T) {
	w := Wait{Chat: "channel:5", After: 100}
	msgs := []Message{
		{ID: 101, Text: "unrelated chatter"},
		{ID: 102, Text: "ok gitlab", ReplyTo: 100},
		{ID: 103, Out: true, Text: "thanks", ReplyTo: 102},
		{ID: 104, Text: "answer to our thanks", ReplyTo: 103},
		{ID: 105, Text: "answer to someone else", ReplyTo: 101},
	}
	if got := ids(w.Replies(msgs)); len(got) != 2 || got[0] != 102 || got[1] != 104 {
		t.Fatalf("replies %v, want [102 104]", got)
	}
	if !w.Group() || (Wait{Chat: "group:1"}).Group() == false || (Wait{Chat: "user:1"}).Group() {
		t.Fatal("Group")
	}
}

func TestAddListRemove(t *testing.T) {
	dir := t.TempDir()
	if ws, err := List(dir); err != nil || len(ws) != 0 {
		t.Fatalf("empty: %v %v", ws, err)
	}
	a := Wait{Space: "personal", Task: "task-a", Chat: "user:7", After: 10, Who: "Lucía"}
	b := Wait{Space: "personal", Task: "task-b", Chat: "user:8", After: 20}
	for _, w := range []Wait{a, b} {
		if err := Add(dir, w); err != nil {
			t.Fatal(err)
		}
	}
	a2 := a
	a2.After = 15
	if err := Add(dir, a2); err != nil {
		t.Fatal(err)
	}
	ws, err := List(dir)
	if err != nil || len(ws) != 2 {
		t.Fatalf("list: %v %v", ws, err)
	}
	if ws[1].Task != "task-a" || ws[1].After != 15 || ws[1].Since.IsZero() || time.Since(ws[1].Since) > time.Minute {
		t.Fatalf("the same task and chat replaces its wait: %+v", ws)
	}
	n, err := Remove(dir, func(w Wait) bool { return w.Task == "task-b" })
	if err != nil || n != 1 {
		t.Fatalf("remove: %d %v", n, err)
	}
	if ws, _ := List(dir); len(ws) != 1 || ws[0].Task != "task-a" {
		t.Fatalf("after remove: %+v", ws)
	}
}
