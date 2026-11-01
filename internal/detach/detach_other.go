//go:build !unix && !windows

package detach

import "os/exec"

// Cmd does nothing where there are no Unix sessions.
func Cmd(*exec.Cmd) {}

// Start starts the command and returns it.
func Start(cmd *exec.Cmd) (*exec.Cmd, error) { return cmd, cmd.Start() }
