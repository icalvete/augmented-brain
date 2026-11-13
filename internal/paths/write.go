package paths

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// syncFile and syncDir make what was written reach the disk; tests count them.
var (
	syncFile = (*os.File).Sync
	syncDir  = func(dir string) error {
		if runtime.GOOS == "windows" {
			return nil // a folder cannot be opened to sync there, and NTFS does not need it
		}
		d, err := os.Open(dir)
		if err != nil {
			return err
		}
		err = d.Sync()
		if cerr := d.Close(); err == nil {
			err = cerr
		}
		return err
	}
)

// ErrLink is what WriteAtomicNoLink returns for a path that is a symbolic link.
var ErrLink = errors.New("a symbolic link, which augmented brain never writes through; replace it with a plain file")

// WriteAtomic replaces a file in one step (write a temporary file next to it, sync it,
// rename it over the file, then sync the folder), so neither a crash nor a power cut
// leaves half a file or an empty one. When the path is a symlink, the file it points
// to is replaced and the link is kept: that is for the configuration folder, whose
// files some people keep in a repository of their own. Notes use WriteAtomicNoLink.
func WriteAtomic(path string, data []byte, perm os.FileMode) error {
	if real, err := filepath.EvalSymlinks(path); err == nil {
		path = real
	}
	return writeAtomic(path, data, perm, true)
}

// WriteAtomicNoLink is WriteAtomic for a file inside a folder of notes: a path that
// is a symbolic link is refused (ErrLink), never written through, so a link planted
// in the notes cannot have the program write somewhere else.
func WriteAtomicNoLink(path string, data []byte, perm os.FileMode) error {
	if fi, err := os.Lstat(path); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%s is %w", path, ErrLink)
	}
	return writeAtomic(path, data, perm, true)
}

// WriteAtomicQuick replaces a file in one step but does not wait for the disk: for
// the machine's state, written at every message, which can be rebuilt if a power cut
// takes it. A crash of the program still never leaves half a file.
func WriteAtomicQuick(path string, data []byte, perm os.FileMode) error {
	if real, err := filepath.EvalSymlinks(path); err == nil {
		path = real
	}
	return writeAtomic(path, data, perm, false)
}

func writeAtomic(path string, data []byte, perm os.FileMode, durable bool) error {
	// A temporary name nobody can guess, so nothing can be planted there first.
	f, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*"+tmpSuffix)
	if err != nil {
		return err
	}
	tmp := f.Name()
	_, err = f.Write(data)
	if err == nil && durable {
		err = syncFile(f)
	}
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
	if durable {
		return syncDir(filepath.Dir(path))
	}
	return nil
}

const tmpSuffix = ".tmp"

// isTemp reports whether name is a temporary file WriteAtomic makes: <name>.<digits>.tmp.
func isTemp(name string) bool {
	base, ok := strings.CutSuffix(name, tmpSuffix)
	if !ok {
		return false
	}
	i := strings.LastIndexByte(base, '.')
	if i <= 0 || i == len(base)-1 {
		return false
	}
	for _, r := range base[i+1:] {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// CleanTemps removes the temporary files WriteAtomic left in dir and its subfolders
// when a write was cut short (a crash, a power cut), once they are older than age, so
// a write still going on is never touched. Folders named in skip (relative to dir)
// and links are left alone. It returns how many it removed; what it cannot read or
// remove it leaves, since this is only tidying.
func CleanTemps(dir string, age time.Duration, skip ...string) int {
	n := 0
	cutoff := time.Now().Add(-age)
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if d != nil && d.IsDir() && p != dir {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			if rel, rerr := filepath.Rel(dir, p); rerr == nil {
				for _, s := range skip {
					if rel == s {
						return filepath.SkipDir
					}
				}
			}
			return nil
		}
		if !d.Type().IsRegular() || !isTemp(d.Name()) {
			return nil
		}
		if fi, err := d.Info(); err == nil && fi.ModTime().Before(cutoff) && os.Remove(p) == nil {
			n++
		}
		return nil
	})
	return n
}

// SyncDir waits until the entries of a folder (files made, renamed or removed in it)
// are on the disk. It does nothing on Windows, where a folder cannot be synced.
func SyncDir(dir string) error { return syncDir(dir) }
