//go:build !unix && !windows

package hook

import "os/exec"

// detach does nothing where there are no Unix sessions.
func detach(*exec.Cmd) {}
