package main

import (
	"fmt"
	"io"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/icalvete/augmented-brain/internal/i18n"
)

// The help of one command: `augmented-brain help panel`, `augmented-brain panel --help`.
// Its text is the start of the command's section in the reference
// (docs/guide/commands.md and its Spanish twin), which scripts/build-guides.py turns into
// internal/i18n/cmdhelp.go: what the command does, its syntax, its options and an
// example. It needs no configuration, so it works before init too.

// referenceURL is the reference page, in each language.
const referenceURL = "https://icalvete.github.io/augmented-brain/guide/commands"

// helpWidth is how wide the help's lines may get.
const helpWidth = 80

// helpWords are the arguments that ask for a command's help instead of running it.
var helpWords = []string{"--help", "-help", "-h"}

// wantsHelp says whether args ask for help: one of helpWords before a lone "--", after
// which every argument is taken as it is.
func wantsHelp(args []string) bool {
	for _, a := range args {
		if a == "--" {
			return false
		}
		if slices.Contains(helpWords, a) {
			return true
		}
	}
	return false
}

// commandNamed is the command called name, with or without a run of its own (backup
// setup is answered by backup, but it has its help).
func commandNamed(name string) (command, bool) {
	for _, c := range commands {
		if c.name == name {
			return c, true
		}
	}
	return command{}, false
}

// helpName is the command words name: its longest prefix that is a command, so `panel
// stop --help` is panel stop's help and not panel's. ok is false when the first word is
// no command at all.
func helpName(words []string) (string, bool) {
	for n := min(len(words), 2); n >= 1; n-- {
		name := strings.Join(words[:n], " ")
		if _, ok := commandNamed(name); ok {
			return name, true
		}
	}
	if len(words) > 0 {
		for _, c := range commands {
			if slices.Contains(c.aliases, words[0]) {
				return c.name, true
			}
		}
	}
	return "", false
}

// runHelp is help's run. It is set in init: help reads the list `commands`, which
// names it, and Go refuses a list that refers to itself while it is being made.
var runHelp func(args []string, out, errOut io.Writer) int

func init() { runHelp = printHelp }

// printHelp is `help`: the list of commands, or with a command its help.
func printHelp(args []string, out, errOut io.Writer) int {
	l := userLanguage()
	if len(args) == 0 {
		fmt.Fprint(out, i18n.T(l, "usage"))
		return 0
	}
	name, ok := helpName(args)
	if !ok {
		fmt.Fprintf(errOut, "%s\n\n%s", i18n.T(l, "usage.unknown", strings.Join(args, " ")), i18n.T(l, "usage"))
		return 2
	}
	fmt.Fprint(out, commandHelp(l, name))
	return 0
}

// commandHelp is the help of the command name, in lang.
func commandHelp(lang, name string) string {
	key := func(part string) string { return "cmdhelp." + name + "." + part }
	text := func(part string) string {
		if !i18n.Has(key(part)) {
			return ""
		}
		return i18n.T(lang, key(part))
	}
	var b strings.Builder
	fmt.Fprintf(&b, "augmented-brain %s\n\n", name)
	b.WriteString(wrap(text("what"), "", helpWidth))
	if s := text("syntax"); s != "" {
		fmt.Fprintf(&b, "\n%s\n", i18n.T(lang, "cmdhelp.syntax"))
		b.WriteString(syntaxLines(s))
	} else if subs := subcommands(name); len(subs) > 0 {
		// A command made of others (remote, session…): the syntax of each.
		fmt.Fprintf(&b, "\n%s\n", i18n.T(lang, "cmdhelp.commands"))
		for _, sub := range subs {
			b.WriteString(syntaxLines(i18n.T(lang, "cmdhelp."+sub+".syntax")))
		}
	}
	if o := text("options"); o != "" {
		fmt.Fprintf(&b, "\n%s\n", i18n.T(lang, "cmdhelp.options"))
		b.WriteString(options(o))
	}
	if e := text("example"); e != "" {
		fmt.Fprintf(&b, "\n%s\n", i18n.T(lang, "cmdhelp.example"))
		for _, line := range strings.Split(e, "\n") {
			b.WriteString("  " + line + "\n")
		}
	}
	page := referenceURL + ".html"
	if i18n.Known(lang) == "es" {
		page = referenceURL + ".es.html"
	}
	fmt.Fprintf(&b, "\n%s\n", i18n.T(lang, "cmdhelp.more", page+"#"+anchor("augmented-brain "+name)))
	return b.String()
}

// syntaxLines lays out a syntax, a line per form; a form too long for one line goes on
// with a deeper indent, so it is not taken for another form.
func syntaxLines(s string) string {
	var b strings.Builder
	for _, form := range strings.Split(s, "\n") {
		b.WriteString(strings.TrimPrefix(wrap(form, "    ", helpWidth), "  "))
	}
	return b.String()
}

// subcommands are the commands under name that people run (the usage text's), in order.
func subcommands(name string) []string {
	var out []string
	for _, c := range commands {
		if strings.HasPrefix(c.name, name+" ") && !c.hidden && i18n.Has("cmdhelp."+c.name+".syntax") {
			out = append(out, c.name)
		}
	}
	return out
}

// options lays out the options, a line per option with its name and its text split by
// a tab, as two columns: the names, and each text wrapped beside its name. A name too
// long for the column has its text on the next line.
func options(text string) string {
	const most = 16
	lines := strings.Split(text, "\n")
	col := 0
	for _, l := range lines {
		name, _, _ := strings.Cut(l, "\t")
		if n := utf8.RuneCountInString(name); n <= most && n > col {
			col = n
		}
	}
	indent := strings.Repeat(" ", 2+col+2)
	var b strings.Builder
	for _, l := range lines {
		name, desc, _ := strings.Cut(l, "\t")
		w := wrap(desc, indent, helpWidth)
		if utf8.RuneCountInString(name) > col {
			b.WriteString("  " + name + "\n" + w)
			continue
		}
		pad := "  " + name + strings.Repeat(" ", col-utf8.RuneCountInString(name)+2)
		b.WriteString(pad + strings.TrimPrefix(w, indent))
	}
	return b.String()
}

// wrap breaks text into lines of at most width characters, each starting with indent,
// at spaces; a word longer than a line stays whole.
func wrap(text, indent string, width int) string {
	var b strings.Builder
	line := indent
	for _, w := range strings.Fields(text) {
		if line != indent && utf8.RuneCountInString(line)+1+utf8.RuneCountInString(w) > width {
			b.WriteString(line + "\n")
			line = indent
		}
		if line != indent {
			line += " "
		}
		line += w
	}
	b.WriteString(line + "\n")
	return b.String()
}

// anchor is a heading's anchor on the website, as scripts/build-guides.py makes it:
// lowercase, spaces as dashes, other punctuation dropped.
func anchor(heading string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(heading) {
		switch {
		case r == ' ':
			b.WriteRune('-')
		case r == '-' || r == '_' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			b.WriteRune(r)
		}
	}
	return b.String()
}
