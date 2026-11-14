package mcpserver

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/icalvete/augmented-brain/internal/filelock"
	"github.com/icalvete/augmented-brain/internal/remote"
)

// connectLock is connect with the notes lock file at lock.
func connectLock(t *testing.T, open Resolver, lock string) *mcp.ClientSession {
	t.Helper()
	ctx := context.Background()
	st, ct := mcp.NewInMemoryTransports()
	if _, err := New(open, lock).Connect(ctx, st, nil); err != nil {
		t.Fatal(err)
	}
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "test"}, nil).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cs.Close() })
	return cs
}

// task_worktree runs git without the notes locked: while git fetch hangs (the network
// down), another session takes the notes, and another tool of this one answers, at
// once (N-08: another session waited 17 s).
func TestTaskWorktreeGitRunsWithoutTheNotes(t *testing.T) {
	open := space(t)
	lock := filepath.Join(t.TempDir(), "lock")
	cs := connectLock(t, open, lock)
	mustCall(t, cs, "task_new", map[string]any{"title": "Site"})
	repo := filepath.Join(os.Getenv("HOME"), "work", "site")
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	inFetch, letGo := make(chan struct{}), make(chan struct{})
	old := remote.GitRun
	t.Cleanup(func() { remote.GitRun = old })
	remote.GitRun = func(dir string, args ...string) (string, error) {
		switch args[0] {
		case "fetch":
			close(inFetch)
			<-letGo
		case "worktree":
			return "", os.MkdirAll(args[len(args)-2], 0o755)
		}
		return "", nil
	}

	type result struct {
		out   string
		isErr bool
	}
	done := make(chan result, 1)
	go func() {
		res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "task_worktree",
			Arguments: map[string]any{"id": "task-site", "repo": repo}})
		if err != nil {
			done <- result{err.Error(), true}
			return
		}
		var b strings.Builder
		for _, c := range res.Content {
			if tc, ok := c.(*mcp.TextContent); ok {
				b.WriteString(tc.Text)
			}
		}
		done <- result{b.String(), res.IsError}
	}()
	select {
	case <-inFetch:
	case <-time.After(10 * time.Second):
		t.Fatal("task_worktree never reached git fetch")
	}

	// Another session (another process) takes the notes.
	unlock, err := filelock.LockWithin(lock, time.Second)
	if err != nil {
		t.Fatalf("the notes stayed locked during git fetch: %v", err)
	}
	unlock()
	// Another tool of this session answers.
	began := time.Now()
	mustCall(t, cs, "task_journal", map[string]any{"id": "task-site", "entry": "while git fetches"})
	if took := time.Since(began); took > time.Second {
		t.Errorf("task_journal waited %v for git", took)
	}

	close(letGo)
	r := <-done
	if r.isErr || !strings.Contains(r.out, "site-site") {
		t.Fatalf("task_worktree: %s", r.out)
	}
}

// A tool that finds the notes held longer than LockWait says they are busy, instead of
// hanging.
func TestToolsSayTheNotesAreBusy(t *testing.T) {
	open := space(t)
	lock := filepath.Join(t.TempDir(), "lock")
	cs := connectLock(t, open, lock)
	old := LockWait
	t.Cleanup(func() { LockWait = old })
	LockWait = 200 * time.Millisecond

	unlock, err := filelock.Lock(lock)
	if err != nil {
		t.Fatal(err)
	}
	began := time.Now()
	out, isErr := call(t, cs, "capture", map[string]any{"text": "an idea"})
	if took := time.Since(began); took > 2*time.Second {
		t.Errorf("waited %v", took)
	}
	if !isErr || !strings.Contains(out, "the notes are busy") {
		t.Fatalf("with the notes held: %v %s", isErr, out)
	}
	unlock()
	mustCall(t, cs, "capture", map[string]any{"text": "an idea"})
}
