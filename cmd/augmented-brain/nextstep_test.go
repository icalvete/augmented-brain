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
		if got := nextStep("en", "~/augmented-brain/work", false); !strings.Contains(got, "open a new one") {
			t.Errorf("%s with a new PATH: %q says nothing of a new terminal", env, got)
		}
		t.Setenv("AB_NEW_TERMINAL", "")
		if got := nextStep("es", "~/augmented-brain/trabajo", false); strings.Contains(got, "abre una nueva") {
			t.Errorf("%s with no PATH change: %q asks for a new terminal", env, got)
		}
	}
}

// Whatever the last message suggests, it says the first time Claude asks who you are:
// it is the one place the installer tells it (with the desktop app too).
func TestNextStepTellsTheInterview(t *testing.T) {
	want := map[string]string{"en": "who you are", "es": "quién eres"}
	cases := map[string][3]string{ // AB_DESKTOP, AB_LAUNCH, AB_NEW_TERMINAL
		"desktop": {"1", "", ""},
		"launch":  {"", "1", ""},
		"newterm": {"", "", "1"},
		"now":     {"", "", ""},
	}
	for name, env := range cases {
		t.Setenv("AB_DESKTOP", env[0])
		t.Setenv("AB_LAUNCH", env[1])
		t.Setenv("AB_NEW_TERMINAL", env[2])
		for lang, w := range want {
			for _, pick := range []bool{false, true} {
				if got := nextStep(lang, "~/augmented-brain/work", pick); !strings.Contains(got, w) {
					t.Errorf("%s (%s): %q says nothing of the first-session questions", name, lang, got)
				}
			}
		}
	}
}

// With a single space, or with reach spaces, no space acts in the desktop app's own
// folder: the message must not say that one works there without choosing a folder.
func TestNextStepDesktopPicksAFolder(t *testing.T) {
	t.Setenv("AB_DESKTOP", "1")
	t.Setenv("AB_LAUNCH", "")
	t.Setenv("AB_NEW_TERMINAL", "")
	for lang, without := range map[string]string{"en": "Without choosing a folder", "es": "Sin elegir carpeta"} {
		if got := nextStep(lang, "~/augmented-brain/work", true); strings.Contains(got, without) || !strings.Contains(got, "~/augmented-brain/work") {
			t.Errorf("%s with pick: %q", lang, got)
		}
		if got := nextStep(lang, "~/augmented-brain/work", false); !strings.Contains(got, without) {
			t.Errorf("%s without pick: %q", lang, got)
		}
	}
}
