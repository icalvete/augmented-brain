package main

import (
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"regexp"

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

// historyDir is the configuration folder and its language for `config history` and
// `config restore`. They work on the files as they are, so they do not need a
// configuration that loads: a broken config.toml is just when going back is needed. The
// language then comes from the file's own `language` line, or the system's.
func historyDir(errOut io.Writer) (string, string, bool) {
	dir, err := config.Dir()
	if err != nil {
		fmt.Fprintln(errOut, err)
		return "", "", false
	}
	if c, err := config.Load(dir); err == nil {
		return dir, c.Language, true
	} else if !errors.Is(err, config.ErrNotFound) {
		return dir, rawLanguage(dir), true
	}
	if vs, _ := confighistory.List(dir); len(vs) > 0 {
		return dir, rawLanguage(dir), true // config.toml gone, its versions kept
	}
	fmt.Fprintln(errOut, i18n.T(i18n.FromEnv(), "config.missing", filepath.Join(dir, config.FileName)))
	return "", "", false
}

// rawLanguage reads `language = "xx"` from a config.toml that may not load.
func rawLanguage(dir string) string {
	b, _ := os.ReadFile(filepath.Join(dir, config.FileName))
	if m := languageLine.FindSubmatch(b); m != nil {
		return string(m[1])
	}
	return i18n.FromEnv()
}

var languageLine = regexp.MustCompile(`(?m)^\s*language\s*=\s*"(en|es)"`)

// cmdConfigHistory lists the versions of the configuration, or, given a number, says
// what that version changed from the one before it.
func cmdConfigHistory(args []string, out, errOut io.Writer) int {
	if len(args) > 1 {
		fmt.Fprintln(errOut, i18n.T(userLanguage(), "usage.config"))
		return 2
	}
	dir, l, ok := historyDir(errOut)
	if !ok {
		return 1
	}
	vs, err := confighistory.List(dir)
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
	changed, _ := confighistory.Changed(dir)
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
	dir, l, ok := historyDir(errOut)
	if !ok {
		return 1
	}
	vs, err := confighistory.List(dir)
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
	cur, err := confighistory.Snapshot(dir)
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
	before := ""
	if c, err := config.Load(dir); err == nil {
		before = guardPrint(c)
	}
	if err := confighistory.Restore(dir, v.ID, func() error { return confighistory.Check(dir) }); err != nil {
		fmt.Fprintln(errOut, i18n.T(l, "history.badresult", err))
		return 1
	}
	fmt.Fprintln(out, "\n"+i18n.T(l, "history.restored"))
	if c2, err := config.Load(dir); err == nil {
		if seal, now := state.ReadSeal(dir), guardPrint(c2); seal != "" && now != before && now != seal {
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
