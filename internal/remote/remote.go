// Package remote keeps a Claude session reachable from another computer or the phone:
// one systemd user service per space that runs `claude remote-control`, so that after a
// reboot the machine can be reached again from Claude Desktop or claude.ai.
//
// Each service starts in a work folder of its space: the one remote.toml names under
// folders, or else the space's first work folder, where the user already works (and so
// Claude usually trusts it already). A space with no work folder gets one of its own,
// <configuration>/remote/<space>, made a work folder of the space. Every session the
// server opens starts there, so the space of the session (and its rules) is the one the
// service was made for.
//
// Linux only for now. Everything the package touches outside the configuration folder
// goes through System, so tests use fakes and write only to temporary folders.
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
	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/paths"
)

// ErrUnsupported means this system has no way to keep the service yet (macOS and
// Windows); the commands say so.
var ErrUnsupported = errors.New("remote access is not available on this system yet")

// Marker is the line every unit starts with. Uninstall removes only units that have it.
const Marker = "# Written by augmented-brain remote install; remove it with augmented-brain remote uninstall."

const unitPrefix = "augmented-brain-remote-"

// Runner runs a command and returns what it printed on the standard output. A failing
// command is an error that holds its standard error; a missing program wraps
// exec.ErrNotFound.
type Runner func(name string, args ...string) (string, error)

// System is everything outside the configuration folder the package depends on.
type System struct {
	Run      Runner
	LookPath func(string) (string, error)
	Hostname func() (string, error)
	GOOS     string
	// User is who `loginctl enable-linger` is for.
	User string
	// UnitDir is where systemd keeps the user's units (~/.config/systemd/user).
	UnitDir string
	// ClaudeJSON is Claude Code's own file, read (never written) to see which folders
	// the user trusts.
	ClaudeJSON string
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
	if d := os.Getenv("CLAUDE_CONFIG_DIR"); d != "" {
		claude = filepath.Join(d, ".claude.json")
	}
	name := os.Getenv("USER")
	if u, err := user.Current(); err == nil {
		name = u.Username
	}
	return System{Run: realRun, LookPath: exec.LookPath, Hostname: os.Hostname, GOOS: runtime.GOOS,
		User: name, UnitDir: filepath.Join(conf, "systemd", "user"), ClaudeJSON: claude}, nil
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
func (s Service) UnitText(claude, mode string) string {
	var b strings.Builder
	b.WriteString(Marker + "\n")
	b.WriteString("[Unit]\n")
	fmt.Fprintf(&b, "Description=augmented brain: Claude remote access, space %s\n", s.Space)
	b.WriteString("After=network-online.target\nWants=network-online.target\n\n")
	b.WriteString("[Service]\n")
	fmt.Fprintf(&b, "WorkingDirectory=%s\n", unitArg(s.Folder, false))
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
}

// Install sets remote access up, and is safe to repeat: it checks each service's folder
// exists (or, for a space with no work folder, creates its own folder and makes it a
// work folder of the space), writes the unit, reloads systemd and
// enables and starts the service. It does not touch Claude's own files, and never runs
// loginctl enable-linger, which is for root: the result says what to run.
func (s System) Install(c *config.Config, set Settings) (*InstallResult, error) {
	if s.GOOS != "linux" {
		return nil, ErrUnsupported
	}
	if len(set.Spaces) == 0 {
		return nil, errors.New(i18n.T(c.Language, "remote.nospaces", filepath.Join(c.Dir, SettingsFile)))
	}
	claude, err := s.LookPath("claude")
	if err != nil {
		return nil, errors.New(i18n.T(c.Language, "remote.noclaude"))
	}
	if claude, err = filepath.Abs(claude); err != nil {
		return nil, err
	}
	// Ask systemd's user manager first: without one (a container, WSL without systemd,
	// a session with no user bus) the units would be written, enabled set, and nothing
	// would ever run them.
	if _, err := s.Run("systemctl", "--user", "show-environment"); err != nil {
		return nil, errors.New(i18n.T(c.Language, "remote.nosystemd", err))
	}
	res := &InstallResult{Services: Services(c, set), Claude: claude}
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
		text := svc.UnitText(claude, set.PermissionMode)
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
	if _, err := s.Run("systemctl", "--user", "daemon-reload"); err != nil {
		return nil, errors.New(i18n.T(c.Language, "remote.systemctl", "daemon-reload", err))
	}
	for _, svc := range res.Services {
		if _, err := s.Run("systemctl", "--user", "enable", "--now", svc.Unit); err != nil {
			return nil, errors.New(i18n.T(c.Language, "remote.systemctl", "enable --now "+svc.Unit, err))
		}
	}
	for _, unit := range restart {
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
	if s.GOOS != "linux" {
		return nil, ErrUnsupported
	}
	units, _ := s.installedUnits()
	var removed []string
	for _, unit := range units {
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
}

// Running is whether the service is up.
func (st SpaceStatus) Running() bool { return st.Active == "active" }

// Status is how remote access stands: each space the settings name, and linger.
type Status struct {
	Spaces []SpaceStatus
	Linger Linger
}

// Status reads (and changes nothing): the unit files, systemctl, loginctl and Claude's
// own file for the folders it trusts.
func (s System) Status(c *config.Config, set Settings) (*Status, error) {
	if s.GOOS != "linux" {
		return nil, ErrUnsupported
	}
	trusted := s.trustedFolders()
	st := &Status{Linger: s.linger()}
	for _, svc := range Services(c, set) {
		x := SpaceStatus{Service: svc, Trusted: trusted(svc.Folder)}
		if data, err := os.ReadFile(filepath.Join(s.UnitDir, svc.Unit)); err == nil && strings.HasPrefix(string(data), Marker) {
			x.Installed = true
			x.Outdated = !strings.Contains(string(data), "\nStandardError=journal\n")
			// is-active and is-enabled exit non-zero for "inactive" and "disabled":
			// what they print is the answer.
			a, _ := s.Run("systemctl", "--user", "is-active", svc.Unit)
			e, _ := s.Run("systemctl", "--user", "is-enabled", svc.Unit)
			x.Active, x.Enabled = strings.TrimSpace(a), strings.TrimSpace(e)
		}
		st.Spaces = append(st.Spaces, x)
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
				if f.Projects[dir].Accepted {
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
