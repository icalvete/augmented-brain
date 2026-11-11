package remote

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/paths"
)

// The watch brings a service back when the server archived its session (Linux only).
//
// Measured on 8-nov and 11-nov with claude 2.1.319 and 2.1.322: a healthy
// `claude remote-control` keeps one child process per session it serves, alive also
// while the session is idle. When the server archives the session (end_session,
// reason archived), the child exits and the server stays up with no child at all,
// "returning to idle": systemd sees a running service and Restart=always never acts,
// and the session is gone from claude.ai and Claude Desktop.
//
// Restarting is not always enough: Claude keeps the session in
// <claude dir>/projects/<folder>/bridge-pointer.json, and a new server adopts it again
// and its child dies at once. Without the pointer it registers anew and gets a new
// session, after the server lets the old registration go (about three minutes; until
// then it exits with "already served", and Restart=always tries again every 30 s).
//
// So the watch works in two steps, a look apart (WatchEvery): a server seen with no
// child twice, with the same pid, is restarted as it is, which brings back a session
// that was not archived; if the new server too is seen twice with no child, the pointer
// is set aside (renamed, never deleted) and it is restarted again.

// WatchName is the name of the watch's service and timer.
const WatchName = "augmented-brain-watch-remote"

// WatchEvery is how often the timer runs the watch.
const WatchEvery = "2min"

// watchFile keeps what the watch saw, in the configuration's state folder.
const watchFile = "remote-watch.json"

// pointerFile is Claude's record of the session a folder's server serves.
const pointerFile = "bridge-pointer.json"

// PointerAside is the name a pointer gets when the watch sets it aside.
const PointerAside = pointerFile + ".archived"

type watchEntry struct {
	// PID is the server last seen with no child; 0 when there is none.
	PID int `json:"pid,omitempty"`
	// Last is what the watch did last to this service: "restart", "pointer" or "".
	Last string `json:"last,omitempty"`
}

// WatchServiceText is the text of the watch's service: one run of `remote watch`, with
// the configuration folder the services were installed from.
func WatchServiceText(self, cfgDir string) string {
	var b strings.Builder
	b.WriteString(Marker + "\n")
	b.WriteString("[Unit]\n")
	b.WriteString("Description=augmented brain: bring back remote access whose session the server archived\n\n")
	b.WriteString("[Service]\nType=oneshot\n")
	fmt.Fprintf(&b, "Environment=%s\n", unitEnv("AB_CONFIG_DIR="+cfgDir))
	fmt.Fprintf(&b, "ExecStart=%s remote watch\n", unitArg(self, true))
	return b.String()
}

// WatchTimerText is the text of the timer that runs the watch every WatchEvery.
func WatchTimerText() string {
	var b strings.Builder
	b.WriteString(Marker + "\n")
	b.WriteString("[Unit]\n")
	b.WriteString("Description=augmented brain: look at remote access every " + WatchEvery + "\n\n")
	b.WriteString("[Timer]\nOnActiveSec=1min\nOnBootSec=3min\nOnUnitActiveSec=" + WatchEvery + "\nAccuracySec=15s\n\n")
	b.WriteString("[Install]\nWantedBy=timers.target\n")
	return b.String()
}

// writeWatch writes the watch's service and timer when they changed.
func (s System) writeWatch(cfgDir string) error {
	for name, text := range map[string]string{
		WatchName + ".service": WatchServiceText(s.Self, cfgDir),
		WatchName + ".timer":   WatchTimerText(),
	} {
		path := filepath.Join(s.UnitDir, name)
		if old, err := os.ReadFile(path); err == nil && string(old) == text {
			continue
		}
		if err := paths.WriteAtomic(path, []byte(text), 0o644); err != nil {
			return err
		}
	}
	return nil
}

// watchInstalled is whether the watch's timer is there, written by Install.
func (s System) watchInstalled() bool {
	data, err := os.ReadFile(filepath.Join(s.UnitDir, WatchName+".timer"))
	return err == nil && strings.HasPrefix(string(data), Marker)
}

// removeWatch stops the timer and removes the watch's files; it says whether there was
// any.
func (s System) removeWatch() (bool, error) {
	found := false
	for _, name := range []string{WatchName + ".timer", WatchName + ".service"} {
		path := filepath.Join(s.UnitDir, name)
		data, err := os.ReadFile(path)
		if err != nil || !strings.HasPrefix(string(data), Marker) {
			continue
		}
		found = true
		if strings.HasSuffix(name, ".timer") {
			_, _ = s.Run("systemctl", "--user", "disable", "--now", name)
		}
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return found, err
		}
	}
	return found, nil
}

// Watch looks once at every installed service and acts as the package comment says. It
// writes a line to out for every service it restarts. A service that is not running is
// left to systemd.
func (s System) Watch(c *config.Config, out io.Writer) error {
	if s.GOOS != "linux" || s.Children == nil {
		return ErrUnsupported
	}
	l := c.Language
	statePath := filepath.Join(c.Dir, "state", watchFile)
	seen := map[string]watchEntry{}
	if data, err := os.ReadFile(statePath); err == nil {
		_ = json.Unmarshal(data, &seen)
	}
	units, err := s.installedUnits()
	if err != nil {
		return err
	}
	next := map[string]watchEntry{}
	var errs []error
	for _, unit := range units {
		pid, active := s.mainPID(unit)
		e := seen[unit]
		switch {
		case !active || pid == 0:
			next[unit] = watchEntry{Last: e.Last}
			continue
		case len(s.Children(pid)) > 0:
			continue // healthy: forget what was done before
		case e.PID != pid:
			next[unit] = watchEntry{PID: pid, Last: e.Last}
			continue
		}
		if e.Last == "restart" {
			pointer := s.findPointer(pid)
			if pointer == "" {
				fmt.Fprintln(out, i18n.T(l, "remote.watch.nopointer", unit))
			} else {
				fmt.Fprintln(out, i18n.T(l, "remote.watch.pointer", unit, pointer, PointerAside))
			}
			if err := s.restartWithout(unit, pointer); err != nil {
				errs = append(errs, err)
			}
			next[unit] = watchEntry{Last: "pointer"}
			continue
		}
		fmt.Fprintln(out, i18n.T(l, "remote.watch.restart", unit))
		if _, err := s.Run("systemctl", "--user", "restart", unit); err != nil {
			errs = append(errs, errors.New(i18n.T(l, "remote.systemctl", "restart "+unit, err)))
		}
		next[unit] = watchEntry{Last: "restart"}
	}
	data, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(statePath), 0o700); err != nil {
		return err
	}
	if err := paths.WriteAtomic(statePath, append(data, '\n'), 0o600); err != nil {
		return err
	}
	return errors.Join(errs...)
}

// restartWithout stops the service, sets its pointer aside (when there is one) and
// starts it again. Stopped first: a server writes its pointer while it runs.
func (s System) restartWithout(unit, pointer string) error {
	if _, err := s.Run("systemctl", "--user", "stop", unit); err != nil {
		return fmt.Errorf("systemctl --user stop %s: %w", unit, err)
	}
	var moveErr error
	if pointer != "" {
		moveErr = paths.Rename(pointer, filepath.Join(filepath.Dir(pointer), PointerAside))
	}
	if _, err := s.Run("systemctl", "--user", "start", unit); err != nil {
		return errors.Join(moveErr, fmt.Errorf("systemctl --user start %s: %w", unit, err))
	}
	return moveErr
}

// mainPID is the service's main process, and whether systemd says it is active.
func (s System) mainPID(unit string) (int, bool) {
	text, err := s.Run("systemctl", "--user", "show", "--property=MainPID,ActiveState", unit)
	if err != nil {
		return 0, false
	}
	pid, active := 0, false
	for _, line := range strings.Split(text, "\n") {
		k, v, _ := strings.Cut(strings.TrimSpace(line), "=")
		switch k {
		case "MainPID":
			pid, _ = strconv.Atoi(v)
		case "ActiveState":
			active = v == "active"
		}
	}
	return pid, active
}

// findPointer is the pointer of the server with this pid: the one in Claude's projects
// that names it. Found by its pid, not by the folder's name, whose spelling is Claude's.
func (s System) findPointer(pid int) string {
	if s.ClaudeDir == "" {
		return ""
	}
	files, _ := filepath.Glob(filepath.Join(s.ClaudeDir, "projects", "*", pointerFile))
	for _, f := range files {
		var p struct {
			PID int `json:"pid"`
		}
		if data, err := os.ReadFile(f); err == nil && json.Unmarshal(data, &p) == nil && p.PID == pid {
			return f
		}
	}
	return ""
}

// procChildren are the processes whose parent is pid, read from /proc.
func procChildren(pid int) []int {
	dirs, _ := os.ReadDir("/proc")
	var out []int
	for _, d := range dirs {
		n, err := strconv.Atoi(d.Name())
		if err != nil {
			continue
		}
		data, err := os.ReadFile(filepath.Join("/proc", d.Name(), "stat"))
		if err != nil {
			continue
		}
		// pid (comm) state ppid …: comm may hold spaces and parentheses.
		i := strings.LastIndexByte(string(data), ')')
		if i < 0 {
			continue
		}
		f := strings.Fields(string(data[i+1:]))
		if len(f) > 1 {
			if pp, _ := strconv.Atoi(f[1]); pp == pid {
				out = append(out, n)
			}
		}
	}
	return out
}
