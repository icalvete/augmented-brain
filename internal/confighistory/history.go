// Package confighistory keeps the last versions of the configuration on this machine,
// so a bad change can be seen and undone. A version is a copy of every configuration
// file as it was at one moment, with who changed it. Only the configuration: the
// notes, shared/ included, have their history in the backup, and the program's state
// is not configuration.
//
// Versions never leave the machine. config.toml holds the guard's lists, which may
// name work machines; the backup leaves them out on purpose, and so does this.
package confighistory

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/filelock"
	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/paths"
	"github.com/icalvete/augmented-brain/internal/remote"
)

// Dir is the folder, inside the configuration folder, that holds the versions.
const Dir = "history"

// Keep is how many versions are kept; the oldest goes when there is one more.
const Keep = 10

// Files are the configuration files a version holds.
var Files = []string{"config.toml", "guard.toml", "remote.toml", "session.toml", "profile.md", "secrets.toml"}

// Who changed the configuration. A version records one of these, plus a detail (the
// command, the tool, the version restored).
const (
	ByOutside = "outside" // found changed: by hand, or by a program that keeps no history
	ByCommand = "command" // an augmented-brain command
	ByTool    = "tool"    // a tool of the assistant
	ByPanel   = "panel"
	ByRestore = "restore" // going back to an earlier version
	ByFirst   = "first"   // the configuration as it was when the history began
)

// Version is the configuration at one moment.
type Version struct {
	ID     string            `json:"-"` // its file name without .json; sorts by time
	Time   time.Time         `json:"time"`
	By     string            `json:"by"`
	Detail string            `json:"detail,omitempty"`
	Files  map[string]string `json:"files"` // name to content; a file missing then is not here
}

// ErrNotFound is a version that is not (or no longer) kept.
var ErrNotFound = errors.New("there is no such version")

func dir(cfgDir string) string { return filepath.Join(cfgDir, Dir) }

func lock(cfgDir string) (func(), error) {
	return filelock.LockWithin(filepath.Join(dir(cfgDir), ".lock"), 3*time.Second)
}

// current reads the configuration files as they are now.
func current(cfgDir string) (map[string]string, error) {
	files := map[string]string{}
	for _, name := range Files {
		data, err := os.ReadFile(filepath.Join(cfgDir, name))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		files[name] = string(data)
	}
	return files, nil
}

// Changed reports whether the configuration is not what the latest version holds:
// changed since in a way that kept no history. With no version yet it is false.
func Changed(cfgDir string) (bool, error) {
	vs, err := List(cfgDir)
	if err != nil || len(vs) == 0 {
		return false, err
	}
	cur, err := current(cfgDir)
	if err != nil {
		return false, err
	}
	return !maps.Equal(cur, vs[0].Files), nil
}

// Record keeps the configuration as a new version when it differs from the latest one,
// and drops the oldest beyond Keep. It returns whether it kept one. With no
// configuration folder yet it keeps nothing.
func Record(cfgDir, by, detail string) (bool, error) {
	if _, err := os.Stat(filepath.Join(cfgDir, "config.toml")); err != nil {
		return false, nil
	}
	release, err := lock(cfgDir)
	if err != nil {
		return false, err
	}
	defer release()
	return record(cfgDir, by, detail, time.Now())
}

func record(cfgDir, by, detail string, now time.Time) (bool, error) {
	cur, err := current(cfgDir)
	if err != nil {
		return false, err
	}
	vs, err := List(cfgDir)
	if err != nil {
		return false, err
	}
	if len(vs) > 0 && maps.Equal(cur, vs[0].Files) {
		return false, nil
	}
	if len(vs) == 0 && by == ByOutside { // nothing to say who changed it: where it starts
		by, detail = ByFirst, ""
	}
	id := now.UTC().Format("20060102T150405.000000000Z")
	if len(vs) > 0 && id <= vs[0].ID { // a clock that went back: still after the latest
		id = vs[0].ID + "-1"
	}
	data, err := json.MarshalIndent(Version{Time: now, By: by, Detail: detail, Files: cur}, "", "  ")
	if err != nil {
		return false, err
	}
	if err := os.MkdirAll(dir(cfgDir), 0o700); err != nil {
		return false, err
	}
	if err := paths.WriteAtomic(filepath.Join(dir(cfgDir), id+".json"), data, 0o600); err != nil {
		return false, err
	}
	vs = append([]Version{{ID: id}}, vs...)
	for _, old := range vs[min(len(vs), Keep):] {
		if err := os.Remove(filepath.Join(dir(cfgDir), old.ID+".json")); err != nil && !errors.Is(err, os.ErrNotExist) {
			return true, err
		}
	}
	return true, nil
}

// List returns the kept versions, newest first. A file that is not a valid version is
// skipped.
func List(cfgDir string) ([]Version, error) {
	entries, err := os.ReadDir(dir(cfgDir))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var vs []Version
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".json") {
			continue
		}
		id := strings.TrimSuffix(name, ".json")
		if !validID(id) {
			continue
		}
		v, err := read(cfgDir, id)
		if err != nil {
			continue
		}
		vs = append(vs, v)
	}
	slices.SortFunc(vs, func(a, b Version) int { return strings.Compare(b.ID, a.ID) })
	return vs, nil
}

func read(cfgDir, id string) (Version, error) {
	data, err := os.ReadFile(filepath.Join(dir(cfgDir), id+".json"))
	if errors.Is(err, os.ErrNotExist) {
		return Version{}, ErrNotFound
	}
	if err != nil {
		return Version{}, err
	}
	var v Version
	if err := json.Unmarshal(data, &v); err != nil {
		return Version{}, err
	}
	if v.Files == nil {
		v.Files = map[string]string{}
	}
	v.ID = id
	return v, nil
}

// validID is an id this package writes: digits, T, Z, a dot before the fraction and an
// optional -1 suffix. Nothing that could lead out of the folder.
func validID(id string) bool {
	if id == "" || strings.Contains(id, "..") || strings.ContainsAny(id, `/\`) {
		return false
	}
	for _, r := range id {
		if !(r >= '0' && r <= '9' || r == 'T' || r == 'Z' || r == '.' || r == '-') {
			return false
		}
	}
	return true
}

// Get returns one kept version.
func Get(cfgDir, id string) (Version, error) {
	if !validID(id) {
		return Version{}, ErrNotFound
	}
	return read(cfgDir, id)
}

// Restore makes the configuration what version id holds: it writes its files and
// removes the ones it did not have. The configuration as it was is kept first as a
// version, so going back can be undone too. check is run on the result (load the
// configuration, say); if it fails, the files go back to how they were and its error
// comes back. Restore does not touch the guard's seal: the caller decides whether the
// user saw the change.
func Restore(cfgDir, id string, check func() error) error {
	v, err := Get(cfgDir, id)
	if err != nil {
		return err
	}
	release, err := lock(cfgDir)
	if err != nil {
		return err
	}
	defer release()
	if _, err := record(cfgDir, ByOutside, "", time.Now()); err != nil {
		return err
	}
	before, err := current(cfgDir)
	if err != nil {
		return err
	}
	if err := write(cfgDir, v.Files); err != nil {
		_ = write(cfgDir, before)
		return err
	}
	if check != nil {
		if err := check(); err != nil {
			if werr := write(cfgDir, before); werr != nil {
				return fmt.Errorf("%w (and putting it back failed: %v)", err, werr)
			}
			return err
		}
	}
	_, err = record(cfgDir, ByRestore, id, time.Now())
	return err
}

// write makes the configuration files exactly files: each one written, and each one
// not in it removed.
func write(cfgDir string, files map[string]string) error {
	for _, name := range Files {
		path := filepath.Join(cfgDir, name)
		text, ok := files[name]
		if !ok {
			if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
			continue
		}
		if err := paths.WriteAtomic(path, []byte(text), 0o600); err != nil {
			return err
		}
	}
	return nil
}

// Who says, in lang, who made version v.
func Who(lang string, v Version) string {
	switch v.By {
	case ByPanel:
		// The page, by its name in the panel's menu: "/guard/list" is the Guard page.
		seg, _, _ := strings.Cut(strings.TrimPrefix(v.Detail, "/"), "/")
		if seg != "" && i18n.All("panel." + seg)[0] != "" {
			return i18n.T(lang, "history.by.panel", i18n.T(lang, "panel."+seg))
		}
		return i18n.T(lang, "history.by.panelany")
	case ByCommand, ByTool:
		return i18n.T(lang, "history.by."+v.By, v.Detail)
	case ByRestore:
		return i18n.T(lang, "history.by.restore")
	case ByFirst:
		return i18n.T(lang, "history.by.first")
	}
	return i18n.T(lang, "history.by.outside")
}

// Index finds a version by its number in List (1 the newest) or by its id.
func Index(vs []Version, which string) (int, bool) {
	for i, v := range vs {
		if v.ID == which || strconv.Itoa(i+1) == which {
			return i, true
		}
	}
	return 0, false
}

// Snapshot is the configuration as it is now, as a version would hold it.
func Snapshot(cfgDir string) (map[string]string, error) { return current(cfgDir) }

// Check loads every configuration file the program reads, so a version that would
// leave one unreadable is refused (see Restore).
func Check(cfgDir string) error {
	c, err := config.Load(cfgDir)
	if err != nil {
		return err
	}
	if _, err := config.LoadGuard(cfgDir); err != nil {
		return err
	}
	if _, err := config.LoadSession(cfgDir); err != nil {
		return err
	}
	host, herr := os.Hostname()
	if herr != nil || host == "" {
		host = "localhost"
	}
	_, err = remote.LoadSettings(c, host)
	return err
}
