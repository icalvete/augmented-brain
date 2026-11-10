package main

import (
	"os"
	"testing"

	"github.com/icalvete/augmented-brain/internal/testenv"
)

// Some tests make throwaway git repositories: without this, run from git's pre-push
// hook they would make them inside the real one.
func TestMain(m *testing.M) {
	testenv.CleanGit()
	os.Exit(m.Run())
}
