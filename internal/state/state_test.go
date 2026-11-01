package state

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestSessionsAndActiveTask(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC)
	if err := Touched(dir, "work", "task-a", now, true); err != nil {
		t.Fatal(err)
	}
	t1, t2 := filepath.Join(dir, "s1.jsonl"), filepath.Join(dir, "s2.jsonl")
	for _, f := range []string{t1, t2} {
		if err := os.WriteFile(f, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(f, now, now); err != nil {
			t.Fatal(err)
		}
	}
	if err := Started(dir, "work", "s1", t1, now); err != nil {
		t.Fatal(err)
	}
	if err := Started(dir, "work", "s2", t2, now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := Ended(dir, "work", "s2", now.Add(2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	s := Read(dir, "work")
	if s.Task != "task-a" || !s.Since.Equal(now) {
		t.Errorf("active task: %+v", s)
	}
	un := s.Unclosed("s3", now.Add(3*time.Hour))
	if len(un) != 1 || un[0].ID != "s1" || un[0].Task != "task-a" || un[0].Transcript != t1 {
		t.Errorf("unclosed: %+v", un)
	}
	if len(s.Unclosed("s1", now.Add(3*time.Hour))) != 0 {
		t.Error("the current session was offered")
	}
	if len(s.Unclosed("s3", now.Add(5*time.Minute))) != 0 {
		t.Error("a session still writing its transcript was offered")
	}
	// A read makes a task active without putting off the reminder; a write does.
	later := now.Add(time.Minute)
	if err := Touched(dir, "work", "task-a", later, false); err != nil || !Read(dir, "work").Since.Equal(now) {
		t.Errorf("a read moved Since: %v", Read(dir, "work").Since)
	}
	if err := Touched(dir, "work", "task-a", later, true); err != nil || !Read(dir, "work").Since.Equal(later) {
		t.Errorf("a write did not move Since: %v", Read(dir, "work").Since)
	}
	if err := Released(dir, "work", "task-b", now); err != nil || Read(dir, "work").Task != "task-a" {
		t.Error("releasing another task changed the active one")
	}
	if err := Released(dir, "work", "task-a", now); err != nil || Read(dir, "work").Task != "" {
		t.Error("the parked task is still active")
	}
	// Old sessions are forgotten (their transcripts are gone too).
	if err := Started(dir, "work", "s4", "", now.Add(40*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if got := Read(dir, "work").Sessions; len(got) != 1 || got[0].ID != "s4" {
		t.Errorf("after 40 days: %+v", got)
	}
	if Read(dir, "personal").Task != "" {
		t.Error("spaces share state")
	}
}

// The transcript fixture mimics Claude Code's JSONL: text of the user and the assistant
// comes back, tool calls and their output do not, nor anything before since, nor a line
// that is not JSON.
func TestConversation(t *testing.T) {
	lines := []string{
		`{"type":"user","timestamp":"2026-10-20T08:00:00.000Z","message":{"role":"user","content":"too early"}}`,
		`{"type":"summary","summary":"x"}`,
		`not json at all`,
		`{"type":"user","timestamp":"2026-10-20T10:00:00.000Z","message":{"role":"user","content":"Let's use port 8443."}}`,
		`{"type":"assistant","timestamp":"2026-10-20T10:00:05.000Z","message":{"role":"assistant","content":[{"type":"thinking","thinking":"hmm"},{"type":"text","text":"Agreed: 8443."},{"type":"tool_use","id":"t1","name":"Bash","input":{"command":"ls"}}]}}`,
		`{"type":"user","timestamp":"2026-10-20T10:00:06.000Z","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"t1","content":"secret output"}]}}`,
		`{"type":"user","isMeta":true,"timestamp":"2026-10-20T10:00:07.000Z","message":{"role":"user","content":"meta"}}`,
		`{"type":"assistant","timestamp":"2026-10-20T10:01:00.000Z","message":{"role":"assistant","content":[{"type":"text","text":"Done."}]}}`,
	}
	path := filepath.Join(t.TempDir(), "s.jsonl")
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	since := time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC)
	got, err := Conversation(path, since, 10000)
	if err != nil {
		t.Fatal(err)
	}
	want := "user: Let's use port 8443.\n\nassistant: Agreed: 8443.\n\nassistant: Done."
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
	// Capped: the most recent part is kept.
	if got, _ := Conversation(path, since, 20); !strings.Contains(got, "Done.") || strings.Contains(got, "8443") {
		t.Errorf("capped: %q", got)
	}
	if _, err := Conversation(filepath.Join(t.TempDir(), "gone.jsonl"), since, 100); err == nil {
		t.Error("a missing transcript must say so")
	}
}

// The cap counts characters and never cuts one in two.
func TestConversationCutsOnCharacters(t *testing.T) {
	path := filepath.Join(t.TempDir(), "s.jsonl")
	line := `{"type":"user","timestamp":"2026-10-20T10:00:00Z","message":{"content":"ñandú ñandú ñandú"}}`
	if err := os.WriteFile(path, []byte(line+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for max := 1; max < 20; max++ {
		got, err := Conversation(path, time.Time{}, max)
		if err != nil || !utf8.ValidString(got) || utf8.RuneCountInString(got) > max+1 {
			t.Errorf("max %d: %q %v", max, got, err)
		}
	}
}
