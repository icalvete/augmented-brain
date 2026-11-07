package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/signal"
	"os/user"
	"strings"
	"syscall"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/panel"
	"github.com/icalvete/augmented-brain/internal/tailnet"
	"rsc.io/qr"
)

// panelTailnet is the Tailscale `panel remote` uses; tests replace it with a fake.
var panelTailnet = tailnet.New

// tailscaleAdmin is the page of Tailscale's admin console where HTTPS is turned on.
const tailscaleAdmin = "https://login.tailscale.com/admin/dns"

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
	sub, remote := "", false
	noBrowser := os.Getenv("AB_NO_BROWSER") == "1"
	for _, a := range args {
		switch a {
		case "--" + panelNoBrowser:
			noBrowser = true
		case "--remote": // panel serve --remote, the detached server of panel remote
			remote = true
		case "stop", "serve", "remote":
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
		tn := panelTailnet()
		// What the panel needs to answer through the network; also asked later, when
		// `panel remote` finds this panel running and has it switch in place.
		publish := func() (*panel.Remote, error) {
			st, err := tn.Status()
			if err != nil {
				return nil, err
			}
			return &panel.Remote{Host: st.Host(), Owner: st.Owner, Unpublish: tn.Unserve}, nil
		}
		var r *panel.Remote
		if remote {
			var err error
			if r, err = publish(); err != nil {
				fmt.Fprintln(errOut, err)
				return 1
			}
		}
		if err := panel.ServeWith(ctx, dir, panel.Idle, r, publish); err != nil {
			fmt.Fprintln(errOut, err)
			return 1
		}
		return 0
	case "remote":
		return panelRemote(c, out, errOut)
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

// panelRemote is `panel remote`: the panel, also published to the user's own Tailscale
// network, with its address and a QR code to open it from the phone. What is missing
// (Tailscale, HTTPS in the network, the operator setting) is explained, never done.
func panelRemote(c *config.Config, out, errOut io.Writer) int {
	l := c.Language
	tn := panelTailnet()
	st, err := tn.Status()
	if err == nil && !st.HTTPS {
		err = tailnet.ErrNoHTTPS
	}
	if err == nil {
		var cur string
		if cur, err = tn.Served(); err == nil && cur == "other" {
			err = tailnet.ErrPortTaken
		}
	}
	if err != nil {
		fmt.Fprintln(errOut, explainTailnet(l, err))
		return 1
	}
	self, err := os.Executable()
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	info, how, err := panel.GoRemote(c.Dir, self)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	if err := tn.Serve(info.Port); err != nil {
		if how == panel.Switched {
			// The panel was open and went remote for this: it goes back to local only,
			// and the pages open in the browser go on as they were.
			_ = panel.GoLocal(c.Dir)
		}
		fmt.Fprintln(errOut, explainTailnet(l, err))
		return 1
	}
	if how == panel.Restarted {
		fmt.Fprintln(out, i18n.T(l, "panel.remote.restarted"))
	}
	_ = tailnet.MarkUsed(c.Dir)
	wd, _ := os.Getwd()
	address := panelAddress(c, info.RemoteURL(), wd)
	fmt.Fprintln(out, i18n.T(l, "panel.remote.open", address))
	if code := qrText(address); code != "" {
		fmt.Fprintln(out, "\n"+i18n.T(l, "panel.remote.qr"))
		fmt.Fprint(out, code)
	}
	fmt.Fprintln(out, "\n"+i18n.T(l, "panel.remote.idle", st.Owner))
	return 0
}

// explainTailnet says in the user's language what is missing for the remote panel.
func explainTailnet(l string, err error) string {
	switch {
	case errors.Is(err, tailnet.ErrMissing):
		return i18n.T(l, "panel.remote.missing")
	case errors.Is(err, tailnet.ErrStopped):
		return i18n.T(l, "panel.remote.stopped")
	case errors.Is(err, tailnet.ErrNoHTTPS):
		return i18n.T(l, "panel.remote.nohttps", tailscaleAdmin)
	case errors.Is(err, tailnet.ErrOperator):
		name := os.Getenv("USER")
		if u, uerr := user.Current(); uerr == nil {
			name = u.Username
		}
		return i18n.T(l, "panel.remote.operator", "sudo tailscale set --operator="+name)
	case errors.Is(err, tailnet.ErrPortTaken):
		return i18n.T(l, "panel.remote.taken", tailnet.Port)
	}
	return err.Error()
}

// qrText draws text as a QR code with half blocks, two rows of modules per line, for a
// terminal: light modules are drawn and dark ones left blank, as qrencode does, which
// reads right on the usual dark terminal. "" when it cannot be encoded.
func qrText(text string) string {
	code, err := qr.Encode(text, qr.L)
	if err != nil {
		return ""
	}
	const quiet = 2
	light := func(x, y int) bool {
		if x < 0 || y < 0 || x >= code.Size || y >= code.Size {
			return true
		}
		return !code.Black(x, y)
	}
	var b strings.Builder
	for y := -quiet; y < code.Size+quiet; y += 2 {
		b.WriteString("  ")
		for x := -quiet; x < code.Size+quiet; x++ {
			top, bottom := light(x, y), light(x, y+1)
			switch {
			case top && bottom:
				b.WriteString("█")
			case top:
				b.WriteString("▀")
			case bottom:
				b.WriteString("▄")
			default:
				b.WriteString(" ")
			}
		}
		b.WriteString("\n")
	}
	return b.String()
}
