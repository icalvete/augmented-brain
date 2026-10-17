//go:build windows

package hook

import (
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

// detach starts the command with no console and in its own process group, so it
// outlives the assistant that ends and no window flashes on screen.
func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: windows.DETACHED_PROCESS | windows.CREATE_NEW_PROCESS_GROUP,
		HideWindow:    true,
	}
}
