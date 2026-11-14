package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/icalvete/augmented-brain/internal/paths"
)

// GuardFile, in the configuration folder, holds what the guard must never allow:
// today the untouchable repositories. A file of its own, not a setting of config.toml:
// a program that does not know it never reads it (earlier programs refuse an unknown
// setting of config.toml), and a missing file means nothing is untouchable.
const GuardFile = "guard.toml"

// Guard is the content of guard.toml.
type Guard struct {
	// UntouchableRepos are repositories where the assistant never commits, pushes or
	// merges, whatever the other lists say. Same form as git_allowed_repos: a path
	// (with ~), a * in its last part, or a worktree of one of them. For the whole
	// machine, not for one space.
	UntouchableRepos []string `toml:"untouchable_repos"`
	// Reach is where augmented brain acts: ReachMachine (or empty, the default) in
	// every Claude Code session of this computer, ReachSpaces only in the folders of
	// the spaces. Outside them, with ReachSpaces, sessions get no memory and no notices,
	// and the guard keeps only what protects its own files and what the user named in
	// the lists (production profiles, untouchable hosts, the repositories of the lists).
	Reach string `toml:"reach"`
}

// The values of Guard.Reach.
const (
	ReachMachine = "machine"
	ReachSpaces  = "spaces"
)

// OnlySpaces says whether augmented brain keeps to the folders of the spaces.
func (g Guard) OnlySpaces() bool { return g.Reach == ReachSpaces }

// LoadGuard reads guard.toml in the configuration folder. A missing file is an empty
// Guard; a file that cannot be read is an error, so the caller can say so.
func LoadGuard(cfgDir string) (Guard, error) {
	var g Guard
	path := filepath.Join(cfgDir, GuardFile)
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return g, nil
	}
	if _, err := toml.DecodeFile(path, &g); err != nil {
		return Guard{}, fmt.Errorf("%s: %w", path, err)
	}
	g.UntouchableRepos = unique(g.UntouchableRepos)
	if err := absAll(g.UntouchableRepos); err != nil { // as the lists of config.toml: absolute, ~ expanded
		return Guard{}, fmt.Errorf("%s: %w", path, err)
	}
	switch g.Reach {
	case "", ReachMachine, ReachSpaces:
	default:
		// Unknown: the guard cannot tell how far it should act, so it reads the file
		// as broken and asks, rather than guess either way.
		return Guard{}, fmt.Errorf("%s: reach = %q: it must be %q or %q", path, g.Reach, ReachMachine, ReachSpaces)
	}
	return g, nil
}

// SaveGuard writes guard.toml (0600, replacing the file in one step).
func SaveGuard(cfgDir string, g Guard) error {
	var b strings.Builder
	b.WriteString("# What the guard never allows. The panel's Guard page writes this file.\n")
	b.WriteString("# untouchable_repos: no commit, push or merge there, whatever the other lists say.\n")
	b.WriteString("untouchable_repos = [\n")
	for _, r := range tildeAll(unique(g.UntouchableRepos)) {
		b.WriteString("  " + fmt.Sprintf("%q", r) + ",\n")
	}
	b.WriteString("]\n")
	if g.Reach != "" {
		b.WriteString("# reach: \"machine\" acts in every Claude Code session; \"spaces\" only in the spaces' folders.\n")
		b.WriteString("reach = " + fmt.Sprintf("%q", g.Reach) + "\n")
	}
	return paths.WriteAtomic(filepath.Join(cfgDir, GuardFile), []byte(b.String()), 0o600)
}
