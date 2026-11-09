package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/remote"
)

// useFakeSystem points `remote` at a system of fakes: nothing runs, every path is in h.
func useFakeSystem(t *testing.T, h, goos string) *[]string {
	t.Helper()
	var calls []string
	old := remoteSystem
	t.Cleanup(func() { remoteSystem = old })
	remoteSystem = func() (remote.System, error) {
		return remote.System{
			Run: func(name string, args ...string) (string, error) {
				calls = append(calls, name+" "+strings.Join(args, " "))
				if name == "loginctl" {
					return "Linger=no\n", nil
				}
				if len(args) > 1 && args[1] == "is-active" {
					return "inactive\n", exec.ErrNotFound
				}
				return "", nil
			},
			LookPath:   func(string) (string, error) { return filepath.Join(h, "bin", "claude"), nil },
			Hostname:   func() (string, error) { return "box", nil },
			GOOS:       goos,
			User:       "lucia",
			UnitDir:    filepath.Join(h, "units"),
			ClaudeJSON: filepath.Join(h, "claude.json"),
		}, nil
	}
	return &calls
}

func TestRemoteCommands(t *testing.T) {
	h, _ := home(t)
	calls := useFakeSystem(t, h, "linux")
	var out, errOut bytes.Buffer

	if code := run([]string{"remote", "status"}, nil, &out, &errOut); code != 0 || !strings.Contains(out.String(), "remote install") {
		t.Errorf("status before install: %d\n%s%s", code, out.String(), errOut.String())
	}

	out.Reset()
	if code := run([]string{"remote", "install"}, nil, &out, &errOut); code != 0 {
		t.Fatalf("install: %d\n%s%s", code, out.String(), errOut.String())
	}
	folder := filepath.Join(h, "augmented-brain", "personal") // the space's first work folder
	for _, want := range []string{"box-personal", "cd " + folder + " && claude", "sudo loginctl enable-linger lucia"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("install output lacks %q (in Spanish, the configuration's language):\n%s", want, out.String())
		}
	}
	if !strings.Contains(out.String(), "acceso remoto está listo") {
		t.Errorf("install must speak the configuration's language:\n%s", out.String())
	}
	if _, err := os.Stat(filepath.Join(h, "units", "augmented-brain-remote-personal.service")); err != nil {
		t.Error("no unit written")
	}
	for _, c := range *calls {
		if strings.Contains(c, "enable-linger") {
			t.Errorf("ran %q", c)
		}
	}

	out.Reset()
	if code := run([]string{"remote", "status"}, nil, &out, &errOut); code != 0 || !strings.Contains(out.String(), "box-personal") || !strings.Contains(out.String(), "parado") {
		t.Errorf("status: %d\n%s%s", code, out.String(), errOut.String())
	}

	out.Reset()
	if code := run([]string{"remote", "uninstall"}, nil, &out, &errOut); code != 0 || !strings.Contains(out.String(), "augmented-brain-remote-personal.service") {
		t.Errorf("uninstall: %d\n%s%s", code, out.String(), errOut.String())
	}
	if _, err := os.Stat(folder); err != nil {
		t.Error("the folder must stay")
	}
}

func TestRemoteUsageAndSystems(t *testing.T) {
	h, _ := home(t)
	useFakeSystem(t, h, "freebsd")
	for _, args := range [][]string{{"remote"}, {"remote", "start"}, {"remote", "install", "now"}} {
		var out, errOut bytes.Buffer
		if code := run(args, nil, &out, &errOut); code != 2 || !strings.Contains(errOut.String(), "remote install | uninstall | status") {
			t.Errorf("%v: %d %q", args, code, errOut.String())
		}
	}
	var out, errOut bytes.Buffer
	if code := run([]string{"remote", "install"}, nil, &out, &errOut); code != 1 || !strings.Contains(errOut.String(), "Linux, macOS y Windows") {
		t.Errorf("freebsd: %d %q", code, errOut.String())
	}
}
