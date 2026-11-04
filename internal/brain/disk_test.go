package brain

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/format"
)

// A repository is on this computer when a work folder has it; a closed task's missing
// repository is not reported, and nothing is changed.
func TestMismatches(t *testing.T) {
	b := open(t, "en")
	work := b.Space.WorkFolders[0]
	home := filepath.Dir(work)
	for _, d := range []string{filepath.Join(work, "code", "app"), filepath.Join(home, "elsewhere")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	live, err := b.NewTask(NewTaskInput{Title: "Ship the app"})
	if err != nil {
		t.Fatal(err)
	}
	done, err := b.NewTask(NewTaskInput{Title: "Old work"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.Close(done.Header.ID, "done"); err != nil {
		t.Fatal(err)
	}
	b.Mem.Notes[live.Header.ID].Header.Repos = []string{"code/app", "code/gone", "~/elsewhere", "~/nowhere", " "}
	b.Mem.Notes[done.Header.ID].Header.Repos = []string{"code/gone"}

	want := []string{"code/gone", "~/nowhere"}
	if got := b.MissingRepos(b.Mem.Notes[live.Header.ID]); !reflect.DeepEqual(got, want) {
		t.Errorf("missing: %v, want %v", got, want)
	}
	got := b.Mismatches()
	if len(got) != 2 || got[0].Note != live.Header.ID || got[0].Repo != "code/gone" || got[1].Repo != "~/nowhere" {
		t.Errorf("mismatches: %+v (a closed task is left out)", got)
	}

	b.Space.WorkFolders = nil
	if got := b.MissingRepos(b.Mem.Notes[live.Header.ID]); !reflect.DeepEqual(got, []string{"~/nowhere"}) {
		t.Errorf("no work folders, so only paths written in full can be told: %v", got)
	}
}

// A space whose notes come from a newer program is not opened: writing could break them.
func TestOpenRefusesNewerNotes(t *testing.T) {
	t.Setenv("AB_LANGUAGE", "en")
	b := open(t, "en")
	if err := format.Write(b.Space.Memory, 99); err != nil {
		t.Fatal(err)
	}
	c, err := config.Load(b.ConfigDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Open(c, c.Spaces["work"]); err == nil || !strings.Contains(err.Error(), "newer augmented brain") {
		t.Errorf("Open: %v", err)
	}
}
