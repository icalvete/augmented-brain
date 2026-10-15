package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/icalvete/augmented-brain/internal/backup"
	"github.com/icalvete/augmented-brain/internal/claudecode"
	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/paths"
)

// cmdBackup: `backup` saves the current space (or every space with --all);
// `backup setup <url>` connects the current space to a private repository.
func cmdBackup(args []string, out, errOut io.Writer) int {
	if len(args) > 0 && args[0] == "setup" {
		if len(args) != 2 {
			fmt.Fprintln(errOut, "usage: augmented-brain backup setup <url of a private git repository>")
			return 2
		}
		c, s, ok := currentSpace(errOut)
		if !ok {
			return 1
		}
		if err := backup.Setup(c, s, args[1]); err != nil {
			fmt.Fprintln(errOut, err)
			return 1
		}
		fmt.Fprintln(out, i18n.T(c.Language, "backup.setup", s.Name, args[1]))
		return 0
	}

	fs := flag.NewFlagSet("backup", flag.ContinueOnError)
	fs.SetOutput(errOut)
	all := fs.Bool("all", false, "back up every space, not only this folder's")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	var c *config.Config
	var spaces []*config.Space
	if *all {
		dir, err := config.Dir()
		if err == nil {
			c, err = config.Load(dir)
		}
		if err != nil {
			fmt.Fprintln(errOut, err)
			return 1
		}
		for _, n := range c.SpaceNames() {
			spaces = append(spaces, c.Spaces[n])
		}
	} else {
		cc, s, ok := currentSpace(errOut)
		if !ok {
			return 1
		}
		c, spaces = cc, []*config.Space{s}
	}
	host, _ := os.Hostname()
	code := 0
	for _, s := range spaces {
		res, err := backup.Run(c, s, host)
		switch {
		case err != nil:
			fmt.Fprintln(errOut, i18n.T(c.Language, "backup.failed", s.Name, err))
			code = 1
		case res.Committed:
			fmt.Fprintln(out, i18n.T(c.Language, "backup.saved", s.Name, s.Backup))
		default:
			fmt.Fprintln(out, i18n.T(c.Language, "backup.same", s.Name, s.Backup))
		}
	}
	return code
}

// cmdRestore brings a space back from its backup on a new machine.
func cmdRestore(args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("restore", flag.ContinueOnError)
	fs.SetOutput(errOut)
	o := backup.RestoreOptions{}
	fs.StringVar(&o.Memory, "memory", "", "where to put the space's notes (default: where they were, inside the work folder)")
	fs.StringVar(&o.WorkFolder, "work", "", "work folder that selects the space (default: the one saved in the backup)")
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		o.URL, args = args[0], args[1:]
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if o.URL == "" {
		fmt.Fprintln(errOut, "usage: augmented-brain restore <url of the space's backup> [--memory dir] [--work dir]")
		return 2
	}
	dir, err := config.Dir()
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	s, err := backup.Restore(dir, o)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	lang := i18n.FromEnv()
	if c, err := config.Load(dir); err == nil {
		lang = c.Language
	}
	fmt.Fprintln(out, i18n.T(lang, "restore.done", s.Name, paths.Tilde(s.Memory)))
	// A new machine has no plugin yet: install it, as init does.
	if ok, err := claudecode.Installed(claudecode.Exec); err == nil && !ok {
		installPlugin(&asker{out: out}, lang, out, errOut)
	}
	if len(s.WorkFolders) > 0 {
		fmt.Fprintln(out, i18n.T(lang, "restore.next", paths.Tilde(s.WorkFolders[0])))
	}
	return 0
}
