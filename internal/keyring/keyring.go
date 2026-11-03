// Package keyring says whether a secret is in the system keychain of this computer,
// never what it holds. It asks the keychain's own command: secret-tool on Linux (the
// Secret Service, attributes service and name), security on macOS (a generic password,
// service and account). Whatever the command prints is thrown away unread.
package keyring

import (
	"context"
	"io"
	"os/exec"
	"runtime"
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
	cmd.Stdout, cmd.Stderr = io.Discard, io.Discard
	err := cmd.Run()
	if ctx.Err() != nil {
		return Unknown
	}
	if err == nil {
		return Present
	}
	if _, ok := err.(*exec.ExitError); ok {
		return Missing
	}
	return Unknown
}
