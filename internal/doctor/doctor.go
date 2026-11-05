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
	"slices"
	"strings"
	"time"

	"github.com/icalvete/augmented-brain/internal/brain"
	"github.com/icalvete/augmented-brain/internal/claudecode"
	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/hostmem"
	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/memory"
	"github.com/icalvete/augmented-brain/internal/paths"
	"github.com/icalvete/augmented-brain/internal/remote"
	"github.com/icalvete/augmented-brain/internal/state"
	"github.com/icalvete/augmented-brain/internal/version"
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
	// ClaudeDir is Claude Code's folder (settings, projects); "" means its usual one.
	ClaudeDir string
	// Remote is the system remote access is read from; nil means the real one.
	Remote *remote.System
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
	if _, err := os.Stat(filepath.Join(c.Dir, config.AskSafetyFile)); err == nil && len(profiles)+len(hosts) == 0 {
		// Restored from a backup, which never carries the lists: say so until they are asked.
		add(Warning, i18n.T(l, "doctor.nolists"), i18n.T(l, "doctor.nolists.fix", filepath.Join(c.Dir, config.FileName)))
	} else {
		add(Info, i18n.T(l, "doctor.lists", len(profiles), len(hosts)), "")
	}
	roots, bad := c.RootExceptions()
	for _, b := range bad {
		add(Warning, i18n.T(l, "doctor.rootask.bad", b), i18n.T(l, "doctor.rootask.fix", filepath.Join(c.Dir, config.FileName)))
	}
	for _, r := range roots {
		until := r.Until.Format("2006-01-02 15:04")
		if time.Now().Before(r.Until) {
			add(Info, i18n.T(l, "doctor.rootask", r.Host, until), "")
		} else {
			add(Info, i18n.T(l, "doctor.rootask.expired", r.Host, until), i18n.T(l, "doctor.rootask.expired.fix", filepath.Join(c.Dir, config.FileName)))
		}
	}

	out = append(out, guardLists(c, l)...)
	for _, name := range c.SpaceNames() {
		out = append(out, space(c, c.Spaces[name])...)
	}
	out = append(out, remoteChecks(e, c)...)
	out = append(out, autoMemory(e, c)...)
	if old := state.OldServers(c.Dir, version.Version); len(old) > 0 {
		add(Warning, i18n.T(l, "doctor.oldservers", len(old), version.Version), i18n.T(l, "doctor.oldservers.fix"))
	}
	out = append(out, memoryHogs(l)...)
	return append(out, tools(e)...)
}

func space(c *config.Config, s *config.Space) []Check {
	l := c.Language
	var private []Check
	if paths.Exposed(s.Memory) {
		private = append(private, Check{Warning, i18n.T(l, "doctor.exposed", s.Name, paths.Tilde(s.Memory)), i18n.T(l, "doctor.exposed.fix")})
	}
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
	return append(checks, private...)
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

// memoryHogs warns about every Claude window on this machine whose host process (VS
// Code's extension host, or claude in a terminal) is past the first step of the Stop
// hook's memory warning: the system may kill it, with every session in it.
func memoryHogs(lang string) []Check {
	if hostmem.Local == nil {
		return nil
	}
	first := hostmem.Steps(os.Getenv)[0] << 20
	var out []Check
	for _, h := range hostmem.Hosts(hostmem.Local) {
		if h.RSS >= first {
			out = append(out, Check{Warning, i18n.T(lang, "doctor.memory", hostmem.GB(h.RSS, lang), i18n.T(lang, "doctor.memory."+h.Kind), h.PID),
				i18n.T(lang, "doctor.memory.fix")})
		}
	}
	return out
}

// autoMemory warns when Claude Code keeps a memory of its own for a work folder: its
// auto memory, a second memory next to the space's, which loads at the start too and
// drifts from it. It is off with "autoMemoryEnabled": false in Claude Code's settings.
func autoMemory(e Env, c *config.Config) []Check {
	dir := e.ClaudeDir
	if dir == "" {
		dir = claudecode.ConfigDir()
	}
	if dir == "" {
		return nil
	}
	var settings map[string]any
	if data, err := os.ReadFile(filepath.Join(dir, "settings.json")); err == nil {
		_ = json.Unmarshal(data, &settings)
	}
	if on, ok := settings["autoMemoryEnabled"].(bool); ok && !on {
		return nil
	}
	var found []string
	for _, name := range c.SpaceNames() {
		for _, w := range c.Spaces[name].WorkFolders {
			// Claude Code names the folder after the path it was opened with, which may
			// or may not have its links resolved (/var and /private/var on macOS).
			ways := []string{w}
			if real, err := filepath.EvalSymlinks(w); err == nil && real != w {
				ways = append(ways, real)
			}
			for _, p := range ways {
				mem := filepath.Join(dir, "projects", claudecode.ProjectSlug(p), "memory")
				if _, err := os.Stat(mem); err == nil && !slices.Contains(found, paths.Tilde(mem)) {
					found = append(found, paths.Tilde(mem))
				}
			}
		}
	}
	if len(found) == 0 {
		return nil
	}
	return []Check{{Warning, i18n.T(c.Language, "doctor.automemory", strings.Join(found, ", ")),
		i18n.T(c.Language, "doctor.automemory.fix", filepath.Join(dir, "settings.json"))}}
}

// guardLists reports on guard.toml (the untouchable repositories) and on whether the
// guard's lists are the ones the user last saw.
func guardLists(c *config.Config, l string) []Check {
	var out []Check
	g, err := config.LoadGuard(c.Dir)
	if err != nil {
		return []Check{{Warning, i18n.T(l, "doctor.guardfile.bad", err.Error()), i18n.T(l, "doctor.guardfile.fix", filepath.Join(c.Dir, config.GuardFile))}}
	}
	if len(g.UntouchableRepos) > 0 {
		out = append(out, Check{Info, i18n.T(l, "doctor.untouchable", len(g.UntouchableRepos)), ""})
	}
	allowed, ask := c.GitLists()
	for _, r := range g.UntouchableRepos {
		if slices.Contains(allowed, r) || slices.Contains(ask, r) {
			out = append(out, Check{Warning, i18n.T(l, "doctor.untouchable.both", paths.Tilde(r)), i18n.T(l, "doctor.untouchable.both.fix")})
		}
	}
	if seal := state.ReadSeal(c.Dir); seal != "" && seal != c.GuardFingerprint(g) {
		out = append(out, Check{Warning, i18n.T(l, "doctor.guardchanged"), i18n.T(l, "doctor.guardchanged.fix")})
	}
	return out
}

// remoteChecks is a line per space when remote access is set up (`remote install`): a
// warning when its service is not running, Claude does not trust its folder, or linger
// is off. Nothing when it is not set up, or on a system that has no remote access yet.
func remoteChecks(e Env, c *config.Config) []Check {
	l := c.Language
	sys := e.Remote
	if sys == nil {
		s, err := remote.NewSystem()
		if err != nil {
			return nil
		}
		sys = &s
	}
	host, _ := sys.Hostname()
	set, err := remote.LoadSettings(c, host)
	if err != nil {
		return []Check{{Warning, i18n.T(l, "remote.doctor.bad", err), i18n.T(l, "remote.doctor.bad.fix")}}
	}
	if !set.Enabled {
		return nil
	}
	st, err := sys.Status(c, set)
	if err != nil {
		return nil
	}
	var out []Check
	for _, x := range st.Spaces {
		what, fix, ok := sys.DoctorLine(l, x, st.Linger)
		status := OK
		if !ok {
			status = Warning
		}
		out = append(out, Check{status, what, fix})
	}
	return out
}
