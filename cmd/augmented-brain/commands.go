package main

import (
	"flag"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/version"
)

// command is one command of the program, as a person types it after augmented-brain.
// The list is what the reference page (docs/guide/commands.md) must cover, what the
// usage text must list, and what run answers: there is no other list of commands.
type command struct {
	name string // the words after augmented-brain: "backup setup"
	// aliases are other names for it, as typed: --version, -v.
	aliases []string
	// run answers it, given the arguments after its name. nil: the command's first word
	// answers it (backup setup is backup's, with setup among its arguments).
	run func(args []string, in io.Reader, out, errOut io.Writer) int
	// hidden commands are not in the usage text: the program runs them, or they repeat
	// another. The reference documents them all the same.
	hidden bool
	// flags are the names of the flags it reads, without the dashes; nil: none.
	flags func() []string
	// words are other names its section must give: the events of hook.
	words func() []string
}

// commands are every command run answers. A new one goes here, in the usage text
// (internal/i18n/commands.go) and in the reference, in English and Spanish.
var commands = []command{
	{name: "init", run: cmdInit, flags: func() []string { fs, _, _, _, _, _ := initFlags(io.Discard); return flagNames(fs) }},
	{name: "space", run: func(_ []string, _ io.Reader, out, errOut io.Writer) int { return cmdSpace(out, errOut) }},
	{name: "space add", run: cmdSpaceAdd, flags: func() []string { fs, _, _ := spaceFlags("space add", io.Discard); return flagNames(fs) }},
	{name: "check", run: func(_ []string, _ io.Reader, out, errOut io.Writer) int { return cmdCheck(out, errOut) }},
	{name: "doctor", run: func(_ []string, _ io.Reader, out, errOut io.Writer) int { return cmdDoctor(out, errOut) }},
	{name: "import", run: withoutIn(cmdImport), flags: func() []string { fs, _, _, _ := importFlags(io.Discard); return flagNames(fs) }},
	{name: "backup", run: withoutIn(cmdBackup), flags: func() []string { fs, _, _, _ := backupFlags(io.Discard); return flagNames(fs) }},
	{name: "backup setup"},
	{name: "restore", run: withoutIn(cmdRestore), flags: func() []string { fs, _ := restoreFlags(io.Discard); return flagNames(fs) }},
	{name: "panel", run: withoutIn(cmdPanel), flags: func() []string { return []string{panelNoBrowser} }},
	{name: "panel stop"},
	{name: "panel serve", hidden: true}, // the detached server `panel` starts
	// remote is hidden: its three subcommands are the ones the usage text lists.
	{name: "remote", hidden: true, run: withoutIn(cmdRemote)},
	{name: "remote install"},
	{name: "remote uninstall"},
	{name: "remote status"},
	{name: "update", run: withoutIn(cmdUpdate), flags: func() []string { fs, _ := updateFlags(io.Discard); return flagNames(fs) }},
	{name: "uninstall", run: cmdUninstall, flags: func() []string { fs, _, _, _ := uninstallFlags(io.Discard); return flagNames(fs) }},
	{name: "mcp", run: func(_ []string, _ io.Reader, _, errOut io.Writer) int { return cmdMCP(errOut) }},
	{name: "hook", run: cmdHook, words: func() []string {
		var out []string
		for _, e := range hookEvents {
			out = append(out, e.name)
		}
		return out
	}},
	{name: "version", aliases: []string{"--version", "-v"}, run: func(_ []string, _ io.Reader, out, _ io.Writer) int {
		fmt.Fprintln(out, version.Version)
		return 0
	}},
	// It prints the usage text.
	{name: "help", aliases: []string{"--help", "-h"}, hidden: true, run: func(_ []string, _ io.Reader, out, _ io.Writer) int {
		fmt.Fprint(out, i18n.T(userLanguage(), "usage"))
		return 0
	}},
}

// withoutIn is a command that reads nothing from the standard input.
func withoutIn(f func(args []string, out, errOut io.Writer) int) func([]string, io.Reader, io.Writer, io.Writer) int {
	return func(args []string, _ io.Reader, out, errOut io.Writer) int { return f(args, out, errOut) }
}

// find is the command args name and the arguments after its name: the longest name
// with a run (space add before space), or an alias. ok is false for an unknown one.
func find(args []string) (c command, rest []string, ok bool) {
	for n := min(len(args), 2); n >= 1; n-- {
		for _, c := range commands {
			// Word by word: one argument "space add" is not the command space add.
			if c.run != nil && slices.Equal(strings.Split(c.name, " "), args[:n]) {
				return c, args[n:], true
			}
		}
	}
	for _, c := range commands {
		if slices.Contains(c.aliases, args[0]) {
			return c, args[1:], true
		}
	}
	return command{}, nil, false
}

// flagNames are the names of fs's flags, in order.
func flagNames(fs *flag.FlagSet) []string {
	var out []string
	fs.VisitAll(func(f *flag.Flag) { out = append(out, f.Name) })
	return out
}
