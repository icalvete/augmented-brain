//go:build unix

package hook

import (
	"os/exec"
	"syscall"
)

// detach puts the command in its own session, so it outlives the assistant that ends.
func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
