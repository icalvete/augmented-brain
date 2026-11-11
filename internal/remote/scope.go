package remote

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/i18n"
)

// Claude's daemon and the remote services (Linux only).
//
// A session served by a remote service that runs `claude --bg` starts Claude's daemon
// when none runs. The daemon leaves its parent (systemd --user takes it), but it stays
// in the service's control group, and so does every background session it starts. A
// service is stopped by killing its whole group (KillMode=control-group): restarting a
// remote service then took the daemon and every task's session with it (seen on
// 11-nov).
//
// So two things keep it out:
//   - session start and the daemon's own start run in a scope of their own
//     (systemd-run --user --scope, OwnScope), so the daemon is born outside;
//   - before a remote service is stopped or restarted, and on every look of the watch,
//     whatever in its group does not come from the server (its main process and what
//     hangs from it) is moved to a new scope of its own, through systemd's
//     StartTransientUnit with the processes' ids (measured on 11-nov: a process moved
//     that way outlives a stop of the service). That also keeps anything else a
//     session left running in the background, which the restart used to kill too.

// keptPrefix names the scopes processes are moved to.
const keptPrefix = "augmented-brain-kept-"

// errNotMoved says some processes could not leave a service's group.
var errNotMoved = errors.New("could not move them out of the service")

// scoped is the command that runs name with args in a scope of its own, when the
// system has OwnScope; otherwise name and args as they are.
func (s System) scoped(name string, args ...string) (string, []string) {
	if !s.OwnScope {
		return name, args
	}
	return "systemd-run", append([]string{"--user", "--scope", "--collect", "--quiet", "--", name}, args...)
}

// hasOwnScope is whether systemd-run can give this user a scope: Linux, the program
// there, and the user manager's bus.
func hasOwnScope(look func(string) (string, error)) bool {
	if _, err := look("systemd-run"); err != nil {
		return false
	}
	dir := os.Getenv("XDG_RUNTIME_DIR")
	if dir == "" {
		dir = filepath.Join("/run/user", strconv.Itoa(os.Getuid()))
	}
	_, err := os.Stat(filepath.Join(dir, "bus"))
	return err == nil
}

// unitGroup is the control group of a unit, as systemd names it ("" when it has none).
func (s System) unitGroup(unit string) string {
	text, err := s.Run("systemctl", "--user", "show", "--property=ControlGroup", unit)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(text, "\n") {
		if v, ok := strings.CutPrefix(strings.TrimSpace(line), "ControlGroup="); ok {
			return v
		}
	}
	return ""
}

// strays are the processes in the unit's group that do not come from its main process.
func (s System) strays(group string, main int) []int {
	procs := s.GroupProcs(group)
	if len(procs) == 0 {
		return nil
	}
	keep := map[int]bool{}
	if main > 0 {
		todo := []int{main}
		for len(todo) > 0 {
			p := todo[len(todo)-1]
			todo = todo[:len(todo)-1]
			if keep[p] {
				continue
			}
			keep[p] = true
			if s.Children != nil {
				todo = append(todo, s.Children(p)...)
			}
		}
	}
	var out []int
	for _, p := range procs {
		if !keep[p] {
			out = append(out, p)
		}
	}
	slices.Sort(out)
	return out
}

// KeepOutside moves out of a remote service's group whatever does not come from its
// server, so stopping the service does not kill it (Claude's daemon and its sessions
// above all). It says how many processes it moved. Nothing to do, or no way to look
// (not Linux, GroupProcs nil, a unit not running), is 0 and no error. A process that
// starts while they move is caught by the next round; after three, what is left is an
// error, and the caller must not stop the service.
func (s System) KeepOutside(unit string) (int, error) {
	if s.GOOS != "linux" || s.GroupProcs == nil {
		return 0, nil
	}
	main, active := s.mainPID(unit)
	if !active {
		return 0, nil
	}
	group := s.unitGroup(unit)
	if group == "" {
		return 0, nil
	}
	moved := 0
	var lastErr error
	for range 3 {
		pids := s.strays(group, main)
		if len(pids) == 0 {
			return moved, nil
		}
		args := []string{"--user", "call", "org.freedesktop.systemd1", "/org/freedesktop/systemd1",
			"org.freedesktop.systemd1.Manager", "StartTransientUnit", "ssa(sv)a(sa(sv))",
			keptPrefix + strconv.Itoa(pids[0]) + ".scope", "fail", "2",
			"Description", "s", "Kept by augmented brain when it moved them out of " + unit,
			"PIDs", "au", strconv.Itoa(len(pids))}
		for _, p := range pids {
			args = append(args, strconv.Itoa(p))
		}
		args = append(args, "0")
		if out, err := s.Run("busctl", args...); err != nil {
			lastErr = errors.New(strings.TrimSpace(out + " " + err.Error()))
			continue
		}
		moved += len(pids)
		// systemd moves them when the scope's job runs, a moment after the call.
		for range 20 {
			if !slices.ContainsFunc(s.strays(group, main), func(p int) bool { return slices.Contains(pids, p) }) {
				break
			}
			s.sleep(50 * time.Millisecond)
		}
	}
	if pids := s.strays(group, main); len(pids) > 0 {
		if lastErr != nil {
			return moved, errors.Join(errNotMoved, lastErr)
		}
		return moved, errNotMoved
	}
	return moved, nil
}

// keepAll runs KeepOutside on every unit, and says which one it could not empty.
func (s System) keepAll(c *config.Config, units []string) error {
	for _, unit := range units {
		if _, err := s.KeepOutside(unit); err != nil {
			return errors.New(i18n.T(c.Language, "remote.keep.refused", unit, err))
		}
	}
	return nil
}

// procGroup reads the processes of a control group of this machine.
func procGroup(group string) []int {
	data, err := os.ReadFile(filepath.Join("/sys/fs/cgroup", group, "cgroup.procs"))
	if err != nil {
		return nil
	}
	var out []int
	for _, f := range strings.Fields(string(data)) {
		if n, err := strconv.Atoi(f); err == nil {
			out = append(out, n)
		}
	}
	return out
}
