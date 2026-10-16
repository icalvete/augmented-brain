package hook

import (
	"io"
	"os"
	"os/exec"
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

// SessionEnd backs up the space the session worked in, when it has a backup set up:
// the user never has to remember to. AB_NO_AUTO_BACKUP=1 turns it off.
func SessionEnd(in io.Reader, cfgDir string) error {
	i := read(in)
	if os.Getenv("AB_NO_AUTO_BACKUP") == "1" {
		return nil
	}
	c, s, err := load(cfgDir, i.Cwd)
	if err != nil || c == nil || s == nil || s.Backup == "" {
		return nil
	}
	return StartBackup(s.Name)
}
