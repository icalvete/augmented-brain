package remote

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf16"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/paths"
)

// Windows: one task of the Task Scheduler per space, \augmented-brain\remote-<space>,
// registered with `schtasks /Create /XML` from a file of its own kept in UnitDir (that
// copy carries the marker, so uninstall removes only these). What shaped it, measured
// on Windows 11 (9-nov-2026, as a standard user, with no administrator):
//
//   - It runs as the user, only while they are logged in (InteractiveToken). "Run
//     whether the user is logged on or not" without a stored password (S4U) is refused to
//     a standard user ("Access is denied"), and a stored password is out of the question.
//   - A console program started by a task opens a window (Windows Terminal) that stays
//     as long as it runs, and closing it would stop the service. conhost.exe --headless
//     runs it with no window at all.
//   - "Restart on failure" does not restart a program that exits on its own: a time
//     trigger that repeats every minute does, since a task already running ignores it
//     (MultipleInstancesPolicy IgnoreNew). The logon trigger starts it at once.
//   - `schtasks /End` stops only the task's first process (conhost): cmd and claude go
//     on. Stopping it also ends, with taskkill /T, the cmd whose line names the
//     service's log, which is unique to it.
//   - schtasks reads the XML as UTF-16.
//
// cmd sets PATH (the folders of claude and of this program, then the user's own) and
// sends claude's stderr to the log (overwritten each start: the last error is what
// counts) and stdout, where claude redraws its screen, to NUL.

const taskFolder = `\augmented-brain\`

func taskName(space string) string { return taskFolder + "remote-" + space }

// TaskText is the task of a service, as XML.
func (svc Service) TaskText(claude, mode, self, log, user, sysRoot string) string {
	cmd := sysRoot + `\System32\cmd.exe`
	line := fmt.Sprintf(`--headless %s /d /s /c "set "PATH=%s;%%PATH%%" & "%s" remote-control --name %s --permission-mode %s >NUL 2>"%s""`,
		cmd, servicePath(";", claude, self), claude, svc.Name, mode, log)
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-16"?>` + "\r\n")
	b.WriteString(markerComment + "\r\n")
	b.WriteString(`<Task version="1.2" xmlns="http://schemas.microsoft.com/windows/2004/02/mit/task">` + "\r\n")
	fmt.Fprintf(&b, "  <RegistrationInfo><Description>augmented brain: Claude remote access, space %s</Description></RegistrationInfo>\r\n", xmlEscape(svc.Space))
	b.WriteString("  <Triggers>\r\n")
	fmt.Fprintf(&b, "    <LogonTrigger><Enabled>true</Enabled><UserId>%s</UserId></LogonTrigger>\r\n", xmlEscape(user))
	b.WriteString("    <TimeTrigger><Enabled>true</Enabled><StartBoundary>2026-01-01T00:00:00</StartBoundary>" +
		"<Repetition><Interval>PT1M</Interval><StopAtDurationEnd>false</StopAtDurationEnd></Repetition></TimeTrigger>\r\n")
	b.WriteString("  </Triggers>\r\n")
	fmt.Fprintf(&b, "  <Principals><Principal id=\"Author\"><UserId>%s</UserId><LogonType>InteractiveToken</LogonType><RunLevel>LeastPrivilege</RunLevel></Principal></Principals>\r\n", xmlEscape(user))
	b.WriteString("  <Settings>\r\n" +
		"    <MultipleInstancesPolicy>IgnoreNew</MultipleInstancesPolicy>\r\n" +
		"    <DisallowStartIfOnBatteries>false</DisallowStartIfOnBatteries>\r\n" +
		"    <StopIfGoingOnBatteries>false</StopIfGoingOnBatteries>\r\n" +
		"    <ExecutionTimeLimit>PT0S</ExecutionTimeLimit>\r\n" +
		"    <IdleSettings><StopOnIdleEnd>false</StopOnIdleEnd><RestartOnIdle>false</RestartOnIdle></IdleSettings>\r\n" +
		"  </Settings>\r\n")
	fmt.Fprintf(&b, "  <Actions Context=\"Author\"><Exec><Command>%s</Command><Arguments>%s</Arguments><WorkingDirectory>%s</WorkingDirectory></Exec></Actions>\r\n",
		xmlEscape(sysRoot+`\System32\conhost.exe`), xmlEscape(line), xmlEscape(svc.Folder))
	b.WriteString("</Task>\r\n")
	return b.String()
}

// utf16File is text as schtasks reads it: UTF-16, little endian, with its mark.
func utf16File(text string) []byte {
	units := utf16.Encode([]rune(text))
	out := make([]byte, 2+2*len(units))
	out[0], out[1] = 0xFF, 0xFE
	for i, u := range units {
		binary.LittleEndian.PutUint16(out[2+2*i:], u)
	}
	return out
}

// fromUTF16 reads a file written by utf16File; anything else is taken as it is.
func fromUTF16(b []byte) string {
	if len(b) < 2 || b[0] != 0xFF || b[1] != 0xFE {
		return string(b)
	}
	b = b[2:]
	units := make([]uint16, len(b)/2)
	for i := range units {
		units[i] = binary.LittleEndian.Uint16(b[2*i:])
	}
	return string(utf16.Decode(units))
}

func (s System) taskFile(space string) string {
	return filepath.Join(s.UnitDir, unitPrefix+space+".xml")
}

// taskStates asks the Task Scheduler how the tasks of the folder stand: name to state
// (Ready, Running, Disabled, Queued), the same words in every language of Windows.
func (s System) taskStates() map[string]string {
	out, _ := s.Run("powershell", "-NoProfile", "-NonInteractive", "-Command",
		"Get-ScheduledTask -TaskPath '"+taskFolder+"' -ErrorAction SilentlyContinue | ForEach-Object { $_.TaskName + '=' + $_.State }")
	states := map[string]string{}
	for _, l := range strings.Split(out, "\n") {
		if name, state, ok := strings.Cut(strings.TrimSpace(l), "="); ok {
			states[taskFolder+name] = state
		}
	}
	return states
}

// stopTask ends a running task and what it started.
func (s System) stopTask(space string) {
	_, _ = s.Run("schtasks", "/End", "/TN", taskName(space))
	log := strings.ReplaceAll(s.logFile(space), "'", "''")
	_, _ = s.Run("powershell", "-NoProfile", "-NonInteractive", "-Command",
		"Get-CimInstance Win32_Process | Where-Object { $_.Name -eq 'cmd.exe' -and $_.CommandLine -and $_.CommandLine.Contains('"+log+"') } | "+
			"ForEach-Object { taskkill.exe /T /F /PID $_.ProcessId | Out-Null }")
}

func (s System) installTasks(c *config.Config, set Settings) (*InstallResult, error) {
	res, err := s.prepare(c, set)
	if err != nil {
		return nil, err
	}
	for _, p := range []string{"schtasks", "conhost"} {
		if _, err := s.LookPath(p); err != nil {
			return nil, errors.New(i18n.T(c.Language, "remote.notasks", p, err))
		}
	}
	if err := s.folders(c, res); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(s.UnitDir, 0o755); err != nil {
		return nil, err
	}
	sysRoot := s.SystemRoot
	if sysRoot == "" {
		sysRoot = `C:\Windows`
	}
	states := s.taskStates()
	for _, svc := range res.Services {
		name, path := taskName(svc.Space), s.taskFile(svc.Space)
		text := svc.TaskText(res.Claude, set.PermissionMode, s.Self, s.logFile(svc.Space), s.User, sysRoot)
		old, readErr := os.ReadFile(path)
		changed := readErr != nil || fromUTF16(old) != text
		state, registered := states[name]
		if changed {
			if err := paths.WriteAtomic(path, utf16File(text), 0o644); err != nil {
				return nil, err
			}
		}
		if changed || !registered {
			if _, err := s.Run("schtasks", "/Create", "/TN", name, "/XML", path, "/F"); err != nil {
				return nil, errors.New(i18n.T(c.Language, "remote.schtasks", "/Create "+name, err))
			}
		}
		if changed && state == "Running" { // a running task keeps its old command
			s.stopTask(svc.Space)
		}
		if _, err := s.Run("schtasks", "/Run", "/TN", name); err != nil {
			return nil, errors.New(i18n.T(c.Language, "remote.schtasks", "/Run "+name, err))
		}
	}
	return s.finish(c, res)
}

func (s System) uninstallTasks(c *config.Config) ([]string, error) {
	var removed []string
	states := s.taskStates()
	for _, f := range s.installedFiles(unitPrefix, ".xml", fromUTF16) {
		space := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(f), unitPrefix), ".xml")
		name := taskName(space)
		s.stopTask(space)
		if _, ok := states[name]; ok {
			if _, err := s.Run("schtasks", "/Delete", "/TN", name, "/F"); err != nil {
				return removed, errors.New(i18n.T(c.Language, "remote.schtasks", "/Delete "+name, err))
			}
		}
		if err := os.Remove(f); err != nil && !errors.Is(err, os.ErrNotExist) {
			return removed, err
		}
		removed = append(removed, name)
	}
	if len(removed) > 0 && len(states) == len(removed) {
		// The folder \augmented-brain\ is empty now: take it away too.
		_, _ = s.Run("schtasks", "/Delete", "/TN", strings.TrimSuffix(taskFolder, `\`), "/F")
	}
	return removed, SetEnabled(c.Dir, false)
}

func (s System) statusTasks(c *config.Config, set Settings) (*Status, error) {
	trusted := s.trustedFolders()
	st := &Status{Linger: LingerNone}
	states := s.taskStates()
	for _, svc := range s.services(c, set) {
		x := SpaceStatus{Service: svc, Trusted: trusted(svc.Folder), Log: s.logFile(svc.Space)}
		if data, err := os.ReadFile(s.taskFile(svc.Space)); err == nil && strings.Contains(fromUTF16(data), markerComment) {
			x.Installed = true
			switch states[svc.Unit] {
			case "Running":
				x.Active, x.Enabled = "active", "enabled"
			case "Queued":
				x.Active, x.Enabled = "activating", "enabled"
			case "Ready":
				x.Active, x.Enabled = "inactive", "enabled"
			default: // Disabled, or not registered at all
				x.Active, x.Enabled = "inactive", "disabled"
			}
		}
		st.Spaces = append(st.Spaces, x)
	}
	return st, nil
}
