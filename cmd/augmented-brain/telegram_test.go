package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/telegramprog"
)

// Without the Telegram program, status tells a session saved in the keychain (connected,
// only the program is missing) from no session at all; it never says "not connected" to
// someone who is.
func TestTelegramStatusWithoutTheProgram(t *testing.T) {
	if !telegramprog.Supported() {
		t.Skip("Telegram is not supported here")
	}
	t.Setenv("AB_CONFIG_DIR", t.TempDir()) // no configuration: the language comes from AB_LANGUAGE
	t.Setenv("AB_LANGUAGE", "en")
	oldPath, oldSaved := telegramPath, telegramprog.SessionSaved
	t.Cleanup(func() { telegramPath, telegramprog.SessionSaved = oldPath, oldSaved })
	telegramPath = func() string { return "" }
	for _, saved := range []bool{true, false} {
		telegramprog.SessionSaved = func() bool { return saved }
		var out, errOut bytes.Buffer
		if code := cmdTelegram([]string{"status"}, strings.NewReader(""), &out, &errOut); code != 0 {
			t.Fatalf("saved %v: exit %d, %s", saved, code, errOut.String())
		}
		got := out.String()
		if saved && (!strings.Contains(got, "session is saved") || !strings.Contains(got, "telegram connect")) {
			t.Errorf("saved session: %q", got)
		}
		if !saved && !strings.Contains(got, "not connected") {
			t.Errorf("no session: %q", got)
		}
	}
}
