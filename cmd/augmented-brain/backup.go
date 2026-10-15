package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/icalvete/augmented-brain/internal/backup"
	"github.com/icalvete/augmented-brain/internal/config"
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
		fmt.Fprintf(out, "Space %s will back up to %s. Run `augmented-brain backup` to make the first copy.\n", s.Name, args[1])
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
			fmt.Fprintf(errOut, "✗ %s: %v\n", s.Name, err)
			code = 1
		case res.Committed:
			fmt.Fprintf(out, "✓ %s: changes saved and pushed to %s\n", s.Name, s.Backup)
		default:
			fmt.Fprintf(out, "✓ %s: nothing new; %s is up to date\n", s.Name, s.Backup)
		}
	}
	return code
}

// cmdRestore brings a space back from its backup on a new machine.
func cmdRestore(args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("restore", flag.ContinueOnError)
	fs.SetOutput(errOut)
	o := backup.RestoreOptions{}
	fs.StringVar(&o.Memory, "memory", "", "where to put the space's notes (default ~/augmented-brain/<space>)")
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
	fmt.Fprintf(out, "Space %s restored in %s.\nWork folder: %s\nNext: run `augmented-brain doctor`.\n",
		s.Name, s.Memory, strings.Join(s.WorkFolders, ", "))
	return 0
}
