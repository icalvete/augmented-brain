package telegram

import (
	"errors"
	"testing"
	"time"
)

func TestLockOneAtATime(t *testing.T) {
	dir := t.TempDir()
	unlock, err := Lock(dir, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Lock(dir, 20*time.Millisecond); !errors.Is(err, ErrBusy) {
		t.Fatalf("second lock while the first is held: %v, want ErrBusy", err)
	}
	unlock()
	again, err := Lock(dir, time.Second)
	if err != nil {
		t.Fatalf("lock after release: %v", err)
	}
	again()
}
