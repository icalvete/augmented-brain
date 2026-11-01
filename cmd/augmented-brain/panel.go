package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/panel"
)

// cmdPanel opens the panel: it starts the server in a detached process (or finds the
// one running), prints its address, opens the browser, and returns at once.
// AB_NO_BROWSER=1 or --no-browser only prints the address.
func cmdPanel(args []string, out, errOut io.Writer) int {
	dir, err := config.Dir()
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	c, err := config.Load(dir)
	if errors.Is(err, config.ErrNotFound) {
		fmt.Fprintln(errOut, i18n.T(i18n.FromEnv(), "config.missing", err))
		return 1
	}
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	l := c.Language
	sub := ""
	noBrowser := os.Getenv("AB_NO_BROWSER") == "1"
	for _, a := range args {
		switch a {
		case "--no-browser":
			noBrowser = true
		case "stop", "serve":
			sub = a
		default:
			fmt.Fprintln(errOut, "usage: augmented-brain panel [--no-browser] | panel stop")
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
	if started {
		fmt.Fprintln(out, i18n.T(l, "panel.cli.started", info.URL()))
	} else {
		fmt.Fprintln(out, i18n.T(l, "panel.cli.running", info.URL()))
	}
	if noBrowser || panel.OpenBrowser(info.URL()) != nil {
		fmt.Fprintln(out, i18n.T(l, "panel.cli.nobrowser"))
	}
	fmt.Fprintln(out, i18n.T(l, "panel.cli.idle"))
	return 0
}
