// Package config reads the configuration file, where the spaces are declared, and
// decides which space a folder belongs to.
//
// See docs/spec/memory-format.md, "Spaces" and "Configuration".
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/icalvete/augmented-brain/internal/paths"
)

// FileName is the configuration file inside the configuration folder.
const FileName = "config.toml"

// AskSafetyFile, in the configuration folder, says a restore brought no guard lists
// (no backup carries them): the next session offers, once, to ask for them again.
const AskSafetyFile = ".ask-safety"

// Languages the program can speak.
var Languages = []string{"en", "es"}

// Space keeps one part of the user's life apart from the others (work, personal…).
type Space struct {
	Name                  string   `toml:"-"`
	Memory                string   `toml:"memory"`
	WorkFolders           []string `toml:"work_folders"`
	Backup                string   `toml:"backup,omitempty"`
	ProductionAWSProfiles []string `toml:"production_aws_profiles,omitempty"`
	UntouchableHosts      []string `toml:"untouchable_hosts,omitempty"`
	GitAllowedRepos       []string `toml:"git_allowed_repos,omitempty"`
	GitAskRepos           []string `toml:"git_ask_repos,omitempty"`
}

// Config is the whole configuration file.
type Config struct {
	// Format is the version of the configuration folder (this file and the profile);
	// 0 when the file does not say, which means 1 (see package format).
	Format   int    `toml:"format,omitzero"`
	Language string `toml:"language"`
	// DefaultSpace is used when the assistant runs in a folder of no space, such as the
	// scratch folder the Claude desktop app opens a new session in. Empty: none.
	DefaultSpace string `toml:"default_space,omitempty"`
	// ProductionAWSProfiles and UntouchableHosts are machine-wide: the guard applies them
	// in every space and outside every space, together with any list a space still has
	// of its own (older configurations kept them per space).
	ProductionAWSProfiles []string          `toml:"production_aws_profiles,omitempty"`
	UntouchableHosts      []string          `toml:"untouchable_hosts,omitempty"`
	Spaces                map[string]*Space `toml:"spaces"`

	// Dir is the folder the configuration was read from.
	Dir string `toml:"-"`
}

// Dir returns the configuration folder: $AB_CONFIG_DIR if set, otherwise
// $XDG_CONFIG_HOME/augmented-brain or ~/.config/augmented-brain on Linux and macOS,
// and the platform's user configuration folder on Windows.
func Dir() (string, error) {
	if d := os.Getenv("AB_CONFIG_DIR"); d != "" {
		return d, nil
	}
	if runtime.GOOS == "windows" {
		base, err := os.UserConfigDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(base, "augmented-brain"), nil
	}
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "augmented-brain"), nil
	}
	home, err := paths.Home()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "augmented-brain"), nil
}

// ErrNotFound means there is no configuration file yet (run `augmented-brain init`).
var ErrNotFound = errors.New("no configuration found")

// Load reads and checks the configuration in dir. Paths come back absolute, with ~ expanded.
func Load(dir string) (*Config, error) {
	path := filepath.Join(dir, FileName)
	var c Config
	md, err := toml.DecodeFile(path, &c)
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("%w in %s", ErrNotFound, dir)
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if extra := md.Undecoded(); len(extra) > 0 {
		keys := make([]string, len(extra))
		for i, k := range extra {
			keys[i] = k.String()
		}
		return nil, fmt.Errorf("%s: unknown settings: %s", path, strings.Join(keys, ", "))
	}
	c.Dir = dir
	if err := c.normalize(); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &c, nil
}

func (c *Config) normalize() error {
	if c.Language == "" {
		c.Language = "en"
	}
	if !slices.Contains(Languages, c.Language) {
		return fmt.Errorf("language %q is not one of %s", c.Language, strings.Join(Languages, ", "))
	}
	if len(c.Spaces) == 0 {
		return errors.New("no spaces declared")
	}
	if c.DefaultSpace != "" {
		if _, ok := c.Spaces[c.DefaultSpace]; !ok {
			return fmt.Errorf("default_space %q is not one of the spaces (%s)", c.DefaultSpace, strings.Join(c.SpaceNames(), ", "))
		}
	}
	for name, s := range c.Spaces {
		s.Name = name
		if s.Memory == "" {
			return fmt.Errorf("space %q: memory folder is missing", name)
		}
		m, err := absPath(s.Memory)
		if err != nil {
			return fmt.Errorf("space %q: %w", name, err)
		}
		s.Memory = m
		for _, list := range [][]string{s.WorkFolders, s.GitAllowedRepos, s.GitAskRepos} {
			if err := absAll(list); err != nil {
				return fmt.Errorf("space %q: %w", name, err)
			}
		}
	}
	return nil
}

// absAll makes every path of list absolute, in place (see absPath).
func absAll(list []string) error {
	for i, p := range list {
		a, err := absPath(p)
		if err != nil {
			return err
		}
		list[i] = a
	}
	return nil
}

// Save writes the configuration to its folder, creating the folder if needed. Paths
// under the home folder are written with ~ so the file is easy to read and to move.
func (c *Config) Save() error {
	if err := os.MkdirAll(c.Dir, 0o700); err != nil {
		return err
	}
	out := Config{Format: c.Format, Language: c.Language, DefaultSpace: c.DefaultSpace, Spaces: map[string]*Space{},
		ProductionAWSProfiles: c.ProductionAWSProfiles, UntouchableHosts: c.UntouchableHosts}
	for name, s := range c.Spaces {
		cp := s.forFile()
		if cp.WorkFolders == nil {
			cp.WorkFolders = []string{} // a whole file always says work_folders, even empty
		}
		out.Spaces[name] = &cp
	}
	var b strings.Builder
	b.WriteString("# augmented brain configuration. See docs/spec/memory-format.md, \"Configuration\".\n\n")
	enc := toml.NewEncoder(&b)
	enc.Indent = ""
	if err := enc.Encode(out); err != nil {
		return err
	}
	// The encoder writes an empty [spaces] table before the [spaces.x] ones: drop it.
	text := strings.Replace(b.String(), "[spaces]\n", "", 1)
	text = strings.ReplaceAll(text, "\n[spaces.", "\n\n[spaces.")
	text = strings.Replace(text, "\n\n\n[spaces.", "\n\n[spaces.", 1)
	return writeAtomic(filepath.Join(c.Dir, FileName), text)
}

// forFile returns a copy of the space as it is written to the file: paths under the
// home folder with ~, and empty lists left out.
func (s Space) forFile() Space {
	s.Memory = paths.Tilde(s.Memory)
	s.WorkFolders = tildeAll(s.WorkFolders)
	s.GitAllowedRepos = tildeAll(s.GitAllowedRepos)
	s.GitAskRepos = tildeAll(s.GitAskRepos)
	return s
}

// tildeAll writes each path under the home folder as ~/…; an empty list becomes nil, so
// the encoder leaves it out.
func tildeAll(list []string) []string {
	if len(list) == 0 {
		return nil
	}
	out := make([]string, len(list))
	for i, p := range list {
		out[i] = paths.Tilde(p)
	}
	return out
}

// GuardLists returns the production AWS profiles and the untouchable hosts the guard
// applies: the machine-wide lists and every space's own, without repeats.
func (c *Config) GuardLists() (profiles, hosts []string) {
	profiles = append(profiles, c.ProductionAWSProfiles...)
	hosts = append(hosts, c.UntouchableHosts...)
	for _, name := range c.SpaceNames() {
		profiles = append(profiles, c.Spaces[name].ProductionAWSProfiles...)
		hosts = append(hosts, c.Spaces[name].UntouchableHosts...)
	}
	return unique(profiles), unique(hosts)
}

func unique(list []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, x := range list {
		if x != "" && !seen[x] {
			seen[x] = true
			out = append(out, x)
		}
	}
	return out
}

// SharedFolder is the folder, inside the configuration folder, of the shared notes:
// the rules and knowledge that apply in every space. It is also the name an import map
// uses for it, so no space can take it.
const SharedFolder = "shared"

// SharedDir returns the folder of the shared notes.
func (c *Config) SharedDir() string { return filepath.Join(c.Dir, SharedFolder) }

// SpaceNames returns the declared spaces, sorted.
func (c *Config) SpaceNames() []string {
	names := make([]string, 0, len(c.Spaces))
	for n := range c.Spaces {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// SpaceFor returns the space whose work folder contains dir. When folders are nested,
// the deepest one wins. It returns nil when dir belongs to no space.
func (c *Config) SpaceFor(dir string) *Space {
	d, err := absPath(dir)
	if err != nil {
		return nil
	}
	var best *Space
	bestLen := -1
	for _, name := range c.SpaceNames() {
		s := c.Spaces[name]
		for _, f := range s.WorkFolders {
			if paths.Within(d, f) && len(f) > bestLen {
				best, bestLen = s, len(f)
			}
		}
	}
	return best
}

// SpaceAt returns the space for a session in dir: the one whose work folder holds it,
// or else the default space, if there is one. isDefault says which.
func (c *Config) SpaceAt(dir string) (s *Space, isDefault bool) {
	if s := c.SpaceFor(dir); s != nil {
		return s, false
	}
	if s, ok := c.Spaces[c.DefaultSpace]; ok {
		return s, true
	}
	return nil, false
}

// FormatVersion is the version of the configuration folder: 1 when the file says none.
func (c *Config) FormatVersion() int {
	if c.Format < 1 {
		return 1
	}
	return c.Format
}

// absPath expands a leading ~, makes the path absolute and resolves symlinks, also
// through the part that does not exist yet, so the same folder always compares equal.
func absPath(p string) (string, error) { return paths.Resolve(p) }
