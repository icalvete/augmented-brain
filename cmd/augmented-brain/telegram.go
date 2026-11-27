package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/remote"
	"github.com/icalvete/augmented-brain/internal/telegramprog"
)

func telegramFlags(errOut io.Writer) (*flag.FlagSet, *bool) {
	fs := flag.NewFlagSet("telegram disconnect", flag.ContinueOnError)
	fs.SetOutput(errOut)
	forget := fs.Bool("forget", false, "also take the application (api_id, api_hash) out of the keychain")
	return fs, forget
}

// telegramPath is where the Telegram program is, or ""; a variable, for tests.
var telegramPath = telegramprog.Path

// telegramInstall downloads it; a variable, for tests.
var telegramInstall = telegramprog.Install

// setTelegramTimer and unsetTelegramTimer set and remove the timer that looks for the
// replies tasks wait for; variables, so tests never touch the real systemd or launchd.
var (
	setTelegramTimer   = remote.SetTelegramTimer
	unsetTelegramTimer = remote.UnsetTelegramTimer
)

// afterTelegram keeps the timer in step with the connection: set after connect, gone
// after disconnect. A timer that cannot be set is said, and nothing else fails.
func afterTelegram(sub string, out io.Writer, l string) {
	switch sub {
	case "connect":
		dir, err := config.Dir()
		if err == nil {
			err = setTelegramTimer(dir)
		}
		if err != nil {
			fmt.Fprintln(out, i18n.T(l, "telegram.timer.failed", err))
		}
	case "disconnect":
		_, _ = unsetTelegramTimer()
	}
}

// cmdTelegram runs augmented brain's Telegram program with the same words. connect
// downloads it first, when it is not there yet: only who turns Telegram on has it.
func cmdTelegram(args []string, in io.Reader, out, errOut io.Writer) int {
	if len(args) > 0 && (args[0] == "check" || args[0] == "waits") {
		c, ok := loadConfig(errOut)
		if !ok {
			return 1
		}
		if args[0] == "check" {
			if len(args) != 1 {
				fmt.Fprintln(errOut, i18n.T(c.Language, "usage.telegram"))
				return 2
			}
			return telegramCheck(c, out, errOut)
		}
		return telegramWaits(c, args[1:], out, errOut)
	}
	l := userLanguage()
	fs, _ := telegramFlags(errOut)
	rest, ok := parseArgs(fs, args, 1, errOut)
	if !ok || len(rest) != 1 || (rest[0] != "connect" && rest[0] != "status" && rest[0] != "disconnect") {
		fmt.Fprintln(errOut, i18n.T(l, "usage.telegram"))
		return 2
	}
	if !telegramprog.Supported() {
		fmt.Fprintln(errOut, i18n.T(l, "telegram.unsupported"))
		return 1
	}
	p := telegramPath()
	if p == "" {
		if rest[0] != "connect" {
			fmt.Fprintln(out, i18n.T(l, "telegram.status.off"))
			return 0
		}
		fmt.Fprintln(out, i18n.T(l, "telegram.installing"))
		var err error
		if p, err = telegramInstall(); err != nil {
			fmt.Fprintln(errOut, i18n.T(l, "telegram.installfailed", err))
			return 1
		}
	}
	// unbounded: connect waits for a person to scan a QR code and type a password; the
	// Telegram program bounds its own waits (5 minutes for a login, 1 for a call).
	cmd := exec.Command(p, args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = in, out, errOut
	cmd.Env = append(os.Environ(), "AB_LANGUAGE="+l)
	if err := cmd.Run(); err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return exit.ExitCode()
		}
		fmt.Fprintln(errOut, err)
		return 1
	}
	afterTelegram(rest[0], out, l)
	return 0
}
