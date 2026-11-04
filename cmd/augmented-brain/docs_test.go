package main

import (
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/doccheck"
	"github.com/icalvete/augmented-brain/internal/i18n"
)

// usageCommands are the commands the usage text lists in lang, each with the flags its
// line names: "  augmented-brain backup setup URL   connect…" is backup setup.
func usageCommands(lang string) map[string][]string {
	out := map[string][]string{}
	flagRE := regexp.MustCompile(`--[a-z][a-z-]*`)
	for _, line := range strings.Split(i18n.T(lang, "usage"), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "augmented-brain ") {
			continue
		}
		head, desc, _ := strings.Cut(line, "  ")
		var words []string
		for _, w := range strings.Fields(head)[1:] {
			if w != strings.ToLower(w) {
				break // an argument: DIR, URL, EVENT
			}
			words = append(words, w)
		}
		name := strings.Join(words, " ")
		for _, f := range flagRE.FindAllString(desc, -1) {
			out[name] = append(out[name], strings.TrimPrefix(f, "--"))
		}
		if out[name] == nil {
			out[name] = []string{}
		}
	}
	return out
}

// The usage text and the list of commands say the same: every command listed (but the
// hidden ones), and each flag the usage names is one the command reads.
func TestUsageListsEveryCommand(t *testing.T) {
	for _, lang := range []string{"en", "es"} {
		usage := usageCommands(lang)
		for _, c := range commands {
			flags, listed := usage[c.name]
			if c.hidden {
				if listed {
					t.Errorf("%s: %q is in the usage text but marked hidden", lang, c.name)
				}
				continue
			}
			if !listed && strings.Contains(c.name, " ") && strings.Contains(i18n.T(lang, "usage"), "("+c.name) {
				continue // named on its parent's line: (panel stop: stop it)
			}
			if !listed {
				t.Errorf("%s: the usage text does not list %q", lang, c.name)
				continue
			}
			var known []string
			if c.flags != nil {
				known = c.flags()
			}
			for _, f := range flags {
				if !slices.Contains(known, f) {
					t.Errorf("%s: the usage text gives %q the flag --%s, which it does not read", lang, c.name, f)
				}
			}
		}
		for name := range usage {
			if !slices.ContainsFunc(commands, func(c command) bool { return c.name == name }) {
				t.Errorf("%s: the usage text lists %q, which is not in commands", lang, name)
			}
		}
	}
	// The hook's events are the ones the usage names.
	for _, e := range hookEvents {
		if !strings.Contains(i18n.T("en", "usage"), e.name) || !strings.Contains(i18n.T("es", "usage"), e.name) {
			t.Errorf("the usage text does not name the hook event %s", e.name)
		}
	}
}

// The reference page of the commands, in each language, has a section per command
// (`## augmented-brain <command>`) that gives each of its flags as `--flag` and, for
// hook, each event. It fails until the page covers the program: docs cannot drift.
// AB_DOCS_DIR points it at pages elsewhere.
func TestCommandsDocumented(t *testing.T) {
	var order []string
	want := map[string][]string{}
	for _, c := range commands {
		h := "augmented-brain " + c.name
		order = append(order, h)
		want[h] = []string{}
		if c.flags != nil {
			for _, f := range c.flags() {
				want[h] = append(want[h], "--"+f)
			}
		}
		if c.words != nil {
			want[h] = append(want[h], c.words()...)
		}
	}
	dir, err := doccheck.Dir()
	if err != nil {
		t.Fatal(err)
	}
	ours := func(h string) bool { return strings.HasPrefix(h, "augmented-brain ") }
	for _, f := range doccheck.Files("commands") {
		p, err := doccheck.Read(dir, f)
		if err != nil {
			t.Errorf("%s: %v", f, err)
			continue
		}
		for _, g := range doccheck.Check(p, order, want, ours, "command") {
			t.Error(g)
		}
	}
}
