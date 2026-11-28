package telegram

import (
	"errors"
	"path/filepath"
	"time"

	"github.com/icalvete/augmented-brain/internal/filelock"
	"github.com/icalvete/augmented-brain/internal/i18n"
)

// LockName is the lock file, in augmented brain's configuration folder, that every
// connection to Telegram holds.
//
// gotd keeps the session in the keychain and writes it back as a connection goes on
// (its keys, the server it settled on). Two processes connected at once (the panel,
// the MCP servers of two sessions, the timer that looks for replies) would each write
// their own copy over the other's. So one connection at a time: they are short.
const LockName = "telegram.lock"

// ErrBusy means another connection to Telegram held the lock all the time Lock waited.
var ErrBusy = i18n.Errorf("telegram.err.busy")

// Lock waits at most wait for the lock in dir and returns the function that releases it.
func Lock(dir string, wait time.Duration) (func(), error) {
	unlock, err := filelock.LockWithin(filepath.Join(dir, LockName), wait)
	if errors.Is(err, filelock.ErrBusy) {
		return nil, ErrBusy
	}
	return unlock, err
}
