package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/icalvete/augmented-brain/internal/brain"
	"github.com/icalvete/augmented-brain/internal/config"
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

// `session stop --when-idle --keep-worktrees`, what parking a task starts, detaches the
// copy that keeps the worktrees, not the one closing a task uses.
func TestSessionStopKeepWorktreesDetaches(t *testing.T) {
	h, cfg := home(t)
	useFakeSystem(t, h, "linux")
	dir := filepath.Join(cfg, "state", "sessions")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "task-site.json"), []byte(`{"id":"abc12345","task":"task-site","folder":"/x"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	var plain, keep []string
	oldPlain, oldKeep := stopDetached, stopDetachedKeep
	t.Cleanup(func() { stopDetached, stopDetachedKeep = oldPlain, oldKeep })
	stopDetached = func(cfgDir, task string) error { plain = append(plain, task); return nil }
	stopDetachedKeep = func(cfgDir, task string) error { keep = append(keep, task); return nil }
	var out, errOut bytes.Buffer
	if code := run([]string{"session", "stop", "task-site", "--when-idle", "--keep-worktrees"}, nil, &out, &errOut); code != 0 {
		t.Errorf("stop: %d\n%s%s", code, out.String(), errOut.String())
	}
	if len(plain) != 0 || len(keep) != 1 || keep[0] != "task-site" {
		t.Errorf("plain %v keep %v", plain, keep)
	}
}

// Started again without --folder, a task's session Claude retired comes back where it
// worked, with its conversation (claude respawn), not as a second session in the
// space's folder: on 22-nov a session opened with --folder and started again without
// it opened a new one, and the first stayed in claude.ai, disconnected.
func TestSessionStartWakesInItsOwnFolder(t *testing.T) {
	h, cfg := home(t)
	calls := useFakeSystem(t, h, "linux")
	work := filepath.Join(h, "augmented-brain", "trabajo")
	t.Chdir(work)
	c, err := config.Load(cfg)
	if err != nil {
		t.Fatal(err)
	}
	b, err := brain.Open(c, c.Spaces["trabajo"])
	if err != nil {
		t.Fatal(err)
	}
	n, err := b.NewTask(brain.NewTaskInput{Title: "Site"})
	if err != nil {
		t.Fatal(err)
	}
	own := filepath.Join(work, "sesiones", "site")
	if err := os.MkdirAll(own, 0o755); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(cfg, "state", "sessions")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	rec := `{"id":"abc12345","task":"` + n.Header.ID + `","folder":"` + own + `"}`
	if err := os.WriteFile(filepath.Join(dir, n.Header.ID+".json"), []byte(rec), 0o600); err != nil {
		t.Fatal(err)
	}
	// Retired: listed with no pid until respawn, then running.
	respawned := false
	fake := remoteSystem
	remoteSystem = func() (remote.System, error) {
		s, err := fake()
		run := s.Run
		s.Run = func(name string, args ...string) (string, error) {
			if len(args) > 0 && args[0] == "agents" {
				if respawned {
					return `[{"id":"abc12345","pid":7,"kind":"background","status":"idle"}]`, nil
				}
				return `[{"id":"abc12345","kind":"background","state":"done"}]`, nil
			}
			if len(args) > 0 && args[0] == "respawn" {
				respawned = true
			}
			return run(name, args...)
		}
		return s, err
	}
	var out, errOut bytes.Buffer
	if code := run([]string{"session", "start", n.Header.ID}, nil, &out, &errOut); code != 0 || !strings.Contains(out.String(), "Recuperada") {
		t.Fatalf("start: %d\n%s%s", code, out.String(), errOut.String())
	}
	all := strings.Join(*calls, "\n")
	if !strings.Contains(all, "respawn abc12345") || strings.Contains(all, "--bg") {
		t.Errorf("calls:\n%s", all)
	}
}
