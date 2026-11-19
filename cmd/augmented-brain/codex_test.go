package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/codex"
)

func TestCodexCommand(t *testing.T) {
	t.Setenv("AB_CONFIG_DIR", t.TempDir())
	t.Setenv("AB_LANGUAGE", "en")
	available, env := codexAvailable, codexEnv
	t.Cleanup(func() { codexAvailable, codexEnv = available, env })

	var out, errOut bytes.Buffer
	if code := run([]string{"codex"}, nil, &out, &errOut); code != 2 || !strings.Contains(errOut.String(), "codex install | uninstall | status") {
		t.Errorf("no subcommand: %d %q", code, errOut.String())
	}

	codexAvailable = func() bool { return false }
	out.Reset()
	errOut.Reset()
	if code := run([]string{"codex", "install"}, nil, &out, &errOut); code != 1 || !strings.Contains(errOut.String(), "Install Codex first") {
		t.Errorf("without Codex: %d %q", code, errOut.String())
	}

	codexAvailable = func() bool { return true }
	codexEnv = func(dir string) (codex.Env, error) {
		return codex.Env{Dir: codex.Dir(dir), Serve: func() (codex.Server, error) { return nil, errors.New("no app server") },
			Run: func(string, ...string) (string, error) { return "", errors.New("exit status 1") }}, nil
	}
	out.Reset()
	errOut.Reset()
	if code := run([]string{"codex", "status"}, nil, &out, &errOut); code != 1 || !strings.Contains(out.String(), "no app server") {
		t.Errorf("status when Codex does not answer: %d %q", code, out.String())
	}
	if code := run([]string{"codex", "install"}, nil, &out, &errOut); code != 1 || !strings.Contains(errOut.String(), "adding the marketplace") {
		t.Errorf("install when Codex fails: %d %q", code, errOut.String())
	}
}

func TestCodexStatusLines(t *testing.T) {
	full := codex.Status{Installed: true, Hooks: codex.HookCount, Trusted: codex.HookCount, TTYOff: true}
	if l := codexStatusLines("en", full); len(l) != 1 || !strings.Contains(l[0], "is installed") {
		t.Errorf("ok: %q", l)
	}
	if l := codexStatusLines("en", codex.Status{}); len(l) != 1 || !strings.Contains(l[0], "not installed") {
		t.Errorf("none: %q", l)
	}
	broken := full
	broken.TTYOff, broken.Trusted = false, 2
	l := codexStatusLines("en", broken)
	if len(l) != 3 || !strings.Contains(l[2], codex.Fix) {
		t.Errorf("broken: %q", l)
	}
}
