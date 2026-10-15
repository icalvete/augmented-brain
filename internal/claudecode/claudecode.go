// Package claudecode installs and checks the augmented brain plugin in Claude Code,
// through Claude Code's own command line, so its settings are only changed the way
// Claude Code itself changes them.
package claudecode

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Plugin is the plugin id in Claude Code: plugin name @ marketplace name.
const Plugin = "brain@augmented-brain"

// Marketplace returns where the marketplace comes from: the public repository, or
// $AB_MARKETPLACE (a path or another repository) to test unpublished changes.
func Marketplace() string {
	if m := os.Getenv("AB_MARKETPLACE"); m != "" {
		return m
	}
	return "icalvete/augmented-brain"
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

// Install adds the marketplace and installs the plugin for the user. Both steps are
// safe to repeat.
func Install(run Runner) error {
	if out, err := run("claude", "plugin", "marketplace", "add", Marketplace()); err != nil &&
		!strings.Contains(out, "already") {
		return fmt.Errorf("adding the marketplace: %v: %s", err, strings.TrimSpace(out))
	}
	if out, err := run("claude", "plugin", "install", Plugin, "--scope", "user"); err != nil &&
		!strings.Contains(out, "already") {
		return fmt.Errorf("installing the plugin: %v: %s", err, strings.TrimSpace(out))
	}
	return nil
}
