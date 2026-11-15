//go:build unix

package runlimit

import (
	"os/exec"
	"syscall"
)

// group puts the program in a process group of its own and makes the end of the
// context stop the whole group, not only the program.
func group(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
	cmd.Cancel = func() error {
		// The group has the program's pid; a negative pid names the group.
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		if err != nil {
			return cmd.Process.Kill()
		}
		return nil
	}
}
