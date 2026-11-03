package paths

import (
	"os"
	"runtime"
	"time"
)

// renameTries is how many times Rename tries on Windows, a short pause apart.
var renameTries = 10

// Rename moves a file or folder, replacing a file already at the destination. On
// Windows a rename fails while another program (an antivirus, the search indexer, an
// editor) has the destination open for a moment, so it tries again briefly there.
func Rename(from, to string) error {
	err := os.Rename(from, to)
	if err == nil || runtime.GOOS != "windows" {
		return err
	}
	for i := 1; i < renameTries; i++ {
		time.Sleep(time.Duration(i) * 10 * time.Millisecond)
		if err = os.Rename(from, to); err == nil {
			return nil
		}
	}
	return err
}

// Remove deletes a file; one already gone is not an error. On Windows a file cannot be
// deleted while another process reads it, so it tries again briefly there, as Rename.
func Remove(path string) error {
	err := os.Remove(path)
	for i := 1; err != nil && !os.IsNotExist(err) && runtime.GOOS == "windows" && i < renameTries; i++ {
		time.Sleep(time.Duration(i) * 10 * time.Millisecond)
		err = os.Remove(path)
	}
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
