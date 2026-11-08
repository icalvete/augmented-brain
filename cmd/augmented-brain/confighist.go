package main

import (
	"fmt"
	"io"
	"maps"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/confighistory"
	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/state"
)

// cmdConfig answers a bare `augmented-brain config`: what it takes.
func cmdConfig(_ []string, errOut io.Writer) int {
	fmt.Fprintln(errOut, i18n.T(userLanguage(), "usage.config"))
	return 2
}

// cmdConfigHistory lists the versions of the configuration, or, given a number, says
// what that version changed from the one before it.
func cmdConfigHistory(args []string, out, errOut io.Writer) int {
	if len(args) > 1 {
		fmt.Fprintln(errOut, i18n.T(userLanguage(), "usage.config"))
		return 2
	}
	c, ok := loadConfig(errOut)
	if !ok {
		return 1
	}
	l := c.Language
	vs, err := confighistory.List(c.Dir)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	if len(vs) == 0 {
		fmt.Fprintln(out, i18n.T(l, "history.none"))
		return 0
	}
	if len(args) == 1 {
		i, ok := confighistory.Index(vs, args[0])
		if !ok {
			fmt.Fprintln(errOut, i18n.T(l, "history.badnumber", args[0]))
			return 1
		}
		v := vs[i]
		when := v.Time.Local().Format("2006-01-02 15:04")
		if i == len(vs)-1 {
			fmt.Fprintln(out, i18n.T(l, "history.first", i+1, when, confighistory.Who(l, v)))
			return 0
		}
		fmt.Fprintln(out, i18n.T(l, "history.what", i+1, when, confighistory.Who(l, v)))
		printDiff(out, l, confighistory.Compare(vs[i+1].Files, v.Files, 2))
		return 0
	}
	fmt.Fprintln(out, i18n.T(l, "history.head", confighistory.Keep))
	changed, _ := confighistory.Changed(c.Dir)
	for i, v := range vs {
		mark := ""
		if i == 0 && !changed {
			mark = "  · " + i18n.T(l, "history.current")
		}
		fmt.Fprintf(out, "  %2d  %s  %s%s\n", i+1, v.Time.Local().Format("2006-01-02 15:04"), confighistory.Who(l, v), mark)
	}
	if changed {
		fmt.Fprintln(out, "\n"+i18n.T(l, "history.changedsince"))
	}
	fmt.Fprintln(out, "\n"+i18n.T(l, "history.tip"))
	return 0
}

// cmdConfigRestore goes back to one version. It does not seal the guard's lists: a
// command can be run by the assistant, so a change of the lists stays one the guard
// asks about until the user accepts it (the panel's Guard page, or a new session).
func cmdConfigRestore(args []string, out, errOut io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(errOut, i18n.T(userLanguage(), "usage.config"))
		return 2
	}
	c, ok := loadConfig(errOut)
	if !ok {
		return 1
	}
	l := c.Language
	vs, err := confighistory.List(c.Dir)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	i, ok := confighistory.Index(vs, args[0])
	if !ok {
		fmt.Fprintln(errOut, i18n.T(l, "history.badnumber", args[0]))
		return 1
	}
	v := vs[i]
	cur, err := confighistory.Snapshot(c.Dir)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	if maps.Equal(cur, v.Files) {
		fmt.Fprintln(out, i18n.T(l, "history.same"))
		return 0
	}
	fmt.Fprintln(out, i18n.T(l, "history.restoring", i+1, v.Time.Local().Format("2006-01-02 15:04")))
	printDiff(out, l, confighistory.Compare(cur, v.Files, 2))
	before := guardPrint(c)
	if err := confighistory.Restore(c.Dir, v.ID, func() error { return confighistory.Check(c.Dir) }); err != nil {
		fmt.Fprintln(errOut, i18n.T(l, "history.badresult", err))
		return 1
	}
	fmt.Fprintln(out, "\n"+i18n.T(l, "history.restored"))
	if c2, err := config.Load(c.Dir); err == nil {
		if seal, now := state.ReadSeal(c.Dir), guardPrint(c2); seal != "" && now != before && now != seal {
			fmt.Fprintln(out, i18n.T(l, "history.guardasks"))
		}
	}
	return 0
}

func guardPrint(c *config.Config) string {
	g, err := config.LoadGuard(c.Dir)
	if err != nil {
		return ""
	}
	return c.GuardFingerprint(g)
}

// printDiff writes a comparison as a text diff, file by file.
func printDiff(out io.Writer, l string, ds []confighistory.FileDiff) {
	for _, d := range ds {
		head := "\n" + d.Name
		switch {
		case d.Gone:
			head += " " + i18n.T(l, "history.filegone")
		case d.New:
			head += " " + i18n.T(l, "history.filenew")
		}
		fmt.Fprintln(out, head)
		for _, line := range d.Lines {
			if line.Op == '~' {
				fmt.Fprintln(out, "  …")
				continue
			}
			fmt.Fprintf(out, "  %c %s\n", line.Op, line.Text)
		}
	}
}
