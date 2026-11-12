package brain

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/icalvete/augmented-brain/internal/memory"
)

// A task whose journal grows past the limit keeps its recent entries and moves the old
// ones, as they were, to archive/journals/; nothing is lost.
func TestJournalArchived(t *testing.T) {
	for _, lang := range []string{"en", "es"} {
		t.Run(lang, func(t *testing.T) {
			b := open(t, lang)
			task, err := b.NewTask(NewTaskInput{Title: "Long one", Summary: "grows"})
			if err != nil {
				t.Fatal(err)
			}
			id := task.Header.ID
			day := time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC)
			var all []string // every entry written, in order
			for i := 0; i < 120; i++ {
				b.Now = func() time.Time { return day.AddDate(0, 0, i/4) }
				entry := fmt.Sprintf("entry %d %s\n  - a detail line under it\n  - and another", i, strings.Repeat("x", 200))
				if _, err := b.Journal(id, entry); err != nil {
					t.Fatal(err)
				}
				all = append(all, fmt.Sprintf("entry %d ", i))
			}
			n := b.Mem.Notes[id]
			data, _ := n.Render()
			if len(data) > memory.JournalArchiveAt {
				t.Errorf("the note is %d bytes, over %d", len(data), memory.JournalArchiveAt)
			}
			files, _ := filepath.Glob(filepath.Join(b.Mem.Dir, "archive", "journals", id+"-*.md"))
			if len(files) == 0 {
				t.Fatal("no archive file was written")
			}
			// Put every archive back where its line is: the journal is whole again.
			whole := n.Body
			for range files {
				whole, _ = memory.ExpandJournal(whole, func(rel string) (string, bool) {
					d, err := os.ReadFile(filepath.Join(b.Mem.Dir, filepath.FromSlash(rel)))
					return string(d), err == nil
				})
			}
			last := -1
			for _, e := range all {
				i := strings.Index(whole, e)
				if i < 0 || i < last {
					t.Fatalf("%q is missing or out of order once the archives are put back", e)
				}
				last = i
			}
			if strings.Count(whole, "a detail line under it") != len(all) {
				t.Error("detail lines were lost")
			}
			// The link line is a dated line that names its file, in the user's language.
			found := false
			for _, l := range strings.Split(n.Body, "\n") {
				if rel, ok := memory.ArchiveLink(l); ok {
					found = true
					if _, err := os.Stat(filepath.Join(b.Mem.Dir, filepath.FromSlash(rel))); err != nil {
						t.Errorf("the line names %s, which is not there", rel)
					}
					if lang == "es" && !strings.Contains(l, "entradas antiguas") {
						t.Errorf("not in Spanish: %q", l)
					}
				}
			}
			if !found {
				t.Error("no line in the journal names the archive")
			}
			if !strings.Contains(n.Body, "entry 119 ") {
				t.Error("the last entry left the note")
			}
		})
	}
}

// A small task is left as it is, and so are other notes.
func TestJournalSmallStays(t *testing.T) {
	b := open(t, "en")
	task, err := b.NewTask(NewTaskInput{Title: "Short", Summary: "small"})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10; i++ {
		if _, err := b.Journal(task.Header.ID, "a line"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := os.Stat(filepath.Join(b.Mem.Dir, "archive", "journals")); err == nil {
		t.Error("an archive was made for a small task")
	}
	if got := b.Oversized(); len(got) != 0 {
		t.Errorf("Oversized = %v", got)
	}
}

// A task written past the limit by hand (or by an earlier program) is archived by
// ArchiveJournals, which says what it wrote.
func TestArchiveJournals(t *testing.T) {
	b := open(t, "en")
	task, err := b.NewTask(NewTaskInput{Title: "Hand made", Summary: "big"})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(b.Mem.Dir, filepath.FromSlash(task.Path))
	data, _ := os.ReadFile(path)
	var extra strings.Builder
	for i := 0; i < 200; i++ {
		fmt.Fprintf(&extra, "- 2026-11-%02d — by hand %d %s\n", 1+i/10, i, strings.Repeat("y", 150))
	}
	if err := os.WriteFile(path, append(data, extra.String()...), 0o600); err != nil {
		t.Fatal(err)
	}
	b = reopen(t, b)
	if got := b.Oversized(); len(got) != 1 || got[0] != task.Header.ID {
		t.Fatalf("Oversized = %v", got)
	}
	files, err := b.ArchiveJournals()
	if err != nil || len(files) != 1 {
		t.Fatalf("ArchiveJournals = %v, %v", files, err)
	}
	if got := b.Oversized(); len(got) != 0 {
		t.Errorf("still oversized: %v", got)
	}
	archived, _ := os.ReadFile(filepath.Join(b.Mem.Dir, filepath.FromSlash(files[0])))
	if !strings.Contains(string(archived), "by hand 0 ") || !strings.HasPrefix(string(archived), "# Hand made · journal from 2026-10-15 to 2026-11-") {
		t.Errorf("archive:\n%.300s", archived)
	}
}

// reopen loads the space again from disk, as the next command would.
func reopen(t *testing.T, b *Brain) *Brain {
	t.Helper()
	m, problems, err := memory.Load(b.Mem.Dir, nil)
	if err != nil || len(problems) > 0 {
		t.Fatal(err, problems)
	}
	b.Mem = m
	return b
}
