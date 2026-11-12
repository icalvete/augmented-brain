//go:build !unix

package main

import "os"

// processAlive is whether a process with this id runs. Without a cheap check here, a
// process that can be opened counts as running; watch also stops waiting after --max.
func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	_ = p.Release()
	return true
}
