//go:build unix

package detach

import (
	"os/exec"
	"syscall"
)

// Cmd puts the command in its own session, so it outlives the process that started it.
func Cmd(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}

// Start starts a command prepared with Cmd and returns the one that runs: here, the
// same one.
func Start(cmd *exec.Cmd) (*exec.Cmd, error) { return cmd, cmd.Start() }
