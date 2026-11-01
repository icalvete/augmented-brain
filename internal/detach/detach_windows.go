//go:build windows

package detach

import (
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

const flags = windows.DETACHED_PROCESS | windows.CREATE_NEW_PROCESS_GROUP

// Cmd starts the command with no console and in its own process group, so it
// outlives the assistant that ends and no window flashes on screen. It also asks to
// leave the job object of its parent (CREATE_BREAKAWAY_FROM_JOB): a program that runs
// the assistant inside a job could otherwise end the command along with the job.
func Cmd(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: flags | windows.CREATE_BREAKAWAY_FROM_JOB,
		HideWindow:    true,
	}
}

// Start starts a command prepared with Cmd and returns the one that runs. A job that
// does not allow leaving it refuses the start: then it starts again without asking to
// leave, in a fresh copy of the command (an exec.Cmd is not started twice).
func Start(cmd *exec.Cmd) (*exec.Cmd, error) {
	err := cmd.Start()
	if err == nil || cmd.SysProcAttr == nil || cmd.SysProcAttr.CreationFlags&windows.CREATE_BREAKAWAY_FROM_JOB == 0 {
		return cmd, err
	}
	again := exec.Command(cmd.Path, cmd.Args[1:]...)
	again.Args = cmd.Args
	again.Dir, again.Env = cmd.Dir, cmd.Env
	again.Stdin, again.Stdout, again.Stderr = cmd.Stdin, cmd.Stdout, cmd.Stderr
	again.SysProcAttr = &syscall.SysProcAttr{CreationFlags: flags, HideWindow: true}
	return again, again.Start()
}
