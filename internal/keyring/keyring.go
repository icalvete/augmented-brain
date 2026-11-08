// Package keyring says whether a secret is in the system keychain of this computer,
// never what it holds. It asks the keychain's own command: secret-tool on Linux (the
// Secret Service, attributes service and name), security on macOS (a generic password,
// service and account). Whatever the command prints is thrown away unread.
package keyring

import (
	"context"
	"errors"
	"io"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// State is what can be said of one secret here.
type State int

const (
	Unknown State = iota // no keychain command here, or it did not answer
	Present
	Missing
)

// Has asks whether the keychain holds service/name.
func Has(service, name string) State {
	var cmd *exec.Cmd
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	switch runtime.GOOS {
	case "linux", "freebsd", "openbsd":
		if _, err := exec.LookPath("secret-tool"); err != nil {
			return Unknown
		}
		cmd = exec.CommandContext(ctx, "secret-tool", "lookup", "service", service, "name", name)
	case "darwin":
		// Without -w, security prints the item's attributes, not the password.
		cmd = exec.CommandContext(ctx, "security", "find-generic-password", "-s", service, "-a", name)
	default:
		return Unknown
	}
	// The error output is read only to tell "not there" from "could not ask": it never
	// holds the secret (without -w, security prints no password; lookup's output, the
	// value, goes nowhere).
	var stderr strings.Builder
	cmd.Stdout, cmd.Stderr = io.Discard, &stderr
	err := cmd.Run()
	if ctx.Err() != nil {
		return Unknown
	}
	return classify(runtime.GOOS, err, stderr.String())
}

// notFoundMac is the exit code of `security` for an item that is not in the keychain
// (errSecItemNotFound). Others mean it could not look: a locked keychain, no access.
const notFoundMac = 44

// classify reads how the keychain's command ended. secret-tool ends with 1 both when
// the secret is not there (saying nothing) and when it cannot reach the keychain (no
// D-Bus session, no Secret Service: it says why), so only a silent 1 is "missing".
// Taking that error for "missing" told a person to store again a secret they had.
func classify(goos string, err error, stderr string) State {
	if err == nil {
		return Present
	}
	var exit *exec.ExitError
	if !errors.As(err, &exit) {
		return Unknown
	}
	switch goos {
	case "darwin":
		if exit.ExitCode() == notFoundMac {
			return Missing
		}
	default:
		if exit.ExitCode() == 1 && strings.TrimSpace(stderr) == "" {
			return Missing
		}
	}
	return Unknown
}

// HowToStore is the command that stores the secret service/name in this computer's
// keychain, asking for its value (never written in the command), or "" where it is not
// known.
func HowToStore(service, name string) string {
	switch runtime.GOOS {
	case "linux", "freebsd", "openbsd":
		return "secret-tool store --label=" + service + "/" + name + " service " + service + " name " + name
	case "darwin":
		return "security add-generic-password -s " + service + " -a " + name + " -w"
	}
	return ""
}
