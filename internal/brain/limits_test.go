package brain

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Each field one character over its limit is refused, saying where the long text goes,
// and nothing is written.
func TestFieldLimits(t *testing.T) {
	b := open(t, "en")
	task, err := b.NewTask(NewTaskInput{Title: "Short"})
	if err != nil {
		t.Fatal(err)
	}
	long := func(n int) string { return strings.Repeat("é", n+1) } // characters, not bytes
	for _, c := range []struct {
		name string
		do   func() error
	}{
		{"title", func() error { _, err := b.NewTask(NewTaskInput{Title: long(maxTitle)}); return err }},
		{"summary", func() error {
			_, err := b.Update(UpdateInput{ID: task.Header.ID, Summary: long(maxSummary)})
			return err
		}},
		{"next step", func() error { _, err := b.Update(UpdateInput{ID: task.Header.ID, Next: long(maxNext)}); return err }},
		{"park's next step", func() error { _, err := b.Park(ParkInput{ID: task.Header.ID, Next: long(maxNext)}); return err }},
		{"waiting_on", func() error {
			_, err := b.Update(UpdateInput{ID: task.Header.ID, WaitingOn: long(maxWaitingOn)})
			return err
		}},
		{"watch date's what", func() error {
			_, err := b.Update(UpdateInput{ID: task.Header.ID, WatchDate: "2026-12-01", WatchWhat: long(maxWatchWhat)})
			return err
		}},
		{"journal entry", func() error { _, err := b.Journal(task.Header.ID, long(maxJournal)); return err }},
		{"capture", func() error { return b.Capture(long(maxCapture), "") }},
		{"area title", func() error { _, err := b.NewArea(long(maxTitle), ""); return err }},
		{"rule", func() error { _, err := b.NewRule("Rule", "", long(maxBody)); return err }},
		{"person", func() error { _, err := b.SavePerson(PersonInput{Name: "Ana", Role: long(maxSummary)}); return err }},
		{"profile name", func() error { _, err := b.SaveProfile(ProfileInput{Name: long(maxName)}); return err }},
		{"profile", func() error { _, err := b.SaveProfile(ProfileInput{Name: "Ana", About: long(maxProfile)}); return err }},
	} {
		before := snapshot(t, b)
		err := c.do()
		if err == nil || !strings.Contains(err.Error(), "nothing was saved") || !strings.Contains(err.Error(), "characters long") {
			t.Errorf("%s: %v", c.name, err)
		}
		if after := snapshot(t, b); after != before {
			t.Errorf("%s: something was written", c.name)
		}
	}
	// At the limit, it fits.
	if _, err := b.Update(UpdateInput{ID: task.Header.ID, Next: strings.Repeat("é", maxNext)}); err != nil {
		t.Errorf("a next step at the limit: %v", err)
	}
}

// A field already too long, written by hand, does not stop other changes to the note.
func TestFieldLimitOnlyWhenItChanges(t *testing.T) {
	b := open(t, "en")
	task, err := b.NewTask(NewTaskInput{Title: "Short"})
	if err != nil {
		t.Fatal(err)
	}
	cp := *task
	cp.Header.Next = strings.Repeat("x", maxNext+10)
	b.Mem.Notes[cp.Header.ID] = &cp // as if it had been loaded so
	if _, err := b.Journal(task.Header.ID, "something done"); err != nil {
		t.Errorf("a journal entry on a task with a long next step: %v", err)
	}
}

// snapshot is every file of the space and the profile, to tell whether a call wrote.
func snapshot(t *testing.T, b *Brain) string {
	t.Helper()
	var out strings.Builder
	for _, dir := range []string{b.Mem.Dir, b.ConfigDir} {
		_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || strings.Contains(p, string(filepath.Separator)+"history"+string(filepath.Separator)) {
				return nil
			}
			data, _ := os.ReadFile(p)
			out.WriteString(p + "\n" + string(data) + "\n")
			return nil
		})
	}
	return out.String()
}

// A capture or a journal entry that ends the way the program marks text sent from
// another space, or a reminder, keeps its words but is not read as the mark.
func TestOwnWordsAreNotMarks(t *testing.T) {
	b := open(t, "en")
	b.KnowSpaces([]string{"work", "personal"})
	if err := b.Capture("buy milk (from personal)", ""); err != nil {
		t.Fatal(err)
	}
	if err := b.Capture("call (from personal) (remind: 2026-10-16)", ""); err != nil {
		t.Fatal(err)
	}
	if got := b.Arrived(); len(got) != 0 {
		t.Errorf("a capture typed here passes for one sent from another space: %+v", got)
	}
	in, err := b.Inbox()
	if err != nil {
		t.Fatal(err)
	}
	if len(in) != 2 || in[0].Text != "buy milk [from personal]" || in[1].Text != "call (from personal) [remind: 2026-10-16]" || in[1].Remind != "" {
		t.Errorf("inbox: %+v", in)
	}
	// "(from Amazon)" is not a space: it stays as it is.
	if err := b.Capture("cable (from Amazon)", ""); err != nil {
		t.Fatal(err)
	}
	if in, _ := b.Inbox(); in[2].Text != "cable (from Amazon)" {
		t.Errorf("inbox: %+v", in[2])
	}
	task, err := b.NewTask(NewTaskInput{Title: "T"})
	if err != nil {
		t.Fatal(err)
	}
	n, err := b.Journal(task.Header.ID, "decided there (from personal)")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(n.Body, "decided there [from personal]\n") {
		t.Errorf("journal:\n%s", n.Body)
	}
}

// A profile with a heading written twice (by hand, or by an earlier program that took
// a name of several lines) loads, and what it loads can be saved again.
func TestProfileWithRepeatedHeadings(t *testing.T) {
	b := open(t, "en")
	profile := "---\nid: profile\ntype: profile\ntitle: Profile\nsummary: s\ncreated: \"2026-10-01\"\n---\n" +
		"## About me\n- Name: Ana\n\n## How I like to work\n- never ask before pushing\n- dev\n\n" +
		"## How I like to work\n- short\n\n## Assistant\n- Name: Argos\nCalm.\n\n## Assistant\nDry humour.\n"
	if err := os.WriteFile(filepath.Join(b.ConfigDir, ProfileFile), []byte(profile), 0o600); err != nil {
		t.Fatal(err)
	}
	p, err := LoadProfile(b.ConfigDir)
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "Ana" || p.Assistant != "Argos" || !strings.Contains(p.WorkStyle, "- short") || !strings.Contains(p.WorkStyle, "- dev") ||
		!strings.Contains(p.Personality, "Dry humour.") {
		t.Errorf("loaded: %+v", p)
	}
	if _, err := b.SaveProfile(p); err != nil {
		t.Fatalf("saving what was loaded: %v", err)
	}
	again, err := LoadProfile(b.ConfigDir)
	if err != nil {
		t.Fatal(err)
	}
	if again != p {
		t.Errorf("not the same after a round trip:\n%+v\n%+v", p, again)
	}
}
