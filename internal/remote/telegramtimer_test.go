package remote

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func telegramSystem(t *testing.T, goos string) (System, *[]string) {
	t.Helper()
	var calls []string
	dir := t.TempDir()
	return System{
		GOOS:     goos,
		Self:     "/opt/ab/augmented-brain",
		UnitDir:  filepath.Join(dir, "units"),
		LogDir:   filepath.Join(dir, "logs"),
		UID:      501,
		LookPath: func(name string) (string, error) { return "/usr/bin/" + name, nil },
		Run: func(name string, args ...string) (string, error) {
			calls = append(calls, name+" "+strings.Join(args, " "))
			if name == "launchctl" && args[0] == "print" {
				return "", exec.ErrNotFound // not loaded
			}
			return "", nil
		},
	}, &calls
}

func TestTelegramTimerLinux(t *testing.T) {
	s, calls := telegramSystem(t, "linux")
	if err := s.InstallTelegramTimer("/home/x/.config/augmented-brain"); err != nil {
		t.Fatal(err)
	}
	svc, err := os.ReadFile(filepath.Join(s.UnitDir, TelegramTimerName+".service"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(svc), "ExecStart=/opt/ab/augmented-brain telegram check") ||
		!strings.Contains(string(svc), "Environment=AB_CONFIG_DIR=/home/x/.config/augmented-brain") {
		t.Errorf("service:\n%s", svc)
	}
	if !strings.Contains(TelegramTimerText(), "OnUnitActiveSec=5min") {
		t.Errorf("timer:\n%s", TelegramTimerText())
	}
	if got := strings.Join(*calls, "; "); !strings.Contains(got, "systemctl --user enable --now "+TelegramTimerName+".timer") {
		t.Errorf("calls: %s", got)
	}
	if !s.TelegramTimerSet() {
		t.Error("set")
	}
	found, err := s.RemoveTelegramTimer()
	if err != nil || !found || s.TelegramTimerSet() {
		t.Fatalf("remove: %v %v", found, err)
	}
	if got := strings.Join(*calls, "; "); !strings.Contains(got, "systemctl --user disable --now "+TelegramTimerName+".timer") {
		t.Errorf("calls: %s", got)
	}
	if found, _ := s.RemoveTelegramTimer(); found {
		t.Error("removed twice")
	}
}

func TestTelegramTimerKeepsForeignFiles(t *testing.T) {
	s, _ := telegramSystem(t, "linux")
	path := filepath.Join(s.UnitDir, TelegramTimerName+".timer")
	if err := os.MkdirAll(s.UnitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("[Timer]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if found, err := s.RemoveTelegramTimer(); found || err != nil {
		t.Fatalf("a file not ours: %v %v", found, err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("a file not ours was removed")
	}
}

func TestTelegramTimerMacOS(t *testing.T) {
	s, calls := telegramSystem(t, "darwin")
	if err := s.InstallTelegramTimer("/Users/x/.config/augmented-brain"); err != nil {
		t.Fatal(err)
	}
	plist, err := os.ReadFile(filepath.Join(s.UnitDir, TelegramTimerLabel+".plist"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"<string>telegram</string>", "<string>check</string>", "<integer>300</integer>", "<string>/Users/x/.config/augmented-brain</string>"} {
		if !strings.Contains(string(plist), want) {
			t.Errorf("plist lacks %s:\n%s", want, plist)
		}
	}
	if got := strings.Join(*calls, "; "); !strings.Contains(got, "launchctl bootstrap gui/501") {
		t.Errorf("calls: %s", got)
	}
	if found, err := s.RemoveTelegramTimer(); !found || err != nil {
		t.Fatalf("remove: %v %v", found, err)
	}
}

func TestTelegramTimerWindows(t *testing.T) {
	s, _ := telegramSystem(t, "windows")
	if err := s.InstallTelegramTimer(`C:\x`); err != ErrNoTimer {
		t.Fatalf("windows: %v", err)
	}
}
