package state

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// LastModel reads only the end of a long transcript, where the first line may be cut.
func TestLastModelLongTranscript(t *testing.T) {
	p := filepath.Join(t.TempDir(), "s.jsonl")
	filler := `{"type":"assistant","message":{"model":"claude-opus-5-5","content":"` + strings.Repeat("x", 4000) + `"}}` + "\n"
	body := strings.Repeat(filler, 300) + `{"type":"assistant","message":{"model":"claude-sonnet-5-5"}}` + "\n" +
		`{"type":"user","message":{"content":"assistant"}}` + "\n"
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := LastModel(p); got != "claude-sonnet-5-5" {
		t.Errorf("got %q", got)
	}
	if got := LastModel(filepath.Join(t.TempDir(), "none.jsonl")); got != "" {
		t.Errorf("missing file: %q", got)
	}
}

// LastAnswer tells a session whose turn ended in an API error, as Claude Code writes it
// when the login has expired.
func TestLastAnswer(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	prompt := `{"type":"user","message":{"role":"user","content":"start the task"}}` + "\n"
	failed := `{"type":"assistant","message":{"model":"<synthetic>","role":"assistant","content":[{"type":"text","text":"Login reached its time limit · Please run /login"}]},"error":"authentication_failed","isApiErrorMessage":true}` + "\n"
	ok := `{"type":"assistant","message":{"model":"claude-opus-5-5","content":[{"type":"text","text":"On it."}]}}` + "\n"
	tool := `{"type":"user","message":{"role":"user","content":[{"type":"tool_result","content":"x"}]}}` + "\n"
	cases := []struct {
		name, body string
		answered   bool
		err        string
	}{
		{"none.jsonl", "", false, ""},
		{"prompt.jsonl", prompt, false, ""},
		{"failed.jsonl", prompt + failed, true, "Login reached its time limit · Please run /login"},
		{"failed-tool.jsonl", prompt + failed + tool, true, "Login reached its time limit · Please run /login"},
		{"retried.jsonl", prompt + failed + prompt, true, ""},
		{"recovered.jsonl", prompt + failed + prompt + ok, true, ""},
		{"ok.jsonl", prompt + ok, true, ""},
	}
	for _, c := range cases {
		p := filepath.Join(dir, "missing")
		if c.body != "" {
			p = write(c.name, c.body)
		}
		answered, err := LastAnswer(p)
		if answered != c.answered || err != c.err {
			t.Errorf("%s: got %v %q, want %v %q", c.name, answered, err, c.answered, c.err)
		}
	}
}
