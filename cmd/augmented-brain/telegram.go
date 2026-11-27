package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/icalvete/augmented-brain/internal/i18n"
)

// telegramProgram is the name of augmented brain's Telegram program. It is a program
// of its own, next to this one, because Telegram's library triples a program's size:
// only who turns Telegram on downloads it.
const telegramProgram = "augmented-brain-telegram"

func telegramFlags(errOut io.Writer) (*flag.FlagSet, *bool) {
	fs := flag.NewFlagSet("telegram disconnect", flag.ContinueOnError)
	fs.SetOutput(errOut)
	forget := fs.Bool("forget", false, "also take the application (api_id, api_hash) out of the keychain")
	return fs, forget
}

// telegramPath is where the Telegram program is, or "" when it is not installed: next
// to this program, where update puts it, or else in PATH. A variable, for tests.
var telegramPath = func() string {
	name := telegramProgram
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	if self, err := os.Executable(); err == nil {
		if self, err = filepath.EvalSymlinks(self); err == nil {
			p := filepath.Join(filepath.Dir(self), name)
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
	}
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	return ""
}

func cmdTelegram(args []string, in io.Reader, out, errOut io.Writer) int {
	l := userLanguage()
	fs, _ := telegramFlags(errOut)
	rest, ok := parseArgs(fs, args, 1, errOut)
	if !ok || len(rest) != 1 || (rest[0] != "connect" && rest[0] != "status" && rest[0] != "disconnect") {
		fmt.Fprintln(errOut, i18n.T(l, "usage.telegram"))
		return 2
	}
	p := telegramPath()
	if p == "" {
		if rest[0] != "connect" {
			fmt.Fprintln(out, i18n.T(l, "telegram.status.off"))
			return 0
		}
		var err error
		if p, err = installTelegram(l, out); err != nil {
			fmt.Fprintln(errOut, err)
			return 1
		}
	}
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
	return 0
}

// installTelegram downloads the Telegram program of this version next to this program.
func installTelegram(l string, out io.Writer) (string, error) {
	return "", errors.New(i18n.T(l, "telegram.notinstalled"))
}
