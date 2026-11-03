package brain

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/config"
)

// Found in the rehearsal of 0.5.6: the tools wrote a key into a task's journal, and the
// backup then refused the whole space. A write with a secret in it is refused, and saves
// nothing; a note that already held one (an import) can still be changed.
func TestNoSecretsInNotes(t *testing.T) {
	b := open(t, "en")
	key := "AK" + "IA" + "QWERTYUIOPASDFGH" // built here, so this file holds no key-shaped text
	n, err := b.NewTask(NewTaskInput{Title: "Ship", Summary: "Ship it"})
	if err != nil {
		t.Fatal(err)
	}
	before := read(t, b, n.Path)
	if _, err := b.Journal(n.Header.ID, "the key is "+key); err == nil || !strings.Contains(err.Error(), "an AWS access key") {
		t.Errorf("journal with a key: %v", err)
	}
	if read(t, b, n.Path) != before {
		t.Error("the refused entry was written")
	}
	if _, err := b.NewTask(NewTaskInput{Title: "Other", Summary: key}); err == nil {
		t.Error("a new task with a key was saved")
	}
	if err := b.Capture("key "+key, ""); err == nil {
		t.Error("a capture with a key was saved")
	}
	if _, err := b.SaveProfile(ProfileInput{Name: "Ana", About: key}); err == nil {
		t.Error("a profile with a key was saved")
	}

	// A note that came with a key (an import) can still get a journal entry.
	path := filepath.Join(b.Mem.Dir, n.Path)
	if err := os.WriteFile(path, []byte(strings.Replace(before, "Ship it", "Ship it "+key, 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := config.Load(b.ConfigDir)
	if err != nil {
		t.Fatal(err)
	}
	b2, err := Open(c, b.Space)
	if err != nil {
		t.Fatal(err)
	}
	b2.Now = b.Now
	if _, err := b2.Journal(n.Header.ID, "took the key out of the summary next"); err != nil {
		t.Errorf("a note that already held a key cannot be changed: %v", err)
	}
}

// Found in the rehearsal of 0.5.6: a task's note stayed as it was opened. The goal and
// "done when" go into the Definition, and parking and closing keep the Essentials
// current: where it is and the next step.
func TestTaskNoteKeepsUp(t *testing.T) {
	for _, lang := range []string{"en", "es"} {
		b := open(t, lang)
		n, err := b.NewTask(NewTaskInput{Title: "Certs", Summary: "Renew the certificates", Goal: "no expired certificate", DoneWhen: "every check is green"})
		if err != nil {
			t.Fatal(err)
		}
		labels := map[string][2]string{"en": {"- Goal: no expired certificate", "- Done when: every check is green"},
			"es": {"- Objetivo: no expired certificate", "- Hecho cuando: every check is green"}}[lang]
		for _, want := range labels {
			if !strings.Contains(read(t, b, n.Path), want) {
				t.Errorf("%s: new task lacks %q:\n%s", lang, want, read(t, b, n.Path))
			}
		}
		if _, err := b.Park(ParkInput{ID: n.Header.ID, Next: "run the renewal on db01"}); err != nil {
			t.Fatal(err)
		}
		parked := read(t, b, n.Path)
		want := map[string][2]string{"en": {"- Where it is: parked on 2026-10-15.", "- Next: run the renewal on db01"},
			"es": {"- Dónde está: aparcada el 2026-10-15.", "- Siguiente: run the renewal on db01"}}[lang]
		if !strings.Contains(parked, want[0]) || !strings.Contains(parked, want[1]) {
			t.Errorf("%s: parked task:\n%s", lang, parked)
		}
		if _, err := b.Close(n.Header.ID, "all renewed"); err != nil {
			t.Fatal(err)
		}
		closed := read(t, b, n.Path)
		done := map[string]string{"en": "- Where it is: closed on 2026-10-15: all renewed", "es": "- Dónde está: cerrada el 2026-10-15: all renewed"}[lang]
		if !strings.Contains(closed, done) || strings.Contains(closed, want[1]+"\n") {
			t.Errorf("%s: closed task:\n%s", lang, closed)
		}
	}
}
