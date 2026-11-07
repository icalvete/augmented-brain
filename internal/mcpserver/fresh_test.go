package mcpserver

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/icalvete/augmented-brain/internal/fresh"
)

// twoSessions starts two MCP servers on the same space and lock file, as two Claude
// processes on one machine would, each with its own seen file.
func twoSessions(t *testing.T) (a, b *mcp.ClientSession, cfgDir string, tasks string) {
	t.Helper()
	open := space(t)
	cfgDir = filepath.Join(os.Getenv("HOME"), "cfg")
	lock := filepath.Join(t.TempDir(), "lock")
	conn := func(key string) *mcp.ClientSession {
		ctx := context.Background()
		st, ct := mcp.NewInMemoryTransports()
		if _, err := New(open, lock, WithFresh(cfgDir, key)).Connect(ctx, st, nil); err != nil {
			t.Fatal(err)
		}
		cs, err := mcp.NewClient(&mcp.Implementation{Name: "test"}, nil).Connect(ctx, ct, nil)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { cs.Close() })
		return cs
	}
	b0, _ := open()
	tasks = filepath.Join(b0.Mem.Dir, "tasks")
	return conn("pid-1001"), conn("pid-1002"), cfgDir, tasks
}

func mustCall(t *testing.T, cs *mcp.ClientSession, name string, args map[string]any) string {
	t.Helper()
	out, isErr := call(t, cs, name, args)
	if isErr {
		t.Fatalf("%s failed: %s", name, out)
	}
	return out
}

const notice = "changed outside it"

func TestFreshAcrossSessions(t *testing.T) {
	a, b, cfgDir, tasks := twoSessions(t)
	mustCall(t, b, "task_new", map[string]any{"title": "Logo"})
	mustCall(t, b, "task_new", map[string]any{"title": "Other"})
	// A starts: SessionStart takes its baseline.
	sp := fresh.SpaceOf("work", filepath.Dir(tasks))
	if err := fresh.Baseline(cfgDir, "pid-1001", "sess-a", sp, time.Now(), false); err != nil {
		t.Fatal(err)
	}

	// Nothing changed: no notice.
	if out := mustCall(t, a, "pending", nil); strings.Contains(out, notice) {
		t.Fatalf("notice with no change:\n%s", out)
	}

	// B closes Logo: A hears it on its next tool, once.
	mustCall(t, b, "task_close", map[string]any{"id": "task-logo", "result": "done by B"})
	out := mustCall(t, a, "where_am_i", nil)
	if !strings.Contains(out, notice) || !strings.Contains(out, "task-logo") || !strings.Contains(out, "closed") {
		t.Fatalf("A did not hear of the close:\n%s", out)
	}
	if out := mustCall(t, a, "where_am_i", nil); strings.Contains(out, notice) {
		t.Fatalf("reported twice:\n%s", out)
	}

	// A's own writes are never reported back to it.
	mustCall(t, a, "task_journal", map[string]any{"id": "task-other", "entry": "mine"})
	mustCall(t, a, "task_update", map[string]any{"id": "task-other", "next": "my next"})
	if out := mustCall(t, a, "pending", nil); strings.Contains(out, notice) {
		t.Fatalf("own write reported:\n%s", out)
	}

	// B writes a journal entry in the task A worked on: A hears it.
	mustCall(t, b, "task_journal", map[string]any{"id": "task-other", "entry": "from B"})
	if out := mustCall(t, a, "pending", nil); !strings.Contains(out, "task-other") {
		t.Fatalf("a journal entry in A's task was not reported:\n%s", out)
	}

	// B creates and journals a task A never touched: only the new task is news.
	mustCall(t, b, "task_new", map[string]any{"title": "Side"})
	if out := mustCall(t, a, "pending", nil); !strings.Contains(out, "task-side") || !strings.Contains(out, "new task") {
		t.Fatalf("new task not reported:\n%s", out)
	}
	mustCall(t, b, "task_journal", map[string]any{"id": "task-side", "entry": "quiet"})
	if out := mustCall(t, a, "pending", nil); strings.Contains(out, notice) {
		t.Fatalf("a journal entry in a task A never touched was reported:\n%s", out)
	}

	// A notice also comes with a tool that fails.
	mustCall(t, b, "task_update", map[string]any{"id": "task-side", "status": "parked", "next": "later"})
	out, isErr := call(t, a, "task_get", map[string]any{"id": "task-missing"})
	if !isErr || !strings.Contains(out, "task-side") {
		t.Fatalf("the failed tool lost the notice (error %v):\n%s", isErr, out)
	}

	// B's own view: everything it did was its own, but A's writes are news to it.
	sb := fresh.SpaceOf("work", filepath.Dir(tasks))
	if err := fresh.Baseline(cfgDir, "pid-1002", "sess-b", sb, time.Now(), false); err != nil {
		t.Fatal(err)
	}
	mustCall(t, a, "task_update", map[string]any{"id": "task-other", "status": "parked", "next": "A parked it"})
	if out := mustCall(t, b, "pending", nil); !strings.Contains(out, "task-other") || !strings.Contains(out, "A parked it") {
		t.Fatalf("B did not hear of A's change:\n%s", out)
	}
}

// Many calls of one session at once (parallel tools, subagents): none of its own
// writes is ever reported back to it.
func TestFreshOwnConcurrentWrites(t *testing.T) {
	a, _, cfgDir, tasks := twoSessions(t)
	mustCall(t, a, "task_new", map[string]any{"title": "Busy"})
	mustCall(t, a, "task_new", map[string]any{"title": "Quiet"})
	if err := fresh.Baseline(cfgDir, "pid-1001", "sess-a", fresh.SpaceOf("work", filepath.Dir(tasks)), time.Now(), false); err != nil {
		t.Fatal(err)
	}
	const n = 20
	var wg sync.WaitGroup
	got := make(chan string, 2*n)
	for i := 0; i < n; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			got <- mustCall(t, a, "task_journal", map[string]any{"id": "task-busy", "entry": fmt.Sprintf("e%02d", i)})
		}(i)
		go func(i int) {
			defer wg.Done()
			got <- mustCall(t, a, "task_update", map[string]any{"id": "task-quiet", "next": fmt.Sprintf("n%02d", i)})
		}(i)
	}
	wg.Wait()
	close(got)
	for out := range got {
		if strings.Contains(out, notice) {
			t.Fatalf("an own write was reported:\n%s", out)
		}
	}
}

// Two sessions writing the same task at once: each hears of the other's writes and
// never of its own, and no write goes unheard by the other.
func TestFreshBothWritingAtOnce(t *testing.T) {
	a, b, cfgDir, tasks := twoSessions(t)
	mustCall(t, a, "task_new", map[string]any{"title": "Shared"})
	sp := fresh.SpaceOf("work", filepath.Dir(tasks))
	for _, k := range []string{"pid-1001", "pid-1002"} {
		if err := fresh.Baseline(cfgDir, k, "s", sp, time.Now(), false); err != nil {
			t.Fatal(err)
		}
	}
	// both touch it first, so any change of it is news to the other
	mustCall(t, a, "task_get", map[string]any{"id": "task-shared"})
	mustCall(t, b, "task_get", map[string]any{"id": "task-shared"})
	const n = 10
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			mustCall(t, a, "task_journal", map[string]any{"id": "task-shared", "entry": fmt.Sprintf("a%02d", i)})
		}(i)
		go func(i int) {
			defer wg.Done()
			mustCall(t, b, "task_journal", map[string]any{"id": "task-shared", "entry": fmt.Sprintf("b%02d", i)})
		}(i)
	}
	wg.Wait()
	// After the storm, one last write by B: A must hear of it.
	mustCall(t, b, "task_journal", map[string]any{"id": "task-shared", "entry": "b-last"})
	if out := mustCall(t, a, "pending", nil); !strings.Contains(out, "task-shared") {
		t.Fatalf("A missed B's last write:\n%s", out)
	}
	if out := mustCall(t, a, "pending", nil); strings.Contains(out, notice) {
		t.Fatalf("A heard something twice or its own write:\n%s", out)
	}
}

// Without a key (not under Claude Code) the tools say nothing of it and write no file.
func TestFreshOff(t *testing.T) {
	cs := connect(t, space(t))
	mustCall(t, cs, "task_new", map[string]any{"title": "X"})
	if out := mustCall(t, cs, "pending", nil); strings.Contains(out, notice) {
		t.Fatal(out)
	}
	if _, err := os.Stat(filepath.Join(os.Getenv("HOME"), "cfg", fresh.Dir)); !os.IsNotExist(err) {
		t.Fatal("seen files were written with fresh off")
	}
}
