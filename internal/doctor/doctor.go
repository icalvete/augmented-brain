// Package doctor checks that augmented brain is set up and says how to fix what is not.
package doctor

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/icalvete/augmented-brain/internal/claudecode"
	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/memory"
)

// Status of one check.
type Status int

const (
	OK      Status = iota
	Warning        // works, but something is worth a look
	Failed         // must be fixed
)

// Check is one line of the report.
type Check struct {
	Status Status
	What   string // what was checked and what was found
	Fix    string // how to fix it; empty when OK
}

// Env is what the checks depend on, so tests can replace it.
type Env struct {
	ConfigDir string
	WorkDir   string
	LookPath  func(string) (string, error)
	// Plugin reports whether the Claude Code plugin is installed; nil means ask
	// Claude Code itself.
	Plugin func() (bool, error)
}

// Run returns every check, in the order they are shown. Later checks are skipped when
// an earlier one makes them meaningless (no configuration, no memory folder).
func Run(e Env) []Check {
	if e.LookPath == nil {
		e.LookPath = exec.LookPath
	}
	var out []Check
	add := func(s Status, what, fix string) { out = append(out, Check{s, what, fix}) }

	c, err := config.Load(e.ConfigDir)
	switch {
	case errors.Is(err, config.ErrNotFound):
		add(Failed, "no configuration in "+e.ConfigDir, "run `augmented-brain init`")
		return append(out, tools(e)...)
	case err != nil:
		add(Failed, err.Error(), "fix the file, or move it away and run `augmented-brain init`")
		return append(out, tools(e)...)
	}
	add(OK, "configuration: "+filepath.Join(c.Dir, config.FileName), "")

	profile := filepath.Join(c.Dir, "profile.md")
	if _, err := os.Stat(profile); err != nil {
		add(Warning, "no profile at "+profile, "write who you are and how you like to work there")
	} else {
		add(OK, "profile: "+profile, "")
	}

	if s := c.SpaceFor(e.WorkDir); s != nil {
		add(OK, fmt.Sprintf("this folder belongs to space %q", s.Name), "")
	} else {
		add(Warning, fmt.Sprintf("this folder belongs to no space (spaces: %v)", c.SpaceNames()),
			"open your assistant inside a space's work folder, or run `augmented-brain space add`")
	}

	for _, name := range c.SpaceNames() {
		out = append(out, space(c, c.Spaces[name])...)
	}
	return append(out, tools(e)...)
}

func space(c *config.Config, s *config.Space) []Check {
	m, problems, err := memory.Load(s.Memory)
	if err != nil {
		return []Check{{Failed, fmt.Sprintf("space %s: memory folder %s: %v", s.Name, s.Memory, err),
			"create it, or fix the path in " + filepath.Join(c.Dir, config.FileName)}}
	}
	if len(problems) > 0 {
		return []Check{{Failed, fmt.Sprintf("space %s: %d notes are not valid", s.Name, len(problems)),
			"run `augmented-brain check` inside the space to see them"}}
	}
	notes := fmt.Sprintf("%d notes", len(m.Notes))
	if len(m.Notes) == 1 {
		notes = "1 note"
	}
	checks := []Check{{OK, fmt.Sprintf("space %s: %s, all valid", s.Name, notes), ""}}
	current, _ := os.ReadFile(filepath.Join(s.Memory, memory.IndexFile))
	if string(current) != m.Index(c.Language) {
		checks = append(checks, Check{Warning, fmt.Sprintf("space %s: %s is out of date", s.Name, memory.IndexFile),
			"run `augmented-brain check` inside the space to rebuild it"})
	}
	return checks
}

func tools(e Env) []Check {
	var out []Check
	for _, t := range []struct{ bin, why, fix string }{
		{"claude", "Claude Code", "install it: https://docs.claude.com/en/docs/claude-code"},
		{"git", "git, for backups", "install git with your system's package manager"},
	} {
		if p, err := e.LookPath(t.bin); err == nil {
			out = append(out, Check{OK, t.why + ": " + p, ""})
		} else {
			out = append(out, Check{Warning, t.why + ": not found", t.fix})
		}
	}
	return append(out, plugin(e)...)
}

func plugin(e Env) []Check {
	if _, err := e.LookPath("claude"); err != nil {
		return nil // already reported: Claude Code not found
	}
	installed := e.Plugin
	if installed == nil {
		installed = func() (bool, error) { return claudecode.Installed(claudecode.Exec) }
	}
	ok, err := installed()
	switch {
	case err != nil:
		return []Check{{Warning, "Claude Code plugin: cannot tell (" + err.Error() + ")", ""}}
	case !ok:
		return []Check{{Warning, "Claude Code plugin " + claudecode.Plugin + ": not installed",
			"claude plugin marketplace add " + claudecode.Marketplace() + " && claude plugin install " + claudecode.Plugin + " --scope user"}}
	}
	return []Check{{OK, "Claude Code plugin " + claudecode.Plugin + ": installed", ""}}
}

// Worst returns the most serious status in the report.
func Worst(checks []Check) Status {
	w := OK
	for _, c := range checks {
		if c.Status > w {
			w = c.Status
		}
	}
	return w
}
