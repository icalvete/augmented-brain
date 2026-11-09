package remote

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/paths"
)

// macOS: one LaunchAgent per space, ~/Library/LaunchAgents/<label>.plist, loaded into
// the user's own launchd domain (gui/<uid>) with `launchctl bootstrap` and taken out
// with `launchctl bootout`. launchd starts it when the user logs in (RunAtLoad), starts
// it again 30 seconds after it stops (KeepAlive, ThrottleInterval), and writes what it
// says on stderr to ~/Library/Logs/augmented-brain/augmented-brain-remote-<space>.log;
// stdout, where claude redraws its screen every second, goes to /dev/null. A
// LaunchAgent runs only while the user is logged in: macOS has no "linger" for an
// ordinary user (a LaunchDaemon, which runs with nobody logged in, needs root).

const launchdPrefix = "com.augmented-brain.remote."

func launchdLabel(space string) string { return launchdPrefix + space }

// launchdPATH is the PATH of a LaunchAgent after the folders of claude and of this
// program: launchd's own is only /usr/bin:/bin:/usr/sbin:/sbin, without Homebrew.
var launchdPATH = []string{"/opt/homebrew/bin", "/usr/local/bin", "/usr/bin", "/bin", "/usr/sbin", "/sbin"}

// PlistText is the LaunchAgent of a service: claude is the absolute path of the claude
// program, mode its permission mode, self this program's path and log where its errors
// go.
func (svc Service) PlistText(claude, mode, self, log string) string {
	var b strings.Builder
	str := func(v string) string { return "<string>" + xmlEscape(v) + "</string>" }
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(markerComment + "\n")
	b.WriteString(`<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">` + "\n")
	b.WriteString(`<plist version="1.0">` + "\n<dict>\n")
	fmt.Fprintf(&b, "\t<key>Label</key>\n\t%s\n", str(launchdLabel(svc.Space)))
	b.WriteString("\t<key>ProgramArguments</key>\n\t<array>\n")
	for _, a := range []string{claude, "remote-control", "--name", svc.Name, "--permission-mode", mode} {
		fmt.Fprintf(&b, "\t\t%s\n", str(a))
	}
	b.WriteString("\t</array>\n")
	fmt.Fprintf(&b, "\t<key>WorkingDirectory</key>\n\t%s\n", str(svc.Folder))
	fmt.Fprintf(&b, "\t<key>EnvironmentVariables</key>\n\t<dict>\n\t\t<key>PATH</key>\n\t\t%s\n\t</dict>\n",
		str(servicePath(":", claude, self, launchdPATH...)))
	b.WriteString("\t<key>RunAtLoad</key>\n\t<true/>\n")
	b.WriteString("\t<key>KeepAlive</key>\n\t<true/>\n")
	b.WriteString("\t<key>ThrottleInterval</key>\n\t<integer>30</integer>\n")
	fmt.Fprintf(&b, "\t<key>StandardOutPath</key>\n\t%s\n", str("/dev/null"))
	fmt.Fprintf(&b, "\t<key>StandardErrorPath</key>\n\t%s\n", str(log))
	b.WriteString("</dict>\n</plist>\n")
	return b.String()
}

func xmlEscape(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&apos;").Replace(s)
}

func (s System) domain() string { return "gui/" + strconv.Itoa(s.UID) }

// loaded is whether launchd has the agent now.
func (s System) loaded(label string) (string, bool) {
	out, err := s.Run("launchctl", "print", s.domain()+"/"+label)
	return out, err == nil
}

func (s System) installLaunchd(c *config.Config, set Settings) (*InstallResult, error) {
	res, err := s.prepare(c, set)
	if err != nil {
		return nil, err
	}
	// launchctl is part of macOS; without it (or without a login session to load the
	// agents into) nothing is written.
	if _, err := s.LookPath("launchctl"); err != nil {
		return nil, errors.New(i18n.T(c.Language, "remote.nolaunchctl", err))
	}
	if err := s.folders(c, res); err != nil {
		return nil, err
	}
	for _, dir := range []string{s.UnitDir, s.LogDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}
	for _, svc := range res.Services {
		label := launchdLabel(svc.Space)
		path := filepath.Join(s.UnitDir, label+".plist")
		text := svc.PlistText(res.Claude, set.PermissionMode, s.Self, s.logFile(svc.Space))
		old, readErr := os.ReadFile(path)
		changed := readErr != nil || string(old) != text
		if changed {
			if err := paths.WriteAtomic(path, []byte(text), 0o644); err != nil {
				return nil, err
			}
		}
		_, isLoaded := s.loaded(label)
		if isLoaded && !changed {
			continue
		}
		if isLoaded { // a loaded agent keeps its old command until it is loaded again
			if _, err := s.Run("launchctl", "bootout", s.domain()+"/"+label); err != nil {
				return nil, errors.New(i18n.T(c.Language, "remote.launchctl", "bootout "+label, err))
			}
		}
		// An agent the user disabled (launchctl disable) refuses to load.
		_, _ = s.Run("launchctl", "enable", s.domain()+"/"+label)
		if _, err := s.Run("launchctl", "bootstrap", s.domain(), path); err != nil {
			return nil, errors.New(i18n.T(c.Language, "remote.launchctl", "bootstrap "+label, err))
		}
	}
	return s.finish(c, res)
}

func (s System) uninstallLaunchd(c *config.Config) ([]string, error) {
	var removed []string
	for _, f := range s.installedFiles(launchdPrefix, ".plist", func(b []byte) string { return string(b) }) {
		label := strings.TrimSuffix(filepath.Base(f), ".plist")
		if _, isLoaded := s.loaded(label); isLoaded {
			if _, err := s.Run("launchctl", "bootout", s.domain()+"/"+label); err != nil {
				if _, still := s.loaded(label); still {
					return removed, errors.New(i18n.T(c.Language, "remote.launchctl", "bootout "+label, err))
				}
			}
		}
		if err := os.Remove(f); err != nil && !errors.Is(err, os.ErrNotExist) {
			return removed, err
		}
		removed = append(removed, label)
	}
	return removed, SetEnabled(c.Dir, false)
}

func (s System) statusLaunchd(c *config.Config, set Settings) (*Status, error) {
	trusted := s.trustedFolders()
	st := &Status{Linger: LingerNone}
	// print-disabled lists `"label" => disabled` (older systems: `=> true`).
	disabled, _ := s.Run("launchctl", "print-disabled", s.domain())
	for _, svc := range s.services(c, set) {
		x := SpaceStatus{Service: svc, Trusted: trusted(svc.Folder), Log: s.logFile(svc.Space)}
		data, err := os.ReadFile(filepath.Join(s.UnitDir, svc.Unit+".plist"))
		if err == nil && strings.Contains(string(data), markerComment) {
			x.Installed = true
			x.Active, x.Enabled = "inactive", "enabled"
			if out, ok := s.loaded(svc.Unit); ok {
				x.Active = launchdState(out)
			}
			for _, l := range strings.Split(disabled, "\n") {
				if strings.Contains(l, `"`+svc.Unit+`"`) && (strings.Contains(l, "=> disabled") || strings.Contains(l, "=> true")) {
					x.Enabled = "disabled"
				}
			}
		}
		st.Spaces = append(st.Spaces, x)
	}
	return st, nil
}

// launchdState reads `launchctl print`: "state = running" is active; anything else
// (launchd waiting to start it again after it stopped) is inactive.
func launchdState(out string) string {
	for _, l := range strings.Split(out, "\n") {
		l = strings.TrimSpace(l)
		if v, ok := strings.CutPrefix(l, "state = "); ok {
			if v == "running" {
				return "active"
			}
			return "inactive"
		}
	}
	return "inactive"
}
