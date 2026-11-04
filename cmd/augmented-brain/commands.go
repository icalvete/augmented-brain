package main

import (
	"flag"
	"io"
)

// command is one command of the program, as a person types it after augmented-brain.
// The list is what the reference page (docs/guide/commands.md) must cover, and what
// the usage text must list; run is where each one is answered.
type command struct {
	name string // the words after augmented-brain: "backup setup"
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
// Not listed: --version, -v, --help and -h, the same as version and help.
var commands = []command{
	{name: "init", flags: func() []string { fs, _, _, _, _, _ := initFlags(io.Discard); return flagNames(fs) }},
	{name: "space"},
	{name: "space add", flags: func() []string { fs, _, _, _ := spaceFlags("space add", io.Discard); return flagNames(fs) }},
	{name: "check"},
	{name: "doctor"},
	{name: "import", flags: func() []string { fs, _, _, _ := importFlags(io.Discard); return flagNames(fs) }},
	{name: "backup", flags: func() []string { fs, _, _, _ := backupFlags(io.Discard); return flagNames(fs) }},
	{name: "backup setup"},
	{name: "restore", flags: func() []string { fs, _ := restoreFlags(io.Discard); return flagNames(fs) }},
	{name: "panel", flags: func() []string { return []string{panelNoBrowser} }},
	{name: "panel stop"},
	{name: "panel serve", hidden: true}, // the detached server `panel` starts
	{name: "update", flags: func() []string { fs, _ := updateFlags(io.Discard); return flagNames(fs) }},
	{name: "uninstall", flags: func() []string { fs, _, _, _ := uninstallFlags(io.Discard); return flagNames(fs) }},
	{name: "mcp"},
	{name: "hook", words: func() []string {
		var out []string
		for _, e := range hookEvents {
			out = append(out, e.name)
		}
		return out
	}},
	{name: "version"},
	{name: "help", hidden: true}, // prints the usage text
}

// flagNames are the names of fs's flags, in order.
func flagNames(fs *flag.FlagSet) []string {
	var out []string
	fs.VisitAll(func(f *flag.Flag) { out = append(out, f.Name) })
	return out
}
