package remote

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/paths"
)

// What macOS and Windows share: the steps before a service is written (claude found,
// each space's folder there or made), and the PATH the service carries.

// prepare checks what a service needs before anything is written: the spaces, claude,
// and each space's folder (a space with no work folder gets its own, made a work folder
// of the space, as on Linux).
func (s System) prepare(c *config.Config, set Settings) (*InstallResult, error) {
	if len(set.Spaces) == 0 {
		return nil, errors.New(i18n.T(c.Language, "remote.nospaces", filepath.Join(c.Dir, SettingsFile)))
	}
	claude, err := s.FindClaude()
	if err != nil {
		return nil, errors.New(i18n.T(c.Language, "remote.noclaude"))
	}
	return &InstallResult{Services: s.services(c, set), Claude: claude, Linger: LingerNone}, nil
}

// folders makes or checks each service's folder, once the system is known to be able
// to keep the services.
func (s System) folders(c *config.Config, res *InstallResult) error {
	for _, svc := range res.Services {
		if svc.Own {
			if err := os.MkdirAll(svc.Folder, 0o700); err != nil {
				return err
			}
			added, err := addWorkFolder(c, svc)
			if err != nil {
				return err
			}
			if added {
				res.Added = append(res.Added, svc)
			}
		} else if fi, err := os.Stat(svc.Folder); err != nil || !fi.IsDir() {
			return errors.New(i18n.T(c.Language, "remote.nofolder", paths.Tilde(svc.Folder), svc.Space))
		}
	}
	return nil
}

// finish is the end of every install: remote access set on, and the folders Claude
// does not trust yet.
func (s System) finish(c *config.Config, res *InstallResult) (*InstallResult, error) {
	if err := SetEnabled(c.Dir, true); err != nil {
		return nil, err
	}
	trusted := s.trustedFolders()
	for _, svc := range res.Services {
		if !trusted(svc.Folder) {
			res.Untrusted = append(res.Untrusted, svc)
		}
	}
	return res, nil
}

// services are Services with the unit named as this system names it: a launchd label
// on macOS, a task on Windows.
func (s System) services(c *config.Config, set Settings) []Service {
	out := Services(c, set)
	for i := range out {
		switch s.GOOS {
		case "darwin":
			out[i].Unit = launchdLabel(out[i].Space)
		case "windows":
			out[i].Unit = taskName(out[i].Space)
		}
	}
	return out
}

// logFile is where a service's errors go on macOS and Windows.
func (s System) logFile(space string) string {
	return filepath.Join(s.LogDir, unitPrefix+space+".log")
}

// servicePath is the PATH a service starts with, first the folders of claude and of
// this program (a service starts with the system's bare PATH, where neither may be: the
// sessions' hooks run augmented-brain, and claude starts its own tools), then rest.
func servicePath(sep string, claude, self string, rest ...string) string {
	var dirs []string
	seen := map[string]bool{}
	add := func(d string) {
		if d == "" || d == "." || seen[d] {
			return
		}
		seen[d] = true
		dirs = append(dirs, d)
	}
	add(dirOf(claude))
	if self != "" {
		add(dirOf(self))
	}
	for _, d := range rest {
		add(d)
	}
	return strings.Join(dirs, sep)
}

// dirOf is the folder of a path written for the system the service runs on, which in
// tests may not be this one.
func dirOf(p string) string {
	i := strings.LastIndexAny(p, `/\`)
	if i <= 0 {
		return ""
	}
	return p[:i]
}

// markerComment is the marker as an XML comment, for the plist and the task.
const markerComment = "<!-- " + markerWords + " -->"

const markerWords = "Written by augmented-brain remote install; remove it with augmented-brain remote uninstall."

// installedFiles are the files in the unit folder that start with prefix, end with ext
// and carry the marker (read through decode), sorted by name.
func (s System) installedFiles(prefix, ext string, decode func([]byte) string) []string {
	files, _ := filepath.Glob(filepath.Join(s.UnitDir, prefix+"*"+ext))
	var out []string
	for _, f := range files {
		if data, err := os.ReadFile(f); err == nil && strings.Contains(decode(data), markerComment) {
			out = append(out, f)
		}
	}
	return out // Glob sorts
}
