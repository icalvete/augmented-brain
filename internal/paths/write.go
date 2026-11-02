package paths

import (
	"os"
	"path/filepath"
)

// WriteAtomic replaces a file in one step (write a temporary file next to it, then
// rename), so a crash never leaves half a file. When the path is a symlink, the file it
// points to is replaced and the link is kept.
func WriteAtomic(path string, data []byte, perm os.FileMode) error {
	if real, err := filepath.EvalSymlinks(path); err == nil {
		path = real
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, perm); err != nil {
		return err
	}
	if err := Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}
