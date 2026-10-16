package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/icalvete/augmented-brain/internal/claudecode"
	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/paths"
)

// cmdUninstall takes augmented brain away: the Claude Code plugin and the program. The
// notes always stay, and the configuration too unless --config; Claude Code, its desktop
// app, git and the PATH line (Claude Code lives in the same folder) are left alone.
func cmdUninstall(args []string, in io.Reader, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("uninstall", flag.ContinueOnError)
	fs.SetOutput(errOut)
	yes := fs.Bool("yes", false, "do not ask")
	withConfig := fs.Bool("config", false, "also remove the configuration and the profile")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	lang := i18n.FromEnv()
	cfgDir, _ := config.Dir()
	var notes []string
	if c, err := config.Load(cfgDir); err == nil {
		lang = c.Language
		for _, n := range c.SpaceNames() {
			notes = append(notes, paths.Tilde(c.Spaces[n].Memory))
		}
	}
	self, _ := os.Executable()

	fmt.Fprintln(out, i18n.T(lang, "uninstall.plan"))
	fmt.Fprintln(out, "  - "+i18n.T(lang, "uninstall.plugin"))
	if self != "" {
		fmt.Fprintln(out, "  - "+paths.Tilde(self))
	}
	if *withConfig {
		fmt.Fprintln(out, "  - "+paths.Tilde(cfgDir))
	}
	if len(notes) > 0 {
		fmt.Fprintln(out, i18n.T(lang, "uninstall.keep", strings.Join(notes, ", ")))
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
	if claudecode.Available() {
		if err := claudecode.Uninstall(claudecode.Exec); err != nil {
			fmt.Fprintln(errOut, err)
			code = 1
		}
	}
	if *withConfig && cfgDir != "" {
		if err := os.RemoveAll(cfgDir); err != nil {
			fmt.Fprintln(errOut, err)
			code = 1
		}
	}
	// Last, the program itself: Unix lets a running program remove its own file.
	if self != "" && filepath.Base(self) == "augmented-brain" {
		if err := os.Remove(self); err != nil {
			fmt.Fprintln(errOut, err)
			code = 1
		}
	}
	if code == 0 {
		fmt.Fprintln(out, i18n.T(lang, "uninstall.done"))
	}
	return code
}
