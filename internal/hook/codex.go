package hook

import (
	"errors"
	"io"
	"path/filepath"
	"strings"

	"github.com/icalvete/augmented-brain/internal/backup"
	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/filelock"
	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/mcpserver"
	"github.com/icalvete/augmented-brain/internal/session"
)

// CodexStartPart is SessionStartPart for Codex (`AB_CODEX_START_HOOK=1`): part n of
// the same profile, rules and index, measured in bytes as Codex cuts them, for the
// space of the folder alone (work or notes; no default space, as the tasks-only MCP
// server). Part 1 adds a note on what differs from Claude Code (codexNote) and what
// the user must hear about the notes: backup, update, migration, unknown settings.
// What only Claude Code has stays out: its sessions, transcripts and model, Remote
// Control, and the guard's notices and seal, which belong to the sessions it guards.
func CodexStartPart(in io.Reader, out io.Writer, cfgDir string, n int) error {
	i := read(in)
	empty := output{HookSpecificOutput: specific{HookEventName: "SessionStart"}}
	c, err := config.Load(cfgDir)
	var s *config.Space
	if err == nil {
		s = c.SpaceOf(i.Cwd)
	}
	if n > 1 && s == nil {
		return write(out, empty)
	}
	var text string
	switch {
	case errors.Is(err, config.ErrNotFound):
		text = "augmented brain is installed but not set up yet: no memory is loaded. The user can run `augmented-brain init`."
	case err != nil:
		text = "augmented brain cannot read its configuration, so no memory is loaded. Tell the user: " + err.Error() + ". `augmented-brain doctor` shows the problem."
	case s == nil:
		text = i18n.T(c.Language, "codex.nospace", i.Cwd, strings.Join(c.SpaceNames(), ", "))
	case n > 1:
		var ok bool
		if text, ok = waitParts(cfgDir, i, s.Name, n, Now(), PartWait); !ok {
			text = session.CodexPart(c, s, codexNote(c.Language), "", n)
		}
	default:
		note := codexNote(c.Language) + backupNote(c, s, cfgDir) + updateNote(c, cfgDir) +
			codexMigrate(c, s, cfgDir) + unknownSettingsNote(c, cfgDir)
		all := session.CodexParts(c, s, note, "")
		if n == 0 {
			text = strings.Join(all, "\n")
		} else if saveParts(cfgDir, i, s.Name, all) {
			text = all[0]
		} else {
			text = session.CodexPart(c, s, note, "", 1)
		}
	}
	empty.HookSpecificOutput.AdditionalContext = text
	return write(out, empty)
}

// codexNote is what part 1 tells the assistant in Codex about what differs from the
// rules, written for Claude Code: the guard, the tools it has, and how to reach the user.
func codexNote(lang string) string {
	return i18n.T(lang, "codex.note", i18n.T(lang, "codex.guard"), strings.Join(mcpserver.TasksOnlyTools, ", "))
}

// codexMigrate brings the notes to the current format, as part 1 of Claude Code's
// start does (startWork), without the baseline of tasks seen: the tasks-only server
// does not tell a session which tasks changed outside it.
func codexMigrate(c *config.Config, s *config.Space, cfgDir string) string {
	unlock, err := filelock.LockWithin(filepath.Join(cfgDir, backup.NotesLock), StartWait)
	if err != nil {
		return "" // busy or failing: the next session does it, and the tools say what is wrong
	}
	defer unlock()
	return migrate(c, s, cfgDir)
}
