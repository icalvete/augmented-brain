//go:build unix || windows

package filelock

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// Lock makes the folder and the file, and while it is held LockWithin gives up with
// ErrBusy after waiting; once released, LockWithin takes it at once. The file stays.
func TestLockAndLockWithin(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a", "b", "notes.lock")
	release, err := Lock(path)
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	if _, err := LockWithin(path, 120*time.Millisecond); !errors.Is(err, ErrBusy) {
		t.Fatalf("a held lock: %v, want ErrBusy", err)
	}
	if waited := time.Since(start); waited < 100*time.Millisecond {
		t.Errorf("gave up after %v, before the wait was over", waited)
	}
	release()
	again, err := LockWithin(path, time.Second)
	if err != nil {
		t.Fatalf("a free lock: %v", err)
	}
	again()
	if _, err := os.Stat(path); err != nil {
		t.Errorf("the lock file is gone: %v", err)
	}
}

// LockWithin waits for a lock released while it waits.
func TestLockWithinWaits(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.lock")
	release, err := Lock(path)
	if err != nil {
		t.Fatal(err)
	}
	time.AfterFunc(100*time.Millisecond, release)
	got, err := LockWithin(path, 5*time.Second)
	if err != nil {
		t.Fatalf("released while waiting: %v", err)
	}
	got()
}

// A lock whose folder cannot be made is an error, not a lock that holds nothing.
func TestLockErrors(t *testing.T) {
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Lock(filepath.Join(file, "notes.lock")); err == nil {
		t.Error("Lock under a file worked")
	}
	if _, err := LockWithin(filepath.Join(file, "notes.lock"), time.Millisecond); err == nil {
		t.Error("LockWithin under a file worked")
	}
}

// A Writer lets one holder at a time, in this process, with the file or without one;
// LockProcess holds the same mutex.
func TestWriter(t *testing.T) {
	for _, path := range []string{"", filepath.Join(t.TempDir(), "notes.lock")} {
		w := NewWriter(path)
		var mu sync.Mutex
		inside, most := 0, 0
		var wg sync.WaitGroup
		for range 8 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				unlock, err := w.Lock()
				if err != nil {
					t.Error(err)
					return
				}
				mu.Lock()
				inside++
				most = max(most, inside)
				mu.Unlock()
				time.Sleep(5 * time.Millisecond)
				mu.Lock()
				inside--
				mu.Unlock()
				unlock()
			}()
		}
		wg.Wait()
		if most != 1 {
			t.Errorf("path %q: %d holders at once", path, most)
		}
		unlock := w.LockProcess()
		done := make(chan struct{})
		go func() {
			u, err := w.Lock()
			if err == nil {
				u()
			}
			close(done)
		}()
		select {
		case <-done:
			t.Errorf("path %q: Lock got in while LockProcess held the mutex", path)
		case <-time.After(50 * time.Millisecond):
		}
		unlock()
		<-done
	}
}

// A Writer whose lock file cannot be made gives the mutex back.
func TestWriterError(t *testing.T) {
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	w := NewWriter(filepath.Join(file, "notes.lock"))
	if _, err := w.Lock(); err == nil {
		t.Fatal("a lock under a file worked")
	}
	w.LockProcess()() // would block forever if the mutex was kept
}
