package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/detach"
	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/remote"
	"github.com/icalvete/augmented-brain/internal/state"
	"github.com/icalvete/augmented-brain/internal/version"
)

// RestartLog is where the detached `session restart --when-idle` writes how it went,
// in the configuration's state folder: nobody reads its output otherwise.
const RestartLog = "session-restart.log"

// restartDetached is set in the environment of the detached copy, which does the
// waiting.
const restartDetached = "AB_RESTART_DETACHED"

// restartWait is how long a restart waits for a session to finish its answer.
const restartWait = 30 * time.Minute

func sessionRestartFlags(errOut io.Writer) (*flag.FlagSet, *bool, *bool) {
	fs := flag.NewFlagSet("session restart", flag.ContinueOnError)
	fs.SetOutput(errOut)
	whenIdle := fs.Bool("when-idle", false, "wait until the session has finished its answer (to restart the session you run it from)")
	old := fs.Bool("old", false, "restart every idle session still running an earlier augmented brain (what update does)")
	return fs, whenIdle, old
}

// sessionRestart is `session restart SESSION [--when-idle] | --old`.
func sessionRestart(c *config.Config, sys remote.System, args []string, out, errOut io.Writer) int {
	l := c.Language
	fs, whenIdle, old := sessionRestartFlags(errOut)
	var key string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		key, args = args[0], args[1:]
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if key == "" && fs.NArg() == 1 {
		key = fs.Arg(0)
	} else if fs.NArg() > 0 {
		key = ""
	}
	if (key == "") == !*old || (*old && *whenIdle) || fs.NArg() > 1 {
		fmt.Fprintln(errOut, i18n.T(l, "usage.session"))
		return 2
	}
	claude, err := sys.FindClaude()
	if err != nil {
		return sessionErr(l, err, errOut)
	}
	if *old {
		return restartOld(c, sys, claude, out, errOut)
	}
	se, err := sys.FindSession(claude, c.Dir, key)
	switch {
	case errors.Is(err, remote.ErrNoSession):
		fmt.Fprintln(errOut, i18n.T(l, "restart.notfound", key))
		return 1
	case errors.Is(err, remote.ErrAmbiguous):
		fmt.Fprintln(errOut, i18n.T(l, "restart.ambiguous", key))
		return 1
	case err != nil:
		return sessionErr(l, err, errOut)
	}
	if !*whenIdle {
		return restartSaid(l, sys.Restart(claude, []remote.Session{se}), "", out, errOut)
	}
	if os.Getenv(restartDetached) == "" {
		// The session that runs this is answering until the command returns: the wait
		// goes on in a detached copy, out of any remote service's group.
		logPath, err := detachRestart(c.Dir, sys, se.Key())
		if err != nil {
			return sessionErr(l, err, errOut)
		}
		fmt.Fprintln(out, i18n.T(l, "restart.waiting", se.Label(), int(restartWait.Minutes()), tilde(logPath)))
		return 0
	}
	stamp := time.Now().Format("2006-01-02 15:04:05") + " "
	r := sys.RestartWhenIdle(claude, se, restartWait)
	if errors.Is(r.Err, remote.ErrStillBusy) {
		fmt.Fprintln(out, stamp+i18n.T(l, "restart.stillbusy", se.Label(), int(restartWait.Minutes())))
		return 1
	}
	return restartSaid(l, []remote.RestartResult{r}, stamp, out, errOut)
}

// restartOld restarts the idle sessions still running an earlier augmented brain.
// It exits 0 only when none is left.
func restartOld(c *config.Config, sys remote.System, claude string, out, errOut io.Writer) int {
	l := c.Language
	servers := state.OldServers(c.Dir, version.Version)
	if len(servers) == 0 {
		fmt.Fprintln(out, i18n.T(l, "restart.none"))
		return 0
	}
	list, unknown, err := sys.OldSessions(claude, servers)
	if err != nil {
		return sessionErr(l, err, errOut)
	}
	// Only a task's own session is brought back with respawn: any other background
	// session may have been archived in Claude Desktop, which Claude is not told.
	remote.MarkUnlaunched(c.Dir, list)
	code := 0
	if len(list) > 0 {
		fmt.Fprintln(out, i18n.T(l, "restart.old", len(list)))
		code = restartSaid(l, sys.Restart(claude, list), "  ", out, errOut)
	}
	// A server whose session could not be told (no parent to read, or a session Claude
	// does not list) is left, and said.
	if unknown > 0 {
		fmt.Fprintln(out, i18n.T(l, "restart.unknown", unknown))
		code = 1
	}
	return code
}

// restartSaid writes a line for each result; the exit code is 0 only when every one
// was restarted or no longer runs.
func restartSaid(l string, rs []remote.RestartResult, prefix string, out, errOut io.Writer) int {
	code := 0
	for _, r := range rs {
		se := r.Session
		switch r.Outcome {
		case remote.Restarted:
			fmt.Fprintln(out, prefix+i18n.T(l, "restart.done", se.Label()))
		case remote.ServerRestarted:
			fmt.Fprintln(out, prefix+i18n.T(l, "restart.server", se.Label()))
		case remote.Ended:
			fmt.Fprintln(out, prefix+i18n.T(l, "restart.ended", se.Label()))
		case remote.Gone:
			fmt.Fprintln(out, prefix+i18n.T(l, "restart.gone", se.Label()))
		case remote.Busy:
			fmt.Fprintln(out, prefix+i18n.T(l, "restart.busy", se.Label(), se.Key()))
			code = 1
		case remote.Self:
			fmt.Fprintln(out, prefix+i18n.T(l, "restart.self", se.Label(), se.Key()))
			code = 1
		case remote.SharedBusy:
			fmt.Fprintln(out, prefix+i18n.T(l, "restart.shared", se.Label(), se.Unit))
			code = 1
		case remote.Unsupported:
			fmt.Fprintln(out, prefix+i18n.T(l, "restart.unsupported", se.Label()))
			code = 1
		default:
			fmt.Fprintln(errOut, prefix+i18n.T(l, "restart.failed", se.Label(), r.Err))
			code = 1
		}
	}
	return code
}

// detachRestart starts the copy of this command that waits for the session's answer to
// end, its output appended to RestartLog (started afresh past 64 KB). Tests replace it.
var detachRestart = func(cfgDir string, sys remote.System, key string) (string, error) {
	self, err := os.Executable()
	if err != nil {
		return "", err
	}
	name, args := sys.Scoped(self, "session", "restart", key, "--when-idle")
	cmd := exec.Command(name, args...)
	cmd.Env = append(os.Environ(), "AB_CONFIG_DIR="+cfgDir, restartDetached+"=1")
	cmd.Stdin = nil
	logPath := filepath.Join(cfgDir, state.Dir, RestartLog)
	if fi, err := os.Stat(logPath); err == nil && fi.Size() > 64<<10 {
		_ = os.Remove(logPath)
	}
	if err := os.MkdirAll(filepath.Dir(logPath), 0o700); err == nil {
		if f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600); err == nil {
			defer f.Close() // the child keeps its own copy
			cmd.Stdout, cmd.Stderr = f, f
		}
	}
	detach.Cmd(cmd)
	_, err = detach.Start(cmd)
	return logPath, err
}
