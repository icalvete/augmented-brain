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
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

// FileName is the configuration file inside the configuration folder.
const FileName = "config.toml"

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
}

// Config is the whole configuration file.
type Config struct {
	Language string            `toml:"language"`
	Spaces   map[string]*Space `toml:"spaces"`

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
	home, err := os.UserHomeDir()
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
	if !contains(Languages, c.Language) {
		return fmt.Errorf("language %q is not one of %s", c.Language, strings.Join(Languages, ", "))
	}
	if len(c.Spaces) == 0 {
		return errors.New("no spaces declared")
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
		for i, f := range s.WorkFolders {
			if s.WorkFolders[i], err = absPath(f); err != nil {
				return fmt.Errorf("space %q: %w", name, err)
			}
		}
		for i, f := range s.GitAllowedRepos {
			if s.GitAllowedRepos[i], err = absPath(f); err != nil {
				return fmt.Errorf("space %q: %w", name, err)
			}
		}
	}
	return nil
}

// Save writes the configuration to its folder, creating the folder if needed. Paths
// under the home folder are written with ~ so the file is easy to read and to move.
func (c *Config) Save() error {
	if err := os.MkdirAll(c.Dir, 0o700); err != nil {
		return err
	}
	out := Config{Language: c.Language, Spaces: map[string]*Space{}}
	for name, s := range c.Spaces {
		cp := *s
		cp.Memory = tilde(s.Memory)
		cp.WorkFolders = make([]string, len(s.WorkFolders))
		for i, f := range s.WorkFolders {
			cp.WorkFolders[i] = tilde(f)
		}
		cp.GitAllowedRepos = make([]string, len(s.GitAllowedRepos))
		for i, f := range s.GitAllowedRepos {
			cp.GitAllowedRepos[i] = tilde(f)
		}
		if len(cp.GitAllowedRepos) == 0 {
			cp.GitAllowedRepos = nil
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
	path := filepath.Join(c.Dir, FileName)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(text), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// tilde writes a path under the home folder as ~/…
func tilde(p string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return p
	}
	if h, err := filepath.EvalSymlinks(home); err == nil {
		home = h
	}
	if p == home {
		return "~"
	}
	if rel, err := filepath.Rel(home, p); err == nil && within(p, home) {
		return "~/" + filepath.ToSlash(rel)
	}
	return p
}

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
			if within(d, f) && len(f) > bestLen {
				best, bestLen = s, len(f)
			}
		}
	}
	return best
}

// within reports whether path is folder itself or inside it.
func within(path, folder string) bool {
	rel, err := filepath.Rel(folder, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

// absPath expands a leading ~, makes the path absolute and resolves symlinks when the
// path exists, so the same folder always compares equal.
func absPath(p string) (string, error) {
	if p == "~" || strings.HasPrefix(p, "~/") || strings.HasPrefix(p, `~\`) {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		p = filepath.Join(home, p[1:])
	}
	p, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	if r, err := filepath.EvalSymlinks(p); err == nil {
		p = r
	}
	return filepath.Clean(p), nil
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
