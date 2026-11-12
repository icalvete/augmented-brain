package main

import (
	"bytes"
	"strings"
	"testing"
)

// session restart takes a session or --old, never both, and --when-idle only with a
// session: a wrong mix prints the usage and touches nothing.
func TestSessionRestartUsage(t *testing.T) {
	home(t)
	for _, args := range [][]string{
		{"restart"},
		{"restart", "--when-idle"},
		{"restart", "--old", "--when-idle"},
		{"restart", "task-x", "--old"},
		{"restart", "task-x", "task-y"},
	} {
		var out, errOut bytes.Buffer
		if code := cmdSession(args, &out, &errOut); code != 2 || !strings.Contains(errOut.String(), "restart --old") {
			t.Errorf("%v: %d\n%s%s", args, code, out.String(), errOut.String())
		}
	}
}
