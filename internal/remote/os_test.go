package remote

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/i18n"
)

// osFake is a System for macOS or Windows whose commands are recorded, never run:
// launchd and the Task Scheduler are a map of what is loaded and how it stands.
type osFake struct {
	*fake
	loaded   map[string]string // label or task -> state (launchd: "running"; tasks: "Running", "Ready"…)
	disabled []string          // labels launchctl print-disabled lists as disabled
	missing  string            // a program LookPath does not find
}

func newOSFake(t *testing.T, goos string) *osFake {
	t.Helper()
	f := &osFake{fake: newFake(t), loaded: map[string]string{}}
	f.sys.GOOS = goos
	f.sys.Run = f.runOS
	f.sys.LookPath = func(name string) (string, error) {
		if name == f.missing {
			return "", exec.ErrNotFound
		}
		if name == "claude" {
			return filepath.Join(f.root, ".local", "bin", "claude"), nil
		}
		return name, nil
	}
	f.sys.Self = filepath.Join(f.root, "bin", "augmented-brain")
	switch goos {
	case "darwin":
		f.sys.UID = 501
		f.sys.UnitDir = filepath.Join(f.root, "Library", "LaunchAgents")
		f.sys.LogDir = filepath.Join(f.root, "Library", "Logs", "augmented-brain")
	case "windows":
		f.sys.User = `BOX\lucia`
		f.sys.SystemRoot = `C:\Windows`
		f.sys.UnitDir = filepath.Join(f.root, "AppData", "Local", "augmented-brain", "remote")
		f.sys.LogDir = f.sys.UnitDir
	}
	return f
}

func (f *osFake) runOS(name string, args ...string) (string, error) {
	call := name + " " + strings.Join(args, " ")
	f.calls = append(f.calls, call)
	if f.failOn != "" && strings.Contains(call, f.failOn) {
		return "", errors.New("boom")
	}
	switch {
	case name == "launchctl" && args[0] == "print":
		label := args[1][strings.LastIndex(args[1], "/")+1:]
		if st, ok := f.loaded[label]; ok {
			return "gui/501/" + label + " = {\n\tstate = " + st + "\n}\n", nil
		}
		return "", errors.New("exit status 113")
	case name == "launchctl" && args[0] == "print-disabled":
		var b strings.Builder
		for _, l := range f.disabled {
			b.WriteString("\t\"" + l + "\" => disabled\n")
		}
		return "disabled services = {\n" + b.String() + "}\n", nil
	case name == "launchctl" && args[0] == "bootstrap":
		f.loaded[strings.TrimSuffix(filepath.Base(args[2]), ".plist")] = "running"
	case name == "launchctl" && args[0] == "bootout":
		delete(f.loaded, args[1][strings.LastIndex(args[1], "/")+1:])
	case name == "schtasks" && args[0] == "/Create":
		f.loaded[args[2]] = "Ready"
	case name == "schtasks" && args[0] == "/Run":
		f.loaded[args[2]] = "Running"
	case name == "schtasks" && args[0] == "/Delete":
		delete(f.loaded, args[2])
	case name == "powershell" && strings.Contains(call, "Get-ScheduledTask"):
		var b strings.Builder
		for task, st := range f.loaded {
			b.WriteString(strings.TrimPrefix(task, taskFolder) + "=" + st + "\r\n")
		}
		return b.String(), nil
	}
	return "", nil
}

func TestPlistText(t *testing.T) {
	svc := Service{Space: "personal", Name: "box-personal", Folder: "/Users/lucia/R&D projects"}
	got := svc.PlistText("/Users/lucia/.local/bin/claude", "default", "/opt/homebrew/bin/augmented-brain", "/Users/lucia/Library/Logs/augmented-brain/augmented-brain-remote-personal.log")
	for _, want := range []string{
		`<?xml version="1.0" encoding="UTF-8"?>` + "\n" + markerComment + "\n",
		"<key>Label</key>\n\t<string>com.augmented-brain.remote.personal</string>",
		"<string>/Users/lucia/.local/bin/claude</string>\n\t\t<string>remote-control</string>\n\t\t<string>--name</string>\n\t\t<string>box-personal</string>\n\t\t<string>--permission-mode</string>\n\t\t<string>default</string>",
		"<key>WorkingDirectory</key>\n\t<string>/Users/lucia/R&amp;D projects</string>",
		// claude's folder first, then this program's, then Homebrew and the system's,
		// each once.
		"<key>PATH</key>\n\t\t<string>/Users/lucia/.local/bin:/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin</string>",
		"<key>RunAtLoad</key>\n\t<true/>", "<key>KeepAlive</key>\n\t<true/>",
		"<key>ThrottleInterval</key>\n\t<integer>30</integer>",
		"<key>StandardOutPath</key>\n\t<string>/dev/null</string>",
		"<key>StandardErrorPath</key>\n\t<string>/Users/lucia/Library/Logs/augmented-brain/augmented-brain-remote-personal.log</string>",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the plist lacks %q:\n%s", want, got)
		}
	}
}

func TestLaunchdInstallStatusUninstall(t *testing.T) {
	f := newOSFake(t, "darwin")
	c, set := f.load()
	res, err := f.sys.Install(c, set)
	if err != nil {
		t.Fatal(err)
	}
	if res.Linger != LingerNone || len(res.Untrusted) != 1 {
		t.Errorf("result: %+v", res)
	}
	plist := filepath.Join(f.sys.UnitDir, "com.augmented-brain.remote.personal.plist")
	data, err := os.ReadFile(plist)
	if err != nil || !strings.Contains(string(data), markerComment) {
		t.Fatalf("plist: %v\n%s", err, data)
	}
	if got := filterCalls(f.calls, "launchctl bootstrap"); !slices.Equal(got, []string{"launchctl bootstrap gui/501 " + plist}) {
		t.Errorf("bootstrap: %q", got)
	}
	if fi, err := os.Stat(f.sys.LogDir); err != nil || !fi.IsDir() {
		t.Error("no folder for the log")
	}
	if got, _ := LoadSettings(c, "box.local"); !got.Enabled {
		t.Error("not enabled")
	}

	// Again, the same: nothing reloaded. Another mode: unloaded and loaded again.
	f.calls = nil
	if _, err := f.sys.Install(c, set); err != nil {
		t.Fatal(err)
	}
	if got := filterCalls(f.calls, "launchctl boot"); len(got) != 0 {
		t.Errorf("an unchanged agent was reloaded: %q", got)
	}
	set.PermissionMode = "plan"
	if _, err := f.sys.Install(c, set); err != nil {
		t.Fatal(err)
	}
	if got := filterCalls(f.calls, "launchctl boot"); !slices.Equal(got, []string{"launchctl bootout gui/501/com.augmented-brain.remote.personal", "launchctl bootstrap gui/501 " + plist}) {
		t.Errorf("a changed agent: %q", got)
	}

	f.trust(filepath.Join(f.root, "p"))
	st, err := f.sys.Status(c, set)
	if err != nil {
		t.Fatal(err)
	}
	x := st.Spaces[0]
	if !x.Installed || !x.Running() || x.Enabled != "enabled" || !x.Trusted || x.Unit != "com.augmented-brain.remote.personal" || !strings.HasSuffix(x.Log, "augmented-brain-remote-personal.log") {
		t.Errorf("status: %+v", x)
	}
	if adv := f.sys.Advice("en", x, st.Linger); len(adv) != 0 {
		t.Errorf("advice when all is well: %q", adv)
	}
	lines := strings.Join(f.sys.StatusLines("en", st), "\n")
	if strings.Contains(lines, "linger") || !strings.Contains(lines, "log in") {
		t.Errorf("status lines on macOS:\n%s", lines)
	}
	what, _, ok := f.sys.DoctorLine("en", x, st.Linger)
	if !ok || strings.Contains(what, "linger") {
		t.Errorf("doctor: %q %v", what, ok)
	}

	// Stopped and disabled: the log says why.
	f.loaded["com.augmented-brain.remote.personal"] = "not running"
	f.disabled = []string{"com.augmented-brain.remote.personal"}
	st, _ = f.sys.Status(c, set)
	x = st.Spaces[0]
	if x.Running() || x.Enabled != "disabled" {
		t.Errorf("stopped: %+v", x)
	}
	if adv := strings.Join(f.sys.Advice("en", x, st.Linger), " "); !strings.Contains(adv, x.Log) || strings.Contains(adv, "journalctl") {
		t.Errorf("advice: %q", adv)
	}

	// Uninstall takes out only what carries the marker.
	other := filepath.Join(f.sys.UnitDir, "com.augmented-brain.remote.mine.plist")
	if err := os.WriteFile(other, []byte("<plist/>"), 0o644); err != nil {
		t.Fatal(err)
	}
	removed, err := f.sys.Uninstall(c)
	if err != nil || !slices.Equal(removed, []string{"com.augmented-brain.remote.personal"}) {
		t.Errorf("uninstall: %q %v", removed, err)
	}
	if _, err := os.Stat(plist); !errors.Is(err, os.ErrNotExist) {
		t.Error("the plist stayed")
	}
	if _, err := os.Stat(other); err != nil {
		t.Error("a plist without the marker was removed")
	}
	if len(f.loaded) != 0 {
		t.Errorf("still loaded: %v", f.loaded)
	}
	if got, _ := LoadSettings(c, "box.local"); got.Enabled {
		t.Error("still enabled")
	}
}

func TestLaunchdErrors(t *testing.T) {
	f := newOSFake(t, "darwin")
	c, set := f.load()
	f.missing = "launchctl"
	if _, err := f.sys.Install(c, set); err == nil || !strings.Contains(err.Error(), "launchctl") {
		t.Errorf("no launchctl: %v", err)
	}
	if _, err := os.Stat(f.sys.UnitDir); err == nil {
		t.Error("written without launchctl")
	}
	f.missing = ""
	f.failOn = "bootstrap"
	if _, err := f.sys.Install(c, set); err == nil || !strings.Contains(err.Error(), "boom") {
		t.Errorf("bootstrap refused: %v", err)
	}
	if got, _ := LoadSettings(c, "box.local"); got.Enabled {
		t.Error("enabled after launchd refused the agent")
	}
}

func TestTaskText(t *testing.T) {
	svc := Service{Space: "work", Name: "box-work", Folder: `C:\Users\lucia\R&D`}
	got := svc.TaskText(`C:\Users\lucia\.local\bin\claude.exe`, "default", `C:\Users\lucia\AppData\Local\Programs\augmented-brain\augmented-brain.exe`,
		`C:\Users\lucia\AppData\Local\augmented-brain\remote\augmented-brain-remote-work.log`, `BOX\lucia`, `C:\Windows`)
	for _, want := range []string{
		`<?xml version="1.0" encoding="UTF-16"?>` + "\r\n" + markerComment + "\r\n",
		`<LogonTrigger><Enabled>true</Enabled><UserId>BOX\lucia</UserId></LogonTrigger>`,
		`<Repetition><Interval>PT1M</Interval><StopAtDurationEnd>false</StopAtDurationEnd></Repetition>`,
		`<LogonType>InteractiveToken</LogonType><RunLevel>LeastPrivilege</RunLevel>`,
		`<MultipleInstancesPolicy>IgnoreNew</MultipleInstancesPolicy>`,
		`<DisallowStartIfOnBatteries>false</DisallowStartIfOnBatteries>`,
		`<ExecutionTimeLimit>PT0S</ExecutionTimeLimit>`,
		`<Command>C:\Windows\System32\conhost.exe</Command>`,
		`<Arguments>--headless C:\Windows\System32\cmd.exe /d /s /c &quot;set &quot;PATH=C:\Users\lucia\.local\bin;C:\Users\lucia\AppData\Local\Programs\augmented-brain;%PATH%&quot; &amp; &quot;C:\Users\lucia\.local\bin\claude.exe&quot; remote-control --name box-work --permission-mode default &gt;NUL 2&gt;&quot;C:\Users\lucia\AppData\Local\augmented-brain\remote\augmented-brain-remote-work.log&quot;&quot;</Arguments>`,
		`<WorkingDirectory>C:\Users\lucia\R&amp;D</WorkingDirectory>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the task lacks %q:\n%s", want, got)
		}
	}
	if back := fromUTF16(utf16File(got)); back != got {
		t.Error("UTF-16 does not read back")
	}
	if b := utf16File("é"); len(b) != 4 || b[0] != 0xFF || b[1] != 0xFE || b[2] != 0xE9 {
		t.Errorf("UTF-16: % x", b)
	}
}

func TestTasksInstallStatusUninstall(t *testing.T) {
	f := newOSFake(t, "windows")
	c, set := f.load()
	res, err := f.sys.Install(c, set)
	if err != nil {
		t.Fatal(err)
	}
	if res.Linger != LingerNone {
		t.Errorf("linger: %v", res.Linger)
	}
	file := filepath.Join(f.sys.UnitDir, "augmented-brain-remote-personal.xml")
	data, err := os.ReadFile(file)
	if err != nil || !strings.Contains(fromUTF16(data), markerComment) {
		t.Fatalf("task file: %v", err)
	}
	task := `\augmented-brain\remote-personal`
	if got := filterCalls(f.calls, "schtasks"); !slices.Equal(got, []string{"schtasks /Create /TN " + task + " /XML " + file + " /F", "schtasks /Run /TN " + task}) {
		t.Errorf("schtasks: %q", got)
	}

	// Again, the same: not registered again, and a run that is ignored while it runs.
	f.calls = nil
	if _, err := f.sys.Install(c, set); err != nil {
		t.Fatal(err)
	}
	if got := filterCalls(f.calls, "schtasks"); !slices.Equal(got, []string{"schtasks /Run /TN " + task}) {
		t.Errorf("unchanged: %q", got)
	}
	// Another mode while it runs: registered again, stopped with what it started, run.
	f.calls = nil
	set.PermissionMode = "plan"
	if _, err := f.sys.Install(c, set); err != nil {
		t.Fatal(err)
	}
	if got := filterCalls(f.calls, "schtasks"); !slices.Equal(got, []string{"schtasks /Create /TN " + task + " /XML " + file + " /F", "schtasks /End /TN " + task, "schtasks /Run /TN " + task}) {
		t.Errorf("changed: %q", got)
	}
	if got := filterCalls(f.calls, "powershell"); len(got) != 2 || !strings.Contains(got[1], "taskkill.exe /T /F") || !strings.Contains(got[1], "augmented-brain-remote-personal.log") {
		t.Errorf("what it started is not stopped: %q", got)
	}

	f.trust(filepath.ToSlash(filepath.Join(f.root, "p"))) // as Claude Code writes it on Windows
	st, err := f.sys.Status(c, set)
	if err != nil {
		t.Fatal(err)
	}
	x := st.Spaces[0]
	if !x.Installed || !x.Running() || x.Enabled != "enabled" || !x.Trusted || x.Unit != task {
		t.Errorf("status: %+v", x)
	}
	f.loaded[task] = "Disabled"
	st, _ = f.sys.Status(c, set)
	if x = st.Spaces[0]; x.Running() || x.Enabled != "disabled" {
		t.Errorf("disabled: %+v", x)
	}
	x.Trusted = false
	if adv := strings.Join(f.sys.Advice("en", x, st.Linger), " "); !strings.Contains(adv, `cd "`) || strings.Contains(adv, "&&") {
		t.Errorf("advice on Windows: %q", adv)
	}

	// Uninstall: only the files with the marker, and their tasks.
	other := filepath.Join(f.sys.UnitDir, "augmented-brain-remote-mine.xml")
	if err := os.WriteFile(other, utf16File("<Task/>"), 0o644); err != nil {
		t.Fatal(err)
	}
	f.calls = nil
	removed, err := f.sys.Uninstall(c)
	if err != nil || !slices.Equal(removed, []string{task}) {
		t.Errorf("uninstall: %q %v", removed, err)
	}
	if !slices.Contains(f.calls, "schtasks /Delete /TN "+task+" /F") || !slices.Contains(f.calls, "schtasks /End /TN "+task) {
		t.Errorf("calls: %q", f.calls)
	}
	if _, err := os.Stat(file); !errors.Is(err, os.ErrNotExist) {
		t.Error("the task file stayed")
	}
	if _, err := os.Stat(other); err != nil {
		t.Error("a file without the marker was removed")
	}
	if got, _ := LoadSettings(c, "box.local"); got.Enabled {
		t.Error("still enabled")
	}
}

func TestTasksErrors(t *testing.T) {
	f := newOSFake(t, "windows")
	c, set := f.load()
	f.missing = "conhost"
	if _, err := f.sys.Install(c, set); err == nil || !strings.Contains(err.Error(), "conhost") {
		t.Errorf("no conhost: %v", err)
	}
	if _, err := os.Stat(f.sys.UnitDir); err == nil {
		t.Error("written without the Task Scheduler")
	}
	f.missing = ""
	f.failOn = "/Create"
	if _, err := f.sys.Install(c, set); err == nil || !strings.Contains(err.Error(), "boom") {
		t.Errorf("schtasks refused: %v", err)
	}
	if got, _ := LoadSettings(c, "box.local"); got.Enabled {
		t.Error("enabled after schtasks refused the task")
	}
}

// The words people read on macOS and Windows exist in both languages.
func TestAtLoginMessages(t *testing.T) {
	for _, k := range []string{"remote.install.atlogin", "remote.status.atlogin", "remote.doctor.atlogin", "remote.fix.restart.log", "remote.nolaunchctl", "remote.notasks"} {
		if en, es := i18n.T("en", k), i18n.T("es", k); en == k || es == k || en == es {
			t.Errorf("%s: %q / %q", k, en, es)
		}
	}
}
