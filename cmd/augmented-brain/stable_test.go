package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/stable"
)

// The commands people type are part of the compatibility promise: every command the
// usage text lists keeps its name, and every flag it reads; hook keeps its events, which
// the plugin calls by name. They are frozen in testdata/stable-commands.json. Hidden
// commands (panel serve, remote watch…) are the program's own and are left out.
func TestStableCommands(t *testing.T) {
	now := stable.Surface{}
	for _, c := range commands {
		if c.hidden {
			continue
		}
		flags := []string{}
		if c.flags != nil {
			flags = c.flags()
		}
		// The events of hook: a plugin of another version calls them by name.
		if c.words != nil {
			for _, w := range c.words() {
				flags = append(flags, "event "+w)
			}
		}
		now[c.name] = flags
		// A name kept for compatibility must still run, not only be listed.
		if _, _, ok := find(strings.Split(c.name, " ")); !ok {
			t.Errorf("%q is listed but nothing runs it", c.name)
		}
	}
	stable.Check(t, filepath.Join("testdata", "stable-commands.json"), now)
}
