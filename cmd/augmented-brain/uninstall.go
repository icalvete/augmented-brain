package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/icalvete/augmented-brain/internal/claudecode"
	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/paths"
)

// cmdUninstall takes augmented brain away: the Claude Code plugin, what Claude Code
// keeps of it (its cache, its data, the MCP server's logs) and the program. The notes
// stay, and the configuration too unless --config. --all starts from scratch: it also
// removes the configuration, the profile and every space's notes, and a space's folder
// when nothing of the user's is left in it. Backups in other repositories, Claude Code,
// its desktop app, git and the PATH line (Claude Code lives in the same folder) are left
// alone.
func cmdUninstall(args []string, in io.Reader, out, errOut io.Writer) int {
	flags := flag.NewFlagSet("uninstall", flag.ContinueOnError)
	flags.SetOutput(errOut)
	yes := flags.Bool("yes", false, "do not ask")
	withConfig := flags.Bool("config", false, "also remove the configuration and the profile")
	all := flags.Bool("all", false, "start from scratch: also the configuration, the profile and the notes of every space")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *all {
		*withConfig = true
	}
	lang := i18n.FromEnv()
	cfgDir, _ := config.Dir()
	type space struct{ memory, work, backup string }
	var spaces []space
	if c, err := config.Load(cfgDir); err == nil {
		lang = c.Language
		for _, n := range c.SpaceNames() {
			s := c.Spaces[n]
			sp := space{memory: s.Memory, backup: s.Backup}
			if len(s.WorkFolders) > 0 {
				sp.work = s.WorkFolders[0]
			}
			spaces = append(spaces, sp)
		}
	}
	self, _ := os.Executable()

	fmt.Fprintln(out, i18n.T(lang, "uninstall.plan"))
	fmt.Fprintln(out, "  - "+i18n.T(lang, "uninstall.plugin"))
	fmt.Fprintln(out, "  - "+i18n.T(lang, "uninstall.leftovers"))
	if self != "" {
		fmt.Fprintln(out, "  - "+paths.Tilde(self))
	}
	if *withConfig {
		fmt.Fprintln(out, "  - "+paths.Tilde(cfgDir))
	}
	var notes, backups []string
	for _, s := range spaces {
		if *all {
			fmt.Fprintln(out, "  - "+i18n.T(lang, "uninstall.notes", paths.Tilde(s.memory), countNotes(s.memory)))
		} else {
			notes = append(notes, paths.Tilde(s.memory))
		}
		if s.backup != "" {
			backups = append(backups, s.backup)
		}
	}
	if len(notes) > 0 {
		fmt.Fprintln(out, i18n.T(lang, "uninstall.keep", strings.Join(notes, ", ")))
	} else {
		fmt.Fprintln(out, i18n.T(lang, "uninstall.keeptools"))
	}
	if *all && len(backups) > 0 {
		fmt.Fprintln(out, i18n.T(lang, "uninstall.backups", strings.Join(backups, ", ")))
	}
	if !*yes {
		fmt.Fprint(out, i18n.T(lang, "uninstall.ask")+" ")
		answer, _ := bufio.NewReader(in).ReadString('\n')
		if a := strings.ToLower(strings.TrimSpace(answer)); !strings.HasPrefix(a, "y") && !strings.HasPrefix(a, "s") {
			fmt.Fprintln(out, i18n.T(lang, "uninstall.nothing"))
			return 1
		}
	}

	code := 0
	fail := func(err error) {
		if err != nil {
			fmt.Fprintln(errOut, err)
			code = 1
		}
	}
	if claudecode.Available() {
		fail(claudecode.Uninstall(claudecode.Exec))
	}
	fail(claudecode.RemoveLeftovers())
	if *withConfig && cfgDir != "" {
		fail(os.RemoveAll(cfgDir))
	}
	if *all {
		for _, s := range spaces {
			fail(os.RemoveAll(s.memory))
			// The space's folder, and the folder that holds the spaces, only when empty:
			// anything of the user's in them stays.
			if s.work != "" {
				os.Remove(s.work)
				os.Remove(filepath.Dir(s.work))
			}
		}
	}
	// Last, the program itself, if it has its own name (not a test binary or go run).
	if b := filepath.Base(self); self != "" && (b == "augmented-brain" || b == "augmented-brain.exe") {
		fail(removeSelf(self))
	}
	if code == 0 {
		if *all {
			fmt.Fprintln(out, i18n.T(lang, "uninstall.doneall"))
		} else {
			fmt.Fprintln(out, i18n.T(lang, "uninstall.done"))
		}
	}
	return code
}

// countNotes counts the Markdown files under a notes folder.
func countNotes(dir string) int {
	n := 0
	filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(d.Name(), ".md") {
			n++
		}
		return nil
	})
	return n
}
