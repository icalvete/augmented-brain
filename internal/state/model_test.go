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
