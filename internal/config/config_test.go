package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestLoadAndSpaceFor(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	for _, d := range []string{"work", "work/clients", "personal"} {
		if err := os.MkdirAll(filepath.Join(home, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	cfgDir := t.TempDir()
	write(t, cfgDir, `
language = "es"

[spaces.work]
memory = "~/brains/work"
work_folders = ["~/work"]
production_aws_profiles = ["prod"]

[spaces.clients]
memory = "~/brains/clients"
work_folders = ["~/work/clients"]

[spaces.personal]
memory = "~/brains/personal"
work_folders = ["~/personal"]
`)
	c, err := Load(cfgDir)
	if err != nil {
		t.Fatal(err)
	}
	if c.Language != "es" {
		t.Errorf("language = %q", c.Language)
	}
	if got := c.Spaces["work"].Memory; got != filepath.Join(home, "brains/work") {
		t.Errorf("memory not expanded: %q", got)
	}

	cases := map[string]string{
		"work":              "work",
		"work/project/src":  "work",
		"work/clients":      "clients", // the deepest folder wins
		"work/clients/acme": "clients",
		"personal":          "personal",
		"workshop":          "", // a name that only starts like a work folder
		"elsewhere":         "",
	}
	for rel, want := range cases {
		s := c.SpaceFor(filepath.Join(home, rel))
		got := ""
		if s != nil {
			got = s.Name
		}
		if got != want {
			t.Errorf("SpaceFor(%s) = %q, want %q", rel, got, want)
		}
	}
}

func TestLoadErrors(t *testing.T) {
	cases := map[string]string{
		"": "no configuration found",
		`language = "fr"` + "\n[spaces.a]\nmemory = \"/m\"": "language",
		`language = "en"`:                            "no spaces",
		"[spaces.a]\nwork_folders = [\"/w\"]":        "memory folder is missing",
		"[spaces.a]\nmemory = \"/m\"\nbakup = \"x\"": "unknown settings",
	}
	for body, want := range cases {
		dir := t.TempDir()
		if body != "" {
			write(t, dir, body)
		}
		_, err := Load(dir)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("config %q: error %v, want it to mention %q", body, err, want)
		}
		if body == "" && !errors.Is(err, ErrNotFound) {
			t.Errorf("missing file: want ErrNotFound, got %v", err)
		}
	}
}

func TestDir(t *testing.T) {
	t.Setenv("AB_CONFIG_DIR", "/tmp/ab-test")
	if d, _ := Dir(); d != "/tmp/ab-test" {
		t.Errorf("AB_CONFIG_DIR ignored: %q", d)
	}
	t.Setenv("AB_CONFIG_DIR", "")
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg")
	if d, _ := Dir(); d != "/tmp/xdg/augmented-brain" {
		t.Errorf("XDG_CONFIG_HOME ignored: %q", d)
	}
}
