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
	// A temporary name nobody can guess, so nothing can be planted there first.
	f, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	_, err = f.Write(data)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Chmod(tmp, perm)
	}
	if err != nil {
		os.Remove(tmp)
		return err
	}
	if err := Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}
