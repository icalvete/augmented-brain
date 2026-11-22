package panel

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/icalvete/augmented-brain/internal/remote"
)

// Closing a task from the panel ends its own session, as task_close does (P6 of the
// documentation review): once its answer ends, through the detached stop.
func TestCloseFromThePanelStopsTheSession(t *testing.T) {
	s, cfg := fixture(t, "en")
	if w := s.post(t, "/task/new", url.Values{"s": {"work"}, "title": {"Site"}}); w.Code != http.StatusSeeOther {
		t.Fatalf("new task: %d %s", w.Code, w.Body)
	}
	if err := os.MkdirAll(filepath.Join(cfg, "state", "sessions"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg, "state", "sessions", "task-site.json"), []byte(`{"id":"abc12345","task":"task-site"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	oldSys, oldStop := newRemoteSystem, stopWhenIdle
	t.Cleanup(func() { newRemoteSystem, stopWhenIdle = oldSys, oldStop })
	newRemoteSystem = func() (remote.System, error) {
		return remote.System{GOOS: "linux",
			LookPath: func(string) (string, error) { return "/usr/bin/claude", nil },
			Run: func(name string, args ...string) (string, error) {
				if len(args) > 0 && args[0] == "agents" {
					return `[{"pid":4242,"id":"abc12345","status":"busy","state":"working"}]`, nil
				}
				return "", nil
			}}, nil
	}
	var stopped []string
	stopWhenIdle = func(dir, task string) error { stopped = append(stopped, dir+" "+task); return nil }
	if w := s.post(t, "/task/close", url.Values{"s": {"work"}, "id": {"task-site"}, "result": {"Done"}}); w.Code != http.StatusSeeOther {
		t.Fatalf("close: %d %s", w.Code, w.Body)
	}
	if len(stopped) != 1 || stopped[0] != cfg+" task-site" {
		t.Errorf("stopped %v", stopped)
	}
}

// Parking a task from the panel ends its own session and keeps its worktrees, as
// task_park does.
func TestParkFromThePanelStopsTheSession(t *testing.T) {
	s, cfg := fixture(t, "en")
	if w := s.post(t, "/task/new", url.Values{"s": {"work"}, "title": {"Site"}}); w.Code != http.StatusSeeOther {
		t.Fatalf("new task: %d %s", w.Code, w.Body)
	}
	if err := os.MkdirAll(filepath.Join(cfg, "state", "sessions"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg, "state", "sessions", "task-site.json"), []byte(`{"id":"abc12345","task":"task-site"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	oldSys, oldStop, oldKeep := newRemoteSystem, stopWhenIdle, stopWhenIdleKeep
	t.Cleanup(func() { newRemoteSystem, stopWhenIdle, stopWhenIdleKeep = oldSys, oldStop, oldKeep })
	newRemoteSystem = func() (remote.System, error) {
		return remote.System{GOOS: "linux",
			LookPath: func(string) (string, error) { return "/usr/bin/claude", nil },
			Run: func(name string, args ...string) (string, error) {
				if len(args) > 0 && args[0] == "agents" {
					return `[{"pid":4242,"id":"abc12345","status":"busy","state":"working"}]`, nil
				}
				return "", nil
			}}, nil
	}
	var stopped, kept []string
	stopWhenIdle = func(dir, task string) error { stopped = append(stopped, task); return nil }
	stopWhenIdleKeep = func(dir, task string) error { kept = append(kept, dir+" "+task); return nil }
	if w := s.post(t, "/task/park", url.Values{"s": {"work"}, "id": {"task-site"}, "next": {"Go on"}}); w.Code != http.StatusSeeOther {
		t.Fatalf("park: %d %s", w.Code, w.Body)
	}
	if len(kept) != 1 || kept[0] != cfg+" task-site" || len(stopped) != 0 {
		t.Errorf("kept %v stopped %v", kept, stopped)
	}
	// Moving it to validation ends its session the same way.
	kept = nil
	if w := s.post(t, "/task/update", url.Values{"s": {"work"}, "id": {"task-site"}, "status": {"validation"}}); w.Code != http.StatusSeeOther {
		t.Fatalf("update: %d %s", w.Code, w.Body)
	}
	if len(kept) != 1 || kept[0] != cfg+" task-site" || len(stopped) != 0 {
		t.Errorf("validation: kept %v stopped %v", kept, stopped)
	}
}
