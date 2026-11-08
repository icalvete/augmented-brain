package main

import (
	"bytes"
	"errors"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/version"
)

// The commands, whole, where the coverage map of the beta tests found nothing: what a
// person types and reads, not the package underneath.

func chdir(t *testing.T, dir string) {
	t.Helper()
	wd, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(wd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
}

// space says the space of the folder, the default one outside every work folder, and
// what to do when there is none.
func TestSpaceCommand(t *testing.T) {
	h, cfg := home(t)
	var out, errOut bytes.Buffer
	chdir(t, filepath.Join(h, "augmented-brain", "trabajo"))
	if code := run([]string{"space"}, nil, &out, &errOut); code != 0 || !strings.Contains(out.String(), "espacio: trabajo") {
		t.Errorf("in trabajo: %d %q %q", code, out.String(), errOut.String())
	}
	out.Reset()
	chdir(t, h)
	if code := run([]string{"space"}, nil, &out, &errOut); code != 0 || !strings.Contains(out.String(), "espacio: personal") {
		t.Errorf("outside every work folder, the default: %d %q %q", code, out.String(), errOut.String())
	}
	if err := config.SetDefaultSpace(cfg, ""); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	errOut.Reset()
	if code := run([]string{"space"}, nil, &out, &errOut); code == 0 || !strings.Contains(errOut.String(), "no está dentro de la carpeta de trabajo de ningún espacio") || !strings.Contains(errOut.String(), "personal, trabajo") {
		t.Errorf("no default space: %d %q", code, errOut.String())
	}
}

// check makes notes others could read private again, and says so.
func TestCheckMakesNotesPrivate(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission bits are Unix")
	}
	h, _ := home(t)
	notes := filepath.Join(h, "augmented-brain", "trabajo", "notas")
	if err := os.Chmod(notes, 0o755); err != nil {
		t.Fatal(err)
	}
	chdir(t, filepath.Join(h, "augmented-brain", "trabajo"))
	var out, errOut bytes.Buffer
	if code := run([]string{"check"}, nil, &out, &errOut); code != 0 {
		t.Fatalf("check: %d %s", code, errOut.String())
	}
	if fi, _ := os.Stat(notes); fi.Mode().Perm()&0o077 != 0 { // windows-ok: skipped above
		t.Errorf("still open: %v", fi.Mode().Perm())
	}
	if !strings.Contains(out.String(), "notas") {
		t.Errorf("check does not say it: %q", out.String())
	}
}

// import of a folder that is not there says so and writes nothing.
func TestImportMissingSource(t *testing.T) {
	h, _ := home(t)
	chdir(t, h)
	var out, errOut bytes.Buffer
	missing := filepath.Join(h, "no-such-memory")
	if code := run([]string{"import", missing, "--apply"}, nil, &out, &errOut); code == 0 || !strings.Contains(errOut.String(), "no-such-memory") {
		t.Errorf("missing source: %d %q %q", code, out.String(), errOut.String())
	}
}

// backup --space backs up that space from anywhere, and names an unknown one.
func TestBackupSpaceFlag(t *testing.T) {
	git, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git is needed")
	}
	h, cfg := home(t)
	t.Setenv("PATH", filepath.Dir(git))
	remote := filepath.Join(h, "copia.git")
	if err := exec.Command("git", "init", "-q", "--bare", "-b", "main", remote).Run(); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	chdir(t, filepath.Join(h, "augmented-brain", "trabajo"))
	if code := run([]string{"backup", "setup", remote}, nil, &out, &errOut); code != 0 {
		t.Fatalf("setup: %d %s", code, errOut.String())
	}
	chdir(t, h) // outside every work folder: the default would be personal
	out.Reset()
	if code := run([]string{"backup", "--space", "trabajo"}, nil, &out, &errOut); code != 0 || !strings.Contains(out.String(), "trabajo") {
		t.Errorf("--space trabajo: %d %q %q", code, out.String(), errOut.String())
	}
	if o, err := exec.Command("git", "--git-dir", remote, "log", "--oneline", "-1").Output(); err != nil || len(o) == 0 {
		t.Errorf("nothing reached the backup: %v", err)
	}
	errOut.Reset()
	if code := run([]string{"backup", "--space", "nope"}, nil, &out, &errOut); code == 0 || !strings.Contains(errOut.String(), `"nope"`) {
		t.Errorf("unknown space: %d %q", code, errOut.String())
	}
	_ = cfg
}

// uninstall --config takes the configuration too, and leaves the notes.
func TestUninstallConfig(t *testing.T) {
	h, cfg := home(t)
	var out, errOut bytes.Buffer
	if code := cmdUninstall([]string{"--config", "--yes"}, strings.NewReader(""), &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s%s", code, out.String(), errOut.String())
	}
	if exists(filepath.Join(cfg, config.FileName)) {
		t.Error("the configuration is still there")
	}
	if !exists(filepath.Join(h, "augmented-brain", "trabajo", "notas", "MEMORY.md")) {
		t.Error("the notes went with it")
	}
}

// The program with nothing prints the list of commands, and an unknown one says so.
func TestNoArguments(t *testing.T) {
	home(t)
	var out, errOut bytes.Buffer
	if code := run(nil, nil, &out, &errOut); code != 2 || !strings.Contains(errOut.String(), "doctor") || !strings.Contains(errOut.String(), "backup") {
		t.Errorf("no arguments: %d %q", code, errOut.String())
	}
	errOut.Reset()
	if code := run([]string{"frobnicate"}, nil, &out, &errOut); code != 2 || !strings.Contains(errOut.String(), "frobnicate") {
		t.Errorf("unknown command: %d %q", code, errOut.String())
	}
}

// update with nothing newer says so and only updates the plugin (here there is no
// Claude Code, so nothing), and an update check that cannot reach GitHub says that.
func TestUpdateNothingNewer(t *testing.T) {
	home(t)
	old := latestRelease
	t.Cleanup(func() { latestRelease = old })
	latestRelease = func(*http.Client) (string, error) { return version.Version, nil }
	var out, errOut bytes.Buffer
	if code := cmdUpdate(nil, &out, &errOut); code != 0 || !strings.Contains(out.String(), version.Version) {
		t.Errorf("nothing newer: %d %q %q", code, out.String(), errOut.String())
	}
	latestRelease = func(*http.Client) (string, error) { return "", errors.New("no network") }
	out.Reset()
	errOut.Reset()
	if code := cmdUpdate(nil, &out, &errOut); code == 0 || !strings.Contains(errOut.String(), "GitHub") {
		t.Errorf("offline: %d %q", code, errOut.String())
	}
}
