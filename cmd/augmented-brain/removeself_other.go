//go:build !windows

package main

import "os"

// removeSelf deletes the running program: Unix lets a running program remove its own
// file.
func removeSelf(self string) error { return os.Remove(self) }
