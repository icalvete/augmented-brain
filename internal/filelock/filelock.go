// Package filelock serializes writers across processes: two assistant sessions on the
// same machine each run their own MCP server, and both may write the same memory.
package filelock

import (
	"os"
	"path/filepath"
)

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
