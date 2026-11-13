package main

import (
	"errors"
	"fmt"
	"io"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/icalvete/augmented-brain/internal/backup"
	"github.com/icalvete/augmented-brain/internal/brain"
	"github.com/icalvete/augmented-brain/internal/filelock"
	"github.com/icalvete/augmented-brain/internal/state"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/remote"
)

// remoteSystem is the system `remote` acts on; tests replace it with fakes.
var remoteSystem = remote.NewSystem

// cmdRemote is `remote install | uninstall | status`: Claude kept reachable from
// another computer, one service per space. The logic is in internal/remote.
func cmdRemote(args []string, out, errOut io.Writer) int {
	if len(args) != 1 || (args[0] != "install" && args[0] != "uninstall" && args[0] != "status" && args[0] != "watch") {
		fmt.Fprintln(errOut, i18n.T(userLanguage(), "usage.remote"))
		return 2
	}
	c, ok := loadConfig(errOut)
	if !ok {
		return 1
	}
	l := c.Language
	sys, err := remoteSystem()
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	host, _ := sys.Hostname()
	set, err := remote.LoadSettings(c, host)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	var runErr error
	switch args[0] {
	case "install":
		runErr = remoteInstall(c, set, sys, out)
	case "uninstall":
		runErr = remoteUninstall(c, sys, out)
	case "status":
		runErr = remoteStatus(c, set, sys, out)
	case "watch": // run by the watch's timer, every two minutes
		runErr = sys.Watch(c, out)
		if !errors.Is(runErr, remote.ErrUnsupported) {
			runErr = errors.Join(runErr, remoteQuota(c, sys, out))
		}
	}
	if errors.Is(runErr, remote.ErrUnsupported) {
		fmt.Fprintln(errOut, i18n.T(l, "remote.unsupported.os"))
		return 1
	}
	if runErr != nil {
		fmt.Fprintln(errOut, runErr)
		return 1
	}
	return 0
}

// remoteQuota is the watch's look at the tasks' sessions the usage limit stopped
// (remote.WakeQuota). The user hears of it twice, whatever the number of sessions: when
// the first one stops and when the last one is back, from the next hook that can show
// them a message; each woken task gets a line in its journal.
func remoteQuota(c *config.Config, sys remote.System, out io.Writer) error {
	l := c.Language
	now := time.Now()
	rep, err := sys.WakeQuota(c, func(task string) string { return i18n.T(l, "quota.wake.message", task) })
	notice := func(text string) {
		fmt.Fprintln(out, text)
		if nErr := state.AddQuotaNotice(c.Dir, text, now); nErr != nil {
			err = errors.Join(err, nErr)
		}
	}
	if rep.Began {
		notice(i18n.T(l, "quota.notice.began", quotaHour(rep.Until, now)))
	}
	if rep.Woken != "" {
		fmt.Fprintln(out, i18n.T(l, "quota.woken", rep.Woken))
		if jErr := quotaNote(c, rep.Woken); jErr != nil {
			fmt.Fprintln(out, jErr)
		}
	}
	if rep.Problem != "" {
		fmt.Fprintln(out, i18n.T(l, "quota.problem", rep.Task, rep.Problem))
		if rep.GaveUp {
			notice(i18n.T(l, "quota.notice.gaveup", rep.Task, rep.Problem, rep.Task))
		}
	}
	if rep.Ended {
		notice(i18n.T(l, "quota.notice.ended", rep.Count))
	}
	return err
}

// quotaHour is the hour of t as the user's clock shows it, with the day when it is not
// today's.
func quotaHour(t, now time.Time) string {
	t = t.Local()
	if y, m, d := now.Local().Date(); t.Year() != y || t.Month() != m || t.Day() != d {
		return t.Format("15:04 (2006-01-02)")
	}
	return t.Format("15:04")
}

// quotaNote writes in the task's journal that its session was woken, in whichever
// space the task is.
func quotaNote(c *config.Config, task string) error {
	unlock, err := filelock.Lock(filepath.Join(c.Dir, backup.NotesLock))
	if err != nil {
		return err
	}
	defer unlock()
	for _, name := range slices.Sorted(maps.Keys(c.Spaces)) {
		b, err := brain.Open(c, c.Spaces[name])
		if err != nil {
			continue
		}
		if _, err := b.Task(task); err != nil {
			continue
		}
		_, err = b.Journal(task, i18n.T(c.Language, "quota.journal"))
		return err
	}
	return nil
}

func remoteInstall(c *config.Config, set remote.Settings, sys remote.System, out io.Writer) error {
	l := c.Language
	res, err := sys.Install(c, set)
	if err != nil {
		return err
	}
	fmt.Fprintln(out, i18n.T(l, "remote.install.done"))
	for _, s := range res.Services {
		fmt.Fprintln(out, i18n.T(l, "remote.install.space", s.Space, s.Name, tilde(s.Folder)))
	}
	if res.Watch {
		fmt.Fprintln(out, i18n.T(l, "remote.install.watch", strings.TrimSuffix(remote.WatchEvery, "min")))
	}
	for _, s := range res.Added {
		fmt.Fprintln(out, i18n.T(l, "remote.install.added", tilde(s.Folder), s.Space))
	}
	if len(res.Untrusted) > 0 {
		fmt.Fprintln(out, "\n"+i18n.T(l, "remote.install.trust"))
		for _, s := range res.Untrusted {
			key := "remote.install.trust.cmd"
			if sys.GOOS == "windows" {
				key += ".win"
			}
			fmt.Fprintln(out, i18n.T(l, key, s.Folder))
		}
		fmt.Fprintln(out, i18n.T(l, "remote.install.trust.after"))
	}
	switch res.Linger {
	case remote.LingerOff:
		fmt.Fprintln(out, "\n"+i18n.T(l, "remote.install.linger"))
		fmt.Fprintln(out, "  "+sys.LingerCommand())
	case remote.LingerUnknown:
		fmt.Fprintln(out, "\n"+i18n.T(l, "remote.install.linger.unknown"))
	case remote.LingerNone:
		fmt.Fprintln(out, "\n"+i18n.T(l, "remote.install.atlogin"))
	}
	if len(res.Untrusted) == 0 && (res.Linger == remote.LingerOn || res.Linger == remote.LingerNone) {
		for _, s := range res.Services {
			fmt.Fprintln(out, "\n"+i18n.T(l, "remote.install.connect", s.Name))
		}
	}
	return nil
}

func remoteUninstall(c *config.Config, sys remote.System, out io.Writer) error {
	removed, err := sys.Uninstall(c)
	if len(removed) > 0 {
		fmt.Fprintln(out, i18n.T(c.Language, "remote.uninstall.done", strings.Join(removed, ", ")))
	} else if err == nil {
		fmt.Fprintln(out, i18n.T(c.Language, "remote.uninstall.none"))
	}
	return err
}

func remoteStatus(c *config.Config, set remote.Settings, sys remote.System, out io.Writer) error {
	st, err := sys.Status(c, set)
	if err != nil {
		return err
	}
	if len(st.Spaces) == 0 || (!set.Enabled && !anyInstalled(st)) {
		fmt.Fprintln(out, i18n.T(c.Language, "remote.status.off"))
		return nil
	}
	for _, line := range sys.StatusLines(c.Language, st) {
		fmt.Fprintln(out, line)
	}
	return nil
}

func anyInstalled(st *remote.Status) bool {
	for _, s := range st.Spaces {
		if s.Installed {
			return true
		}
	}
	return false
}

func tilde(p string) string { return tildes([]string{p})[0] }
