package mcpserver

import (
	"os"
	"testing"

	"github.com/icalvete/augmented-brain/internal/testenv"
)

// The tests here run git: see testenv.CleanGit.
func TestMain(m *testing.M) {
	testenv.CleanGit()
	setTimer = func(string) error { return nil } // never the real systemd or launchd
	os.Exit(m.Run())
}
