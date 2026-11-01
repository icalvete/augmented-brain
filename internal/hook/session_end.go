package hook

import (
	"io"
	"os"
	"os/exec"

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
	detach(cmd)
	return cmd.Start()
}

// SessionEnd marks the session as ended normally (so the next one does not offer to
// recover it) and backs up the space the session worked in, when it has a backup set
// up: the user never has to remember to. AB_NO_AUTO_BACKUP=1 turns the backup off.
//
// Claude Code gives SessionEnd hooks 1.5 seconds in all, whatever the plugin asks for:
// only quick work here (a small state file), and anything longer in a detached process.
func SessionEnd(in io.Reader, cfgDir string) error {
	i := read(in)
	c, s, err := load(cfgDir, i.Cwd)
	if err != nil || c == nil || s == nil {
		return nil
	}
	_ = state.Ended(cfgDir, s.Name, i.SessionID, Now())
	if os.Getenv("AB_NO_AUTO_BACKUP") == "1" || s.Backup == "" {
		return nil
	}
	return StartBackup(s.Name)
}
