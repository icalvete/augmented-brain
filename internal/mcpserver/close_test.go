package mcpserver

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/remote"
)

// Closing a task with an open watch date says so: it keeps coming up at the start of
// the session (the case of 7-nov: a check for the 8th in a task closed on the 7th).
func TestCloseTellsOpenWatches(t *testing.T) {
	cs := connect(t, space(t))
	mustCall(t, cs, "task_new", map[string]any{"title": "Rollback"})
	mustCall(t, cs, "task_update", map[string]any{"id": "task-rollback", "watch_date": "2026-11-08", "watch_what": "descriptions still in Jenkins"})
	mustCall(t, cs, "task_update", map[string]any{"id": "task-rollback", "watch_date": "2026-11-01", "watch_what": "already checked"})
	mustCall(t, cs, "task_update", map[string]any{"id": "task-rollback", "watch_done": "2026-11-01"})
	out := mustCall(t, cs, "task_close", map[string]any{"id": "task-rollback", "result": "done"})
	if !strings.Contains(out, "1 open watch date") || !strings.Contains(out, "2026-11-08: descriptions still in Jenkins") || strings.Contains(out, "already checked") {
		t.Fatalf("task_close:\n%s", out)
	}
	// it still comes up: pending lists the watch of the closed task
	if p := mustCall(t, cs, "pending", map[string]any{"within_days": 400}); !strings.Contains(p, "descriptions still in Jenkins") {
		t.Fatalf("pending:\n%s", p)
	}
	mustCall(t, cs, "task_new", map[string]any{"title": "Plain"})
	if out := mustCall(t, cs, "task_close", map[string]any{"id": "task-plain", "result": "done"}); strings.Contains(out, "watch") {
		t.Fatalf("no watches, yet:\n%s", out)
	}
}

// Closing a task whose own session runs has a process of its own close that session
// once it has finished its answer (often the closing one): never at once.
func TestCloseStopsTheTaskSession(t *testing.T) {
	cs := connect(t, space(t))
	cfgDir := filepath.Join(os.Getenv("HOME"), "cfg")
	mustCall(t, cs, "task_new", map[string]any{"title": "Site"})
	if err := os.MkdirAll(filepath.Join(cfgDir, "state", "sessions"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "state", "sessions", "task-site.json"), []byte(`{"id":"abc12345","task":"task-site"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	oldSys, oldStop := NewRemoteSystem, StopSessionWhenIdle
	t.Cleanup(func() { NewRemoteSystem, StopSessionWhenIdle = oldSys, oldStop })
	NewRemoteSystem = func() (remote.System, error) {
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
	StopSessionWhenIdle = func(dir, task string) error { stopped = append(stopped, dir+" "+task); return nil }
	out := mustCall(t, cs, "task_close", map[string]any{"id": "task-site", "result": "done"})
	if len(stopped) != 1 || stopped[0] != cfgDir+" task-site" || !strings.Contains(out, "closes when it finishes this answer") {
		t.Fatalf("stopped %v:\n%s", stopped, out)
	}
	// A task whose session Claude retired (listed with no pid, 11-nov): closed at once,
	// no process waits for it.
	mustCall(t, cs, "task_new", map[string]any{"title": "Docs"})
	if err := os.WriteFile(filepath.Join(cfgDir, "state", "sessions", "task-docs.json"), []byte(`{"id":"def67890","task":"task-docs"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	var calls []string
	NewRemoteSystem = func() (remote.System, error) {
		return remote.System{GOOS: "linux",
			LookPath: func(string) (string, error) { return "/usr/bin/claude", nil },
			Run: func(name string, args ...string) (string, error) {
				calls = append(calls, strings.Join(args, " "))
				if len(args) > 0 && args[0] == "agents" {
					return `[{"id":"def67890","state":"blocked"}]`, nil
				}
				return "", nil
			}}, nil
	}
	out = mustCall(t, cs, "task_close", map[string]any{"id": "task-docs", "result": "done"})
	if len(stopped) != 1 || !strings.Contains(out, "Claude had stopped") || !slices.Contains(calls, "rm def67890") {
		t.Fatalf("stopped %v, calls %v:\n%s", stopped, calls, out)
	}
	if _, err := os.Stat(filepath.Join(cfgDir, "state", "sessions", "task-docs.json")); !os.IsNotExist(err) {
		t.Errorf("the record of the retired session stays: %v", err)
	}
	// A task with no session of its own starts nothing.
	mustCall(t, cs, "task_new", map[string]any{"title": "Plain"})
	mustCall(t, cs, "task_close", map[string]any{"id": "task-plain", "result": "done"})
	if len(stopped) != 1 {
		t.Errorf("stopped %v", stopped)
	}
}

// task_worktree notes a worktree for the task; closing the task removes it when it
// holds nothing of its own, and says why it stays otherwise.
func TestTaskWorktreeTool(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git")
	}
	cs := connect(t, space(t))
	work := filepath.Join(os.Getenv("HOME"), "work")
	mustCall(t, cs, "task_new", map[string]any{"title": "Site"})
	if out, isErr := call(t, cs, "task_worktree", map[string]any{"id": "task-site"}); !isErr {
		t.Errorf("neither repo nor folder: %s", out)
	}
	if out, isErr := call(t, cs, "task_worktree", map[string]any{"id": "task-site", "folder": t.TempDir()}); !isErr || !strings.Contains(out, "not inside the work folders") {
		t.Errorf("outside the work folders: %s", out)
	}
	git := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=Lucia", "-c", "user.email=lucia@ejemplo.com", "-c", "init.defaultBranch=main"}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	repo := filepath.Join(work, "site")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	git(repo, "init", "-q")
	git(repo, "commit", "-q", "--allow-empty", "-m", "one")
	wt := filepath.Join(work, "site-fix")
	git(repo, "worktree", "add", "-q", "-b", "fix", wt)
	out := mustCall(t, cs, "task_worktree", map[string]any{"id": "task-site", "folder": wt})
	if !strings.Contains(out, "Noted") {
		t.Fatalf("note: %s", out)
	}
	// No remote: its commit is on no remote, so it stays, and closing says why.
	out = mustCall(t, cs, "task_close", map[string]any{"id": "task-site", "result": "done"})
	if !strings.Contains(out, "site-fix stays") || !strings.Contains(out, "commits not pushed") {
		t.Fatalf("close:\n%s", out)
	}
	if _, err := os.Stat(wt); err != nil {
		t.Errorf("a worktree with work was removed: %v", err)
	}
}
