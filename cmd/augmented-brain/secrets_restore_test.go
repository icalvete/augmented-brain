package main

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/brain"
)

// After a restore, the secrets the backup names: those the keychain lacks, with the
// command to store each; and, when the keychain cannot be asked (no D-Bus session, a
// locked keychain), those it could not check, instead of calling them missing or saying
// nothing.
func TestRestoreNamesTheSecrets(t *testing.T) {
	var keychain, missing, broken string
	switch runtime.GOOS {
	case "linux":
		keychain, missing, broken = "secret-tool", "exit 1", "echo 'secret-tool: Cannot autolaunch D-Bus' >&2; exit 1"
	case "darwin":
		keychain, missing, broken = "security", "exit 44", "echo 'User interaction is not allowed.' >&2; exit 36"
	default:
		t.Skip("a keychain command to fake: Linux and macOS")
	}
	dir := t.TempDir()
	if err := brain.SaveSecretList(dir, []string{"mail/smtp: sending mail"}); err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	fake := func(body string) {
		if err := os.WriteFile(filepath.Join(bin, keychain), []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	var out bytes.Buffer
	fake(missing)
	missingSecrets("en", dir, &out)
	if !strings.Contains(out.String(), "keychain does not have") || !strings.Contains(out.String(), "mail/smtp") {
		t.Errorf("missing:\n%s", out.String())
	}

	out.Reset()
	fake(broken)
	missingSecrets("en", dir, &out)
	if strings.Contains(out.String(), "does not have") || !strings.Contains(out.String(), "could not be checked") || !strings.Contains(out.String(), "mail/smtp") {
		t.Errorf("a keychain that cannot be asked:\n%s", out.String())
	}

	out.Reset()
	fake("exit 0")
	missingSecrets("en", dir, &out)
	if out.Len() != 0 {
		t.Errorf("all there, yet:\n%s", out.String())
	}
}
