// Package doctor checks that augmented brain is set up and says how to fix what is not.
package doctor

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

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
	// Hook runs the PreToolUse hook as Claude Code does, with input on stdin, and returns
	// what it prints; nil means run the installed program.
	Hook func(input []byte) ([]byte, error)
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

	if names := c.SpaceNames(); len(names) > 1 && c.DefaultSpace == "" {
		add(Warning, i18n.T(l, "doctor.nodefault"), i18n.T(l, "doctor.nodefault.fix", names[0], filepath.Join(c.Dir, config.FileName)))
	}

	profiles, hosts := c.GuardLists()
	add(Info, i18n.T(l, "doctor.lists", len(profiles), len(hosts)), "")

	for _, name := range c.SpaceNames() {
		out = append(out, space(c, c.Spaces[name])...)
	}
	return append(out, tools(e)...)
}

func space(c *config.Config, s *config.Space) []Check {
	l := c.Language
	m, problems, err := memory.LoadSpace(s.Memory, c.SharedDir())
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
	return []Check{{OK, i18n.T(e.Lang, "doctor.plugin.ok", claudecode.Plugin), ""}, guardWorks(e)}
}

// guardWorks runs the PreToolUse hook end to end with a command it must block. If the
// program is not where the plugin looks for it, or answers wrong, every command would go
// through unchecked without anyone noticing.
func guardWorks(e Env) Check {
	run := e.Hook
	if run == nil {
		run = runHook
	}
	in, _ := json.Marshal(map[string]any{"cwd": e.WorkDir, "hook_event_name": "PreToolUse",
		"tool_name": "Bash", "tool_input": map[string]any{"command": "sudo id"}})
	out, err := run(in)
	var o struct {
		Specific struct {
			Decision string `json:"permissionDecision"`
		} `json:"hookSpecificOutput"`
	}
	// The plugin's hook blocks the tool call when the program cannot run or prints
	// something that is not an answer, so then the assistant can do nothing; when it
	// answers but does not block, risky commands go through.
	if err == nil && len(bytes.TrimSpace(out)) > 0 {
		err = json.Unmarshal(out, &o)
	}
	if err != nil {
		return Check{Failed, i18n.T(e.Lang, "doctor.guard.cannotrun", err.Error()), i18n.T(e.Lang, "doctor.guard.fix")}
	}
	if o.Specific.Decision != "deny" {
		why := strings.TrimSpace(string(out))
		if why == "" {
			why = "no answer"
		}
		return Check{Failed, i18n.T(e.Lang, "doctor.guard.fail", why), i18n.T(e.Lang, "doctor.guard.fix")}
	}
	return Check{OK, i18n.T(e.Lang, "doctor.guard.ok"), ""}
}

// runHook runs `augmented-brain hook pre-tool-use` found the way the plugin's hooks find
// it: on the PATH, or in ~/.local/bin.
func runHook(input []byte) ([]byte, error) {
	bin, err := exec.LookPath("augmented-brain")
	if err != nil {
		home, herr := paths.Home()
		if herr != nil {
			return nil, err
		}
		name := "augmented-brain"
		if runtime.GOOS == "windows" {
			name += ".exe"
		}
		bin = filepath.Join(home, ".local", "bin", name)
		if _, serr := os.Stat(bin); serr != nil {
			return nil, err
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "hook", "pre-tool-use")
	cmd.Stdin = bytes.NewReader(input)
	return cmd.Output()
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
