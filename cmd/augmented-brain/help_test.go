package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/i18n"
)

// Every command has its help, in both languages: what it does, and its syntax or the
// commands it is made of. Each flag it reads is among its options, so the help never
// leaves one out. The texts come from the reference (scripts/build-guides.py writes
// internal/i18n/cmdhelp.go); a command missing here is missing there.
func TestEveryCommandHasHelp(t *testing.T) {
	for _, c := range commands {
		for _, lang := range []string{"en", "es"} {
			if !i18n.Has("cmdhelp." + c.name + ".what") {
				t.Errorf("%s: no help: its section in docs/guide/commands.md is missing (then run scripts/build-guides.py)", c.name)
				continue
			}
			h := commandHelp(lang, c.name)
			if !strings.HasPrefix(h, "augmented-brain "+c.name+"\n\n") {
				t.Errorf("%s (%s): the help does not start with the command:\n%s", c.name, lang, h)
			}
			syntax := i18n.Has("cmdhelp." + c.name + ".syntax")
			if !syntax && len(subcommands(c.name)) == 0 {
				t.Errorf("%s: no syntax and no commands of its own: give its section a Syntax line", c.name)
			}
			if syntax && !strings.Contains(h, "  augmented-brain "+c.name) {
				t.Errorf("%s (%s): the syntax does not name the command:\n%s", c.name, lang, h)
			}
			if c.flags == nil {
				continue
			}
			for _, f := range c.flags() {
				if !strings.Contains(h, "\n  --"+f+" ") && !strings.Contains(h, "\n  --"+f+"\n") {
					t.Errorf("%s (%s): option --%s is not among its options: add it to its section of the reference", c.name, lang, f)
				}
			}
		}
	}
}

// `help <command>`, `<command> --help` and `-h` print the help and run nothing, with
// no configuration at all, in the user's language.
func TestHelpAsked(t *testing.T) {
	t.Setenv("AB_CONFIG_DIR", t.TempDir())
	t.Setenv("AB_LANGUAGE", "es")
	cases := []struct {
		args []string
		want string // the help of
	}{
		{[]string{"help", "panel"}, "panel"},
		{[]string{"panel", "--help"}, "panel"},
		{[]string{"panel", "stop", "--help"}, "panel stop"},
		{[]string{"help", "panel", "stop"}, "panel stop"},
		{[]string{"session", "start", "-h"}, "session start"},
		{[]string{"session", "start", "task-x", "--help"}, "session start"},
		{[]string{"space", "add", "-help"}, "space add"},
		{[]string{"backup", "setup", "--help"}, "backup setup"},
		{[]string{"init", "--yes", "--help"}, "init"},
		{[]string{"uninstall", "--all", "--yes", "-h"}, "uninstall"},
		{[]string{"hook", "stop", "--help"}, "hook"},
		{[]string{"help", "--version"}, "version"},
		{[]string{"help", "help"}, "help"},
		{[]string{"help", "--help"}, "help"},
	}
	for _, c := range cases {
		var out, errOut bytes.Buffer
		code := run(c.args, strings.NewReader(""), &out, &errOut)
		if code != 0 || errOut.Len() > 0 {
			t.Errorf("%v: code %d, %s", c.args, code, errOut.String())
			continue
		}
		if want := commandHelp("es", c.want); out.String() != want {
			t.Errorf("%v: printed\n%s\nwant the help of %s", c.args, out.String(), c.want)
		}
		if !strings.Contains(out.String(), "Sintaxis:") && !strings.Contains(out.String(), "Sus comandos") {
			t.Errorf("%v: not in Spanish:\n%s", c.args, out.String())
		}
	}

	// Alone, help is the list; an unknown command says so.
	var out, errOut bytes.Buffer
	if code := run([]string{"help"}, nil, &out, &errOut); code != 0 || out.String() != i18n.T("es", "usage") {
		t.Errorf("help: code %d, %q", code, out.String())
	}
	out.Reset()
	if code := run([]string{"help", "nope"}, nil, &out, &errOut); code != 2 || !strings.Contains(errOut.String(), `"nope"`) {
		t.Errorf("help nope: code %d, %q", code, errOut.String())
	}
}

// After a lone "--", --help is an argument like any other.
func TestWantsHelp(t *testing.T) {
	for args, want := range map[string]bool{
		"--help":         true,
		"x -h":           true,
		"x -help y":      true,
		"-- --help":      false,
		"x":              false,
		"--helpful":      false,
		"--folder -- -h": false,
	} {
		if got := wantsHelp(strings.Fields(args)); got != want {
			t.Errorf("wantsHelp(%q) = %v", args, got)
		}
	}
}

// Options sit in two columns, wrapped under their text; a long name takes its own line.
func TestHelpLayout(t *testing.T) {
	got := options("--a\tone two three\n--bb\t" + strings.Repeat("word ", 30) + "\n--a-very-long-name\tshort")
	for _, line := range strings.Split(strings.TrimSuffix(got, "\n"), "\n") {
		if len(line) > helpWidth {
			t.Errorf("line longer than %d: %q", helpWidth, line)
		}
	}
	if !strings.HasPrefix(got, "  --a   one two three\n  --bb  word") {
		t.Errorf("columns:\n%s", got)
	}
	if !strings.Contains(got, "\n        word") || !strings.Contains(got, "\n  --a-very-long-name\n        short\n") {
		t.Errorf("wrapping:\n%s", got)
	}
	if got := syntaxLines("augmented-brain watch pr <number> [--task <task>] [--repo <owner/name>] [--max <time>] [--every <time>]\nx y"); got != "  augmented-brain watch pr <number> [--task <task>] [--repo <owner/name>]\n    [--max <time>] [--every <time>]\n  x y\n" {
		t.Errorf("syntax:\n%s", got)
	}
	if a := anchor("augmented-brain session start"); a != "augmented-brain-session-start" {
		t.Errorf("anchor = %q", a)
	}
}
