package remote

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/paths"
)

// SettingsFile, in the configuration folder, holds the settings of remote access. A
// file of its own, like secrets.toml, not a table of config.toml: a program from before
// this one refuses a setting of config.toml it does not know, and must still read the
// configuration of a computer that has remote access.
const SettingsFile = "remote.toml"

// PermissionModes are the values Claude Code accepts for --permission-mode.
var PermissionModes = []string{"acceptEdits", "auto", "bypassPermissions", "default", "dontAsk", "plan"}

// DefaultMode is the permission mode when remote.toml names none: Claude asks, and the
// guard stays in force.
const DefaultMode = "default"

// Settings are the settings of remote.toml, with their defaults filled in.
type Settings struct {
	// Enabled is whether remote access is set up on this computer: `remote install`
	// turns it on, `remote uninstall` off. Only then does doctor look at it.
	Enabled bool
	// Spaces are the spaces that get a service. Default: the default space.
	Spaces []string
	// NamePrefix starts the name of each session, "<prefix>-<space>". Default: the
	// computer's name.
	NamePrefix string
	// PermissionMode is the --permission-mode of the Claude server.
	PermissionMode string
	// Folders are where a space's session starts, when it is not the space's first work
	// folder: space name to an absolute path. It must lie inside a work folder of the
	// space, so the session still loads that space.
	Folders map[string]string
}

type fileSettings struct {
	Enabled        bool              `toml:"enabled"`
	Spaces         []string          `toml:"spaces"`
	NamePrefix     string            `toml:"name_prefix"`
	PermissionMode string            `toml:"permission_mode"`
	Folders        map[string]string `toml:"folders"`
}

// safeWord is what a name prefix may hold: it goes into a command line and a session
// name, so it is one plain word.
var safeWord = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,60}$`)

// LoadSettings reads remote.toml of the configuration c (absent: all defaults, not
// enabled) and checks it. hostname is the computer's name, the default prefix.
func LoadSettings(c *config.Config, hostname string) (Settings, error) {
	var f fileSettings
	path := filepath.Join(c.Dir, SettingsFile)
	md, err := toml.DecodeFile(path, &f)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return Settings{}, errors.New(path + ": " + err.Error())
	}
	if extra := md.Undecoded(); len(extra) > 0 {
		keys := make([]string, len(extra))
		for i, k := range extra {
			keys[i] = k.String()
		}
		return Settings{}, errors.New(i18n.T(c.Language, "config.unknown", path, strings.Join(keys, ", ")))
	}
	s := Settings{Enabled: f.Enabled, Spaces: f.Spaces, NamePrefix: f.NamePrefix, PermissionMode: f.PermissionMode}
	if s.Spaces == nil && c.DefaultSpace != "" {
		s.Spaces = []string{c.DefaultSpace}
	}
	if s.Spaces == nil && len(c.Spaces) == 1 {
		s.Spaces = c.SpaceNames()
	}
	if s.NamePrefix == "" {
		s.NamePrefix = cleanWord(hostname)
	}
	if s.PermissionMode == "" {
		s.PermissionMode = DefaultMode
	}
	l := c.Language
	if !slices.Contains(PermissionModes, s.PermissionMode) {
		return s, errors.New(i18n.T(l, "remote.badmode", path, s.PermissionMode, strings.Join(PermissionModes, ", ")))
	}
	if !safeWord.MatchString(s.NamePrefix) {
		return s, errors.New(i18n.T(l, "remote.badprefix", path, s.NamePrefix))
	}
	for _, name := range s.Spaces {
		if _, ok := c.Spaces[name]; !ok {
			return s, errors.New(i18n.T(l, "remote.badspace", path, name, strings.Join(c.SpaceNames(), ", ")))
		}
	}
	for name, folder := range f.Folders {
		sp, ok := c.Spaces[name]
		if !ok {
			return s, errors.New(i18n.T(l, "remote.badspace", path, name, strings.Join(c.SpaceNames(), ", ")))
		}
		abs, err := paths.Resolve(folder)
		if err != nil || folder == "" {
			return s, errors.New(i18n.T(l, "remote.badfolder", path, name, folder))
		}
		inside := false
		for _, w := range sp.WorkFolders {
			if paths.Within(abs, w) {
				inside = true
				break
			}
		}
		if !inside {
			return s, errors.New(i18n.T(l, "remote.notwork", path, folder, name))
		}
		if s.Folders == nil {
			s.Folders = map[string]string{}
		}
		s.Folders[name] = abs
	}
	return s, nil
}

// cleanWord turns a computer's name into a safe word: other characters become "-".
func cleanWord(s string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(s) {
		if r < 128 && (r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '.' || r == '_' || r == '-') {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	out := strings.Trim(b.String(), "-.")
	if len(out) > 60 {
		out = out[:60]
	}
	if out == "" {
		return "computer"
	}
	return out
}

const settingsHeader = `# augmented brain: Claude remote access (augmented-brain remote install).
# A file of its own because earlier versions refuse a setting of config.toml they do not know.
# Other settings, all optional:
#   spaces = ["personal"]       # spaces that get a session; default: default_space
#   name_prefix = "my-laptop"   # sessions are called <name_prefix>-<space>; default: this computer's name
#   permission_mode = "default" # acceptEdits, auto, bypassPermissions, default, dontAsk or plan
#   folders = { personal = "~/projects" } # where a session starts; default: the space's first work folder
`

// SetEnabled writes enabled in remote.toml of dir, keeping every other line. Turning it
// on creates the file; turning it off leaves a computer that never had it alone.
func SetEnabled(dir string, on bool) error {
	path := filepath.Join(dir, SettingsFile)
	line := "enabled = false"
	if on {
		line = "enabled = true"
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		if !on {
			return nil
		}
		return paths.WriteAtomic(path, []byte(settingsHeader+line+"\n"), 0o600)
	}
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	done := false
	for i, l := range lines {
		if rest, ok := strings.CutPrefix(strings.TrimSpace(l), "enabled"); ok && strings.HasPrefix(strings.TrimLeft(rest, " \t"), "=") {
			lines[i], done = line, true
			break
		}
	}
	if !done {
		lines = append([]string{line}, lines...)
	}
	return paths.WriteAtomic(path, []byte(strings.Join(lines, "\n")), 0o600)
}
