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
