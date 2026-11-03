package hook

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/icalvete/augmented-brain/internal/backup"
	"github.com/icalvete/augmented-brain/internal/brain"
	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/filelock"
	"github.com/icalvete/augmented-brain/internal/format"
	"github.com/icalvete/augmented-brain/internal/session"
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
				"%s; `augmented-brain doctor` helps.\n",
				what, err, session.TellUserLine(c.Language))
		case r.Migrated():
			fmt.Fprintf(&out, "\naugmented brain was updated and moved %s from format %d to %d (%s). A copy from before is in %s. "+
				"%s.\n",
				what, r.From, r.To, strings.Join(r.Done, "; "), r.Copy, session.TellUserLine(c.Language))
		}
	}
	if unlock, err := filelock.LockWithin(filepath.Join(cfgDir, backup.NotesLock), LockWait); err == nil {
		r, err := format.EnsureConfig(cfgDir, c.FormatVersion(), func(v int) error { return config.SetFormat(cfgDir, v) }, now)
		if err == nil {
			// 0.6.0 kept the secrets' names in config.toml, which earlier programs
			// refuse: they go to their own file.
			err = brain.MoveSecrets(cfgDir)
		}
		unlock()
		say("its configuration and profile", r, err)
	}

	unlock, err := filelock.LockWithin(filepath.Join(cfgDir, backup.NotesLock), LockWait)
	if errors.Is(err, filelock.ErrBusy) {
		return out.String() // a backup or an import is writing: the next session does it
	}
	if err != nil {
		say("the notes of space "+s.Name, format.Result{}, err)
		return out.String()
	}
	defer unlock()
	r, err := format.EnsureNotes(s.Memory, cfgDir, s.Name, now)
	say("the notes of space "+s.Name, r, err)
	if _, err := os.Stat(c.SharedDir()); err == nil {
		r, err = format.EnsureNotes(c.SharedDir(), cfgDir, config.SharedFolder, now)
		say("the shared notes", r, err)
	}
	return out.String()
}

// LockWait is the most a session-start hook waits for the notes lock: hooks have ten
// seconds, and a backup or an import may hold it for longer.
var LockWait = 3 * time.Second
