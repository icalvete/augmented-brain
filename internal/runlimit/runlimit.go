// Package runlimit runs the programs augmented brain waits for (claude, gh, git, ps,
// systemctl…) with a limit that holds. exec.CommandContext alone stops only the program
// it started: a helper that program started (ssh, a credential helper, a child of
// `sh -c`) keeps its output open, and Wait goes on waiting for it, well past the limit.
//
// Command gives the program a process group of its own on Unix and stops the whole
// group when the context ends; on every system it stops waiting for the output
// WaitDelay after that. So a call returns within its limit plus WaitDelay, whatever the
// program left running.
//
// Not for programs meant to outlive the call (internal/detach) nor for interactive
// ones a person answers (gh auth login, the installer of an update).
package runlimit

import (
	"context"
	"os/exec"
	"time"
)

// PsLimit is how long a call to ps may take: it reads the process table, nothing more.
const PsLimit = 5 * time.Second

// WaitDelay is how long a call waits for the output of a program it stopped.
const WaitDelay = 5 * time.Second

// Command is exec.CommandContext with a limit that holds: when ctx ends the program and
// everything it started are stopped, and Wait returns WaitDelay later at most.
func Command(ctx context.Context, name string, arg ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, arg...)
	cmd.WaitDelay = WaitDelay
	group(cmd)
	return cmd
}

// Output runs a command with a limit of its own and returns its standard output, as
// exec.Cmd.Output does.
func Output(limit time.Duration, name string, arg ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), limit)
	defer cancel()
	return Command(ctx, name, arg...).Output()
}
