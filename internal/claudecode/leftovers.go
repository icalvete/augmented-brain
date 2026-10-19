package claudecode

import (
	"os"
	"path/filepath"
)

// ConfigDir is where Claude Code keeps its settings and plugins: $CLAUDE_CONFIG_DIR, or
// ~/.claude.
func ConfigDir() string {
	if d := os.Getenv("CLAUDE_CONFIG_DIR"); d != "" {
		return d
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".claude")
}

// Leftovers are what Claude Code keeps of the plugin after it is uninstalled: its cached
// copy, its data folder, the marketplace's copy if it stayed, and the logs of the MCP
// server, one folder per project in Claude Code's cache (~/.cache on Linux,
// ~/Library/Caches on macOS, %LOCALAPPDATA% on Windows). Only the paths that exist.
func Leftovers() []string {
	var found []string
	add := func(p string) {
		if _, err := os.Stat(p); err == nil {
			found = append(found, p)
		}
	}
	if cfg := ConfigDir(); cfg != "" {
		add(filepath.Join(cfg, "plugins", "cache", MarketplaceName))
		add(filepath.Join(cfg, "plugins", "data", "brain-"+MarketplaceName))
		add(filepath.Join(cfg, "plugins", "marketplaces", MarketplaceName))
	}
	if cache, err := os.UserCacheDir(); err == nil {
		for _, base := range []string{filepath.Join(cache, "claude-cli-nodejs"), filepath.Join(cache, "claude-cli-nodejs", "Cache")} {
			logs, _ := filepath.Glob(filepath.Join(base, "*", "mcp-logs-plugin-brain-"+MarketplaceName))
			found = append(found, logs...)
		}
	}
	return found
}

// RemoveLeftovers removes the leftovers, and the project folder of a log when nothing
// else is left in it. It returns the first error and goes on with the rest.
func RemoveLeftovers() error {
	var first error
	for _, p := range Leftovers() {
		if err := os.RemoveAll(p); err != nil && first == nil {
			first = err
		}
		// A log folder's project folder in the cache, empty now, goes too.
		if filepath.Base(filepath.Dir(filepath.Dir(p))) != "plugins" {
			os.Remove(filepath.Dir(p)) // only succeeds when empty
		}
	}
	return first
}
