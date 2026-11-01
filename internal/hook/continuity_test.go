package hook

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/icalvete/augmented-brain/internal/brain"
	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/state"
)

func clock(t *testing.T, at time.Time) *time.Time {
	t.Helper()
	cur := at
	old := Now
	Now = func() time.Time { return cur }
	t.Cleanup(func() { Now = old })
	return &cur
}

func openTask(t *testing.T, cfgDir string) string {
	t.Helper()
	c, err := config.Load(cfgDir)
	if err != nil {
		t.Fatal(err)
	}
	b, err := brain.Open(c, c.Spaces["work"])
	if err != nil {
		t.Fatal(err)
	}
	n, err := b.NewTask(brain.NewTaskInput{Title: "Move the shop"})
	if err != nil {
		t.Fatal(err)
	}
	return n.Header.ID
}

// The Stop hook reminds to journal after a while without writing, once, and only when a
// task is active.
func TestStopReminder(t *testing.T) {
	cfgDir, work := setupSpace(t)
	start := time.Date(2026, 10, 20, 9, 0, 0, 0, time.Local)
	now := clock(t, start)
	stop := func(active bool) string {
		o := run(t, func(in *bytes.Reader, out *bytes.Buffer) error { return Stop(in, out, cfgDir) },
			map[string]any{"cwd": work, "hook_event_name": "Stop", "stop_hook_active": active})
		if o == nil {
			return ""
		}
		if o["hookEventName"] != "Stop" {
			t.Errorf("event: %v", o)
		}
		return o["additionalContext"].(string)
	}
	*now = start.Add(time.Hour)
	if got := stop(false); got != "" {
		t.Errorf("no active task, yet a reminder: %s", got)
	}
	id := openTask(t, cfgDir)
	if err := state.Touched(cfgDir, "work", id, start, true); err != nil {
		t.Fatal(err)
	}
	*now = start.Add(10 * time.Minute)
	if got := stop(false); got != "" {
		t.Errorf("too soon: %s", got)
	}
	*now = start.Add(25 * time.Minute)
	if got := stop(true); got != "" {
		t.Errorf("while a Stop hook already continues: %s", got)
	}
	got := stop(false)
	if !strings.Contains(got, id) || !strings.Contains(got, "since 09:00") || !strings.Contains(got, "task_journal") {
		t.Errorf("reminder: %s", got)
	}
	*now = start.Add(2 * time.Hour)
	if got := stop(false); got != "" {
		t.Errorf("reminded twice without a write in between: %s", got)
	}
	if err := state.Touched(cfgDir, "work", id, *now, true); err != nil {
		t.Fatal(err)
	}
	*now = now.Add(30 * time.Minute)
	if got := stop(false); got == "" {
		t.Error("no reminder after a new write and another while")
	}
}

// After a compaction the summary is kept in archive/sessions/, secrets out, with a
// journal line in the active task; the next session start asks to journal.
func TestCompaction(t *testing.T) {
	cfgDir, work := setupSpace(t)
	clock(t, time.Date(2026, 10, 20, 11, 30, 0, 0, time.Local))
	id := openTask(t, cfgDir)
	if err := state.Touched(cfgDir, "work", id, Now(), true); err != nil {
		t.Fatal(err)
	}
	token := "gh" + "p_" + strings.Repeat("a", 36)
	pc := func(in *bytes.Reader, _ *bytes.Buffer) error { return PostCompact(in, cfgDir) }
	run(t, pc, map[string]any{"cwd": work, "hook_event_name": "PostCompact", "session_id": "abcdef123456",
		"trigger": "auto", "compact_summary": "We chose port 8443. The token was " + token + "."})
	c, _ := config.Load(cfgDir)
	mem := c.Spaces["work"].Memory
	rel := "archive/sessions/2026-10-20-113000-abcdef12.md"
	data, err := os.ReadFile(filepath.Join(mem, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "port 8443") || strings.Contains(string(data), token) || !strings.Contains(string(data), "removed]") {
		t.Errorf("summary file:\n%s", data)
	}
	task, _ := os.ReadFile(filepath.Join(mem, "tasks", id+".md"))
	if !strings.Contains(string(task), "Claude Code's summary is in "+rel) {
		t.Errorf("no journal line:\n%s", task)
	}

	ss := func(in *bytes.Reader, out *bytes.Buffer) error { return SessionStart(in, out, cfgDir) }
	o := run(t, ss, map[string]any{"cwd": work, "source": "compact", "session_id": "abcdef123456"})
	if text := o["additionalContext"].(string); !strings.Contains(text, "just compacted") || !strings.Contains(text, "task "+id) {
		t.Errorf("after compaction:\n%s", text)
	}
	o = run(t, ss, map[string]any{"cwd": work, "source": "startup", "session_id": "other"})
	if strings.Contains(o["additionalContext"].(string), "just compacted") {
		t.Error("a normal start asks to journal after a compaction")
	}
}

// A session that never ended is offered for recovery once, at the next start; one that
// ended is not.
func TestRecoveryOffer(t *testing.T) {
	cfgDir, work := setupSpace(t)
	clock(t, time.Date(2026, 10, 20, 9, 0, 0, 0, time.Local))
	transcript := filepath.Join(t.TempDir(), "s1.jsonl")
	if err := os.WriteFile(transcript, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(transcript, Now().Add(-time.Hour), Now().Add(-time.Hour)); err != nil { // not running any more
		t.Fatal(err)
	}
	ss := func(in *bytes.Reader, out *bytes.Buffer) error { return SessionStart(in, out, cfgDir) }
	start := func(id string) string {
		o := run(t, ss, map[string]any{"cwd": work, "source": "startup", "session_id": id, "transcript_path": transcript})
		return o["additionalContext"].(string)
	}
	end := func(id string) {
		if err := SessionEnd(strings.NewReader(`{"cwd":"`+filepath.ToSlash(work)+`","session_id":"`+id+`"}`), cfgDir); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("AB_NO_AUTO_BACKUP", "1")
	if text := start("s1"); strings.Contains(text, "session_recover") {
		t.Error("offered with no earlier session")
	}
	end("s1")
	if text := start("s2"); strings.Contains(text, "session_recover") {
		t.Error("a session that ended was offered")
	}
	// s2 dies: no SessionEnd.
	if text := start("s3"); !strings.Contains(text, "did not end normally") || !strings.Contains(text, "session_recover") {
		t.Errorf("no offer for the session that died:\n%s", text)
	}
	end("s3")
	if text := start("s4"); strings.Contains(text, "session_recover") {
		t.Error("offered twice")
	}
}

// Sessions outside every space's folder, which use the default space, leave nothing in
// it: no summary, no reminder, no session record.
func TestNothingFromOutsideTheSpaces(t *testing.T) {
	cfgDir, _ := setupSpace(t)
	if err := config.SetDefaultSpace(cfgDir, "work"); err != nil {
		t.Fatal(err)
	}
	clock(t, time.Date(2026, 10, 20, 11, 30, 0, 0, time.Local))
	outside := t.TempDir()
	run(t, func(in *bytes.Reader, _ *bytes.Buffer) error { return PostCompact(in, cfgDir) },
		map[string]any{"cwd": outside, "session_id": "s1", "compact_summary": "private things"})
	c, _ := config.Load(cfgDir)
	if _, err := os.Stat(filepath.Join(c.Spaces["work"].Memory, "archive", "sessions")); err == nil {
		t.Error("a summary from outside the spaces was saved in the default space")
	}
	run(t, func(in *bytes.Reader, out *bytes.Buffer) error { return SessionStart(in, out, cfgDir) },
		map[string]any{"cwd": outside, "source": "startup", "session_id": "s1"})
	if st := state.Read(cfgDir, "work"); len(st.Sessions) != 0 {
		t.Errorf("a session outside the spaces was recorded: %+v", st.Sessions)
	}
}

// After a restore with no guard lists, the next session offers once to ask for them.
func TestAskSafetyAfterRestore(t *testing.T) {
	cfgDir, work := setupSpace(t)
	if err := config.SetSpaceList(cfgDir, "work", "production_aws_profiles", nil); err != nil {
		t.Fatal(err)
	}
	c, _ := config.Load(cfgDir)
	b, _ := brain.Open(c, c.Spaces["work"])
	if _, err := b.SaveProfile(brain.ProfileInput{Name: "Dana"}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, config.AskSafetyFile), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	ss := func(in *bytes.Reader, out *bytes.Buffer) error { return SessionStart(in, out, cfgDir) }
	if text := run(t, ss, map[string]any{"cwd": work})["additionalContext"].(string); !strings.Contains(text, "safety_save") {
		t.Errorf("no offer:\n%s", text)
	}
	if text := run(t, ss, map[string]any{"cwd": work})["additionalContext"].(string); strings.Contains(text, "backups do not carry") {
		t.Error("offered twice")
	}
}
