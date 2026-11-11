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
	t1, t2 := filepath.Join(dir, "s1.jsonl"), filepath.Join(dir, "s2.jsonl")
	for _, f := range []string{t1, t2} {
		if err := os.WriteFile(f, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(f, now, now); err != nil {
			t.Fatal(err)
		}
	}
	if err := Started(dir, "work", "s1", t1, "pid-10", now); err != nil {
		t.Fatal(err)
	}
	s1 := Who{ID: "s1"}
	if err := Touched(dir, "work", s1, "task-a", now, true); err != nil {
		t.Fatal(err)
	}
	if err := Started(dir, "work", "s2", t2, "pid-20", now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if se, ok := Active(dir, "work", Who{ID: "s2"}); !ok || se.Task != "" {
		t.Errorf("a new session took another one's task: %+v", se)
	}
	if err := Ended(dir, "work", "s2", now.Add(2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	s := Read(dir, "work")
	if se := s.Session(s1); se == nil || se.Task != "task-a" || !se.Since.Equal(now) {
		t.Errorf("active task: %+v", se)
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
	since := func(w Who) time.Time { se, _ := Active(dir, "work", w); return se.Since }
	// A read makes a task active without putting off the reminder; a write does.
	later := now.Add(time.Minute)
	if err := Touched(dir, "work", s1, "task-a", later, false); err != nil || !since(s1).Equal(now) {
		t.Errorf("a read moved Since: %v", since(s1))
	}
	// By the key of its process (the MCP server), the same session.
	if err := Touched(dir, "work", Who{Key: "pid-10"}, "task-a", later, true); err != nil || !since(s1).Equal(later) {
		t.Errorf("a write did not move Since: %v", since(s1))
	}
	// Nobody (the panel) and a key no session runs in change nothing.
	for _, w := range []Who{{}, {Key: "pid-99"}, {ID: "nope"}} {
		if err := Touched(dir, "work", w, "task-z", later, true); err != nil {
			t.Fatal(err)
		}
	}
	for _, se := range Read(dir, "work").Sessions {
		if se.Task == "task-z" {
			t.Errorf("%s got a task it never touched", se.ID)
		}
	}
	if err := Reminded(dir, "work", s1, later); err != nil {
		t.Fatal(err)
	}
	if se, _ := Active(dir, "work", s1); !se.Reminded.Equal(later) {
		t.Errorf("reminded: %+v", se)
	}
	if err := Released(dir, "work", "task-b", now); err != nil {
		t.Fatal(err)
	}
	if se, _ := Active(dir, "work", s1); se.Task != "task-a" {
		t.Error("releasing another task changed the active one")
	}
	if err := Released(dir, "work", "task-a", now); err != nil {
		t.Fatal(err)
	}
	if se, _ := Active(dir, "work", s1); se.Task != "" {
		t.Error("the parked task is still active")
	}
	// Old sessions are forgotten (their transcripts are gone too).
	if err := Started(dir, "work", "s4", "", "", now.Add(40*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if got := Read(dir, "work").Sessions; len(got) != 1 || got[0].ID != "s4" {
		t.Errorf("after 40 days: %+v", got)
	}
	if len(Read(dir, "personal").Sessions) != 0 {
		t.Error("spaces share state")
	}
}

// After /clear the process goes on under a new session id: its key names the new one,
// and a resumed session takes the key of the process it now runs in.
func TestSessionByKey(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC)
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(Started(dir, "work", "a", "", "pid-10", now))
	must(Touched(dir, "work", Who{Key: "pid-10"}, "task-a", now, true))
	must(Ended(dir, "work", "a", now.Add(time.Minute))) // /clear
	must(Started(dir, "work", "b", "", "pid-10", now.Add(time.Minute)))
	must(Touched(dir, "work", Who{Key: "pid-10"}, "task-b", now.Add(2*time.Minute), true))
	if se, _ := Active(dir, "work", Who{ID: "a"}); se.Task != "task-a" {
		t.Errorf("the ended session's task changed: %+v", se)
	}
	if se, _ := Active(dir, "work", Who{Key: "pid-10"}); se.ID != "b" || se.Task != "task-b" {
		t.Errorf("by key: %+v", se)
	}
	must(Started(dir, "work", "a", "", "pid-30", now.Add(time.Hour))) // claude --resume a
	if se, _ := Active(dir, "work", Who{Key: "pid-30"}); se.ID != "a" || se.Task != "task-a" {
		t.Errorf("resumed: %+v", se)
	}
}

// What earlier versions kept for the whole space is not read, and goes at the next write.
func TestOldSpaceTaskIgnored(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, Dir), 0o700); err != nil {
		t.Fatal(err)
	}
	old := `{"task":"task-old","since":"2026-10-20T09:00:00Z","sessions":[{"id":"s","started":"2026-10-20T09:00:00Z","task":"task-old"}]}`
	if err := os.WriteFile(file(dir, "work"), []byte(old), 0o600); err != nil {
		t.Fatal(err)
	}
	if se, ok := Active(dir, "work", Who{ID: "s"}); !ok || se.Task != "task-old" || !se.Since.IsZero() {
		t.Errorf("old session: %+v %v", se, ok)
	}
	if err := Started(dir, "work", "t", "", "", time.Date(2026, 10, 20, 10, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(file(dir, "work"))
	if strings.Contains(string(data), `"since": "2026-10-20T09`) {
		t.Errorf("the space's old active task is still written:\n%s", data)
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
