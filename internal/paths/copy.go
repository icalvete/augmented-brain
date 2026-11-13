package paths

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// CopyTree copies a folder, files and subfolders, into dst, leaving out the folders
// whose path relative to src skip names. Links and other special files are not copied:
// a backup never holds a link.
func CopyTree(src, dst string, skip ...string) error {
	return copyTree(src, dst, false, skip)
}

// CopyTreeExact is CopyTree for a copy that must be able to stand in for the folder,
// the one made before a migration: links are copied as links (a link that cannot be
// made is an error naming it, never left out in silence), any other special file is an
// error too, and every file and folder is synced, so the copy is on the disk before
// anything changes the original.
func CopyTreeExact(src, dst string, skip ...string) error {
	return copyTree(src, dst, true, skip)
}

func copyTree(src, dst string, exact bool, skip []string) error {
	var dirs []string
	err := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			for _, s := range skip {
				if rel == s {
					return filepath.SkipDir
				}
			}
			dirs = append(dirs, target)
			return os.MkdirAll(target, 0o700)
		}
		switch {
		case d.Type().IsRegular():
		case !exact:
			return nil
		case d.Type()&fs.ModeSymlink != 0:
			to, err := os.Readlink(p)
			if err == nil {
				err = os.Symlink(to, target)
			}
			if err != nil {
				return fmt.Errorf("copying the link %s: %w", p, err)
			}
			return nil
		default:
			return fmt.Errorf("%s is not a file, a folder or a link: it cannot be copied", p)
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if !exact {
			return os.WriteFile(target, data, 0o600)
		}
		return writeSynced(target, data)
	})
	if err != nil || !exact {
		return err
	}
	for i := len(dirs) - 1; i >= 0; i-- {
		if err := syncDir(dirs[i]); err != nil {
			return err
		}
	}
	return nil
}

// writeSynced writes a new file and waits for the disk.
func writeSynced(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	if err == nil {
		err = syncFile(f)
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return err
}
