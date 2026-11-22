package remote

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/icalvete/augmented-brain/internal/filelock"
	"github.com/icalvete/augmented-brain/internal/state"
)

// The state folder keeps a few records per task session: the worktrees noted for a task
// (task-worktrees), the worktree session start made (session-worktrees), the lock a
// start holds (session-locks) and the stops on record (session-stops). Some outlived
// what they named: the 22-nov audit found 10 noted worktrees whose folder was gone,
// a session worktree record of a task closed hours before, and 117 start locks from
// the 13-nov on. TidyRecords takes those off; it never touches a folder.

// lockAge is how old a lock file must be before TidyRecords removes it: a start holds
// its lock for seconds, a stop for 30 minutes at most.
const lockAge = 24 * time.Hour

// folderGone says the folder is not there (any other error: it may be, so it stays).
func folderGone(folder string) bool {
	_, err := os.Stat(folder)
	return errors.Is(err, os.ErrNotExist)
}

// TidyRecords takes off the records that name a folder that is gone, and the lock
// files nobody holds that are older than lockAge, of tasks with no session on record.
// It returns how many it took off.
func TidyRecords(cfgDir string, now time.Time) int {
	n := 0
	dir := taskWorktreeRecords(cfgDir)
	if entries, err := os.ReadDir(dir); err == nil {
		for _, e := range entries {
			if e.IsDir() || strings.HasSuffix(e.Name(), ".lock") {
				continue
			}
			_ = changeNoted(cfgDir, e.Name(), func(list []noted) ([]noted, error) {
				var left []noted
				for _, x := range list {
					if folderGone(x.folder) {
						n++
						continue
					}
					left = append(left, x)
				}
				return left, nil
			})
		}
	}
	if entries, err := os.ReadDir(worktreeRecords(cfgDir)); err == nil {
		for _, e := range entries {
			rec := filepath.Join(worktreeRecords(cfgDir), e.Name())
			data, err := os.ReadFile(rec)
			if err != nil || e.IsDir() {
				continue
			}
			if folder := strings.TrimSpace(string(data)); folder == "" || folderGone(folder) {
				if os.Remove(rec) == nil {
					n++
				}
			}
		}
	}
	for _, locks := range []string{filepath.Join(cfgDir, state.Dir, "session-locks"), stopsDir(cfgDir)} {
		entries, err := os.ReadDir(locks)
		if err != nil {
			continue
		}
		for _, e := range entries {
			task, ok := strings.CutSuffix(e.Name(), ".lock")
			if !ok || HasLaunch(cfgDir, task) {
				continue
			}
			if _, err := os.Stat(stopFile(cfgDir, task, ".pending")); err == nil {
				continue // a stop on record still needs its lock
			}
			fi, err := e.Info()
			if err != nil || now.Sub(fi.ModTime()) < lockAge {
				continue
			}
			path := filepath.Join(locks, e.Name())
			release, err := filelock.LockWithin(path, 0)
			if err != nil {
				continue // someone holds it
			}
			release() // Windows removes no file a process has open
			if os.Remove(path) == nil {
				n++
			}
		}
	}
	return n
}
