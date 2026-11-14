package filelock

import (
	"sync"
	"time"
)

// Writer lets one writer of notes at a time: one at a time inside this process (a
// mutex), and one at a time among every augmented brain process of the user (the lock
// file at path). The MCP server and the panel both take it before they read or write a
// space. An empty path means the mutex only.
type Writer struct {
	mu   sync.Mutex
	path string
}

// NewWriter returns a Writer on the lock file at path.
func NewWriter(path string) *Writer { return &Writer{path: path} }

// Lock blocks until both are held, and returns the function that releases them.
func (w *Writer) Lock() (func(), error) {
	w.mu.Lock()
	if w.path == "" {
		return w.mu.Unlock, nil
	}
	unlock, err := Lock(w.path)
	if err != nil {
		w.mu.Unlock()
		return nil, err
	}
	return func() { unlock(); w.mu.Unlock() }, nil
}

// LockWithin is Lock, waiting at most wait for both together: a tool answers that the
// notes are busy rather than hang while a backup, an import or another session holds
// them. It returns ErrBusy when they stayed held.
func (w *Writer) LockWithin(wait time.Duration) (func(), error) {
	deadline := time.Now().Add(wait)
	for !w.mu.TryLock() {
		if time.Now().After(deadline) {
			return nil, ErrBusy
		}
		time.Sleep(10 * time.Millisecond)
	}
	if w.path == "" {
		return w.mu.Unlock, nil
	}
	unlock, err := LockWithin(w.path, max(time.Until(deadline), 0))
	if err != nil {
		w.mu.Unlock()
		return nil, err
	}
	return func() { unlock(); w.mu.Unlock() }, nil
}

// LockProcess takes the mutex only, for work that takes the file lock itself later
// (a backup takes the backup lock first and the notes lock after it, never before).
func (w *Writer) LockProcess() func() {
	w.mu.Lock()
	return w.mu.Unlock
}
