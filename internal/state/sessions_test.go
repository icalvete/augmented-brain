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
		if err := Started(cfg, "work", id, transcript(t, proj, id, mod), "", old); err != nil {
			t.Fatal(err)
		}
	}
	if err := Started(cfg, "work", "gone", filepath.Join(proj, "gone.jsonl"), "", old); err != nil {
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
	if f := se.From(time.Time{}); !f.Equal(start) {
		t.Errorf("nothing written: %v", f)
	}
	write := start.Add(2 * time.Hour)
	at := func(task string, since time.Time) Session { x := se; x.Task, x.Since = task, since; return x }
	if f := at("task-a", write).From(time.Time{}); !f.Equal(write) {
		t.Errorf("written while alive: %v", f)
	}
	if f := at("task-a", start.Add(6*time.Hour)).From(time.Time{}); !f.Equal(start) {
		t.Errorf("written after it died: %v", f)
	}
	if f := at("", write).From(time.Time{}); !f.Equal(start) {
		t.Errorf("no task active: %v", f)
	}
	entry := start.Add(3 * time.Hour)
	if f := at("task-a", write).From(entry); !f.Equal(entry) {
		t.Errorf("a later journal date: %v", f)
	}
}

func TestDeleteTranscript(t *testing.T) {
	cfg, proj := t.TempDir(), t.TempDir()
	now := time.Now()
	old := now.Add(-2 * time.Hour)
	add := func(id, path string, closed bool) {
		t.Helper()
		if err := Started(cfg, "work", id, path, "", old); err != nil {
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

// Unclosed is List's dead sessions, without the one asking: the hook that offers a
// recovery and the panel that lists them never disagree.
func TestUnclosedIsDead(t *testing.T) {
	cfg, proj := t.TempDir(), t.TempDir()
	now := time.Now()
	old := now.Add(-2 * time.Hour)
	for _, id := range []string{"a", "b", "live", "closed"} {
		mod := old
		if id == "live" {
			mod = now
		}
		if err := Started(cfg, "work", id, transcript(t, proj, id, mod), "", old); err != nil {
			t.Fatal(err)
		}
	}
	if err := Started(cfg, "work", "gone", filepath.Join(proj, "gone.jsonl"), "", old); err != nil {
		t.Fatal(err)
	}
	if err := Ended(cfg, "work", "closed", now); err != nil {
		t.Fatal(err)
	}
	st := Read(cfg, "work")
	var dead []string
	for _, l := range st.List(now) {
		if l.Kind == Dead && l.ID != "b" {
			dead = append(dead, l.ID)
		}
	}
	var un []string
	for _, se := range st.Unclosed("b", now) {
		un = append(un, se.ID)
	}
	if len(un) != 1 || un[0] != "a" || len(dead) != 1 || dead[0] != "a" {
		t.Errorf("unclosed %v, dead %v; want [a]", un, dead)
	}
}

func TestReportedAndDismissed(t *testing.T) {
	cfg, proj := t.TempDir(), t.TempDir()
	now := time.Now()
	old := now.Add(-2 * time.Hour)
	if err := Started(cfg, "work", "s", transcript(t, proj, "s", old), "", old); err != nil {
		t.Fatal(err)
	}
	if err := Reported(cfg, "work", "s", now); err != nil {
		t.Fatal(err)
	}
	if un := Read(cfg, "work").Unclosed("", now); len(un) != 1 || !un[0].Reported {
		t.Errorf("after Reported: %+v", un)
	}
	// Unknown or empty ids change nothing and are no error.
	for _, id := range []string{"", "missing"} {
		if err := Dismissed(cfg, "work", id, now); err != nil {
			t.Errorf("%q: %v", id, err)
		}
	}
	if err := Dismissed(cfg, "work", "s", now); err != nil {
		t.Fatal(err)
	}
	if un := Read(cfg, "work").Unclosed("", now); len(un) != 0 {
		t.Errorf("dismissed, still unclosed: %+v", un)
	}
}
