// Package doctor checks that augmented brain is set up and says how to fix what is not.
package doctor

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/icalvete/augmented-brain/internal/brain"
	"github.com/icalvete/augmented-brain/internal/claudecode"
	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/memory"
	"github.com/icalvete/augmented-brain/internal/paths"
)

// Status of one check.
type Status int

const (
	OK      Status = iota
	Info           // worth knowing, not a problem
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
	// Lang is the language of the report when there is no configuration to say it.
	Lang string
}

// Run returns every check, in the order they are shown. Later checks are skipped when
// an earlier one makes them meaningless (no configuration, no memory folder).
func Run(e Env) []Check {
	if e.LookPath == nil {
		e.LookPath = exec.LookPath
	}
	var out []Check
	add := func(s Status, what, fix string) { out = append(out, Check{s, what, fix}) }
	if e.Lang == "" {
		e.Lang = "en"
	}

	c, err := config.Load(e.ConfigDir)
	switch {
	case errors.Is(err, config.ErrNotFound):
		add(Failed, i18n.T(e.Lang, "doctor.noconfig", e.ConfigDir), i18n.T(e.Lang, "doctor.noconfig.fix"))
		return append(out, tools(e)...)
	case err != nil:
		add(Failed, err.Error(), i18n.T(e.Lang, "doctor.badconfig.fix"))
		return append(out, tools(e)...)
	}
	e.Lang = c.Language
	l := e.Lang
	add(OK, i18n.T(l, "doctor.config", filepath.Join(c.Dir, config.FileName)), "")

	profile := filepath.Join(c.Dir, "profile.md")
	if brain.ProfileEmpty(c.Dir) {
		add(Info, i18n.T(l, "doctor.blankprofile"), "")
	} else {
		add(OK, i18n.T(l, "doctor.profile", profile), "")
	}

	if s, isDefault := c.SpaceAt(e.WorkDir); s != nil && !isDefault {
		add(OK, i18n.T(l, "doctor.inspace", s.Name), "")
	} else if s != nil {
		add(Info, i18n.T(l, "doctor.default", s.Name), "")
	} else {
		var where []string
		for _, name := range c.SpaceNames() {
			if f := c.Spaces[name].WorkFolders; len(f) > 0 {
				where = append(where, name+" ("+paths.Tilde(f[0])+")")
			}
		}
		add(Info, i18n.T(l, "doctor.nospace", strings.Join(where, ", ")), "")
	}

	for _, name := range c.SpaceNames() {
		out = append(out, space(c, c.Spaces[name])...)
	}
	return append(out, tools(e)...)
}

func space(c *config.Config, s *config.Space) []Check {
	l := c.Language
	m, problems, err := memory.Load(s.Memory)
	if err != nil {
		return []Check{{Failed, i18n.T(l, "doctor.nomemory", s.Name, s.Memory, err),
			i18n.T(l, "doctor.nomemory.fix", filepath.Join(c.Dir, config.FileName))}}
	}
	if len(problems) > 0 {
		return []Check{{Failed, i18n.T(l, "doctor.invalid", s.Name, len(problems)), i18n.T(l, "doctor.invalid.fix")}}
	}
	checks := []Check{{OK, i18n.T(l, "doctor.valid", s.Name, i18n.Count(l, "notes", len(m.Notes))), ""}}
	current, _ := os.ReadFile(filepath.Join(s.Memory, memory.IndexFile))
	if string(current) != m.Index(c.Language) {
		checks = append(checks, Check{Warning, i18n.T(l, "doctor.stale", s.Name, memory.IndexFile), i18n.T(l, "doctor.stale.fix")})
	}
	return checks
}

func tools(e Env) []Check {
	var out []Check
	for _, t := range []struct{ bin, why, fix string }{
		{"claude", i18n.T(e.Lang, "doctor.claude"), i18n.T(e.Lang, "doctor.claude.fix")},
		{"git", i18n.T(e.Lang, "doctor.git"), i18n.T(e.Lang, "doctor.git.fix", GitInstall())},
	} {
		if p, err := e.LookPath(t.bin); err == nil {
			out = append(out, Check{OK, i18n.T(e.Lang, "doctor.tool", t.why, p), ""})
		} else {
			out = append(out, Check{Failed, i18n.T(e.Lang, "doctor.notool", t.why), t.fix})
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
		return []Check{{Warning, i18n.T(e.Lang, "doctor.plugin.unknown", err), ""}}
	case !ok:
		return []Check{{Failed, i18n.T(e.Lang, "doctor.plugin.missing", claudecode.Plugin),
			"claude plugin marketplace add " + claudecode.Marketplace() + " && claude plugin install " + claudecode.Plugin + " --scope user"}}
	}
	return []Check{{OK, i18n.T(e.Lang, "doctor.plugin.ok", claudecode.Plugin), ""}}
}

// GitInstall is the command that installs git on this system.
func GitInstall() string {
	switch runtime.GOOS {
	case "darwin":
		return "xcode-select --install"
	case "windows":
		return "winget install --id Git.Git -e --source winget"
	}
	for _, pm := range []struct{ bin, cmd string }{
		{"apt-get", "sudo apt install git"},
		{"dnf", "sudo dnf install git"},
		{"pacman", "sudo pacman -S git"},
		{"zypper", "sudo zypper install git"},
	} {
		if _, err := exec.LookPath(pm.bin); err == nil {
			return pm.cmd
		}
	}
	return "your system's package manager"
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
