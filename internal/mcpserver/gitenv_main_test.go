package mcpserver

import (
	"os"
	"testing"

	"github.com/icalvete/augmented-brain/internal/testenv"
)

// The tests here run git: see testenv.CleanGit.
func TestMain(m *testing.M) {
	testenv.CleanGit()
	os.Exit(m.Run())
}
