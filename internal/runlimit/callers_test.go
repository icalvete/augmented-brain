package runlimit

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// unbounded are the files that may start a program with exec.Command, and why: each
// one is meant to outlive the call (internal/detach), is answered by a person, or runs
// in tests only. Anything else waits through Command or Output, so it has a limit.
var unbounded = map[string]string{
	"internal/remote/remote.go":                 "realStartDaemon: Claude's daemon, detached",
	"internal/remote/closetask.go":              "session stop --when-idle, detached",
	"internal/hook/session_end.go":              "the automatic backup, detached",
	"internal/hook/updatecheck.go":              "the check for a new version, detached",
	"internal/panel/backup.go":                  "a backup started from the panel, detached",
	"internal/panel/control.go":                 "the panel's server, detached; the browser it opens",
	"internal/detach/detach_windows.go":         "detach itself",
	"internal/testenv/testenv.go":               "tests only",
	"internal/backup/github.go":                 "gh auth login: the person answers in the browser",
	"cmd/augmented-brain/update.go":             "the installer and session restart --old, which show their progress",
	"cmd/augmented-brain/restart.go":            "session restart --when-idle, detached",
	"cmd/augmented-brain/watch.go":              "the keeper of watch pr, detached",
	"cmd/augmented-brain/telegram.go":           "telegram connect: the person scans a QR code; the Telegram program bounds its own waits",
	"cmd/augmented-brain/removeself_windows.go": "the line that deletes the program after it ends",
}

var (
	plain   = regexp.MustCompile(`\bexec\.Command\(`)
	withCtx = regexp.MustCompile(`\bexec\.CommandContext\(`)
)

// Every program augmented brain waits for has a limit that holds (R-08 of the
// November 2026 review): no exec.CommandContext outside this package, and
// exec.Command only where unbounded says why.
func TestEveryCallHasALimit(t *testing.T) {
	root := filepath.Join("..", "..")
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if n := d.Name(); p != root && (strings.HasPrefix(n, ".") || n == "testdata" || n == "runlimit") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		src, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		if withCtx.Match(src) {
			t.Errorf("%s: exec.CommandContext; use runlimit.Command, whose limit holds", rel)
		}
		if plain.Match(src) {
			if _, ok := unbounded[rel]; !ok {
				t.Errorf("%s: exec.Command with no limit; use runlimit.Command or runlimit.Output, or say in unbounded why it needs none", rel)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
