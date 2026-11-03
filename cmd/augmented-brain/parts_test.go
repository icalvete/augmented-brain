package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/session"
)

// The plugin runs one SessionStart hook per part of the session text: as many as the
// program has, 1 to session.Parts. Found in the review of 2026-11-03: they had to be
// kept in step by hand.
func TestPluginParts(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "plugin", "hooks", "hooks.json"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if n := strings.Count(text, "hook session-start "); n != session.Parts {
		t.Errorf("the plugin runs %d session-start parts, the program has %d", n, session.Parts)
	}
	for n := 1; n <= session.Parts; n++ {
		if !strings.Contains(text, fmt.Sprintf("hook session-start %d\"", n)) {
			t.Errorf("the plugin does not ask for part %d", n)
		}
	}
}

// A plugin newer than the program may ask for a part the program does not have: it
// gets an empty part and the session starts, never an error.
func TestPartBeyondTheProgram(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run([]string{"hook", "session-start", fmt.Sprint(session.Parts + 3)}, strings.NewReader("{}"), &out, &errOut); code != 0 {
		t.Errorf("exit %d: %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), `"hookEventName":"SessionStart"`) {
		t.Errorf("output: %s", out.String())
	}
}
