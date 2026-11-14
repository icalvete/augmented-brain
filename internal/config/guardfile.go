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
	path := filepath.Join(cfgDir, GuardFile)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Guard{}, nil
	}
	if err != nil {
		return Guard{}, fmt.Errorf("%s: %w", path, err)
	}
	g, err := parseGuard(string(data))
	if err != nil {
		return Guard{}, fmt.Errorf("%s: %w", path, err)
	}
	return g, nil
}

// parseGuard reads the text of guard.toml with every check LoadGuard makes.
func parseGuard(text string) (Guard, error) {
	var g Guard
	if _, err := toml.Decode(text, &g); err != nil {
		return Guard{}, err
	}
	g.UntouchableRepos = unique(g.UntouchableRepos)
	if err := absAll(g.UntouchableRepos); err != nil { // as the lists of config.toml: absolute, ~ expanded
		return Guard{}, err
	}
	switch g.Reach {
	case "", ReachMachine, ReachSpaces:
	default:
		// Unknown: the guard cannot tell how far it should act, so it reads the file
		// as broken and asks, rather than guess either way.
		return Guard{}, fmt.Errorf("reach = %q: it must be %q or %q", g.Reach, ReachMachine, ReachSpaces)
	}
	return g, nil
}

// SaveGuard writes the settings of g into guard.toml (0600, replacing the file in one
// step). An existing file is changed one setting at a time, as config.toml is: the
// user's comments and any setting this program does not know (one a newer program
// wrote, say) stay as they were, so a panel left open from before an update never
// takes away a protection it cannot see. The setting changed is written whole again
// (a comment inside its list goes). The result is loaded back before it stays.
func SaveGuard(cfgDir string, g Guard) error {
	path := filepath.Join(cfgDir, GuardFile)
	repos := guardReposLines(g.UntouchableRepos)
	reach := ""
	if g.Reach != "" {
		reach = "reach = " + fmt.Sprintf("%q", g.Reach)
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		var b strings.Builder
		b.WriteString("# What the guard never allows. The panel's Guard page writes this file.\n")
		b.WriteString(reposComment + "\n")
		b.WriteString(strings.Join(repos, "\n") + "\n")
		if reach != "" {
			b.WriteString(reachComment + "\n" + reach + "\n")
		}
		return saveGuardText(path, b.String())
	}
	if err != nil {
		return err
	}
	f := &lineEditor{dir: cfgDir, path: path, old: string(data), lines: strings.Split(string(data), "\n"),
		check: func(text string) error { _, err := parseGuard(text); return err }}
	f.setTop("untouchable_repos", repos, reposComment)
	if reach == "" {
		f.setTop("reach", nil, "")
	} else {
		f.setTop("reach", []string{reach}, reachComment)
	}
	return saveGuardText(path, strings.Join(f.lines, "\n"))
}

// The lines that explain each setting, written with it.
const (
	reposComment = "# untouchable_repos: no commit, push or merge there, whatever the other lists say."
	reachComment = "# reach: \"machine\" acts in every Claude Code session; \"spaces\" only in the spaces' folders."
)

// guardReposLines is untouchable_repos as the file keeps it: one repository a line.
func guardReposLines(repos []string) []string {
	out := []string{"untouchable_repos = ["}
	for _, r := range tildeAll(unique(repos)) {
		out = append(out, "  "+fmt.Sprintf("%q", r)+",")
	}
	return append(out, "]")
}

// saveGuardText checks that text loads as guard.toml and only then replaces the file:
// the guard reads it at every command.
func saveGuardText(path, text string) error {
	if _, err := parseGuard(text); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return paths.WriteAtomic(path, []byte(text), 0o600)
}
