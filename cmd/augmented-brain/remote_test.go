package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/icalvete/augmented-brain/internal/brain"
	"github.com/icalvete/augmented-brain/internal/config"

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

// The watch's look at the sessions the usage limit stopped: the session of a task is
// stopped and resumed with a message once the limit has reset, the task's journal says
// so, and the user gets two notices in all.
func TestRemoteWatchWakesQuota(t *testing.T) {
	h, cfg := home(t)
	c, err := config.Load(cfg)
	if err != nil {
		t.Fatal(err)
	}
	b, err := brain.Open(c, c.Spaces["personal"])
	if err != nil {
		t.Fatal(err)
	}
	task, err := b.NewTask(brain.NewTaskInput{Title: "Un cambio"})
	if err != nil {
		t.Fatal(err)
	}
	id := task.Header.ID
	folder := filepath.Join(h, "augmented-brain", "personal")
	dir := filepath.Join(cfg, "state", "sessions")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	record, _ := json.Marshal(map[string]string{"id": "abc12345", "task": id, "folder": folder})
	if err := os.WriteFile(filepath.Join(dir, id+".json"), record, 0o600); err != nil {
		t.Fatal(err)
	}
	claudeDir := filepath.Join(h, ".claude")
	project := filepath.Join(claudeDir, "projects", "-x")
	if err := os.MkdirAll(project, 0o700); err != nil {
		t.Fatal(err)
	}
	limit := func(resets time.Time) {
		line := fmt.Sprintf(`{"type":"assistant","message":{"model":"<synthetic>","content":[{"type":"text","text":"You've hit your 5-hour limit"}]},"apiError":"usage_limit_reached","quotaLimits":{"resetsAt":%d},"error":"rate_limit","isApiErrorMessage":true}`, resets.Unix())
		if err := os.WriteFile(filepath.Join(project, "abc12345-0000.jsonl"), []byte(line+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	pid := 42
	var resumed []string
	fake := remoteSystem
	remoteSystem = func() (remote.System, error) {
		s, err := fake()
		s.ClaudeDir = claudeDir
		s.Children = func(int) []int { return nil }
		s.Sleep = func(time.Duration) {}
		run := s.Run
		s.Run = func(name string, args ...string) (string, error) {
			switch {
			case len(args) > 0 && args[0] == "agents" && pid > 0:
				return fmt.Sprintf(`[{"pid":%d,"id":"abc12345","sessionId":"abc12345-0000","kind":"background","status":"idle","state":"blocked"}]`, pid), nil
			case len(args) > 0 && args[0] == "agents":
				return `[{"id":"abc12345","sessionId":"abc12345-0000","kind":"background","state":"stopped"}]`, nil
			case len(args) > 0 && args[0] == "stop":
				pid = 0
			}
			return run(name, args...)
		}
		s.RunIn = func(dir string, name string, args ...string) (string, error) {
			resumed = append(resumed, dir+" | "+strings.Join(args, " "))
			pid = 43
			return "backgrounded · abc12345 · x\n", nil
		}
		return s, err
	}

	// Before the reset: the user is told once, the session is left alone, and the list
	// says what it waits for.
	limit(time.Now().Add(time.Hour))
	var out, errOut bytes.Buffer
	if code := run([]string{"remote", "watch"}, nil, &out, &errOut); code != 0 {
		t.Fatalf("watch: %d\n%s%s", code, out.String(), errOut.String())
	}
	notice, _ := os.ReadFile(filepath.Join(cfg, "state", "quota-notice"))
	if !strings.Contains(string(notice), "Se despiertan solas cuando vuelva") || len(resumed) != 0 {
		t.Fatalf("before the reset: notice %q, resumed %v\n%s", notice, resumed, out.String())
	}
	out.Reset()
	if code := run([]string{"session", "list"}, nil, &out, &errOut); code != 0 || !strings.Contains(out.String(), id+" · en espera: la paró el límite de uso de Claude") {
		t.Fatalf("list: %d\n%s%s", code, out.String(), errOut.String())
	}
	out.Reset()
	if code := run([]string{"remote", "watch"}, nil, &out, &errOut); code != 0 || out.String() != "" {
		t.Fatalf("a second look says it again: %d\n%s%s", code, out.String(), errOut.String())
	}

	// After it: woken with a message, in its folder, under its own id.
	limit(time.Now().Add(-time.Hour))
	if code := run([]string{"remote", "watch"}, nil, &out, &errOut); code != 0 {
		t.Fatalf("watch: %d\n%s%s", code, out.String(), errOut.String())
	}
	if len(resumed) != 1 || !strings.HasPrefix(resumed[0], folder+" | --bg --resume abc12345-0000 El límite de uso de Claude paró esta sesión") || !strings.Contains(resumed[0], "task_get "+id) {
		t.Fatalf("resumed: %v\n%s", resumed, out.String())
	}
	notice, _ = os.ReadFile(filepath.Join(cfg, "state", "quota-notice"))
	if n := strings.Count(strings.TrimSpace(string(notice)), "\n") + 1; n != 2 || !strings.Contains(string(notice), "ya trabajan otra vez (1)") {
		// one when the wait began, one when it ended
		t.Fatalf("notices: %q", notice)
	}
	if b, err = brain.Open(c, c.Spaces["personal"]); err != nil { // as the notes are now
		t.Fatal(err)
	}
	got, err := b.Task(id)
	if err != nil || !strings.Contains(got.Body, "augmented brain ha despertado la sesión") {
		t.Fatalf("journal: %v\n%s", err, got.Body)
	}
}
