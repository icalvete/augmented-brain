package hook

import (
	"os"
	"os/exec"

	"github.com/icalvete/augmented-brain/internal/detach"
)

// StartUpdateCheck asks for the latest version in a process of its own (`hook
// update-check`) and returns at once: the session start never waits for the network,
// and the news reaches the next session. Tests replace it.
var StartUpdateCheck = func() error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(self, "hook", "update-check")
	cmd.Stdin, cmd.Stdout, cmd.Stderr = nil, nil, nil
	detach.Cmd(cmd)
	_, err = detach.Start(cmd)
	return err
}
