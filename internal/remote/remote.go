// Package remote keeps a Claude session reachable from another computer or the phone:
// one service per space that runs `claude remote-control`, so that after a reboot the
// machine can be reached again from Claude Desktop or claude.ai. The service is a
// systemd user unit on Linux, a LaunchAgent on macOS (launchd.go) and a task of the
// Task Scheduler on Windows (tasks.go).
//
// Each service starts in a work folder of its space: the one remote.toml names under
// folders, or else the space's first work folder, where the user already works (and so
// Claude usually trusts it already). A space with no work folder gets one of its own,
// <configuration>/remote/<space>, made a work folder of the space. Every session the
// server opens starts there, so the space of the session (and its rules) is the one the
// service was made for.
//
// Everything the package touches outside the configuration folder goes through System,
// so tests use fakes and write only to temporary folders.
package remote

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/detach"
	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/paths"
)

// ErrUnsupported means this system has no way to keep the service (a system other than
// Linux, macOS and Windows; for `session start`, anything but Linux); the commands say so.
var ErrUnsupported = errors.New("remote access is not available on this system yet")

// Marker is the line every unit starts with. Uninstall removes only units that have it.
const Marker = "# " + markerWords

const unitPrefix = "augmented-brain-remote-"

// Runner runs a command and returns what it printed on the standard output. A failing
// command is an error that holds its standard error; a missing program wraps
// exec.ErrNotFound.
type Runner func(name string, args ...string) (string, error)

// System is everything outside the configuration folder the package depends on.
type System struct {
	Run Runner
	// RunIn runs a command in a folder, with this program's environment but TaskEnv:
	// how a launched session starts.
	RunIn    func(dir string, name string, args ...string) (string, error)
	LookPath func(string) (string, error)
	Hostname func() (string, error)
	GOOS     string
	// User is who `loginctl enable-linger` is for; on Windows, who the task runs as
	// (DOMAIN\user).
	User string
	// UID is the user's number on macOS: launchd's domain is gui/<uid>.
	UID int
	// UnitDir is where the services are written: systemd's folder of the user's units
	// (~/.config/systemd/user), ~/Library/LaunchAgents on macOS, and on Windows a folder
	// of augmented brain's own that keeps a copy of each task as it was registered.
	UnitDir string
	// LogDir is where a service's errors go on macOS and Windows (on Linux, the journal).
	LogDir string
	// SystemRoot is Windows' own folder (C:\Windows), where cmd.exe and conhost.exe are.
	SystemRoot string
	// ClaudeJSON is Claude Code's own file, read (never written) to see which folders
	// the user trusts.
	ClaudeJSON string
	// ClaudeDir is Claude Code's folder ($CLAUDE_CONFIG_DIR, or ~/.claude): its
	// sessions' transcripts are under projects/, read (never written) to see how a
	// launched session's turn ended, and so is a remote server's bridge-pointer.json,
	// which the watch may rename (watch.go).
	ClaudeDir string
	// Sleep waits between looks at a launched session; nil is time.Sleep.
	Sleep func(time.Duration)
	// StartDaemon starts Claude's daemon in the background and returns at once; nil
	// never starts it.
	StartDaemon func(claude string) error
	// Home is the user's home folder, where Claude Code's installer puts claude when the
	// PATH does not lead to it.
	Home string
	// Self is this program, augmented-brain: its folder goes into the PATH of every unit,
	// so a remote session finds it, and the watch's service runs it.
	Self string
	// Children are the processes whose parent is pid (the watch); nil is none.
	Children func(pid int) []int
	// GroupProcs are the processes of a control group (scope.go); nil never looks.
	GroupProcs func(group string) []int
	// OwnScope runs session start and Claude's daemon in a scope of their own
	// (systemd-run), so the daemon never lives inside a remote service (scope.go).
	OwnScope bool
	// Parent is the parent of a process (restart.go); nil, or 0, when it cannot tell.
	Parent func(pid int) int
	// PID is this process: a restart never touches the session or service it runs in.
	PID int
	// Now is the time, for how long ago the watch last looked; nil is time.Now.
	Now func() time.Time
	// MCPServers are the augmented-brain mcp processes running, and StopServer ends one
	// (restart.go): how a Claude Desktop session gets the program of now. nil, either
	// of them, leaves such a session alone.
	MCPServers func() []int
	StopServer func(pid int) error
}

// NewSystem is the real system.
func NewSystem() (System, error) {
	home, err := paths.Home()
	if err != nil {
		return System{}, err
	}
	conf := os.Getenv("XDG_CONFIG_HOME")
	if conf == "" {
		conf = filepath.Join(home, ".config")
	}
	claude := filepath.Join(home, ".claude.json")
	claudeDir := filepath.Join(home, ".claude")
	if d := os.Getenv("CLAUDE_CONFIG_DIR"); d != "" {
		claude = filepath.Join(d, ".claude.json")
		claudeDir = d
	}
	name := os.Getenv("USER")
	if u, err := user.Current(); err == nil {
		name = u.Username
	}
	self, _ := os.Executable()
	s := System{Run: realRun, RunIn: realRunIn, LookPath: exec.LookPath, Hostname: os.Hostname, GOOS: runtime.GOOS,
		User: name, UID: os.Getuid(), UnitDir: filepath.Join(conf, "systemd", "user"), ClaudeJSON: claude, ClaudeDir: claudeDir,
		Home: home, Self: self, Children: procChildren, Parent: procParent, PID: os.Getpid(),
		MCPServers: procMCPServers, StopServer: stopMCPServer}
	if runtime.GOOS == "linux" {
		s.GroupProcs = procGroup
		s.OwnScope = hasOwnScope(exec.LookPath)
	}
	scope := s.OwnScope
	s.StartDaemon = func(claude string) error { return realStartDaemon(claude, scope) }
	switch runtime.GOOS {
	case "darwin":
		s.UnitDir = filepath.Join(home, "Library", "LaunchAgents")
		s.LogDir = filepath.Join(home, "Library", "Logs", "augmented-brain")
	case "windows":
		local := os.Getenv("LOCALAPPDATA")
		if local == "" {
			local = filepath.Join(home, "AppData", "Local")
		}
		s.UnitDir = filepath.Join(local, "augmented-brain", "remote")
		s.LogDir = s.UnitDir
		s.SystemRoot = os.Getenv("SystemRoot")
	}
	return s, nil
}

func realRunIn(dir string, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, TaskEnv+"=") {
			cmd.Env = append(cmd.Env, e)
		}
	}
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// realStartDaemon starts Claude's daemon as `claude --bg` does when none runs: transient,
// so it exits on its own about a minute after its last client. Every session Claude
// starts later comes from it and gets its environment, so it gets neither TaskEnv nor
// AB_CONFIG_DIR (the folder of whoever closed a task, maybe a test's).
func realStartDaemon(claude string, ownScope bool) error {
	name, args := System{OwnScope: ownScope}.scoped(claude, "daemon", "run", "--origin", "transient")
	cmd := exec.Command(name, args...)
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, TaskEnv+"=") && !strings.HasPrefix(e, "AB_CONFIG_DIR=") {
			cmd.Env = append(cmd.Env, e)
		}
	}
	cmd.Stdin, cmd.Stdout, cmd.Stderr = nil, nil, nil
	detach.Cmd(cmd)
	started, err := detach.Start(cmd)
	if err != nil {
		return err
	}
	return started.Process.Release()
}

func realRun(name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	if err := cmd.Run(); err != nil {
		if msg := strings.TrimSpace(errOut.String()); msg != "" {
			err = fmt.Errorf("%w: %s", err, msg)
		}
		return out.String(), err
	}
	return out.String(), nil
}

// Service is what one space gets.
type Service struct {
	Space  string
	Name   string // the session's name: <prefix>-<space>
	Folder string // where the server starts, so where every session starts
	Unit   string // the unit's file name
	// Own is true when Folder is the service's own folder, <configuration>/remote/<space>,
	// because the space has no work folder: install creates it and adds it to the space.
	Own bool
}

// Services are the services the settings ask for, in the order of the spaces.
func Services(c *config.Config, set Settings) []Service {
	var out []Service
	for _, space := range set.Spaces {
		svc := Service{Space: space, Name: set.NamePrefix + "-" + space, Unit: unitPrefix + space + ".service"}
		switch sp := c.Spaces[space]; {
		case set.Folders[space] != "":
			svc.Folder = set.Folders[space]
		case sp != nil && len(sp.WorkFolders) > 0:
			svc.Folder = sp.WorkFolders[0]
		default:
			svc.Folder, svc.Own = filepath.Join(c.Dir, "remote", space), true
		}
		out = append(out, svc)
	}
	return out
}

// Unit is the text of a service's unit file: claude is the absolute path of the claude
// program and mode its permission mode. Its errors go to the journal, so a service that
// fails at boot can say why (`journalctl --user -u <unit>`; on 7-nov one failed with
// status 1 and nothing to read): claude remote-control writes them on stderr (measured
// on 2.1.319: an untrusted folder, a question no one answered), and its screen and
// prompts on stdout, which still goes nowhere, since it redraws every second. In case a
// later version also writes on stderr all the time, the journal takes at most 200 lines
// a minute from it.
//
// path is the PATH the service runs with (UnitPath): systemd's user manager does not
// read the shell's profile, so without it a remote session found neither claude nor
// augmented-brain (measured on 9-nov, after a reboot).
func (s Service) UnitText(claude, mode, path string) string {
	var b strings.Builder
	b.WriteString(Marker + "\n")
	b.WriteString("[Unit]\n")
	fmt.Fprintf(&b, "Description=augmented brain: Claude remote access, space %s\n", s.Space)
	b.WriteString("After=network-online.target\nWants=network-online.target\n\n")
	b.WriteString("[Service]\n")
	fmt.Fprintf(&b, "WorkingDirectory=%s\n", unitArg(s.Folder, false))
	fmt.Fprintf(&b, "Environment=%s\n", unitEnv("PATH="+path))
	fmt.Fprintf(&b, "ExecStart=%s remote-control --name %s --permission-mode %s\n", unitArg(claude, true), s.Name, mode)
	b.WriteString("Restart=always\nRestartSec=30\nStandardOutput=null\nStandardError=journal\n")
	fmt.Fprintf(&b, "SyslogIdentifier=augmented-brain-remote-%s\n", s.Space)
	b.WriteString("LogRateLimitIntervalSec=60\nLogRateLimitBurst=200\n\n")
	b.WriteString("[Install]\nWantedBy=default.target\n")
	return b.String()
}

// unitArg writes a path so systemd reads it back as is: % and $ doubled, and in quotes
// when it holds spaces (a command line splits on them; WorkingDirectory does not).
func unitArg(p string, command bool) string {
	p = strings.ReplaceAll(p, "%", "%%")
	if command {
		p = strings.ReplaceAll(p, "$", "$$")
		if strings.ContainsAny(p, " \t\"'\\;") {
			p = `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(p) + `"`
		}
	}
	return p
}

// unitEnv writes one VAR=value assignment for Environment=: % doubled, and in quotes
// when it holds a space, a quote or a backslash (systemd does not expand $ there).
func unitEnv(kv string) string {
	kv = strings.ReplaceAll(kv, "%", "%%")
	if strings.ContainsAny(kv, " \t\"'\\") {
		kv = `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(kv) + `"`
	}
	return kv
}

// systemPath is the PATH systemd itself gives a service, the tail of UnitPath.
var systemPath = []string{"/usr/local/sbin", "/usr/local/bin", "/usr/sbin", "/usr/bin", "/sbin", "/bin"}

// UnitPath is the PATH of a service or a launched session: claude's folder and this
// program's first, then the system's, each once.
func (s System) UnitPath(claude string) string {
	var dirs []string
	seen := map[string]bool{}
	add := func(d string) {
		if d != "" && d != "." && !seen[d] {
			seen[d] = true
			dirs = append(dirs, d)
		}
	}
	add(filepath.Dir(claude))
	if s.Self != "" {
		add(filepath.Dir(s.Self))
	}
	for _, d := range systemPath {
		add(d)
	}
	return strings.Join(dirs, string(os.PathListSeparator))
}

// FindClaude is the claude program: the one in the PATH, or else where Claude Code's
// installers put it (~/.local/bin, ~/.claude/local) or next to this program. A session
// started by a service, or by systemd-run, does not always have the shell's PATH.
func (s System) FindClaude() (string, error) {
	if p, err := s.LookPath("claude"); err == nil {
		return filepath.Abs(p)
	}
	var cands []string
	if s.Home != "" {
		cands = append(cands, filepath.Join(s.Home, ".local", "bin", "claude"), filepath.Join(s.Home, ".claude", "local", "claude"))
	}
	if s.Self != "" {
		cands = append(cands, filepath.Join(filepath.Dir(s.Self), "claude"))
	}
	for _, c := range cands {
		if fi, err := os.Stat(c); err == nil && fi.Mode().IsRegular() && fi.Mode()&0o111 != 0 {
			return c, nil
		}
	}
	return "", exec.ErrNotFound
}

// InstallResult is what Install did and what is left for the user.
type InstallResult struct {
	Services []Service
	Claude   string
	// Untrusted are the folders Claude Code does not trust yet: its server refuses to
	// start in them until the user runs claude there once.
	Untrusted []Service
	Linger    Linger
	// Added are the folders put into a space's work_folders just now.
	Added []Service
	// Watch is whether the watch (watch.go) was installed and started.
	Watch bool
}

// Install sets remote access up, and is safe to repeat: it checks each service's folder
// exists (or, for a space with no work folder, creates its own folder and makes it a
// work folder of the space), writes the unit, reloads systemd and
// enables and starts the service. It does not touch Claude's own files, and never runs
// loginctl enable-linger, which is for root: the result says what to run.
func (s System) Install(c *config.Config, set Settings) (*InstallResult, error) {
	switch s.GOOS {
	case "darwin":
		return s.installLaunchd(c, set)
	case "windows":
		return s.installTasks(c, set)
	case "linux":
	default:
		return nil, ErrUnsupported
	}
	if len(set.Spaces) == 0 {
		return nil, errors.New(i18n.T(c.Language, "remote.nospaces", filepath.Join(c.Dir, SettingsFile)))
	}
	claude, err := s.FindClaude()
	if err != nil {
		return nil, errors.New(i18n.T(c.Language, "remote.noclaude"))
	}
	// Ask systemd's user manager first: without one (a container, WSL without systemd,
	// a session with no user bus) the units would be written, enabled set, and nothing
	// would ever run them.
	if _, err := s.Run("systemctl", "--user", "show-environment"); err != nil {
		return nil, errors.New(i18n.T(c.Language, "remote.nosystemd", err))
	}
	res := &InstallResult{Services: Services(c, set), Claude: claude}
	// Before any unit is written: a unit written and not restarted would look current to
	// the next install, which then would not restart it either.
	units := make([]string, 0, len(res.Services))
	for _, svc := range res.Services {
		units = append(units, svc.Unit)
	}
	if err := s.keepAll(c, units); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(s.UnitDir, 0o755); err != nil {
		return nil, err
	}
	var restart []string
	for _, svc := range res.Services {
		if svc.Own {
			if err := os.MkdirAll(svc.Folder, 0o700); err != nil {
				return nil, err
			}
			added, err := addWorkFolder(c, svc)
			if err != nil {
				return nil, err
			}
			if added {
				res.Added = append(res.Added, svc)
			}
		} else if fi, err := os.Stat(svc.Folder); err != nil || !fi.IsDir() {
			return nil, errors.New(i18n.T(c.Language, "remote.nofolder", paths.Tilde(svc.Folder), svc.Space))
		}
		path := filepath.Join(s.UnitDir, svc.Unit)
		text := svc.UnitText(claude, set.PermissionMode, s.UnitPath(claude))
		old, readErr := os.ReadFile(path)
		if readErr != nil || string(old) != text {
			if err := paths.WriteAtomic(path, []byte(text), 0o644); err != nil {
				return nil, err
			}
			if readErr == nil {
				restart = append(restart, svc.Unit) // a running service keeps its old command
			}
		}
	}
	// The watch brings a service back when the server archives its session (watch.go).
	watch := s.Self != ""
	if watch {
		if err := s.writeWatch(c.Dir); err != nil {
			return nil, err
		}
	}
	if _, err := s.Run("systemctl", "--user", "daemon-reload"); err != nil {
		return nil, errors.New(i18n.T(c.Language, "remote.systemctl", "daemon-reload", err))
	}
	for _, svc := range res.Services {
		if _, err := s.Run("systemctl", "--user", "enable", "--now", svc.Unit); err != nil {
			return nil, errors.New(i18n.T(c.Language, "remote.systemctl", "enable --now "+svc.Unit, err))
		}
	}
	if watch {
		if _, err := s.Run("systemctl", "--user", "enable", "--now", WatchName+".timer"); err != nil {
			return nil, errors.New(i18n.T(c.Language, "remote.systemctl", "enable --now "+WatchName+".timer", err))
		}
	}
	for _, unit := range restart {
		if _, err := s.KeepOutside(unit); err != nil {
			return nil, errors.New(i18n.T(c.Language, "remote.keep.refused", unit, err))
		}
		if _, err := s.Run("systemctl", "--user", "restart", unit); err != nil {
			return nil, errors.New(i18n.T(c.Language, "remote.systemctl", "restart "+unit, err))
		}
	}
	// Enabled only once systemd took the services: a failed install does not leave
	// remote access set as on, which doctor would then report as broken.
	if err := SetEnabled(c.Dir, true); err != nil {
		return nil, err
	}
	trusted := s.trustedFolders()
	for _, svc := range res.Services {
		if !trusted(svc.Folder) {
			res.Untrusted = append(res.Untrusted, svc)
		}
	}
	res.Linger = s.linger()
	res.Watch = watch
	return res, nil
}

// addWorkFolder makes the service's folder a work folder of its space, through the
// configuration's line editor; false when it already was.
func addWorkFolder(c *config.Config, svc Service) (bool, error) {
	sp := c.Spaces[svc.Space]
	if sp == nil {
		return false, errors.New(i18n.T(c.Language, "config.nospace", filepath.Join(c.Dir, config.FileName), svc.Space))
	}
	want, err := paths.Resolve(svc.Folder)
	if err != nil {
		return false, err
	}
	for _, f := range sp.WorkFolders {
		if paths.Same(f, want) {
			return false, nil
		}
	}
	list := make([]string, 0, len(sp.WorkFolders)+1)
	for _, f := range sp.WorkFolders {
		list = append(list, paths.Tilde(f))
	}
	list = append(list, paths.Tilde(want))
	if err := config.SetSpaceList(c.Dir, svc.Space, "work_folders", list); err != nil {
		return false, err
	}
	sp.WorkFolders = append(sp.WorkFolders, want)
	return true, nil
}

// Uninstall stops and removes every unit Install wrote (found by their marker, so it
// follows what is on disk, not the settings of today). The folders, the work_folders
// entries and Claude's own files stay. It returns the units removed.
func (s System) Uninstall(c *config.Config) ([]string, error) {
	switch s.GOOS {
	case "darwin":
		return s.uninstallLaunchd(c)
	case "windows":
		return s.uninstallTasks(c)
	case "linux":
	default:
		return nil, ErrUnsupported
	}
	units, _ := s.installedUnits()
	var removed []string
	// Before anything changes: a refusal leaves everything as it was.
	if err := s.keepAll(c, units); err != nil {
		return removed, err
	}
	// The watch first: it would restart what is being stopped.
	watched, err := s.removeWatch()
	if err != nil {
		return removed, err
	}
	if watched {
		removed = append(removed, WatchName+".timer")
	}
	for _, unit := range units {
		if _, err := s.KeepOutside(unit); err != nil {
			return removed, errors.New(i18n.T(c.Language, "remote.keep.refused", unit, err))
		}
		if _, err := s.Run("systemctl", "--user", "disable", "--now", unit); err != nil {
			// Not loaded is fine; still running is not: then the file stays.
			if state, _ := s.Run("systemctl", "--user", "is-active", unit); strings.TrimSpace(state) == "active" {
				return removed, errors.New(i18n.T(c.Language, "remote.systemctl", "disable --now "+unit, err))
			}
		}
		if err := os.Remove(filepath.Join(s.UnitDir, unit)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return removed, err
		}
		removed = append(removed, unit)
	}
	if len(removed) > 0 {
		if _, err := s.Run("systemctl", "--user", "daemon-reload"); err != nil {
			return removed, errors.New(i18n.T(c.Language, "remote.systemctl", "daemon-reload", err))
		}
	}
	return removed, SetEnabled(c.Dir, false)
}

// installedUnits are the units in the user's folder that carry the marker, sorted.
func (s System) installedUnits() ([]string, error) {
	files, err := filepath.Glob(filepath.Join(s.UnitDir, unitPrefix+"*.service"))
	if err != nil {
		return nil, err
	}
	var out []string
	for _, f := range files {
		if data, err := os.ReadFile(f); err == nil && strings.HasPrefix(string(data), Marker) {
			out = append(out, filepath.Base(f))
		}
	}
	sort.Strings(out)
	return out, nil
}

// Linger says whether systemd starts the user's services at boot, without a login.
type Linger int

const (
	LingerUnknown Linger = iota // loginctl is not there
	LingerOn
	LingerOff
	// LingerNone: the system has no such thing (macOS, Windows). The service starts
	// when the user logs in, and only then.
	LingerNone
)

// LingerCommand is what root runs to turn linger on. It is printed, never run.
func (s System) LingerCommand() string { return "sudo loginctl enable-linger " + s.User }

func (s System) linger() Linger {
	out, err := s.Run("loginctl", "show-user", s.User, "--property=Linger")
	if errors.Is(err, exec.ErrNotFound) {
		return LingerUnknown
	}
	// A user with no session and no linger is not known to loginctl: that is "no".
	if strings.TrimSpace(out) == "Linger=yes" {
		return LingerOn
	}
	return LingerOff
}

// SpaceStatus is how one space's service stands.
type SpaceStatus struct {
	Service
	Installed bool   // the unit file exists
	Outdated  bool   // written by an earlier program: its errors still go nowhere
	Active    string // systemctl is-active: active, inactive, failed…; "" if unknown
	Enabled   string // systemctl is-enabled: enabled, disabled…; "" if unknown
	Trusted   bool   // Claude Code trusts the folder
	// Log is the file the service's errors go to (macOS, Windows); "" on Linux, where
	// they go to the journal.
	Log string
}

// Running is whether the service is up.
func (st SpaceStatus) Running() bool { return st.Active == "active" }

// Status is how remote access stands: each space the settings name, and linger.
type Status struct {
	Spaces []SpaceStatus
	Linger Linger
	// Watch is how the watch stands (Linux, watch.go); nil where there is none.
	Watch *WatchStatus
}

// Status reads (and changes nothing): the unit files, systemctl, loginctl and Claude's
// own file for the folders it trusts.
func (s System) Status(c *config.Config, set Settings) (*Status, error) {
	switch s.GOOS {
	case "darwin":
		return s.statusLaunchd(c, set)
	case "windows":
		return s.statusTasks(c, set)
	case "linux":
	default:
		return nil, ErrUnsupported
	}
	trusted := s.trustedFolders()
	st := &Status{Linger: s.linger()}
	for _, svc := range Services(c, set) {
		x := SpaceStatus{Service: svc, Trusted: trusted(svc.Folder)}
		if data, err := os.ReadFile(filepath.Join(s.UnitDir, svc.Unit)); err == nil && strings.HasPrefix(string(data), Marker) {
			x.Installed = true
			// An earlier unit had no PATH (its session found neither claude nor
			// augmented-brain), and before 0.13 its errors went nowhere.
			// Before the watch (0.17.3), nothing brought a service back when the server
			// archived its session.
			x.Outdated = !strings.Contains(string(data), "\nStandardError=journal\n") || !strings.Contains(string(data), "\nEnvironment=") ||
				(s.Self != "" && !s.watchInstalled())
			// is-active and is-enabled exit non-zero for "inactive" and "disabled":
			// what they print is the answer.
			a, _ := s.Run("systemctl", "--user", "is-active", svc.Unit)
			e, _ := s.Run("systemctl", "--user", "is-enabled", svc.Unit)
			x.Active, x.Enabled = strings.TrimSpace(a), strings.TrimSpace(e)
		}
		st.Spaces = append(st.Spaces, x)
	}
	// Self is how Install knows what the watch runs: without it, it writes no watch.
	if s.Self != "" || s.watchInstalled() {
		st.Watch = s.watchStatus()
	}
	return st, nil
}

// trustedFolders reads Claude Code's file and returns whether a folder is trusted, as
// Claude decides: the folder or one of its parents has accepted the trust dialog. The
// file is only read; when it is missing or unreadable, nothing is trusted.
func (s System) trustedFolders() func(string) bool {
	var f struct {
		Projects map[string]struct {
			Accepted bool `json:"hasTrustDialogAccepted"`
		} `json:"projects"`
	}
	if data, err := os.ReadFile(s.ClaudeJSON); err == nil {
		_ = json.Unmarshal(data, &f)
	}
	return func(folder string) bool {
		cands := []string{folder}
		if real, err := paths.Resolve(folder); err == nil && real != folder {
			cands = append(cands, real)
		}
		for _, p := range cands {
			for dir := p; ; dir = filepath.Dir(dir) {
				// Claude Code on Windows writes C:/Users/…, with forward slashes.
				if f.Projects[dir].Accepted || f.Projects[filepath.ToSlash(dir)].Accepted {
					return true
				}
				if filepath.Dir(dir) == dir {
					break
				}
			}
		}
		return false
	}
}
