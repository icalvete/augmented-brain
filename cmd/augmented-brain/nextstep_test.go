package main

import (
	"strings"
	"testing"
)

// The terminal that ran the installer does not find the program yet: whatever the last
// message suggests, it says so when the installer changed the PATH.
func TestNextStepSaysNewTerminal(t *testing.T) {
	for _, env := range []string{"AB_DESKTOP", "AB_LAUNCH"} {
		t.Setenv("AB_DESKTOP", "")
		t.Setenv("AB_LAUNCH", "")
		t.Setenv(env, "1")
		t.Setenv("AB_NEW_TERMINAL", "1")
		if got := nextStep("en", "~/augmented-brain/work"); !strings.Contains(got, "open a new one") {
			t.Errorf("%s with a new PATH: %q says nothing of a new terminal", env, got)
		}
		t.Setenv("AB_NEW_TERMINAL", "")
		if got := nextStep("es", "~/augmented-brain/trabajo"); strings.Contains(got, "abre una nueva") {
			t.Errorf("%s with no PATH change: %q asks for a new terminal", env, got)
		}
	}
}
