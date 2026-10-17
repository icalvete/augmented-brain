//go:build windows

package filelock

import (
	"os"

	"golang.org/x/sys/windows"
)

// Windows locks byte ranges, not whole files: the first byte stands for the file. The
// call blocks until the lock is free, like flock on Unix.
func lock(f *os.File) error {
	return windows.LockFileEx(windows.Handle(f.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK, 0, 1, 0, new(windows.Overlapped))
}

func unlock(f *os.File) error {
	return windows.UnlockFileEx(windows.Handle(f.Fd()), 0, 1, 0, new(windows.Overlapped))
}
