package brain

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/icalvete/augmented-brain/internal/memory"
	"github.com/icalvete/augmented-brain/internal/paths"
)

// archiveJournal moves the oldest entries of a task's journal to a file of their own in
// archive/journals/ when the note has grown past memory.JournalArchiveAt, and leaves in
// their place one dated line naming that file. The file is written before the note, so
// a failure in between leaves a copy too many, never one too few. It changes n's body
// and reports the file it wrote ("" when nothing moved).
func (b *Brain) archiveJournal(n *memory.Note) (string, error) {
	if n.Header.Type != "task" {
		return "", nil
	}
	data, err := n.Render()
	if err != nil || len(data) <= memory.JournalArchiveAt {
		return "", err
	}
	s, ok := memory.SplitJournal(n.Body, memory.JournalKeep)
	if !ok {
		return "", nil
	}
	rel, err := b.journalFile(n.Header.ID, s.First, s.Last)
	if err != nil {
		return "", err
	}
	var file strings.Builder
	fmt.Fprintf(&file, "# %s · %s\n\n", n.Header.Title, fmt.Sprintf(b.text("journal.archive.title"), s.First, s.Last))
	fmt.Fprintf(&file, "%s\n\n", fmt.Sprintf(b.text("journal.archive.from"), n.Path, b.today(), memory.JournalArchiveAt/1024))
	file.WriteString(s.Moved)
	if err := paths.WriteAtomicNoLink(filepath.Join(b.Mem.Dir, filepath.FromSlash(rel)), []byte(file.String()), 0o600); err != nil {
		return "", err
	}
	// Dated as the last entry it stands for, so the journal stays in date order.
	link := memory.DatedLine(s.Last, fmt.Sprintf(b.text("journal.archived"), s.Entries, s.First, s.Last, b.today(), rel))
	n.Body = s.Before + link + s.After
	b.archived = append(b.archived, rel)
	return rel, nil
}

// journalFile is a free name for the entries of task id from first to last, from the
// memory folder: archive/journals/<id>-<first>-<last>.md, or with -2, -3… when taken.
// The dates in the name keep two machines that archive different entries apart.
func (b *Brain) journalFile(id, first, last string) (string, error) {
	dir := filepath.Join(b.Mem.Dir, filepath.FromSlash(memory.JournalArchiveDir))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	base := fmt.Sprintf("%s-%s-%s", id, first, last)
	for i := 1; ; i++ {
		name := base
		if i > 1 {
			name = fmt.Sprintf("%s-%d", base, i)
		}
		if _, err := os.Stat(filepath.Join(dir, name+".md")); errors.Is(err, os.ErrNotExist) {
			return memory.JournalArchiveDir + "/" + name + ".md", nil
		}
	}
}

// Oversized returns the tasks of the space whose journal ArchiveJournals would move.
func (b *Brain) Oversized() []string { return b.Mem.Oversized() }

// ArchiveJournals moves the old journal entries of every task that has grown past the
// limit (written by hand, or by an earlier program), and returns the files written.
func (b *Brain) ArchiveJournals() ([]string, error) {
	b.archived = nil
	for _, id := range b.Oversized() {
		cp := *b.Mem.Notes[id]
		if err := b.save(&cp); err != nil {
			return b.archived, err
		}
	}
	return b.archived, nil
}
