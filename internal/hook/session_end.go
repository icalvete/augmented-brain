package hook

import (
	"io"
	"os"
	"os/exec"

	"github.com/icalvete/augmented-brain/internal/detach"
	"github.com/icalvete/augmented-brain/internal/fresh"
	"github.com/icalvete/augmented-brain/internal/state"
)

// StartBackup starts the backup of a space in the background and returns at once, so
// closing the assistant never waits for git. Tests replace it.
var StartBackup = func(space string) error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(self, "backup", "--auto", "--space", space)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = nil, nil, nil
	detach.Cmd(cmd)
	_, err = detach.Start(cmd)
	return err
}

// SessionEnd marks the session as ended normally (so the next one does not offer to
// recover it) and backs up the space the session worked in, when it has a backup set
// up: the user never has to remember to. AB_NO_AUTO_BACKUP=1 turns the backup off.
//
// Claude Code gives SessionEnd hooks 1.5 seconds in all, whatever the plugin asks for:
// only quick work here (a small state file), and anything longer in a detached process.
func SessionEnd(in io.Reader, cfgDir string) error {
	i := read(in)
	if i.Reason != "clear" {
		// The process ends: its seen file is of no use. After /clear the same process
		// goes on, and SessionStart takes a new one.
		fresh.Remove(cfgDir, fresh.HookKey(Getenv))
	}
	c, s, err := load(cfgDir, i.Cwd)
	if err != nil || c == nil || s == nil || quiet(cfgDir, c, i.Cwd) {
		return nil
	}
	_ = state.Ended(cfgDir, s.Name, i.SessionID, Now())
	if os.Getenv("AB_NO_AUTO_BACKUP") == "1" {
		return nil
	}
	var err1 error
	if s.Backup != "" {
		_ = state.TakeOutside(cfgDir, s.Name, Now()) // its own backup covers it
		err1 = StartBackup(s.Name)
	}
	// Spaces a session of another space wrote in (a task's journal, its next step):
	// backed up now, not at the end of a session of their own, which may be days away.
	for _, name := range c.SpaceNames() {
		o := c.Spaces[name]
		if name == s.Name || o.Backup == "" || !state.TakeOutside(cfgDir, name, Now()) {
			continue
		}
		if err := StartBackup(name); err != nil && err1 == nil {
			err1 = err
		}
	}
	return err1
}
