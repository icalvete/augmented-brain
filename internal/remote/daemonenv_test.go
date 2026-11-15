package remote

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

// What Claude's daemon gets passes by a list: no AB_* switch, no task, no token of the
// terminal that happened to start it; what a session needs to run does pass (R-07).
func TestDaemonEnv(t *testing.T) {
	env := []string{
		"HOME=/home/lucia", "PATH=/usr/bin", "LANG=es_ES.UTF-8", "LC_TIME=es_ES.UTF-8",
		"XDG_RUNTIME_DIR=/run/user/1000", "SSH_AUTH_SOCK=/run/ssh", "CLAUDE_CONFIG_DIR=/c",
		"https_proxy=http://proxy:3128", "SystemRoot=C:\\Windows",
		"AB_CONFIG_DIR=/tmp/throwaway", "AB_STOP_DETACHED=1", "AB_NO_AUTO_BACKUP=1",
		"AB_RESTART_DETACHED=1", "AB_WATCH_RESULT=/tmp/x", TaskEnv + "=task-x",
		"SECRET_TOKEN=s3cret", "GH_TOKEN=gh", "AWS_SECRET_ACCESS_KEY=k", "AWS_PROFILE=prod",
		"=C:=C:\\odd", "NOEQUALS",
	}
	got := daemonEnv(env)
	for _, want := range []string{"HOME=/home/lucia", "PATH=/usr/bin", "LANG=es_ES.UTF-8", "LC_TIME=es_ES.UTF-8",
		"XDG_RUNTIME_DIR=/run/user/1000", "SSH_AUTH_SOCK=/run/ssh", "CLAUDE_CONFIG_DIR=/c",
		"https_proxy=http://proxy:3128", "SystemRoot=C:\\Windows"} {
		if !slices.Contains(got, want) {
			t.Errorf("%s did not pass: %v", want, got)
		}
	}
	for _, kv := range got {
		name, _, _ := strings.Cut(kv, "=")
		if strings.HasPrefix(name, "AB_") || strings.HasPrefix(name, "AUGMENTED_BRAIN") ||
			strings.HasPrefix(name, "AWS_") || name == "SECRET_TOKEN" || name == "GH_TOKEN" || name == "" {
			t.Errorf("%s passed", kv)
		}
	}
}

// The environment a fake claude --bg gets from RunIn, as the daemon would get it.
func TestRunInGivesTheDaemonEnv(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("a shell script stands for claude")
	}
	dir := t.TempDir()
	dump := filepath.Join(dir, "env")
	fake := filepath.Join(dir, "claude")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\nenv > '"+dump+"'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AB_CONFIG_DIR", "/tmp/throwaway-config")
	t.Setenv("AB_STOP_DETACHED", "1")
	t.Setenv("AB_NO_AUTO_BACKUP", "1")
	t.Setenv(TaskEnv, "task-x")
	t.Setenv("SECRET_TOKEN", "s3cret")
	t.Setenv("HOME", dir)
	if _, err := realRunIn(dir, fake, "--bg"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(dump)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(data), "\n")
	for _, l := range lines {
		if strings.HasPrefix(l, "AB_") || strings.HasPrefix(l, TaskEnv+"=") || strings.HasPrefix(l, "SECRET_TOKEN=") {
			t.Errorf("claude got %s", l)
		}
	}
	if !slices.Contains(lines, "HOME="+dir) || !slices.ContainsFunc(lines, func(l string) bool { return strings.HasPrefix(l, "PATH=") }) {
		t.Errorf("HOME or PATH missing:\n%s", data)
	}
}
