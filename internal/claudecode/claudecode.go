// Package claudecode installs and checks the augmented brain plugin in Claude Code,
// through Claude Code's own command line, so its settings are only changed the way
// Claude Code itself changes them.
package claudecode

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/version"
)

// Plugin is the plugin id in Claude Code: plugin name @ marketplace name.
const Plugin = "brain@augmented-brain"

// MarketplaceName is the marketplace's name, as .claude-plugin/marketplace.json says.
const MarketplaceName = "augmented-brain"

// GitHubMarketplace is where the plugin came from before it travelled with the program:
// main of the public repository, whatever was on it that day. Install and Update move
// whoever still has it to the program's own folder.
const GitHubMarketplace = "icalvete/augmented-brain"

// Manual is the command that installs or updates the plugin by hand.
const Manual = "augmented-brain update --plugin"

// marketplace writes the plugin this program carries to its folder (LocalDir of the
// configuration folder) and returns that folder, for Claude Code to read the plugin
// from: so the plugin is always the one of the program installed. $AB_MARKETPLACE (a
// path or a repository) takes its place, to try unpublished changes.
var marketplace = func() (string, error) {
	if m := os.Getenv("AB_MARKETPLACE"); m != "" {
		return m, nil
	}
	cfg, err := config.Dir()
	if err != nil {
		return "", err
	}
	dir := LocalDir(cfg)
	if err := WriteLocal(dir, version.Version, runtime.GOOS); err != nil {
		return "", fmt.Errorf("writing the plugin to %s: %v", dir, err)
	}
	return dir, nil
}

// addMarketplace adds the marketplace, or points the one already there at it: Claude
// Code keeps the plugin installed and moves it to the new source. Never remove and add
// again: removing a marketplace uninstalls its plugin and deletes its data.
func addMarketplace(run Runner) error {
	src, err := marketplace()
	if err != nil {
		return err
	}
	if out, err := run("claude", "plugin", "marketplace", "add", src); err != nil &&
		!strings.Contains(out, "already") {
		return fmt.Errorf("adding the marketplace: %v: %s", err, strings.TrimSpace(out))
	}
	return nil
}

// Runner runs a command and returns its combined output. Tests replace it.
type Runner func(name string, args ...string) (string, error)

// Exec runs real commands.
func Exec(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).CombinedOutput()
	return string(out), err
}

// Available reports whether the claude command is on the PATH.
func Available() bool {
	_, err := exec.LookPath("claude")
	return err == nil
}

// Installed reports whether the plugin is installed for the user.
func Installed(run Runner) (bool, error) {
	out, err := run("claude", "plugin", "list")
	if err != nil {
		return false, fmt.Errorf("claude plugin list: %v: %s", err, strings.TrimSpace(out))
	}
	return strings.Contains(out, Plugin), nil
}

// Update brings the plugin to the one this program carries, moving a marketplace that
// still follows GitHub to the program's folder. Claude Code uses it from the next
// session on.
func Update(run Runner) error {
	if err := addMarketplace(run); err != nil {
		return err
	}
	if out, err := run("claude", "plugin", "marketplace", "update", MarketplaceName); err != nil {
		return fmt.Errorf("updating the marketplace: %v: %s", err, strings.TrimSpace(out))
	}
	if out, err := run("claude", "plugin", "update", Plugin); err != nil {
		return fmt.Errorf("updating the plugin: %v: %s", err, strings.TrimSpace(out))
	}
	return nil
}

// Uninstall removes the plugin and its marketplace, and the files the program wrote for
// it. Something already gone is fine.
func Uninstall(run Runner) error {
	if out, err := run("claude", "plugin", "uninstall", Plugin); err != nil && !gone(out) {
		return fmt.Errorf("removing the plugin: %v: %s", err, strings.TrimSpace(out))
	}
	if out, err := run("claude", "plugin", "marketplace", "remove", MarketplaceName); err != nil && !gone(out) {
		return fmt.Errorf("removing the marketplace: %v: %s", err, strings.TrimSpace(out))
	}
	if cfg, err := config.Dir(); err == nil {
		return RemoveLocal(LocalDir(cfg))
	}
	return nil
}

// gone tells, from what claude printed, that there was nothing to remove ("Plugin …
// not found in installed plugins", "Marketplace '…' not found").
func gone(out string) bool {
	return strings.Contains(out, "not found") || strings.Contains(out, "not installed")
}

// Install adds the marketplace and installs the plugin for the user. Both steps are
// safe to repeat.
func Install(run Runner) error {
	if err := addMarketplace(run); err != nil {
		return err
	}
	if out, err := run("claude", "plugin", "install", Plugin, "--scope", "user"); err != nil &&
		!strings.Contains(out, "already") {
		return fmt.Errorf("installing the plugin: %v: %s", err, strings.TrimSpace(out))
	}
	return nil
}
