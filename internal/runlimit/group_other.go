//go:build !unix

package runlimit

import "os/exec"

// group leaves the default: the end of the context stops the program, and WaitDelay
// stops the wait for whatever it started.
func group(*exec.Cmd) {}
