package state

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// transcript writes an empty transcript named after the session, last written at mod.
func transcript(t *testing.T, dir, id string, mod time.Time) string {
	t.Helper()
	p := filepath.Join(dir, id+".jsonl")
	if err := os.WriteFile(p, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(p, mod, mod); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestListKinds(t *testing.T) {
	cfg, proj := t.TempDir(), t.TempDir()
	now := time.Now()
	old := now.Add(-2 * time.Hour)
	for id, mod := range map[string]time.Time{"live": now.Add(-time.Minute), "dead": old, "closed": old} {
		if err := Started(cfg, "work", id, transcript(t, proj, id, mod), old); err != nil {
			t.Fatal(err)
		}
	}
	if err := Started(cfg, "work", "gone", filepath.Join(proj, "gone.jsonl"), old); err != nil {
		t.Fatal(err)
	}
	if err := Ended(cfg, "work", "closed", now); err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, l := range Read(cfg, "work").List(now) {
		got[l.ID] = l.Kind
		if l.Size != 3 {
			t.Errorf("%s: size %d", l.ID, l.Size)
		}
	}
	want := map[string]string{"live": Live, "dead": Dead, "closed": Closed}
	if len(got) != len(want) {
		t.Errorf("listed %v, want %v (an unended session without a transcript is left out)", got, want)
	}
	for id, k := range want {
		if got[id] != k {
			t.Errorf("%s: %q, want %q", id, got[id], k)
		}
	}
}

func TestFrom(t *testing.T) {
	proj := t.TempDir()
	start := time.Date(2026, 10, 20, 9, 0, 0, 0, time.Local)
	se := Session{ID: "s", Task: "task-a", Started: start, Transcript: transcript(t, proj, "s", start.Add(5*time.Hour))}
	if f := (Space{}).From(se, time.Time{}); !f.Equal(start) {
		t.Errorf("nothing written: %v", f)
	}
	write := start.Add(2 * time.Hour)
	s := Space{Task: "task-a", Since: write}
	if f := s.From(se, time.Time{}); !f.Equal(write) {
		t.Errorf("written while alive: %v", f)
	}
	if f := (Space{Task: "task-a", Since: start.Add(6 * time.Hour)}).From(se, time.Time{}); !f.Equal(start) {
		t.Errorf("written after it died (by another session): %v", f)
	}
	if f := (Space{Task: "task-b", Since: write}).From(se, time.Time{}); !f.Equal(start) {
		t.Errorf("another task written: %v", f)
	}
	entry := start.Add(3 * time.Hour)
	if f := s.From(se, entry); !f.Equal(entry) {
		t.Errorf("a later journal date: %v", f)
	}
}

func TestDeleteTranscript(t *testing.T) {
	cfg, proj := t.TempDir(), t.TempDir()
	now := time.Now()
	old := now.Add(-2 * time.Hour)
	add := func(id, path string, closed bool) {
		t.Helper()
		if err := Started(cfg, "work", id, path, old); err != nil {
			t.Fatal(err)
		}
		if closed {
			if err := Ended(cfg, "work", id, old); err != nil {
				t.Fatal(err)
			}
		}
	}
	add("done", transcript(t, proj, "done", old), true)
	side := filepath.Join(proj, "done", "tool-results")
	if err := os.MkdirAll(side, 0o755); err != nil {
		t.Fatal(err)
	}
	add("dead", transcript(t, proj, "dead", old), false)
	add("fresh", transcript(t, proj, "fresh", now), true)
	other := filepath.Join(proj, "notes.md") // not named after the session
	if err := os.WriteFile(other, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	_ = os.Chtimes(other, old, old)
	add("odd", other, true)

	for _, id := range []string{"dead", "fresh", "odd", "missing", ""} {
		if err := DeleteTranscript(cfg, "work", id, now); !errors.Is(err, ErrNotDeletable) {
			t.Errorf("%q: %v, want ErrNotDeletable", id, err)
		}
	}
	for _, p := range []string{filepath.Join(proj, "dead.jsonl"), filepath.Join(proj, "fresh.jsonl"), other} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("%s was deleted", p)
		}
	}
	if err := DeleteTranscript(cfg, "work", "done", now); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(proj, "done.jsonl")); !errors.Is(err, os.ErrNotExist) {
		t.Error("the transcript is still there")
	}
	if _, err := os.Stat(filepath.Join(proj, "done")); !errors.Is(err, os.ErrNotExist) {
		t.Error("the folder next to it is still there")
	}
	for _, se := range Read(cfg, "work").Sessions {
		if se.ID == "done" && se.Transcript != "" {
			t.Error("the state still points at the transcript")
		}
	}
}
