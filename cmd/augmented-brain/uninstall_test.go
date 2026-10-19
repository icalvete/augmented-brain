package main

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/setup"
)

// home sets up two spaces in a fake home, the way the installer does, plus what Claude
// Code leaves of the plugin, and returns the home and the configuration folder.
func home(t *testing.T) (string, string) {
	t.Helper()
	h := t.TempDir()
	t.Setenv("HOME", h)
	t.Setenv("USERPROFILE", h)
	t.Setenv("XDG_CACHE_HOME", filepath.Join(h, ".cache"))
	t.Setenv("LOCALAPPDATA", filepath.Join(h, "AppData", "Local"))
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(h, ".claude"))
	t.Setenv("PATH", t.TempDir()) // no claude: the plugin commands are not run
	cfg := filepath.Join(h, "cfg")
	t.Setenv("AB_CONFIG_DIR", cfg)
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

func exists(p string) bool { _, err := os.Stat(p); return err == nil }

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
