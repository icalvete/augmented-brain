package hook

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/icalvete/augmented-brain/internal/backup"
	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/filelock"
	"github.com/icalvete/augmented-brain/internal/format"
)

// migrate brings the configuration folder and the space's notes to the format this
// program writes, and says what the session text should add: nothing when there was
// nothing to do, which is nearly always (it reads two small files).
func migrate(c *config.Config, s *config.Space, cfgDir string) string {
	now := Now()
	var out strings.Builder
	say := func(what string, r format.Result, err error) {
		switch {
		case err != nil:
			fmt.Fprintf(&out, "\naugmented brain could not bring %s up to date: %v. Until it can, its tools may refuse to work there. "+
				"Tell the user in one short line of your first answer, in the language they write in (%s if you cannot tell); `augmented-brain doctor` helps.\n",
				what, err, language(c.Language))
		case r.Migrated():
			fmt.Fprintf(&out, "\naugmented brain was updated and moved %s from format %d to %d (%s). A copy from before is in %s. "+
				"Tell the user in one short line of your first answer, in the language they write in (%s if you cannot tell).\n",
				what, r.From, r.To, strings.Join(r.Done, "; "), r.Copy, language(c.Language))
		}
	}
	r, err := format.EnsureConfig(cfgDir, c.FormatVersion(), func(v int) error { return config.SetFormat(cfgDir, v) }, now)
	say("its configuration and profile", r, err)

	unlock, err := filelock.Lock(filepath.Join(cfgDir, backup.NotesLock))
	if err != nil {
		say("the notes of space "+s.Name, format.Result{}, err)
		return out.String()
	}
	defer unlock()
	r, err = format.EnsureNotes(s.Memory, cfgDir, s.Name, now)
	say("the notes of space "+s.Name, r, err)
	return out.String()
}
