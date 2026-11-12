//go:build unix

package main

import (
	"errors"
	"syscall"
)

// processAlive is whether a process with this id runs (EPERM: it runs, as someone else).
func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}
