package mcpserver

import (
	"strings"
	"testing"
)

// The tools the coverage map of the beta tests found tested only underneath, in brain,
// and the errors no test asked for through MCP.

// review_done through MCP: the review is saved, a processed capture leaves the inbox,
// and one that is not there is named, not an error.
func TestReviewDoneThroughMCP(t *testing.T) {
	cs := connect(t, space(t))
	if _, isErr := call(t, cs, "capture", map[string]any{"text": "buy flour"}); isErr {
		t.Fatal("capture failed")
	}
	text, isErr := call(t, cs, "review_done", map[string]any{"summary": "flour goes to the shopping task", "processed": []string{"buy flour", "something never captured"}})
	if isErr || !strings.Contains(text, "Review saved in") || !strings.Contains(text, "something never captured") {
		t.Fatalf("review_done: %v %q", isErr, text)
	}
	if text, _ := call(t, cs, "weekly_review", map[string]any{}); strings.Contains(text, "buy flour") {
		t.Errorf("the processed capture is still in the inbox:\n%s", text)
	}
}

// secret_name_save through MCP: a name is kept, a value is refused.
func TestSecretNameSaveThroughMCP(t *testing.T) {
	cs := connect(t, space(t))
	if text, isErr := call(t, cs, "secret_name_save", map[string]any{"service": "mail", "name": "smtp", "what": "sending mail"}); isErr {
		t.Fatalf("a name: %q", text)
	}
	token := "gh" + "p_" + strings.Repeat("a1B2", 9)
	if text, isErr := call(t, cs, "secret_name_save", map[string]any{"service": "github", "name": token}); !isErr {
		t.Errorf("a value was kept as a name: %q", text)
	}
}

// A required parameter left out is an error that names it, never a note written with
// it empty.
func TestMissingParameters(t *testing.T) {
	cs := connect(t, space(t))
	for _, c := range []struct {
		tool string
		args map[string]any
	}{
		{"task_new", map[string]any{}},
		{"task_get", map[string]any{}},
		{"task_journal", map[string]any{"id": "task-x"}},
		{"capture", map[string]any{}},
		{"secret_name_save", map[string]any{"service": "mail"}},
		{"knowledge_new", map[string]any{}},
	} {
		if text, isErr := call(t, cs, c.tool, c.args); !isErr {
			t.Errorf("%s with %v went through: %q", c.tool, c.args, text)
		}
	}
	if text, _ := call(t, cs, "pending", map[string]any{}); strings.Contains(text, "task-") {
		t.Errorf("a task was made:\n%s", text)
	}
}

// A secret in a task's text is refused through MCP too, not only in a capture.
func TestSecretRefusedInATask(t *testing.T) {
	cs := connect(t, space(t))
	key := "AKIA" + strings.Repeat("Q7", 8)
	if text, isErr := call(t, cs, "task_new", map[string]any{"title": "Rotate keys", "summary": "the key is " + key}); !isErr {
		t.Errorf("a key went into a task: %q", text)
	}
}
