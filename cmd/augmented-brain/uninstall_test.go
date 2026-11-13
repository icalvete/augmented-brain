package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/paths"
	"github.com/icalvete/augmented-brain/internal/remote"
	"github.com/icalvete/augmented-brain/internal/setup"
)

// home sets up two spaces in a fake home, the way the installer does, plus what Claude
// Code leaves of the plugin, and returns the home and the configuration folder.
func home(t *testing.T) (string, string) {
	t.Helper()
	h, _ := filepath.EvalSymlinks(t.TempDir()) // macOS: /var is /private/var
	t.Setenv("HOME", h)
	t.Setenv("USERPROFILE", h)
	t.Setenv("XDG_CACHE_HOME", filepath.Join(h, ".cache"))
	t.Setenv("LOCALAPPDATA", filepath.Join(h, "AppData", "Local"))
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(h, ".claude"))
	t.Setenv("PATH", t.TempDir())                            // no claude: the plugin commands are not run
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(h, ".config")) // never the real systemd units
	t.Setenv("CLAUDE_JOB_DIR", "")
	cfg := filepath.Join(h, "cfg")
	t.Setenv("AB_CONFIG_DIR", cfg)
	useFakeSystem(t, h, "linux") // nothing a test runs reaches the real services
	for _, sp := range []string{"trabajo", "personal"} {
		work := filepath.Join(h, "augmented-brain", sp)
		if err := os.MkdirAll(work, 0o755); err != nil {
			t.Fatal(err)
		}
		opts := setup.Options{Language: "es", Space: sp, WorkFolder: work, Memory: filepath.Join(work, "notas")}
		var err error
		if sp == "trabajo" {
			_, err = setup.Init(cfg, opts)
		} else {
			_, err = setup.AddSpace(cfg, opts)
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	cache, _ := os.UserCacheDir()
	for _, d := range []string{
		filepath.Join(h, ".claude", "plugins", "cache", "augmented-brain", "brain"),
		filepath.Join(h, ".claude", "plugins", "data", "brain-augmented-brain"),
		filepath.Join(cache, "claude-cli-nodejs", "-home-x-augmented-brain-trabajo", "mcp-logs-plugin-brain-augmented-brain"),
		filepath.Join(cache, "claude-cli-nodejs", "-home-x-other", "mcp-logs-plugin-brain-augmented-brain"),
		filepath.Join(cache, "claude-cli-nodejs", "-home-x-other", "mcp-logs-other-server"),
	} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return h, cfg
}

func TestUninstallLeavesNotesButNoLeftovers(t *testing.T) {
	h, cfg := home(t)
	var out, errOut bytes.Buffer
	if code := cmdUninstall([]string{"--yes"}, strings.NewReader(""), &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s%s", code, out.String(), errOut.String())
	}
	cache, _ := os.UserCacheDir()
	for _, gone := range []string{
		filepath.Join(h, ".claude", "plugins", "cache", "augmented-brain"),
		filepath.Join(h, ".claude", "plugins", "data", "brain-augmented-brain"),
		filepath.Join(cache, "claude-cli-nodejs", "-home-x-augmented-brain-trabajo"),
		filepath.Join(cache, "claude-cli-nodejs", "-home-x-other", "mcp-logs-plugin-brain-augmented-brain"),
	} {
		if exists(gone) {
			t.Errorf("still there: %s", gone)
		}
	}
	for _, kept := range []string{
		filepath.Join(cache, "claude-cli-nodejs", "-home-x-other", "mcp-logs-other-server"), // not ours
		filepath.Join(h, "augmented-brain", "trabajo", "notas", "MEMORY.md"),
		filepath.Join(cfg, "config.toml"),
	} {
		if !exists(kept) {
			t.Errorf("removed, but it should stay: %s", kept)
		}
	}
}

func TestUninstallAllStartsFromScratch(t *testing.T) {
	h, cfg := home(t)
	// Something of the user's in the work space's folder: it must stay.
	mine := filepath.Join(h, "augmented-brain", "trabajo", "proyecto.txt")
	if err := os.WriteFile(mine, []byte("mío"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	if code := cmdUninstall([]string{"--all"}, strings.NewReader("sí\n"), &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s%s", code, out.String(), errOut.String())
	}
	if !strings.Contains(out.String(), "notas)") || !strings.Contains(out.String(), "con tus notas y tu configuración") {
		t.Errorf("the plan or the end do not say the notes go:\n%s", out.String())
	}
	for _, gone := range []string{
		cfg,
		filepath.Join(h, "augmented-brain", "trabajo", "notas"),
		filepath.Join(h, "augmented-brain", "personal"),
		filepath.Join(h, ".claude", "plugins", "cache", "augmented-brain"),
	} {
		if exists(gone) {
			t.Errorf("still there: %s", gone)
		}
	}
	if !exists(mine) {
		t.Error("a file of the user's in a space's folder was removed")
	}
	if runtime.GOOS != "windows" && !exists(filepath.Join(h, "augmented-brain")) {
		t.Error("the folder that holds the user's file was removed")
	}
}

func TestUninstallAsksFirst(t *testing.T) {
	h, cfg := home(t)
	var out, errOut bytes.Buffer
	if code := cmdUninstall([]string{"--all"}, strings.NewReader("no\n"), &out, &errOut); code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !exists(cfg) || !exists(filepath.Join(h, "augmented-brain", "trabajo", "notas")) || !exists(filepath.Join(h, ".claude", "plugins", "cache", "augmented-brain")) {
		t.Error("answering no removed something")
	}
}

func uninstall(t *testing.T, args ...string) string {
	t.Helper()
	var out, errOut bytes.Buffer
	if code := cmdUninstall(args, strings.NewReader(""), &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s%s", code, out.String(), errOut.String())
	}
	return out.String()
}

// R-01: notes in the same folder as the work. --all used to remove the folder whole,
// repositories and all.
func TestUninstallAllKeepsNotesThatHoldTheWork(t *testing.T) {
	h, cfg := home(t)
	work := filepath.Join(h, "augmented-brain", "trabajo")
	if err := config.SetSpaceKey(cfg, "trabajo", "memory", work); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"MEMORY.md", filepath.Join("repo-a", "main.go")} {
		os.MkdirAll(filepath.Join(work, filepath.Dir(f)), 0o755)
		if err := os.WriteFile(filepath.Join(work, f), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	out := uninstall(t, "--all", "--yes")
	if !exists(filepath.Join(work, "repo-a", "main.go")) || !exists(filepath.Join(work, "MEMORY.md")) {
		t.Fatal("a notes folder that holds the work folder was touched")
	}
	if !strings.Contains(out, "contiene la carpeta de trabajo") || !strings.Contains(out, "se quedan como estaban") {
		t.Errorf("the plan does not say the folder stays:\n%s", out)
	}
	if exists(filepath.Join(h, "augmented-brain", "personal", "notas")) {
		t.Error("the other space's notes should go")
	}
}

func TestUninstallAllTakesOnlyNotes(t *testing.T) {
	h, _ := home(t)
	notes := filepath.Join(h, "augmented-brain", "personal", "notas")
	mine := filepath.Join(notes, "mio.txt")
	if err := os.WriteFile(mine, []byte("mío"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := uninstall(t, "--all", "--yes")
	if !exists(mine) {
		t.Fatal("a file that is not a note was removed")
	}
	if exists(filepath.Join(notes, "MEMORY.md")) || exists(filepath.Join(notes, "tasks")) || exists(filepath.Join(notes, ".augmented-brain")) {
		t.Error("the notes stayed")
	}
	if !strings.Contains(out, "los otros 1 ficheros o carpetas") {
		t.Errorf("the plan does not say what stays:\n%s", out)
	}
}

func TestUninstallAllRefusesWhatIsNotNotes(t *testing.T) {
	h, cfg := home(t)
	docs := filepath.Join(h, "Documents")
	mine := filepath.Join(docs, "carta.md")
	os.MkdirAll(docs, 0o755)
	if err := os.WriteFile(mine, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := config.SetSpaceKey(cfg, "personal", "memory", docs); err != nil {
		t.Fatal(err)
	}
	out := uninstall(t, "--all", "--yes")
	if !exists(mine) || !strings.Contains(out, "no parece una carpeta de notas") {
		t.Errorf("a folder with no notes was emptied, or the plan does not say:\n%s", out)
	}
	// The home folder, never, whatever it holds.
	if why, _ := checkNotes("en", h, h, nil, false); why == "" {
		t.Error("the home folder may be emptied")
	}
}

func TestUninstallConfigNeedsAConfiguration(t *testing.T) {
	h, _ := home(t)
	other := filepath.Join(h, "other")
	os.MkdirAll(other, 0o755)
	os.WriteFile(filepath.Join(other, "keep.txt"), []byte("x"), 0o644)
	t.Setenv("AB_CONFIG_DIR", other)
	out := uninstall(t, "--config", "--yes")
	if !exists(filepath.Join(other, "keep.txt")) || !strings.Contains(out, "⚠ "+paths.Tilde(other)) {
		t.Errorf("a folder with no configuration was removed:\n%s", out)
	}
}

// R-03: uninstall used to leave the remote access running, with no guard and no
// program to take it away.
func TestUninstallTakesTheRemoteAccessFirst(t *testing.T) {
	h, _ := home(t)
	calls := useFakeSystem(t, h, "linux")
	var out, errOut bytes.Buffer
	if code := run([]string{"remote", "install"}, nil, &out, &errOut); code != 0 {
		t.Fatalf("install: %d\n%s%s", code, out.String(), errOut.String())
	}
	*calls = nil
	plan := uninstall(t, "--yes")
	if !strings.Contains(plan, "acceso remoto") || !strings.Contains(plan, "augmented-brain-remote-personal.service") {
		t.Errorf("the plan does not name the remote access:\n%s", plan)
	}
	units, _ := os.ReadDir(filepath.Join(h, "units"))
	for _, u := range units {
		if data, _ := os.ReadFile(filepath.Join(h, "units", u.Name())); strings.Contains(string(data), "augmented brain") {
			t.Errorf("unit still there: %s", u.Name())
		}
	}
	if !slices.Contains(*calls, "systemctl --user disable --now augmented-brain-remote-personal.service") {
		t.Errorf("the service was not stopped: %q", *calls)
	}
}

func TestUninstallClosesTaskSessionsAndStopsWhenItCannot(t *testing.T) {
	h, cfg := home(t)
	rec := filepath.Join(cfg, "state", "sessions")
	os.MkdirAll(rec, 0o700)
	os.WriteFile(filepath.Join(rec, "task-a.json"), []byte(`{"id":"abcdef12","task":"task-a","folder":"/x"}`), 0o600)
	var calls []string
	failRm := true
	remoteSystem = func() (remote.System, error) {
		return remote.System{
			Run: func(name string, args ...string) (string, error) {
				calls = append(calls, strings.Join(args, " "))
				switch args[0] {
				case "agents":
					return `[{"id":"abcdef12","pid":42}]`, nil
				case "rm":
					if failRm {
						return "", errors.New("busy")
					}
				}
				return "", nil
			},
			LookPath: func(string) (string, error) { return filepath.Join(h, "bin", "claude"), nil },
			Hostname: func() (string, error) { return "box", nil },
			GOOS:     "linux", UnitDir: filepath.Join(h, "units"),
		}, nil
	}
	var out, errOut bytes.Buffer
	if code := cmdUninstall([]string{"--yes"}, strings.NewReader(""), &out, &errOut); code != 1 {
		t.Fatalf("exit %d, want 1: %s%s", code, out.String(), errOut.String())
	}
	if !strings.Contains(out.String(), "task-a") || !strings.Contains(errOut.String(), "No he quitado nada más") {
		t.Errorf("plan or error:\n%s%s", out.String(), errOut.String())
	}
	if !exists(filepath.Join(h, ".claude", "plugins", "cache", "augmented-brain")) {
		t.Error("the plugin went although a session could not be closed")
	}
	failRm = false
	uninstall(t, "--yes")
	if !slices.Contains(calls, "stop abcdef12") || exists(filepath.Join(rec, "task-a.json")) {
		t.Errorf("the session was not closed: %q", calls)
	}
}
