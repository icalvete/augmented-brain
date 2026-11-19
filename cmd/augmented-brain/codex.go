package main

import (
	"fmt"
	"io"

	"github.com/icalvete/augmented-brain/internal/codex"
	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/i18n"
)

// codexAvailable and codexEnv are what `codex` works with; tests replace them.
var (
	codexAvailable = codex.Available
	codexEnv       = codex.Default
)

// cmdCodex is `codex install | uninstall | status`: augmented brain in Codex, as a
// plugin of its own. The logic is in internal/codex.
func cmdCodex(args []string, out, errOut io.Writer) int {
	l := userLanguage()
	if len(args) != 1 || (args[0] != "install" && args[0] != "uninstall" && args[0] != "status") {
		fmt.Fprintln(errOut, i18n.T(l, "usage.codex"))
		return 2
	}
	if !codexAvailable() {
		fmt.Fprintln(errOut, i18n.T(l, "codex.missing"))
		return 1
	}
	dir, err := config.Dir()
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	e, err := codexEnv(dir)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	switch args[0] {
	case "install":
		fmt.Fprintln(out, i18n.T(l, "codex.installing", codex.Plugin))
		if err := codex.Install(e); err != nil {
			fmt.Fprintln(errOut, err)
			return 1
		}
		fmt.Fprintln(out, i18n.T(l, "codex.installed"))
	case "uninstall":
		if err := codex.Uninstall(e); err != nil {
			fmt.Fprintln(errOut, err)
			return 1
		}
		fmt.Fprintln(out, i18n.T(l, "codex.uninstalled"))
	case "status":
		st := codex.Check(e)
		lines := codexStatusLines(l, st)
		for _, line := range lines {
			fmt.Fprintln(out, line)
		}
		if !st.OK() {
			return 1
		}
	}
	return 0
}

// codexStatusLines says how augmented brain stands in Codex: one line when all is well
// or it is not installed, one per problem and the fix otherwise.
func codexStatusLines(l string, st codex.Status) []string {
	switch {
	case st.Err != nil:
		return []string{i18n.T(l, "codex.status.error", st.Err)}
	case st.OK():
		return []string{i18n.T(l, "codex.status.ok", st.Trusted)}
	case st.NotInstalled():
		return []string{i18n.T(l, "codex.status.none")}
	}
	return append(codex.Problems(l, st), i18n.T(l, "codex.status.fix", codex.Fix))
}
