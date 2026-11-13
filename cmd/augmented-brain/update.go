package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/icalvete/augmented-brain/internal/claudecode"
	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/doctor"
	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/state"
	"github.com/icalvete/augmented-brain/internal/update"
	"github.com/icalvete/augmented-brain/internal/version"
)

// updateFlags are update's flags: --plugin, --force, --no-restart and --from.
func updateFlags(errOut io.Writer) (*flag.FlagSet, *bool, *bool, *bool, *string) {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	fs.SetOutput(errOut)
	plugin := fs.Bool("plugin", false, "update only the Claude Code plugin")
	force := fs.Bool("force", false, "update even when the backup before it fails")
	noRestart := fs.Bool("no-restart", false, "do not restart the idle Claude sessions still running the old program")
	from := fs.String("from", "", "for tests: install the release at this address instead of GitHub's latest")
	return fs, plugin, force, noRestart, from
}

// ignoredEnv are the variables of the installer's tests that `update` never obeys: a
// project's .claude/settings.json can set the environment of a Claude session, and the
// session may run `update` for the user. Where the release comes from is said with
// --from, on the command line, where the guard and the user see it.
var ignoredEnv = []string{"AB_RELEASE_URL", "AB_MARKETPLACE"}

// installerEnv are the variables of augmented brain passed on to the installer `update`
// runs; any other AB_ variable is left out (see ignoredEnv). The rest of the
// environment (PATH, HOME, the system's own) goes as it is.
var installerEnv = map[string]bool{"AB_CONFIG_DIR": true, "AB_INSTALL_DIR": true}

// checkFrom accepts the address of --from: https://, file:// (a release in a folder) or
// http:// only to this computer or to the host of a test VM (10.0.2.2), where the
// rehearsal serves its release.
func checkFrom(addr string) error {
	u, err := url.Parse(addr)
	if err != nil {
		return err
	}
	switch u.Scheme {
	case "https", "file":
		return nil
	case "http":
		switch u.Hostname() {
		case "127.0.0.1", "localhost", "::1", "10.0.2.2":
			return nil
		}
	}
	return errors.New(addr)
}

// cmdUpdate brings augmented brain up to date: the program through the published
// installer (the one people run, so there is a single way to install), and then the
// Claude Code plugin. --plugin updates only the plugin; the installer calls it.
//
// Around a new version it does what people used to do by hand: before, the health
// check (its problems are shown, so they are not taken for the new version's) and a
// backup of every space that has one (a failed backup stops it, unless --force); after,
// the idle Claude sessions still running the old program are restarted (unless
// --no-restart), and for those left, which task each space has active, to park before
// restarting Claude.
func cmdUpdate(args []string, out, errOut io.Writer) int {
	fs, pluginOnly, force, noRestart, from := updateFlags(errOut)
	if _, ok := parseArgs(fs, args, 0, errOut); !ok {
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

	for _, v := range ignoredEnv {
		if os.Getenv(v) != "" {
			fmt.Fprintln(errOut, i18n.T(lang, "update.envignored", v))
			os.Unsetenv(v)
		}
	}
	base := *from
	if base != "" {
		if err := checkFrom(base); err != nil {
			fmt.Fprintln(errOut, i18n.T(lang, "update.badfrom", base))
			return 2
		}
		fmt.Fprintln(out, i18n.T(lang, "update.from", base))
	}
	latest := ""
	if base == "" {
		var err error
		latest, err = latestRelease(&http.Client{Timeout: 15 * time.Second})
		if err != nil {
			fmt.Fprintln(errOut, i18n.T(lang, "update.noreach", err))
			return 1
		}
		if !update.Newer(latest, version.Version) {
			fmt.Fprintln(out, i18n.T(lang, "update.current", version.Version))
			return updatePlugin(lang, out, errOut)
		}
		fmt.Fprintln(out, i18n.T(lang, "update.found", version.Version, latest))
	}
	if code := beforeUpdate(lang, *force, out, errOut); code != 0 {
		return code
	}
	if err := runInstaller(lang, base, latest, out, errOut); err != nil {
		fmt.Fprintln(errOut, i18n.T(lang, "update.failed", err))
		return 1
	}
	now := installedVersion()
	if now == "" {
		// The installer checks the new program before it replaces the old one, and puts
		// the old one back when it does not start: this is what is left, said plainly.
		fmt.Fprintln(errOut, i18n.T(lang, "update.broken"))
		return 1
	}
	if latest != "" && update.Newer(latest, now) {
		fmt.Fprintln(errOut, i18n.T(lang, "update.notdone", now, latest))
		return 1
	}
	afterUpdate(lang, now, !*noRestart, out)
	return 0
}

// latestRelease asks GitHub for the latest version; tests replace it.
var latestRelease = update.Latest

// runInstaller downloads the published installer of release latest (or the one at
// base) into a file and runs it: it replaces the program and, as the configuration
// exists, then runs `augmented-brain update --plugin` with the new program. Never
// `curl | sh`: there a download that failed reaches sh as an empty script, which
// succeeds, and the update looked done when nothing had changed.
func runInstaller(lang, base, latest string, out, errOut io.Writer) error {
	name := "install.sh"
	if runtime.GOOS == "windows" {
		name = "install.ps1"
	}
	tmp, err := os.MkdirTemp("", "ab-update-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	script := filepath.Join(tmp, name)
	if err := update.Download(&http.Client{Timeout: time.Minute}, base, latest, name, script); err != nil {
		return err
	}
	cmd := exec.Command("sh", script)
	if runtime.GOOS == "windows" {
		// Windows has no sh: its installer is install.ps1, run by the PowerShell that
		// comes with Windows. It can replace this program while it runs.
		cmd = exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", script)
	}
	// AB_VERSION: the installer takes the very release just checked, not whatever is
	// latest a moment later. AB_EXIT_CODE: install.ps1 ends with an error code when it
	// fails, which it never does in a person's own window.
	// AB_RELEASE_URL only from --from (base).
	cmd.Env = append(installerEnviron(os.Environ()), "AB_LANGUAGE="+lang, "AB_NO_DESKTOP=1", "AB_EXIT_CODE=1")
	if latest != "" {
		cmd.Env = append(cmd.Env, "AB_VERSION="+latest)
	}
	if base != "" {
		cmd.Env = append(cmd.Env, "AB_RELEASE_URL="+base)
	}
	cmd.Stdout, cmd.Stderr = out, errOut
	return cmd.Run()
}

// installerEnviron is env without the variables of augmented brain the installer must not get
// from it: every AB_ one but those of installerEnv.
func installerEnviron(env []string) []string {
	var keep []string
	for _, kv := range env {
		name, _, _ := strings.Cut(kv, "=")
		if strings.HasPrefix(strings.ToUpper(name), "AB_") && !installerEnv[strings.ToUpper(name)] {
			continue
		}
		keep = append(keep, kv)
	}
	return keep
}

// beforeUpdate checks the installation and backs up every space that has a backup,
// before a new version replaces the program. A problem the check finds is shown, not a
// reason to stop: an update makes none of them worse (a missing Claude Code is even
// installed by it). A backup that fails stops it: that copy is the way back. Without a configuration that loads there
// is nothing to check or back up: the update may be what fixes it (a newer format, a
// setting this version does not know). 1 stops the update.
func beforeUpdate(lang string, force bool, out, errOut io.Writer) int {
	dir, err := config.Dir()
	if err != nil {
		return 0
	}
	c, err := config.Load(dir)
	if err != nil {
		return 0
	}
	wd, _ := os.Getwd()
	var failed []doctor.Check
	for _, ch := range doctor.Run(doctor.Env{ConfigDir: dir, WorkDir: wd, Lang: lang}) {
		if ch.Status == doctor.Failed {
			failed = append(failed, ch)
		}
	}
	if len(failed) > 0 {
		fmt.Fprintln(errOut, i18n.T(lang, "update.before.check"))
		for _, ch := range failed {
			fmt.Fprintf(errOut, "✗ %s\n", ch.What)
			if ch.Fix != "" {
				fmt.Fprintf(errOut, "    → %s\n", ch.Fix)
			}
		}
	} else {
		fmt.Fprintln(out, i18n.T(lang, "update.before.ok"))
	}
	var spaces []*config.Space
	for _, n := range c.SpaceNames() {
		if c.Spaces[n].Backup != "" {
			spaces = append(spaces, c.Spaces[n])
		}
	}
	if len(spaces) == 0 {
		fmt.Fprintln(out, i18n.T(lang, "update.nobackup"))
		return 0
	}
	fmt.Fprintln(out, i18n.T(lang, "update.before.backup"))
	if backupSpaces(c, spaces, out, errOut) != 0 && !force {
		fmt.Fprintln(errOut, i18n.T(lang, "update.before.stop"))
		return 1
	}
	return 0
}

// afterUpdate restarts the idle Claude sessions still running the old program, through
// the new one (`session restart --old`: this one may still be the old one, running
// from memory, which knows nothing of how). For any left, or with restart false, it
// says which task each space has active: park it, then close and open Claude.
func afterUpdate(lang, now string, restart bool, out io.Writer) {
	dir, err := config.Dir()
	if err != nil {
		return
	}
	c, err := config.Load(dir)
	if err != nil {
		return
	}
	if now == "" {
		return
	}
	old := state.OldServers(dir, now)
	if len(old) == 0 {
		fmt.Fprintln(out, i18n.T(lang, "update.after.none"))
		return
	}
	if restart && restartOldWith(out) == nil {
		return
	}
	if !restart {
		fmt.Fprintln(out, i18n.T(lang, "update.after.old", len(old)))
	}
	for _, n := range c.SpaceNames() {
		// Each open session has its own active task; two may share one.
		said := map[string]bool{}
		for _, se := range state.Read(dir, n).Sessions {
			if se.Task != "" && !se.Closed && !said[se.Task] {
				said[se.Task] = true
				fmt.Fprintln(out, i18n.T(lang, "update.after.task", n, se.Task))
			}
		}
	}
	fmt.Fprintln(out, i18n.T(lang, "update.after.restart"))
}

// restartOldWith runs the program now installed with `session restart --old`, its
// output going to out; an error when it left some session. Tests replace it.
var restartOldWith = func(out io.Writer) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(exe, "session", "restart", "--old")
	cmd.Stdout, cmd.Stderr = out, out
	return cmd.Run()
}

// installedVersion is the version of the program the installer just put in place (this
// one may still be the old one, running from memory), or "" when it cannot tell.
func installedVersion() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	o, err := exec.Command(exe, "version").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(o))
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
