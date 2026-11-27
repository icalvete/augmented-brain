package remote

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/icalvete/augmented-brain/internal/paths"
)

// The Telegram timer runs `augmented-brain telegram check` every few minutes while
// Telegram is connected: it brings the replies tasks wait for (package replywait). It is
// here because this package already writes the user's services; it does not depend on
// remote access being installed. Telegram works on Linux and macOS only, so there is no
// Windows side.

// TelegramTimerName is the name of the systemd service and timer.
const TelegramTimerName = "augmented-brain-telegram-check"

// TelegramTimerLabel is the LaunchAgent's label on macOS.
const TelegramTimerLabel = "com.augmented-brain.telegram-check"

// TelegramEvery is how often the timer runs the check, in minutes.
const TelegramEvery = 5

// ErrNoTimer means this system has no timer augmented brain can set (not Linux with
// systemd, not macOS).
var ErrNoTimer = errors.New("no timer on this system")

// TelegramServiceText is the systemd service: one run of `telegram check`.
func TelegramServiceText(self, cfgDir string) string {
	var b strings.Builder
	b.WriteString(Marker + "\n")
	b.WriteString("[Unit]\n")
	b.WriteString("Description=augmented brain: bring the replies tasks wait for on Telegram\n\n")
	b.WriteString("[Service]\nType=oneshot\n")
	fmt.Fprintf(&b, "Environment=%s\n", unitEnv("AB_CONFIG_DIR="+cfgDir))
	fmt.Fprintf(&b, "ExecStart=%s telegram check\n", unitArg(self, true))
	return b.String()
}

// TelegramTimerText is the systemd timer.
func TelegramTimerText() string {
	every := fmt.Sprintf("%dmin", TelegramEvery)
	var b strings.Builder
	b.WriteString(Marker + "\n")
	b.WriteString("[Unit]\n")
	b.WriteString("Description=augmented brain: look for Telegram replies every " + every + "\n\n")
	b.WriteString("[Timer]\nOnActiveSec=1min\nOnBootSec=3min\nOnUnitActiveSec=" + every + "\nAccuracySec=30s\n\n")
	b.WriteString("[Install]\nWantedBy=timers.target\n")
	return b.String()
}

// TelegramPlistText is the LaunchAgent on macOS: the check every TelegramEvery minutes.
func TelegramPlistText(self, cfgDir, log string) string {
	var b strings.Builder
	str := func(v string) string { return "<string>" + xmlEscape(v) + "</string>" }
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(markerComment + "\n")
	b.WriteString(`<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">` + "\n")
	b.WriteString(`<plist version="1.0">` + "\n<dict>\n")
	fmt.Fprintf(&b, "\t<key>Label</key>\n\t%s\n", str(TelegramTimerLabel))
	b.WriteString("\t<key>ProgramArguments</key>\n\t<array>\n")
	for _, a := range []string{self, "telegram", "check"} {
		fmt.Fprintf(&b, "\t\t%s\n", str(a))
	}
	b.WriteString("\t</array>\n")
	fmt.Fprintf(&b, "\t<key>EnvironmentVariables</key>\n\t<dict>\n\t\t<key>AB_CONFIG_DIR</key>\n\t\t%s\n\t</dict>\n", str(cfgDir))
	fmt.Fprintf(&b, "\t<key>StartInterval</key>\n\t<integer>%d</integer>\n", TelegramEvery*60)
	fmt.Fprintf(&b, "\t<key>StandardOutPath</key>\n\t%s\n", str("/dev/null"))
	fmt.Fprintf(&b, "\t<key>StandardErrorPath</key>\n\t%s\n", str(log))
	b.WriteString("</dict>\n</plist>\n")
	return b.String()
}

func (s System) telegramFiles() []string {
	if s.GOOS == "darwin" {
		return []string{filepath.Join(s.UnitDir, TelegramTimerLabel+".plist")}
	}
	return []string{filepath.Join(s.UnitDir, TelegramTimerName+".timer"), filepath.Join(s.UnitDir, TelegramTimerName+".service")}
}

// writeIfChanged writes a file when its text changed, and says whether it did.
func writeIfChanged(path, text string) (bool, error) {
	if old, err := os.ReadFile(path); err == nil && string(old) == text {
		return false, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	return true, paths.WriteAtomic(path, []byte(text), 0o644)
}

// InstallTelegramTimer sets the timer that runs the Telegram check, from the
// configuration folder cfgDir. Setting it again changes only what changed.
func (s System) InstallTelegramTimer(cfgDir string) error {
	switch s.GOOS {
	case "linux":
		if _, err := s.LookPath("systemctl"); err != nil {
			return ErrNoTimer
		}
		for name, text := range map[string]string{
			TelegramTimerName + ".service": TelegramServiceText(s.Self, cfgDir),
			TelegramTimerName + ".timer":   TelegramTimerText(),
		} {
			if _, err := writeIfChanged(filepath.Join(s.UnitDir, name), text); err != nil {
				return err
			}
		}
		if _, err := s.Run("systemctl", "--user", "daemon-reload"); err != nil {
			return err
		}
		_, err := s.Run("systemctl", "--user", "enable", "--now", TelegramTimerName+".timer")
		return err
	case "darwin":
		if _, err := s.LookPath("launchctl"); err != nil {
			return ErrNoTimer
		}
		if err := os.MkdirAll(s.LogDir, 0o755); err != nil {
			return err
		}
		path := s.telegramFiles()[0]
		changed, err := writeIfChanged(path, TelegramPlistText(s.Self, cfgDir, filepath.Join(s.LogDir, "telegram-check.log")))
		if err != nil {
			return err
		}
		_, isLoaded := s.loaded(TelegramTimerLabel)
		if isLoaded && !changed {
			return nil
		}
		if isLoaded {
			if _, err := s.Run("launchctl", "bootout", s.domain()+"/"+TelegramTimerLabel); err != nil {
				return err
			}
		}
		_, _ = s.Run("launchctl", "enable", s.domain()+"/"+TelegramTimerLabel)
		_, err = s.Run("launchctl", "bootstrap", s.domain(), path)
		return err
	}
	return ErrNoTimer
}

// RemoveTelegramTimer stops the timer and removes its files; it says whether there was
// any. Files without augmented brain's mark are not its own, and stay.
func (s System) RemoveTelegramTimer() (bool, error) {
	found := false
	for _, path := range s.telegramFiles() {
		data, err := os.ReadFile(path)
		if err != nil || !strings.Contains(string(data), markerWords) {
			continue
		}
		found = true
		switch {
		case s.GOOS == "darwin":
			if _, isLoaded := s.loaded(TelegramTimerLabel); isLoaded {
				_, _ = s.Run("launchctl", "bootout", s.domain()+"/"+TelegramTimerLabel)
			}
		case strings.HasSuffix(path, ".timer"):
			_, _ = s.Run("systemctl", "--user", "disable", "--now", filepath.Base(path))
		}
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return found, err
		}
	}
	return found, nil
}

// TelegramTimerSet says whether the timer's files are there, written by augmented brain.
func (s System) TelegramTimerSet() bool {
	data, err := os.ReadFile(s.telegramFiles()[0])
	return err == nil && strings.Contains(string(data), markerWords)
}

// SetTelegramTimer sets the timer on this computer unless it is there already: after
// connecting Telegram, and before keeping a wait (a computer connected before the timer
// existed gets it then).
func SetTelegramTimer(cfgDir string) error {
	s, err := NewSystem()
	if err != nil {
		return err
	}
	if s.TelegramTimerSet() {
		return nil
	}
	return s.InstallTelegramTimer(cfgDir)
}

// UnsetTelegramTimer removes the timer from this computer, when Telegram is
// disconnected or augmented brain uninstalled.
func UnsetTelegramTimer() (bool, error) {
	s, err := NewSystem()
	if err != nil {
		return false, err
	}
	return s.RemoveTelegramTimer()
}
