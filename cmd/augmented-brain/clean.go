package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/icalvete/augmented-brain/internal/brain"
	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/leftovers"
	"github.com/icalvete/augmented-brain/internal/paths"
	"github.com/icalvete/augmented-brain/internal/remote"
)

func cleanFlags(errOut io.Writer) (fs *flag.FlagSet, yes *bool) {
	fs = flag.NewFlagSet("clean", flag.ContinueOnError)
	fs.SetOutput(errOut)
	yes = fs.Bool("yes", false, "do not ask")
	return fs, yes
}

// cleanEnv is what clean looks at; tests point it at temp folders.
var cleanEnv = leftovers.LocalEnv

// cmdClean shows what working with Claude Code left behind (package leftovers) and
// removes it when the user says yes, or with --yes. A worktree with work of its own is
// only named.
func cmdClean(args []string, in io.Reader, out, errOut io.Writer) int {
	flags, yes := cleanFlags(errOut)
	if _, ok := parseArgs(flags, args, 0, errOut); !ok {
		return 2
	}
	lang := i18n.FromEnv()
	var c *config.Config
	if dir, err := config.Dir(); err == nil {
		if loaded, err := config.Load(dir); err == nil {
			c, lang = loaded, loaded.Language
		}
	}
	env, ok := cleanEnv()
	if !ok {
		fmt.Fprintln(errOut, i18n.T(lang, "clean.noscratch", "~/.claude"))
		return 1
	}
	r := leftovers.Find(env)
	kept := keptWorktrees(r)
	sys, tasks, others := spareSessions(c, lang, errOut)
	answers := bufio.NewReader(in)
	if r.Empty() && len(tasks) == 0 && len(others) == 0 {
		fmt.Fprintln(out, i18n.T(lang, "clean.nothing"))
		printKept(out, lang, kept)
		return 0
	}
	code := 0
	if !r.Empty() || len(tasks) > 0 {
		fmt.Fprintln(out, i18n.T(lang, "clean.plan"))
		if w := r.Removable(); len(w) > 0 {
			fmt.Fprintln(out, "\n"+i18n.T(lang, "clean.worktrees"))
			for _, x := range w {
				line := paths.Tilde(x.Folder)
				if x.Branch != "" {
					line = i18n.T(lang, "clean.worktree.branch", line, x.Branch)
				}
				fmt.Fprintln(out, "  - "+line)
			}
		}
		if len(r.Prunable) > 0 {
			fmt.Fprintln(out, "\n"+i18n.T(lang, "clean.prunable"))
			for _, p := range r.Prunable {
				fmt.Fprintln(out, "  - "+paths.Tilde(p))
			}
		}
		if len(r.Scratch) > 0 {
			fmt.Fprintln(out, "\n"+i18n.T(lang, "clean.scratch", len(r.Scratch), leftovers.HumanSize(r.ScratchSize()), paths.Tilde(env.TempRoot)))
			if !r.AllProfiles {
				fmt.Fprintln(out, i18n.T(lang, "clean.oneprofile", paths.Tilde(env.ClaudeDir)))
			}
		}
		if len(tasks) > 0 {
			fmt.Fprintln(out, "\n"+i18n.T(lang, "clean.sessions"))
			for _, sp := range tasks {
				fmt.Fprintln(out, "  - "+spareTaskLine(lang, sp))
			}
		}
		if !r.ScratchOK {
			fmt.Fprintln(out, "\n"+i18n.T(lang, "clean.noscratch", paths.Tilde(env.ClaudeDir)))
		}
		printKept(out, lang, kept)
		if *yes || said(out, answers, "\n"+i18n.T(lang, "clean.ask")) {
			code = cleanAll(env, r, sys, c, tasks, lang, out, errOut)
		} else {
			fmt.Fprintln(out, i18n.T(lang, "uninstall.nothing"))
			code = 1
		}
	} else {
		printKept(out, lang, kept)
	}
	if len(others) == 0 {
		return code
	}
	// Sessions someone else started: one by one, never with --yes.
	fmt.Fprintln(out, "\n"+i18n.T(lang, "clean.others"))
	for _, sp := range others {
		line := i18n.T(lang, "clean.other", sp.Name, sp.ID, paths.Tilde(sp.Folder), i18n.T(lang, "clean.state."+sp.State))
		if sp.InWorktree() {
			line += " " + i18n.T(lang, "clean.other.worktree")
		}
		fmt.Fprintln(out, "  - "+line)
		if *yes {
			continue
		}
		if !said(out, answers, "    "+i18n.T(lang, "clean.other.ask")) {
			continue
		}
		if err := sys.RemoveSpare(c.Dir, sp); err != nil {
			fmt.Fprintln(errOut, "    "+err.Error())
			code = 1
			continue
		}
		fmt.Fprintln(out, "    "+i18n.T(lang, "clean.other.removed", sp.Name))
	}
	if *yes {
		fmt.Fprintln(out, i18n.T(lang, "clean.others.yes"))
	}
	return code
}

// said asks question and says whether the answer is yes (y, s, yes, sí…).
func said(out io.Writer, in *bufio.Reader, question string) bool {
	fmt.Fprint(out, question+" ")
	answer, _ := in.ReadString('\n')
	a := strings.ToLower(strings.TrimSpace(answer))
	return strings.HasPrefix(a, "y") || strings.HasPrefix(a, "s")
}

// cleanAll removes what clean found, after the yes: the leftovers and the sessions of
// tasks parked, put off, closed or gone.
func cleanAll(env leftovers.Env, r leftovers.Report, sys remote.System, c *config.Config, tasks []remote.Spare, lang string, out, errOut io.Writer) int {
	code := 0
	if !r.Empty() {
		d := leftovers.Clean(env, r)
		if len(d.Worktrees) > 0 {
			fmt.Fprintln(out, i18n.T(lang, "clean.removed.worktrees", len(d.Worktrees)))
		}
		for _, k := range d.Kept {
			fmt.Fprintln(out, i18n.T(lang, "clean.removed.kept", paths.Tilde(k.Folder)))
		}
		if len(d.Pruned) > 0 {
			fmt.Fprintln(out, i18n.T(lang, "clean.removed.pruned", len(d.Pruned)))
		}
		if d.Scratch > 0 {
			fmt.Fprintln(out, i18n.T(lang, "clean.removed.scratch", d.Scratch, leftovers.HumanSize(d.Freed)))
		}
		for _, err := range d.Errors {
			fmt.Fprintln(errOut, err)
		}
		if len(d.Errors) > 0 {
			code = 1
		}
	}
	removed := 0
	for _, sp := range tasks {
		var left *remote.LeftInClaude
		switch err := sys.RemoveSpare(c.Dir, sp); {
		case err == nil:
			removed++
		case errors.As(err, &left):
			fmt.Fprintln(errOut, i18n.T(lang, "session.left", sp.Task, left.ID))
			code = 1
		default:
			fmt.Fprintln(errOut, sp.Task+": "+err.Error())
			code = 1
		}
	}
	if removed > 0 {
		fmt.Fprintln(out, i18n.T(lang, "clean.removed.sessions", removed))
	}
	return code
}

// spareTaskLine names a task's spare session and why it is left over.
func spareTaskLine(lang string, sp remote.Spare) string {
	if sp.TaskStatus == "" {
		return i18n.T(lang, "clean.session.gone", sp.Name, sp.Task)
	}
	return i18n.T(lang, "clean.session", sp.Name, sp.Task, i18n.T(lang, "clean.status."+sp.TaskStatus))
}

// spareSessions are the background sessions left over (remote.SpareSessions): the
// tasks' own and the others. None without a configuration, or where sessions are not
// supported; when Claude's list cannot be read, it says so.
func spareSessions(c *config.Config, lang string, errOut io.Writer) (remote.System, []remote.Spare, []remote.Spare) {
	if c == nil {
		return remote.System{}, nil, nil
	}
	sys, err := remoteSystem()
	if err != nil || sys.GOOS != "linux" {
		return sys, nil, nil
	}
	if _, err := sys.FindClaude(); err != nil {
		return sys, nil, nil
	}
	list, err := sys.SpareSessions(c.Dir, brain.TaskStatuses(c))
	if err != nil {
		fmt.Fprintln(errOut, i18n.T(lang, "clean.sessions.unread", err))
		return sys, nil, nil
	}
	var tasks, others []remote.Spare
	for _, sp := range list {
		if sp.OfTask() {
			tasks = append(tasks, sp)
		} else {
			others = append(others, sp)
		}
	}
	return sys, tasks, others
}

func keptWorktrees(r leftovers.Report) []leftovers.Worktree {
	var kept []leftovers.Worktree
	for _, w := range r.Worktrees {
		if w.Work != "" {
			kept = append(kept, w)
		}
	}
	return kept
}

func printKept(out io.Writer, lang string, kept []leftovers.Worktree) {
	if len(kept) == 0 {
		return
	}
	fmt.Fprintln(out, "\n"+i18n.T(lang, "clean.kept"))
	for _, k := range kept {
		line := i18n.T(lang, "clean.kept.changes", paths.Tilde(k.Folder))
		switch k.Work {
		case remote.KeptCommits:
			line = i18n.T(lang, "clean.kept.commits", paths.Tilde(k.Folder))
		case remote.KeptIgnored:
			line = i18n.T(lang, "clean.kept.ignored", paths.Tilde(k.Folder), remote.IgnoredNames(lang, k.Folder))
		}
		fmt.Fprintln(out, "  - "+line)
	}
}
