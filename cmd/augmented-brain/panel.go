package main

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/panel"
)

// panelNoBrowser is panel's one flag, read by hand (panel takes stop and serve too).
const panelNoBrowser = "no-browser"

// cmdPanel opens the panel: it starts the server in a detached process (or finds the
// one running), prints its address, opens the browser, and returns at once.
// AB_NO_BROWSER=1 or --no-browser only prints the address.
func cmdPanel(args []string, out, errOut io.Writer) int {
	c, ok := loadConfig(errOut)
	if !ok {
		return 1
	}
	dir, l := c.Dir, c.Language
	sub := ""
	noBrowser := os.Getenv("AB_NO_BROWSER") == "1"
	for _, a := range args {
		switch a {
		case "--" + panelNoBrowser:
			noBrowser = true
		case "stop", "serve":
			sub = a
		default:
			fmt.Fprintln(errOut, i18n.T(l, "usage.panel"))
			return 2
		}
	}
	switch sub {
	case "serve":
		// The detached server itself; `augmented-brain panel` starts it.
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		if err := panel.Serve(ctx, dir, panel.Idle); err != nil {
			fmt.Fprintln(errOut, err)
			return 1
		}
		return 0
	case "stop":
		stopped, err := panel.Stop(dir)
		if err != nil {
			fmt.Fprintln(errOut, err)
			return 1
		}
		if stopped {
			fmt.Fprintln(out, i18n.T(l, "panel.cli.stopped"))
		} else {
			fmt.Fprintln(out, i18n.T(l, "panel.cli.none"))
		}
		return 0
	}
	self, err := os.Executable()
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	info, started, err := panel.Start(dir, self)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	wd, _ := os.Getwd()
	address := panelAddress(c, info.URL(), wd)
	if started {
		fmt.Fprintln(out, i18n.T(l, "panel.cli.started", address))
	} else {
		fmt.Fprintln(out, i18n.T(l, "panel.cli.running", address))
	}
	if noBrowser || panel.OpenBrowser(address) != nil {
		fmt.Fprintln(out, i18n.T(l, "panel.cli.nobrowser"))
	}
	fmt.Fprintln(out, i18n.T(l, "panel.cli.idle"))
	return 0
}

// panelAddress is the page to open: Today, on the space of the folder the panel was
// asked for in (so /brain:panel in a work session shows work), or else the panel's
// first page, which shows the default space.
func panelAddress(c *config.Config, base, wd string) string {
	if s := c.SpaceFor(wd); s != nil {
		return base + "today?s=" + url.QueryEscape(s.Name)
	}
	return base
}
