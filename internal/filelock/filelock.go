// Package filelock serializes writers across processes: two assistant sessions on the
// same machine each run their own MCP server, and both may write the same memory.
package filelock

import (
	"errors"
	"os"
	"path/filepath"
	"time"
)

// maxPause is the longest LockWithin sleeps between two tries.
const maxPause = 50 * time.Millisecond

// ErrBusy is what LockWithin returns when the lock stayed held all the time it waited.
var ErrBusy = errors.New("the lock is held by another process")

// Lock blocks until the lock file at path is held, and returns the function that
// releases it. The file is created if needed and never removed.
func Lock(path string) (func(), error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := lock(f); err != nil {
		f.Close()
		return nil, err
	}
	return func() {
		unlock(f)
		f.Close()
	}, nil
}

// LockWithin is Lock, waiting at most wait: a hook has a few seconds, not forever (a
// backup or an import may hold the lock for longer). It returns ErrBusy when the lock
// stayed held. It tries again after 1 ms, then waits twice as long each time up to
// 50 ms: the session start's parts hold the lock for a few milliseconds, and a fixed
// 50 ms wait made them queue for far longer than the work itself.
func LockWithin(path string, wait time.Duration) (func(), error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	deadline := time.Now().Add(wait)
	pause := time.Millisecond
	for {
		ok, err := tryLock(f)
		if err != nil {
			f.Close()
			return nil, err
		}
		if ok {
			return func() {
				unlock(f)
				f.Close()
			}, nil
		}
		left := time.Until(deadline)
		if left <= 0 {
			f.Close()
			return nil, ErrBusy
		}
		time.Sleep(min(pause, left))
		pause = min(2*pause, maxPause)
	}
}
