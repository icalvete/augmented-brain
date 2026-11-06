package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/icalvete/augmented-brain/internal/backup"
	"github.com/icalvete/augmented-brain/internal/brain"
	"github.com/icalvete/augmented-brain/internal/claudecode"
	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/keyring"
	"github.com/icalvete/augmented-brain/internal/paths"
)

// backupFlags are backup's flags: --all, --space and --auto (the automatic backup the
// session's end starts).
func backupFlags(errOut io.Writer) (fs *flag.FlagSet, all *bool, only *string, auto *bool) {
	fs = flag.NewFlagSet("backup", flag.ContinueOnError)
	fs.SetOutput(errOut)
	all = fs.Bool("all", false, "back up every space, not only this folder's")
	only = fs.String("space", "", "back up this space, wherever you are")
	auto = fs.Bool("auto", false, "started by the assistant when a session ends: quiet, never asks for a password")
	return fs, all, only, auto
}

// cmdBackup: `backup` saves the current space (or every space with --all);
// `backup setup <url>` connects the current space to a private repository.
func cmdBackup(args []string, out, errOut io.Writer) int {
	if len(args) > 0 && args[0] == "setup" {
		if len(args) != 2 {
			fmt.Fprintln(errOut, i18n.T(userLanguage(), "usage.backup.setup"))
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

	fs, all, only, auto := backupFlags(errOut)
	if _, ok := parseArgs(fs, args, 0, errOut); !ok {
		return 2
	}
	if *auto {
		// Nobody is there to type a password: git must fail instead of waiting.
		os.Setenv("GIT_TERMINAL_PROMPT", "0")
		if os.Getenv("GIT_SSH_COMMAND") == "" {
			os.Setenv("GIT_SSH_COMMAND", "ssh -o BatchMode=yes")
		}
		out, errOut = io.Discard, io.Discard
	}
	var c *config.Config
	var spaces []*config.Space
	if *all || *only != "" {
		var ok bool
		if c, ok = loadConfig(errOut); !ok {
			return 1
		}
		for _, n := range c.SpaceNames() {
			if *only == "" || n == *only {
				spaces = append(spaces, c.Spaces[n])
			}
		}
		if len(spaces) == 0 {
			fmt.Fprintln(errOut, i18n.T(c.Language, "space.unknown", *only))
			return 1
		}
	} else {
		cc, s, ok := currentSpace(errOut)
		if !ok {
			return 1
		}
		c, spaces = cc, []*config.Space{s}
	}
	return backupSpaces(c, spaces, out, errOut)
}

// backupSpaces backs up each space and says how it went; 1 when any failed. backup.Run
// takes the locks: backups run one at a time, and never during a write of the assistant.
func backupSpaces(c *config.Config, spaces []*config.Space, out, errOut io.Writer) int {
	host, _ := os.Hostname()
	code := 0
	for _, s := range spaces {
		res, err := backup.Run(c, s, host)
		backup.RecordStatusWarning(c.Dir, s.Name, err, res.Warning(c.Language), time.Now())
		if w := res.Warning(c.Language); w != "" && err == nil {
			fmt.Fprintln(errOut, i18n.T(c.Language, "backup.leftout", s.Name, strings.Join(res.LeftOut, ", "), backup.MaxFileSize>>20))
		}
		var se *backup.SecretsError
		var ce *backup.ConflictError
		switch {
		case errors.As(err, &ce):
			fmt.Fprintln(errOut, i18n.T(c.Language, "backup.conflict", s.Name, strings.Join(ce.Notes, ", "), ce.Dir))
			code = 1
		case errors.As(err, &se):
			var l []string
			for _, f := range se.Found {
				l = append(l, f.Path+" ("+i18n.T(c.Language, "secret."+f.Kind)+")")
			}
			fmt.Fprintln(errOut, i18n.T(c.Language, "backup.secrets", s.Name, strings.Join(l, ", ")))
			code = 1
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

// restoreFlags are restore's flags: --memory and --work.
func restoreFlags(errOut io.Writer) (*flag.FlagSet, *backup.RestoreOptions) {
	fs := flag.NewFlagSet("restore", flag.ContinueOnError)
	fs.SetOutput(errOut)
	o := &backup.RestoreOptions{}
	fs.StringVar(&o.Memory, "memory", "", "where to put the space's notes (default: where they were, inside the work folder)")
	fs.StringVar(&o.WorkFolder, "work", "", "work folder that selects the space (default: the one saved in the backup)")
	return fs, o
}

// cmdRestore brings a space back from its backup on a new machine.
func cmdRestore(args []string, out, errOut io.Writer) int {
	fs, o := restoreFlags(errOut)
	rest, ok := parseArgs(fs, args, 1, errOut)
	if !ok {
		return 2
	}
	if len(rest) > 0 {
		o.URL = rest[0]
	}
	if o.URL == "" {
		fmt.Fprintln(errOut, i18n.T(userLanguage(), "usage.restore"))
		return 2
	}
	dir, err := config.Dir()
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	s, err := backup.Restore(dir, *o)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	lang := i18n.FromEnv()
	if c, err := config.Load(dir); err == nil {
		lang = c.Language
	}
	fmt.Fprintln(out, i18n.T(lang, "restore.done", s.Name, paths.Tilde(s.Memory)))
	if _, err := os.Stat(filepath.Join(dir, config.AskSafetyFile)); err == nil {
		fmt.Fprintln(out, i18n.T(lang, "restore.nolists"))
	}
	if len(s.NotRestored) > 0 {
		fmt.Fprintln(out, i18n.T(lang, "restore.norepos", strings.Join(tildes(s.NotRestored), ", "), filepath.Join(dir, config.FileName)))
	}
	missingSecrets(lang, dir, out)
	// A new machine has no plugin yet: install it, as init does.
	if ok, err := claudecode.Installed(claudecode.Exec); err == nil && !ok {
		installPlugin(&asker{out: out}, lang, out, errOut)
	}
	if len(s.WorkFolders) > 0 {
		fmt.Fprintln(out, i18n.T(lang, "restore.next", paths.Tilde(s.WorkFolders[0])))
	}
	return 0
}

// missingSecrets names the secrets of the list that this machine's keychain does not
// have, and how to store each: the backup brings their names, never their values.
func missingSecrets(lang, dir string, out io.Writer) {
	names, _ := brain.SecretNames(dir)
	var missing []brain.SecretName
	for _, n := range names {
		if keyring.Has(n.Service, n.Name) == keyring.Missing {
			missing = append(missing, n)
		}
	}
	if len(missing) == 0 {
		return
	}
	fmt.Fprintln(out, i18n.T(lang, "restore.secrets", len(missing)))
	for _, n := range missing {
		line := "  " + n.Key()
		if n.What != "" {
			line += " · " + n.What
		}
		fmt.Fprintln(out, line)
		if how := keyring.HowToStore(n.Service, n.Name); how != "" {
			fmt.Fprintln(out, "    "+how)
		}
	}
}
