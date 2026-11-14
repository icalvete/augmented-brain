package brain

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/memory"
	"github.com/icalvete/augmented-brain/internal/paths"
)

// Rules and knowledge are changed in place and retired to archive/, never deleted, and
// never with a field the header did not have: earlier programs read headers strictly
// and would take a note with an unknown key for a broken one.

// changeable are the note types UpdateNote and RetireNote work on.
var changeable = []string{"rule", "reference", "lesson"}

// UpdateNote replaces the title, summary and body of a rule, reference or lesson; an
// empty one keeps what it has (area is only for knowledge, and empty keeps it too).
// The version it replaces is kept first in archive/<folder>/<id>-<date>.md.
func (b *Brain) UpdateNote(id, typ, title, summary, area, body string) (*memory.Note, error) {
	old, err := b.changeableNote(id, typ)
	if err != nil {
		return nil, err
	}
	if area != "" {
		if typ == "rule" {
			return nil, i18n.Errorf("err.rule.area")
		}
		if _, err := b.get(area, "area"); err != nil {
			return nil, err
		}
	}
	cp := *old
	if t := strings.TrimSpace(title); t != "" {
		cp.Header.Title = t
	}
	if sm := strings.TrimSpace(summary); sm != "" {
		cp.Header.Summary = sm
	}
	if area != "" {
		cp.Header.Area = area
	}
	if strings.TrimSpace(body) != "" {
		if !strings.HasSuffix(body, "\n") {
			body += "\n"
		}
		cp.Body = body
	}
	// The old version is kept only for a change that will be saved.
	if err := b.refuse(&cp); err != nil {
		return nil, err
	}
	if err := b.archiveCopy(old, "-"+b.today()); err != nil {
		return nil, err
	}
	return &cp, b.save(&cp)
}

// RetireNote takes a rule, reference or lesson out of the memory without deleting it: it
// moves to archive/<folder>/, which is never loaded, so a rule stops reaching the session
// start; a line at its end says when, why, and what replaces it (replacedBy, optional,
// must exist).
func (b *Brain) RetireNote(id, reason, replacedBy string) (string, error) {
	old, err := b.changeableNote(id, "")
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(reason) == "" {
		return "", i18n.Errorf("err.retire.why")
	}
	if replacedBy != "" {
		if replacedBy == id {
			return "", i18n.Errorf("err.retire.self")
		}
		if _, ok := b.Mem.Notes[replacedBy]; !ok {
			return "", i18n.Errorf("err.retire.missing", replacedBy)
		}
	}
	line := i18n.T(b.Language, "note.retired", b.today(), strings.TrimRight(strings.TrimSpace(reason), "."))
	if replacedBy != "" {
		line += " " + i18n.T(b.Language, "note.replaced", replacedBy)
	}
	cp := *old
	cp.Body = strings.TrimRight(cp.Body, "\n") + "\n\n" + line + "\n"
	data, err := cp.Render()
	if err != nil {
		return "", err
	}
	dest, err := b.archivePath(old, "")
	if err != nil {
		return "", err
	}
	if err := paths.WriteAtomicNoLink(dest, data, 0o644); err != nil {
		return "", err
	}
	if err := os.Remove(filepath.Join(b.Mem.Dir, filepath.FromSlash(old.Path))); err != nil {
		return "", err
	}
	delete(b.Mem.Notes, id)
	if _, err := b.Mem.WriteIndex(b.Language); err != nil {
		return "", err
	}
	return dest, nil
}

func (b *Brain) changeableNote(id, typ string) (*memory.Note, error) {
	old, err := b.get(id, typ)
	if err != nil {
		return nil, err
	}
	for _, t := range changeable {
		if old.Header.Type == t {
			return old, nil
		}
	}
	return nil, i18n.Errorf("err.note.unchangeable", id, old.Header.Type)
}

// archiveCopy keeps the file of n as it is now in archive/.
func (b *Brain) archiveCopy(n *memory.Note, suffix string) error {
	data, err := os.ReadFile(filepath.Join(b.Mem.Dir, filepath.FromSlash(n.Path)))
	if err != nil {
		return err
	}
	dest, err := b.archivePath(n, suffix)
	if err != nil {
		return err
	}
	return paths.WriteAtomicNoLink(dest, data, 0o644)
}

// archivePath is a free path for n in archive/<folder>/: <id><suffix>.md, or with -2,
// -3… when that is taken.
func (b *Brain) archivePath(n *memory.Note, suffix string) (string, error) {
	dir := filepath.Join(b.Mem.Dir, "archive", filepath.FromSlash(filepath.ToSlash(filepath.Dir(n.Path))))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	base := n.Header.ID + suffix
	for i := 1; ; i++ {
		name := base
		if i > 1 {
			name = fmt.Sprintf("%s-%d", base, i)
		}
		p := filepath.Join(dir, name+".md")
		if _, err := os.Stat(p); errors.Is(err, os.ErrNotExist) {
			return p, nil
		}
	}
}
