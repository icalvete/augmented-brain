package paths

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
)

// Private makes everything under dir readable by its owner only: folders 0700, files
// 0600 (links are left alone). Notes may hold personal data, and a copy made with a
// loose umask, such as a clone, is not. On Windows, where these modes mean little, it
// does nothing.
func Private(dir string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	return filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		switch {
		case d.Type()&fs.ModeSymlink != 0:
			return nil
		case d.IsDir():
			return os.Chmod(p, 0o700)
		default:
			return os.Chmod(p, 0o600)
		}
	})
}

// Exposed reports whether others than its owner may read dir (always false on
// Windows).
func Exposed(dir string) bool {
	if runtime.GOOS == "windows" {
		return false
	}
	fi, err := os.Stat(dir)
	return err == nil && fi.Mode().Perm()&0o077 != 0
}
