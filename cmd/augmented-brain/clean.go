package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"strings"

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
	if dir, err := config.Dir(); err == nil {
		if c, err := config.Load(dir); err == nil {
			lang = c.Language
		}
	}
	env, ok := cleanEnv()
	if !ok {
		fmt.Fprintln(errOut, i18n.T(lang, "clean.noscratch", "~/.claude"))
		return 1
	}
	r := leftovers.Find(env)
	kept := keptWorktrees(r)
	if r.Empty() {
		fmt.Fprintln(out, i18n.T(lang, "clean.nothing"))
		printKept(out, lang, kept)
		return 0
	}
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
	}
	if !r.ScratchOK {
		fmt.Fprintln(out, "\n"+i18n.T(lang, "clean.noscratch", paths.Tilde(env.ClaudeDir)))
	}
	printKept(out, lang, kept)
	if !*yes {
		fmt.Fprint(out, "\n"+i18n.T(lang, "clean.ask")+" ")
		answer, _ := bufio.NewReader(in).ReadString('\n')
		if a := strings.ToLower(strings.TrimSpace(answer)); !strings.HasPrefix(a, "y") && !strings.HasPrefix(a, "s") {
			fmt.Fprintln(out, i18n.T(lang, "uninstall.nothing"))
			return 1
		}
	}
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
		return 1
	}
	return 0
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
		key := "clean.kept.changes"
		switch k.Work {
		case remote.KeptCommits:
			key = "clean.kept.commits"
		case remote.KeptIgnored:
			key = "clean.kept.ignored"
		}
		fmt.Fprintln(out, "  - "+i18n.T(lang, key, paths.Tilde(k.Folder)))
	}
}
