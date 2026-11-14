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
	"github.com/icalvete/augmented-brain/internal/paths"
	"github.com/icalvete/augmented-brain/internal/session"
)

// migrate brings the configuration folder and the space's notes to the format this
// program writes, and says what the session text should add: nothing when there was
// nothing to do, which is nearly always (it reads two small files). The caller holds
// the notes lock (startWork).
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
	r, err := format.EnsureConfig(cfgDir, c.FormatVersion(), func(v int) error { return config.SetFormat(cfgDir, v) }, c.Language, now)
	if err == nil {
		// 0.6.0 kept the secrets' names in config.toml, which earlier programs
		// refuse: they go to their own file.
		err = brain.MoveSecrets(cfgDir)
	}
	say("its configuration and profile", r, err)

	// What a write cut short by a crash or a power cut left behind; under the lock, so
	// no write of this machine is going on.
	paths.CleanTemps(s.Memory, TempAge, ".git", "docs")
	paths.CleanTemps(cfgDir, TempAge, ".git", format.CopiesDir, "remote")
	r, err = format.EnsureNotes(s.Memory, cfgDir, s.Name, c.Language, now)
	say("the notes of space "+s.Name, r, err)
	if _, err := os.Stat(c.SharedDir()); err == nil {
		r, err = format.EnsureNotes(c.SharedDir(), cfgDir, config.SharedFolder, c.Language, now)
		say("the shared notes", r, err)
	}
	return out.String()
}

// startWork does, with the notes lock taken once, the work part 1 of the session start
// does behind the notes: the migration and the baseline of what this session has seen
// (freshBaseline). It waits StartWait at most, for all of it: when a backup, an import
// or a tool holds the notes, the session starts without that work, and the next one
// does it (N-09). It returns what the session text should add.
func startWork(c *config.Config, s *config.Space, i input, cfgDir string) string {
	unlock, err := filelock.LockWithin(filepath.Join(cfgDir, backup.NotesLock), StartWait)
	if errors.Is(err, filelock.ErrBusy) {
		return ""
	}
	if err != nil {
		return fmt.Sprintf("\naugmented brain could not bring the notes of space %s up to date: %v. Until it can, its tools may refuse to work there. "+
			"%s; `augmented-brain doctor` helps.\n", s.Name, err, session.TellUserLine(c.Language))
	}
	defer unlock()
	note := migrate(c, s, cfgDir)
	freshBaseline(s, i, cfgDir)
	return note
}

// TempAge is how old a temporary file left by a write must be before the session
// start removes it: well past any write still going on (another machine's, through a
// synced folder).
const TempAge = time.Hour

// StartWait is the most part 1 of the session start waits for the notes lock, once,
// for all the work it does under it (startWork): hooks have ten seconds, the rest of
// part 1 needs some of them, and a backup or an import may hold the lock for longer.
var StartWait = 2 * time.Second

// PartWait is the most parts 2 and on wait for part 1, which may be moving the notes
// to a new format.
var PartWait = time.Second
