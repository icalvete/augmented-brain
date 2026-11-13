package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/icalvete/augmented-brain/internal/claudecode"
	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/format"
	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/panel"
	"github.com/icalvete/augmented-brain/internal/paths"
	"github.com/icalvete/augmented-brain/internal/remote"
)

// uninstallFlags are uninstall's flags: --yes, --config and --all.
func uninstallFlags(errOut io.Writer) (fs *flag.FlagSet, yes, withConfig, all *bool) {
	fs = flag.NewFlagSet("uninstall", flag.ContinueOnError)
	fs.SetOutput(errOut)
	yes = fs.Bool("yes", false, "do not ask")
	withConfig = fs.Bool("config", false, "also remove the configuration and the profile")
	all = fs.Bool("all", false, "start from scratch: also the configuration, the profile and the notes of every space")
	return fs, yes, withConfig, all
}

// cmdUninstall takes augmented brain away: the Claude Code plugin, what Claude Code
// keeps of it (its cache, its data, the MCP server's logs) and the program. The notes
// stay, and the configuration too unless --config. --all starts from scratch: it also
// removes the configuration, the profile and every space's notes, and a space's folder
// when nothing of the user's is left in it. Backups in other repositories, Claude Code,
// its desktop app, git and the PATH line (Claude Code lives in the same folder) are left
// alone.
func cmdUninstall(args []string, in io.Reader, out, errOut io.Writer) int {
	flags, yes, withConfig, all := uninstallFlags(errOut)
	if _, ok := parseArgs(flags, args, 0, errOut); !ok {
		return 2
	}
	if *all {
		*withConfig = true
	}
	lang := i18n.FromEnv()
	cfgDir, _ := config.Dir()
	type space struct {
		memory, work, backup string
		why                  string // why the notes stay with --all, or "" when they go
		other                int    // things in the notes folder that are not notes
	}
	var spaces []space
	c, cerr := config.Load(cfgDir)
	if cerr == nil {
		lang = c.Language
		var works []string
		for _, n := range c.SpaceNames() {
			works = append(works, c.Spaces[n].WorkFolders...)
		}
		home, _ := paths.Home()
		for _, n := range c.SpaceNames() {
			s := c.Spaces[n]
			sp := space{memory: s.Memory, backup: s.Backup}
			if len(s.WorkFolders) > 0 {
				sp.work = s.WorkFolders[0]
			}
			if *all {
				sp.why, sp.other = checkNotes(lang, s.Memory, home, works, s.Backup != "")
			}
			spaces = append(spaces, sp)
		}
	} else {
		// The remote access and the sessions are found without the configuration.
		c = &config.Config{Dir: cfgDir, Language: lang}
	}
	// The configuration folder goes only when it holds a configuration: AB_CONFIG_DIR
	// pointing somewhere else must not empty that folder.
	configGoes := *withConfig && cfgDir != "" && exists(filepath.Join(cfgDir, config.FileName))
	self, _ := os.Executable()

	// What would keep running once the plugin and its guard are gone: the remote access
	// services and the sessions launched for tasks. They go first.
	var services []string
	var launches []remote.LaunchStatus
	own := ""
	sys, serr := remoteSystem()
	if serr == nil {
		services = sys.Installed()
		if cfgDir != "" {
			list, err := sys.Launches(cfgDir)
			if err != nil && !errors.Is(err, remote.ErrUnsupported) {
				fmt.Fprintln(errOut, i18n.T(lang, "uninstall.sessions.unknown", err))
			}
			cwd, _ := os.Getwd()
			own = remote.LaunchedTask(cfgDir, "", os.Getenv("CLAUDE_JOB_DIR"), cwd)
			for _, l := range list {
				if l.Task != own {
					launches = append(launches, l)
				}
			}
		}
	}

	fmt.Fprintln(out, i18n.T(lang, "uninstall.plan"))
	if len(services) > 0 {
		fmt.Fprintln(out, "  - "+i18n.T(lang, "uninstall.remote", strings.Join(services, ", ")))
	}
	if len(launches) > 0 {
		var tasks []string
		for _, l := range launches {
			tasks = append(tasks, l.Task)
		}
		fmt.Fprintln(out, "  - "+i18n.T(lang, "uninstall.sessions", strings.Join(tasks, ", ")))
	}
	fmt.Fprintln(out, "  - "+i18n.T(lang, "uninstall.plugin"))
	fmt.Fprintln(out, "  - "+i18n.T(lang, "uninstall.leftovers"))
	if self != "" {
		fmt.Fprintln(out, "  - "+paths.Tilde(self))
	}
	if configGoes {
		fmt.Fprintln(out, "  - "+paths.Tilde(cfgDir))
	}
	var notes, backups, unsaved, kept []string
	for _, s := range spaces {
		switch {
		case *all && s.why != "":
			kept = append(kept, i18n.T(lang, "uninstall.notes.stay", paths.Tilde(s.memory), s.why))
		case *all && s.other > 0:
			fmt.Fprintln(out, "  - "+i18n.T(lang, "uninstall.notes.other", paths.Tilde(s.memory), countNotes(s.memory), s.other))
		case *all:
			fmt.Fprintln(out, "  - "+i18n.T(lang, "uninstall.notes", paths.Tilde(s.memory), countNotes(s.memory)))
		default:
			notes = append(notes, paths.Tilde(s.memory))
		}
		if s.backup != "" {
			backups = append(backups, s.backup)
		} else if s.why == "" {
			unsaved = append(unsaved, paths.Tilde(s.memory))
		}
	}
	if *withConfig && cfgDir != "" && !configGoes && exists(cfgDir) {
		kept = append(kept, i18n.T(lang, "uninstall.config.stay", paths.Tilde(cfgDir), config.FileName))
	}
	if own != "" {
		kept = append(kept, i18n.T(lang, "uninstall.ownsession", own))
	}
	for _, k := range kept {
		fmt.Fprintln(out, k)
	}
	if len(notes) > 0 {
		fmt.Fprintln(out, i18n.T(lang, "uninstall.keep", strings.Join(notes, ", ")))
	} else {
		fmt.Fprintln(out, i18n.T(lang, "uninstall.keeptools"))
	}
	if *all && len(backups) > 0 {
		fmt.Fprintln(out, i18n.T(lang, "uninstall.backups", strings.Join(backups, ", ")))
	}
	// Starting from scratch deletes notes that may exist nowhere else: say so plainly.
	if *all && len(unsaved) > 0 {
		fmt.Fprintln(out, i18n.T(lang, "uninstall.nobackup", strings.Join(unsaved, ", ")))
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
	// First what would run on without a guard. If any of it stays, nothing else goes:
	// without the program, nothing could take it away later.
	if len(services) > 0 {
		_, err := sys.Uninstall(c)
		fail(err)
	}
	for _, l := range launches {
		var left *remote.LeftInClaude
		if err := sys.Stop(cfgDir, l.Task); err != nil && !errors.Is(err, remote.ErrNoSession) && !errors.As(err, &left) {
			fail(fmt.Errorf("%s: %w", l.Task, err))
		}
	}
	if code != 0 {
		fmt.Fprintln(errOut, i18n.T(lang, "uninstall.firstfailed"))
		return code
	}
	if claudecode.Available() {
		fail(claudecode.Uninstall(claudecode.Exec))
	}
	fail(claudecode.RemoveLeftovers())
	// The plugin's guard blocks every command when it cannot run the program, so the
	// program stays until the plugin is surely gone.
	pluginGone := code == 0
	// A panel left running would keep the program in use (on Windows, it could not be
	// removed) and would write notes that are going away.
	if cfgDir != "" {
		_, _ = panel.Stop(cfgDir)
	}
	if configGoes {
		fail(os.RemoveAll(cfgDir))
	}
	if *all {
		for _, s := range spaces {
			if s.why != "" {
				continue
			}
			fail(removeNotes(s.memory, s.backup != ""))
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
		if pluginGone {
			fail(removeSelf(self))
		} else {
			fmt.Fprintln(errOut, i18n.T(lang, "uninstall.keptself", self))
		}
	}
	if code == 0 {
		switch {
		case *all && len(kept) > 0:
			fmt.Fprintln(out, i18n.T(lang, "uninstall.donekept"))
		case *all:
			fmt.Fprintln(out, i18n.T(lang, "uninstall.doneall"))
		default:
			fmt.Fprintln(out, i18n.T(lang, "uninstall.done"))
		}
	}
	return code
}

// notesNames are what a notes folder holds of augmented brain's (docs/spec/memory-format.md):
// all `uninstall --all` takes from it. gitNames are the backup's, when the space has one.
var (
	notesNames = []string{"MEMORY.md", "profile.md", "inbox.md", "areas", "tasks", "knowledge",
		"people", "rules", "reviews", "docs", "archive", format.MetaDir}
	gitNames = []string{".git", ".gitattributes", ".gitignore"}
)

// checkNotes says why a notes folder must stay whole with --all ("" when it may go),
// and how many things in it are not augmented brain's (they stay).
func checkNotes(lang, memory, home string, works []string, backup bool) (why string, other int) {
	switch {
	case home != "" && paths.Within(home, memory):
		return i18n.T(lang, "uninstall.why.home"), 0
	case !exists(filepath.Join(memory, "MEMORY.md")) && !exists(filepath.Join(memory, format.MetaDir, format.File)):
		return i18n.T(lang, "uninstall.why.notnotes"), 0
	case !backup && exists(filepath.Join(memory, ".git")):
		return i18n.T(lang, "uninstall.why.git"), 0
	}
	for _, w := range works {
		if paths.Within(w, memory) {
			return i18n.T(lang, "uninstall.why.work", paths.Tilde(w)), 0
		}
	}
	entries, _ := os.ReadDir(memory)
	for _, e := range entries {
		if !slices.Contains(notesNames, e.Name()) && !(backup && slices.Contains(gitNames, e.Name())) {
			other++
		}
	}
	return "", other
}

// removeNotes takes augmented brain's own out of a notes folder, and the folder when
// nothing else is left in it.
func removeNotes(memory string, backup bool) error {
	names := notesNames
	if backup {
		names = append(slices.Clone(names), gitNames...)
	}
	for _, n := range names {
		if err := os.RemoveAll(filepath.Join(memory, n)); err != nil {
			return err
		}
	}
	os.Remove(memory) // only when empty
	return nil
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

func exists(p string) bool { _, err := os.Stat(p); return err == nil }
