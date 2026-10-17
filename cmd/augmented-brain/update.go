package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/icalvete/augmented-brain/internal/claudecode"
	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/update"
	"github.com/icalvete/augmented-brain/internal/version"
)

// cmdUpdate brings augmented brain up to date: the program through the published
// installer (the one people run, so there is a single way to install), and then the
// Claude Code plugin. --plugin updates only the plugin; the installer calls it.
func cmdUpdate(args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	fs.SetOutput(errOut)
	pluginOnly := fs.Bool("plugin", false, "update only the Claude Code plugin")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	lang := i18n.FromEnv()
	if dir, err := config.Dir(); err == nil {
		if c, err := config.Load(dir); err == nil {
			lang = c.Language
		}
	}
	if *pluginOnly {
		return updatePlugin(lang, out, errOut)
	}

	base := os.Getenv("AB_RELEASE_URL")
	if base == "" {
		latest, err := update.Latest(&http.Client{Timeout: 15 * time.Second})
		if err != nil {
			fmt.Fprintln(errOut, i18n.T(lang, "update.noreach", err))
			return 1
		}
		if !update.Newer(latest, version.Version) {
			fmt.Fprintln(out, i18n.T(lang, "update.current", version.Version))
			return updatePlugin(lang, out, errOut)
		}
		fmt.Fprintln(out, i18n.T(lang, "update.found", version.Version, latest))
		base = "https://github.com/" + update.Repo + "/releases/latest/download"
	}
	// The installer replaces the program and, as the configuration exists, then runs
	// `augmented-brain update --plugin` with the new program.
	cmd := exec.Command("sh", "-c", `curl -fsSL "$1/install.sh" | sh`, "sh", base)
	if runtime.GOOS == "windows" {
		// Windows has no sh: its installer is install.ps1, run by the PowerShell that
		// comes with Windows. It can replace this program while it runs.
		cmd = exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command",
			"irm '"+base+"/install.ps1' | iex")
	}
	cmd.Env = append(os.Environ(), "AB_LANGUAGE="+lang, "AB_NO_DESKTOP=1")
	cmd.Stdout, cmd.Stderr = out, errOut
	if err := cmd.Run(); err != nil {
		fmt.Fprintln(errOut, i18n.T(lang, "update.failed", err))
		return 1
	}
	return 0
}

func updatePlugin(lang string, out, errOut io.Writer) int {
	if !claudecode.Available() {
		return 0 // no Claude Code here: nothing to update
	}
	// After an uninstall the plugin is gone: install it again instead of updating it.
	update := claudecode.Update
	if ok, err := claudecode.Installed(claudecode.Exec); err == nil && !ok {
		update = claudecode.Install
	}
	if err := update(claudecode.Exec); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	fmt.Fprintln(out, i18n.T(lang, "update.plugin"))
	return 0
}
