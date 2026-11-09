package main

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/remote"
)

// remoteSystem is the system `remote` acts on; tests replace it with fakes.
var remoteSystem = remote.NewSystem

// cmdRemote is `remote install | uninstall | status`: Claude kept reachable from
// another computer, one service per space. The logic is in internal/remote.
func cmdRemote(args []string, out, errOut io.Writer) int {
	if len(args) != 1 || (args[0] != "install" && args[0] != "uninstall" && args[0] != "status") {
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
