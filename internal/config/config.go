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
	if !contains(Languages, c.Language) {
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
		for i, f := range s.GitAskRepos {
			if s.GitAskRepos[i], err = absPath(f); err != nil {
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
	out := Config{Format: c.Format, Language: c.Language, DefaultSpace: c.DefaultSpace, Spaces: map[string]*Space{},
		ProductionAWSProfiles: c.ProductionAWSProfiles, UntouchableHosts: c.UntouchableHosts}
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
		cp.GitAskRepos = tildeAll(s.GitAskRepos)
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

// writeAtomic replaces the configuration file in one step.
func writeAtomic(path, text string) error {
	return paths.WriteAtomic(path, []byte(text), 0o600)
}

// AddSpace appends a space to the configuration file in dir, leaving everything else
// in the file (comments included) as it is.
func AddSpace(dir string, s *Space) error {
	path := filepath.Join(dir, FileName)
	old, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	block := &Config{Spaces: map[string]*Space{s.Name: s}, Dir: dir}
	var b strings.Builder
	enc := toml.NewEncoder(&b)
	enc.Indent = ""
	cp := *s
	cp.Memory = tilde(s.Memory)
	cp.WorkFolders = tildeAll(s.WorkFolders)
	cp.GitAllowedRepos = tildeAll(s.GitAllowedRepos)
	cp.GitAskRepos = tildeAll(s.GitAskRepos)
	block.Spaces[s.Name] = &cp
	if err := enc.Encode(struct {
		Spaces map[string]*Space `toml:"spaces"`
	}{block.Spaces}); err != nil {
		return err
	}
	text := strings.TrimRight(string(old), "\n") + "\n\n" + strings.TrimSpace(strings.Replace(b.String(), "[spaces]\n", "", 1)) + "\n"
	return replaceChecked(dir, path, string(old), text)
}

// SetSpaceKey sets one string setting of a space in the configuration file in dir,
// changing only that line (or adding it under the space's header).
func SetSpaceKey(dir, space, key, value string) error {
	return setSpaceLine(dir, space, key, fmt.Sprintf("%s = %q", key, value))
}

// SetTopList sets one machine-wide list setting (production_aws_profiles,
// untouchable_hosts) in the configuration file in dir: it changes only that line, or
// adds it before the first table, and keeps everything else, comments included.
func SetTopList(dir, key string, values []string) error {
	path := filepath.Join(dir, FileName)
	old, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	line := fmt.Sprintf("%s = [%s]", key, quoteAll(values))
	lines := strings.Split(string(old), "\n")
	at := -1
	for i, l := range lines {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "[") {
			if at < 0 {
				at = i
			}
			break
		}
		if strings.HasPrefix(t, key+" ") || strings.HasPrefix(t, key+"=") {
			lines[i] = line
			return replaceChecked(dir, path, string(old), strings.Join(lines, "\n"))
		}
	}
	if at < 0 {
		at = len(lines)
	}
	// Keep a blank line between the top settings and the first table.
	insert := []string{line}
	if at > 0 && strings.TrimSpace(lines[at-1]) == "" {
		at--
	} else if at < len(lines) {
		insert = append(insert, "")
	}
	lines = append(lines[:at], append(insert, lines[at:]...)...)
	return replaceChecked(dir, path, string(old), strings.Join(lines, "\n"))
}

func quoteAll(values []string) string {
	quoted := make([]string, len(values))
	for i, v := range values {
		quoted[i] = fmt.Sprintf("%q", v)
	}
	return strings.Join(quoted, ", ")
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

// SetSpaceList sets one list setting of a space (such as untouchable_hosts) the same way.
func SetSpaceList(dir, space, key string, values []string) error {
	return setSpaceLine(dir, space, key, fmt.Sprintf("%s = [%s]", key, quoteAll(values)))
}

// RemoveSpaceKey takes one setting out of a space in the configuration file in dir,
// changing nothing else. A setting that is not there is fine.
func RemoveSpaceKey(dir, space, key string) error {
	return setSpaceLine(dir, space, key, "")
}

// setSpaceLine puts line in place of the key's line under the space's header, or adds
// it there. An empty line removes the key.
func setSpaceLine(dir, space, key, line string) error {
	path := filepath.Join(dir, FileName)
	old, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(old), "\n")
	header := -1
	for i, l := range lines {
		t := strings.TrimSpace(l)
		if header < 0 {
			if t == "[spaces."+space+"]" || t == `[spaces."`+space+`"]` {
				header = i
			}
			continue
		}
		if strings.HasPrefix(t, "[") {
			break // the next table: the key is not there yet
		}
		if strings.HasPrefix(t, key+" ") || strings.HasPrefix(t, key+"=") {
			if line == "" {
				lines = append(lines[:i], lines[i+1:]...)
			} else {
				lines[i] = line
			}
			return replaceChecked(dir, path, string(old), strings.Join(lines, "\n"))
		}
	}
	if header < 0 {
		return fmt.Errorf("%s has no space %q", path, space)
	}
	if line == "" {
		return nil
	}
	lines = append(lines[:header+1], append([]string{line}, lines[header+1:]...)...)
	return replaceChecked(dir, path, string(old), strings.Join(lines, "\n"))
}

// replaceChecked writes the new text and loads it back; if it does not load, the old
// text is put back and the error returned, so a bad edit never leaves a broken file.
func replaceChecked(dir, path, old, text string) error {
	if err := writeAtomic(path, text); err != nil {
		return err
	}
	if _, err := Load(dir); err != nil {
		_ = writeAtomic(path, old)
		return fmt.Errorf("the change would break %s, so it was not made: %w", path, err)
	}
	return nil
}

func tildeAll(paths []string) []string {
	if len(paths) == 0 {
		return nil
	}
	out := make([]string, len(paths))
	for i, p := range paths {
		out[i] = tilde(p)
	}
	return out
}

// tilde writes a path under the home folder as ~/…
func tilde(p string) string { return paths.Tilde(p) }

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

// SetDefaultSpace writes default_space in the configuration file in dir, right after
// the language, changing nothing else.
func SetDefaultSpace(dir, space string) error {
	path := filepath.Join(dir, FileName)
	old, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	line := fmt.Sprintf("default_space = %q", space)
	lines := strings.Split(string(old), "\n")
	at := -1
	for i, l := range lines {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "[") {
			break
		}
		if strings.HasPrefix(t, "default_space") {
			lines[i] = line
			return replaceChecked(dir, path, string(old), strings.Join(lines, "\n"))
		}
		if strings.HasPrefix(t, "language") {
			at = i + 1
		}
	}
	if at < 0 {
		at = 0
	}
	lines = append(lines[:at], append([]string{line}, lines[at:]...)...)
	return replaceChecked(dir, path, string(old), strings.Join(lines, "\n"))
}

// FormatVersion is the version of the configuration folder: 1 when the file says none.
func (c *Config) FormatVersion() int {
	if c.Format < 1 {
		return 1
	}
	return c.Format
}

// SetFormat records the version of the configuration folder, as the first line of the
// file, keeping everything else as it is. Version 1 is never written: it is what a
// file that says none means, and releases before versions were written refuse a
// setting they do not know.
func SetFormat(dir string, v int) error {
	if v <= 1 {
		return nil
	}
	path := filepath.Join(dir, FileName)
	old, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	line := fmt.Sprintf("format = %d", v)
	lines := strings.Split(string(old), "\n")
	for i, l := range lines {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "[") {
			break
		}
		if strings.HasPrefix(t, "format ") || strings.HasPrefix(t, "format=") {
			if t == line {
				return nil
			}
			lines[i] = line
			return replaceChecked(dir, path, string(old), strings.Join(lines, "\n"))
		}
	}
	return replaceChecked(dir, path, string(old), line+"\n"+string(old))
}

// within reports whether path is folder itself or inside it.
func within(path, folder string) bool { return paths.Within(path, folder) }

// absPath expands a leading ~, makes the path absolute and resolves symlinks, also
// through the part that does not exist yet, so the same folder always compares equal.
func absPath(p string) (string, error) { return paths.Resolve(p) }

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
