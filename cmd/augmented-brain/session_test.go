package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/icalvete/augmented-brain/internal/remote"
)

// `session stop` never says nothing: with no session it says so and fails (on 10-nov
// --when-idle exited 0, silent, and closed nothing); a session Claude retired is listed
// as retired, and stop closes it.
func TestSessionStopSaysWhatHappened(t *testing.T) {
	h, cfg := home(t)
	calls := useFakeSystem(t, h, "linux")
	var out, errOut bytes.Buffer
	for _, args := range [][]string{{"session", "stop", "task-site"}, {"session", "stop", "task-site", "--when-idle"}} {
		out.Reset()
		errOut.Reset()
		if code := run(args, nil, &out, &errOut); code != 1 || !strings.Contains(errOut.String(), "no tiene ninguna sesión abierta") {
			t.Errorf("%v: %d\n%s%s", args, code, out.String(), errOut.String())
		}
	}
	dir := filepath.Join(cfg, "state", "sessions")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "task-site.json"), []byte(`{"id":"abc12345","task":"task-site","folder":"/x"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	// Claude lists it with no pid, as it does a session it retired (11-nov).
	fake := remoteSystem
	remoteSystem = func() (remote.System, error) {
		s, err := fake()
		run := s.Run
		s.Run = func(name string, args ...string) (string, error) {
			if len(args) > 0 && args[0] == "agents" {
				*calls = append(*calls, name+" "+strings.Join(args, " "))
				return `[{"id":"abc12345","kind":"background","state":"blocked"}]`, nil
			}
			return run(name, args...)
		}
		return s, err
	}
	out.Reset()
	if code := run([]string{"session", "list"}, nil, &out, &errOut); code != 0 || !strings.Contains(out.String(), "task-site · retirada") {
		t.Errorf("list: %d\n%s%s", code, out.String(), errOut.String())
	}
	*calls = nil
	out.Reset()
	if code := run([]string{"session", "stop", "task-site"}, nil, &out, &errOut); code != 0 || !strings.Contains(out.String(), "Cerrada la sesión de task-site") {
		t.Errorf("stop: %d\n%s%s", code, out.String(), errOut.String())
	}
	if !strings.Contains(strings.Join(*calls, "\n"), "rm abc12345") {
		t.Errorf("calls: %v", *calls)
	}
}

// A session whose first turn failed on an expired login (11-nov) is listed as failed,
// with Claude's words and what to do, not as active.
func TestSessionListFailed(t *testing.T) {
	h, cfg := home(t)
	useFakeSystem(t, h, "linux")
	dir := filepath.Join(cfg, "state", "sessions")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "task-site.json"), []byte(`{"id":"abc12345","task":"task-site","folder":"/x"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	claudeDir := filepath.Join(h, ".claude")
	project := filepath.Join(claudeDir, "projects", "-x")
	if err := os.MkdirAll(project, 0o700); err != nil {
		t.Fatal(err)
	}
	transcript := `{"type":"user","message":{"role":"user","content":"go"}}` + "\n" +
		`{"type":"assistant","message":{"model":"<synthetic>","content":[{"type":"text","text":"Login reached its time limit · Please run /login"}]},"error":"authentication_failed","isApiErrorMessage":true}` + "\n"
	if err := os.WriteFile(filepath.Join(project, "abc12345-0000.jsonl"), []byte(transcript), 0o600); err != nil {
		t.Fatal(err)
	}
	fake := remoteSystem
	remoteSystem = func() (remote.System, error) {
		s, err := fake()
		s.ClaudeDir = claudeDir
		run := s.Run
		s.Run = func(name string, args ...string) (string, error) {
			if len(args) > 0 && args[0] == "agents" {
				return `[{"pid":42,"id":"abc12345","sessionId":"abc12345-0000","kind":"background","status":"idle","state":"blocked"}]`, nil
			}
			return run(name, args...)
		}
		return s, err
	}
	var out, errOut bytes.Buffer
	code := run([]string{"session", "list"}, nil, &out, &errOut)
	if code != 0 || !strings.Contains(out.String(), "task-site · fallida: caducó el login de Claude Code (Login reached its time limit · Please run /login)") ||
		!strings.Contains(out.String(), "claude auth login") || strings.Contains(out.String(), "active") {
		t.Errorf("list: %d\n%s%s", code, out.String(), errOut.String())
	}
}

// `session stop --when-idle` run from the session it closes cannot wait there: that
// session answers until the command returns, so it would never be idle (#99). With a
// session on record it starts the detached copy and returns at once; the copy
// (StopDetached set) does the waiting.
func TestSessionStopWhenIdleDetaches(t *testing.T) {
	h, cfg := home(t)
	useFakeSystem(t, h, "linux")
	dir := filepath.Join(cfg, "state", "sessions")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "task-site.json"), []byte(`{"id":"abc12345","task":"task-site","folder":"/x"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	var started []string
	old := stopDetached
	t.Cleanup(func() { stopDetached = old })
	stopDetached = func(cfgDir, task string) error { started = append(started, task); return nil }
	var out, errOut bytes.Buffer
	if code := run([]string{"session", "stop", "task-site", "--when-idle"}, nil, &out, &errOut); code != 0 || !strings.Contains(out.String(), "se cerrará cuando acabe su respuesta") || !strings.Contains(out.String(), "session-stop.log") {
		t.Errorf("stop --when-idle: %d\n%s%s", code, out.String(), errOut.String())
	}
	if len(started) != 1 || started[0] != "task-site" {
		t.Errorf("detached: %v", started)
	}
	// The detached copy waits in place, never starting another: here Claude no longer
	// lists the session, so it counts as idle and is closed.
	fake := remoteSystem
	remoteSystem = func() (remote.System, error) {
		s, err := fake()
		run := s.Run
		s.Run = func(name string, args ...string) (string, error) {
			if len(args) > 0 && args[0] == "agents" {
				return `[]`, nil
			}
			return run(name, args...)
		}
		s.Sleep = func(time.Duration) {}
		return s, err
	}
	t.Setenv("AB_STOP_DETACHED", "1")
	started = nil
	out.Reset()
	errOut.Reset()
	run([]string{"session", "stop", "task-site", "--when-idle"}, nil, &out, &errOut)
	if len(started) != 0 || !strings.Contains(out.String(), "Cerrada la sesión de task-site") {
		t.Errorf("copy detached again: %v\n%s%s", started, out.String(), errOut.String())
	}
}
