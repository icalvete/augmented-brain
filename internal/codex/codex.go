// Package codex installs and checks augmented brain in Codex, OpenAI's coding assistant:
// a plugin with the session start (nine SessionStart hooks), the guard (a PreToolUse hook)
// and the MCP server in tasks-only mode, plus what Codex needs around it — trust in those
// hooks and unified_exec_tty = false, without which a terminal escapes the guard.
//
// Codex's own files are only changed by Codex: the plugin through its command line
// (`codex plugin`), the trust and the feature through its app server (`config/batchWrite`),
// which keeps the user's comments. The trust is the hash Codex gives for each hook
// (`hooks/list`): it changes with the hook's content, so a new version re-trusts its hooks
// (Install again) and Check sees a hook whose content changed ("modified").
//
// The app server is experimental in Codex; dev/codex-integration.md says what was measured.
package codex

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/paths"
	"github.com/icalvete/augmented-brain/internal/session"
)

// MarketplaceName is the name of the local marketplace the program writes for Codex.
const MarketplaceName = "augmented-brain"

// PluginName is the plugin's name; Plugin, its id in Codex.
const (
	PluginName = "brain"
	Plugin     = PluginName + "@" + MarketplaceName
)

// Fix is the command that installs, or repairs, augmented brain in Codex.
const Fix = "augmented-brain codex install"

// HookCount is how many hooks the plugin has: the start's parts and the guard.
const HookCount = session.Parts + 1

// Dir is the folder, inside the configuration folder, where the program writes the
// marketplace Codex reads the plugin from.
func Dir(cfgDir string) string {
	return filepath.Join(cfgDir, "codex-marketplace")
}

// Runner runs a command and returns its combined output. Tests replace it.
type Runner func(name string, args ...string) (string, error)

// Server is a connection to Codex's app server. Tests replace it.
type Server interface {
	// Call sends a request and decodes its result into result (nil to drop it).
	Call(method string, params, result any) error
	Close() error
}

// Env is what Install, Uninstall and Check work with.
type Env struct {
	Run     Runner
	Serve   func() (Server, error)
	Dir     string // where the marketplace is written: Dir(config folder)
	Exe     string // the program the hooks and the MCP server run
	Version string
	GOOS    string
	Cwd     string // the folder hooks/list looks from; plugin hooks are the same anywhere
}

// Available reports whether the codex command is on the PATH.
func Available() bool {
	_, err := exec.LookPath("codex")
	return err == nil
}

// files are the marketplace's files, relative to Dir, with their content.
func files(exe, version, goos string) (map[string][]byte, error) {
	run := hookCommand(exe, goos)
	var start []map[string]any
	for n := 1; n <= session.Parts; n++ {
		start = append(start, map[string]any{"type": "command", "timeout": 10,
			"command": run("hook", "session-start", strconv.Itoa(n), "codex")})
	}
	guard := run("hook", "pre-tool-use", "codex")
	if goos != "windows" {
		// A missing program (127) would let the command through: Codex runs it when a
		// hook fails. Exit 2 blocks, and says why.
		guard += `; s=$?; [ $s -eq 127 ] && { echo "augmented brain: its program is missing (` + exe +
			`), so its guard cannot check this. Install it again, or run augmented-brain codex uninstall." >&2; exit 2; }; exit $s`
	}
	out := map[string]any{
		".claude-plugin/marketplace.json": map[string]any{
			"name": MarketplaceName, "owner": map[string]any{"name": "augmented brain"},
			"plugins": []any{map[string]any{"name": PluginName, "source": "./" + PluginName,
				"description": "augmented brain in Codex: the space at the start, the guard and the task tools."}},
		},
		PluginName + "/.codex-plugin/plugin.json": map[string]any{
			"name": PluginName, "version": version,
			"description": "augmented brain in Codex: loads the space's memory and rules when a session starts, refuses risky commands and file changes, and gives the task tools.",
			"homepage":    "https://github.com/icalvete/augmented-brain", "license": "MIT",
			"hooks": "./hooks/hooks.json", "mcpServers": "./.mcp.json",
		},
		PluginName + "/hooks/hooks.json": map[string]any{"hooks": map[string]any{
			"SessionStart": []any{map[string]any{"hooks": start}},
			"PreToolUse": []any{map[string]any{"hooks": []any{
				map[string]any{"type": "command", "timeout": 10, "command": guard}}}},
		}},
		PluginName + "/.mcp.json": map[string]any{"mcpServers": map[string]any{
			// Approved: the tasks-only tools write notes and nothing else, and asking for
			// each journal line would make the assistant stop writing them.
			"augmented-brain": map[string]any{"command": exe, "args": []string{"mcp"},
				"env":                         map[string]string{"AB_MCP_TASKS_ONLY": "1"},
				"default_tools_approval_mode": "approve",
				// Codex gives a server only a few variables of its own: a configuration
				// folder set with AB_CONFIG_DIR must reach it too.
				"env_vars": []string{"AB_CONFIG_DIR"}},
		}},
	}
	b := map[string][]byte{}
	for name, v := range out {
		j, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return nil, err
		}
		b[name] = append(j, '\n')
	}
	return b, nil
}

// hookCommand gives a hook's command line: the program by its full path, quoted for sh.
// On Windows, by its name: the installer puts it on the PATH, and the shell Codex runs
// hooks with there is not sh.
func hookCommand(exe, goos string) func(args ...string) string {
	name := "augmented-brain"
	if goos != "windows" {
		name = shQuote(exe)
	}
	return func(args ...string) string { return name + " " + strings.Join(args, " ") }
}

func shQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// Write writes the marketplace to dir, each file whole.
func Write(dir, exe, version, goos string) error {
	fs, err := files(exe, version, goos)
	if err != nil {
		return err
	}
	for name, b := range fs {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return err
		}
		if err := paths.WriteAtomic(p, b, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// Remove takes away what Write wrote, and the folders it leaves empty.
func Remove(dir string) error {
	fs, err := files("", "", "")
	if err != nil {
		return err
	}
	var first error
	for name := range fs {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) && first == nil {
			first = err
		}
		for d := filepath.Dir(p); d != dir && strings.HasPrefix(d, dir); d = filepath.Dir(d) {
			if os.Remove(d) != nil {
				break
			}
		}
	}
	os.Remove(dir) // only when empty
	return first
}

// Install writes the marketplace, adds it and the plugin to Codex (adding again brings
// a new version), trusts the plugin's hooks and turns unified_exec_tty off. It is safe to
// repeat, and what it changes is only augmented brain's.
func Install(e Env) error {
	if err := Write(e.Dir, e.Exe, e.Version, e.GOOS); err != nil {
		return fmt.Errorf("writing the plugin to %s: %v", e.Dir, err)
	}
	if out, err := e.Run("codex", "plugin", "marketplace", "add", e.Dir); err != nil &&
		!strings.Contains(out, "already") {
		return fmt.Errorf("adding the marketplace: %v: %s", err, strings.TrimSpace(out))
	}
	if out, err := e.Run("codex", "plugin", "add", Plugin); err != nil {
		return fmt.Errorf("adding the plugin: %v: %s", err, strings.TrimSpace(out))
	}
	srv, err := e.Serve()
	if err != nil {
		return err
	}
	defer srv.Close()
	hooks, err := listHooks(srv, e.Cwd)
	if err != nil {
		return err
	}
	if len(hooks) != HookCount {
		return fmt.Errorf("codex shows %d of the plugin's %d hooks", len(hooks), HookCount)
	}
	edits := []configEdit{{KeyPath: "features.unified_exec_tty", MergeStrategy: "upsert", Value: false}}
	for _, h := range hooks {
		edits = append(edits, configEdit{KeyPath: stateKey(h.Key), MergeStrategy: "replace",
			Value: map[string]string{"trusted_hash": h.CurrentHash}})
	}
	if err := srv.Call("config/batchWrite", map[string]any{"edits": edits}, nil); err != nil {
		return fmt.Errorf("trusting the hooks: %v", err)
	}
	st := check(srv, e.Cwd)
	if st.Err != nil {
		return st.Err
	}
	if !st.OK() {
		return fmt.Errorf("codex still does not run augmented brain's hooks: %d of %d trusted", st.Trusted, HookCount)
	}
	return nil
}

// Uninstall removes the plugin, its marketplace, the trust in its hooks and the files the
// program wrote. unified_exec_tty stays off: it only protects. Something already gone is fine.
func Uninstall(e Env) error {
	srv, err := e.Serve()
	if err != nil {
		return err
	}
	defer srv.Close()
	keys, err := trustedKeys(srv)
	if err != nil {
		return err
	}
	if len(keys) > 0 {
		var edits []configEdit
		for _, k := range keys {
			edits = append(edits, configEdit{KeyPath: stateKey(k), MergeStrategy: "replace", Value: nil})
		}
		if err := srv.Call("config/batchWrite", map[string]any{"edits": edits}, nil); err != nil {
			return fmt.Errorf("removing the trust in the hooks: %v", err)
		}
	}
	if out, err := e.Run("codex", "plugin", "remove", Plugin); err != nil && !gone(out) {
		return fmt.Errorf("removing the plugin: %v: %s", err, strings.TrimSpace(out))
	}
	if out, err := e.Run("codex", "plugin", "marketplace", "remove", MarketplaceName); err != nil && !gone(out) {
		return fmt.Errorf("removing the marketplace: %v: %s", err, strings.TrimSpace(out))
	}
	return Remove(e.Dir)
}

// gone tells, from what codex printed, that there was nothing to remove.
func gone(out string) bool {
	o := strings.ToLower(out)
	return strings.Contains(o, "not found") || strings.Contains(o, "not installed") ||
		strings.Contains(o, "is not configured") || strings.Contains(o, "unknown")
}

// Status is how augmented brain stands in Codex.
type Status struct {
	Installed bool  // the plugin is installed and enabled
	Hooks     int   // how many of the plugin's hooks Codex shows
	Trusted   int   // how many of those it runs
	Modified  bool  // some changed after they were trusted (a new version, not installed again)
	TTYOff    bool  // unified_exec_tty is false
	Err       error // Codex could not be asked
}

// OK reports whether everything is in place.
func (s Status) OK() bool {
	return s.Err == nil && s.Installed && s.Hooks == HookCount && s.Trusted == HookCount && s.TTYOff
}

// Check asks Codex how augmented brain stands there.
func Check(e Env) Status {
	srv, err := e.Serve()
	if err != nil {
		return Status{Err: err}
	}
	defer srv.Close()
	return check(srv, e.Cwd)
}

func check(srv Server, cwd string) Status {
	var st Status
	var plugins struct {
		Marketplaces []struct {
			Plugins []struct {
				ID        string `json:"id"`
				Installed bool   `json:"installed"`
				Enabled   bool   `json:"enabled"`
			} `json:"plugins"`
		} `json:"marketplaces"`
	}
	if err := srv.Call("plugin/list", map[string]any{}, &plugins); err != nil {
		st.Err = fmt.Errorf("asking Codex for its plugins: %v", err)
		return st
	}
	for _, m := range plugins.Marketplaces {
		for _, p := range m.Plugins {
			if p.ID == Plugin && p.Installed && p.Enabled {
				st.Installed = true
			}
		}
	}
	hooks, err := listHooks(srv, cwd)
	if err != nil {
		st.Err = err
		return st
	}
	st.Hooks = len(hooks)
	for _, h := range hooks {
		switch h.TrustStatus {
		case "trusted", "managed":
			st.Trusted++
		case "modified":
			st.Modified = true
		}
	}
	var cfg struct {
		Config struct {
			Features map[string]any `json:"features"`
		} `json:"config"`
	}
	if err := srv.Call("config/read", map[string]any{}, &cfg); err != nil {
		st.Err = fmt.Errorf("reading Codex's configuration: %v", err)
		return st
	}
	st.TTYOff = cfg.Config.Features["unified_exec_tty"] == false
	return st
}

type configEdit struct {
	KeyPath       string `json:"keyPath"`
	MergeStrategy string `json:"mergeStrategy"`
	Value         any    `json:"value"`
}

// stateKey is the configuration key of a hook's trust. The hook's key has dots
// ("hooks/hooks.json"), so it goes in quotes, as in TOML.
func stateKey(hook string) string {
	q, _ := json.Marshal(hook)
	return "hooks.state." + string(q)
}

type hookInfo struct {
	Key         string `json:"key"`
	PluginID    string `json:"pluginId"`
	CurrentHash string `json:"currentHash"`
	TrustStatus string `json:"trustStatus"`
}

// listHooks gives the plugin's hooks as Codex sees them.
func listHooks(srv Server, cwd string) ([]hookInfo, error) {
	var res struct {
		Data []struct {
			Hooks []hookInfo `json:"hooks"`
		} `json:"data"`
	}
	if err := srv.Call("hooks/list", map[string]any{"cwds": []string{cwd}}, &res); err != nil {
		return nil, fmt.Errorf("asking Codex for its hooks: %v", err)
	}
	var out []hookInfo
	for _, d := range res.Data {
		for _, h := range d.Hooks {
			if h.PluginID == Plugin {
				out = append(out, h)
			}
		}
	}
	return out, nil
}

// trustedKeys are the trust entries of the plugin's hooks in Codex's configuration,
// whether the plugin is still there or not.
func trustedKeys(srv Server) ([]string, error) {
	var cfg struct {
		Config struct {
			Hooks struct {
				State map[string]json.RawMessage `json:"state"`
			} `json:"hooks"`
		} `json:"config"`
	}
	if err := srv.Call("config/read", map[string]any{}, &cfg); err != nil {
		return nil, fmt.Errorf("reading Codex's configuration: %v", err)
	}
	var keys []string
	for k := range cfg.Config.Hooks.State {
		if strings.HasPrefix(k, Plugin+":") {
			keys = append(keys, k)
		}
	}
	return keys, nil
}

// ErrNoCodex is returned when the codex command is not on the PATH.
var ErrNoCodex = errors.New("codex is not on the PATH")

// NotInstalled reports whether augmented brain is simply not in Codex: no plugin and
// none of its hooks.
func (s Status) NotInstalled() bool {
	return s.Err == nil && !s.Installed && s.Hooks == 0
}

// Problems says, in language l, what is wrong in Codex, one line each; Fix repairs them
// all. Empty when everything is in place, when it is not installed or Codex could not
// be asked.
func Problems(l string, s Status) []string {
	if s.Err != nil || s.OK() || s.NotInstalled() {
		return nil
	}
	var out []string
	if s.Modified {
		out = append(out, i18n.T(l, "codex.status.modified"))
	} else if !s.Installed || s.Hooks != HookCount || s.Trusted != HookCount {
		out = append(out, i18n.T(l, "codex.status.hooks", s.Trusted, HookCount))
	}
	if !s.TTYOff {
		out = append(out, i18n.T(l, "codex.status.tty"))
	}
	return out
}
