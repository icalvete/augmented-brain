package mcpserver

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/icalvete/augmented-brain/internal/brain"
	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/setup"
)

// connect starts the server with the given resolver and returns a connected client.
func connect(t *testing.T, open Resolver) *mcp.ClientSession {
	t.Helper()
	ctx := context.Background()
	st, ct := mcp.NewInMemoryTransports()
	if _, err := New(open, filepath.Join(t.TempDir(), "lock")).Connect(ctx, st, nil); err != nil {
		t.Fatal(err)
	}
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "test"}, nil).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cs.Close() })
	return cs
}

// call runs a tool and returns its text and whether it was an error.
func call(t *testing.T, cs *mcp.ClientSession, name string, args map[string]any) (string, bool) {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("%s: protocol error: %v", name, err)
	}
	var b strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			b.WriteString(tc.Text)
		}
	}
	return b.String(), res.IsError
}

func space(t *testing.T) Resolver {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	work := filepath.Join(home, "work")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	cfgDir := filepath.Join(home, "cfg")
	if _, err := setup.Init(cfgDir, setup.Options{Space: "work", WorkFolder: work, Memory: "~/brain"}); err != nil {
		t.Fatal(err)
	}
	return func() (*brain.Brain, error) {
		c, err := config.Load(cfgDir)
		if err != nil {
			return nil, err
		}
		b, err := brain.Open(c, c.Spaces["work"])
		if b != nil {
			b.Now = func() time.Time { return time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC) }
		}
		return b, err
	}
}

func TestTools(t *testing.T) {
	cs := connect(t, space(t))

	list, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, tl := range list.Tools {
		names = append(names, tl.Name)
	}
	sort.Strings(names)
	want := "area_new backup_setup capture knowledge_new pending person_save profile_save safety_save task_close task_get task_journal task_new task_park task_update where_am_i"
	if got := strings.Join(names, " "); got != want {
		t.Errorf("tools = %s", got)
	}

	if out, isErr := call(t, cs, "task_new", map[string]any{"title": "Backup and restore", "summary": "one per space"}); isErr ||
		!strings.Contains(out, "id: task-backup-and-restore") || !strings.Contains(out, "status: definition") {
		t.Fatalf("task_new: %s", out)
	}
	if out, isErr := call(t, cs, "task_update", map[string]any{"id": "task-backup-and-restore", "status": "analysis", "next": "Measure"}); isErr ||
		!strings.Contains(out, "status: analysis · waits on: me · next: Measure") {
		t.Errorf("task_update: %s", out)
	}
	if _, isErr := call(t, cs, "task_journal", map[string]any{"id": "task-backup-and-restore", "entry": "Measured: 2 MB."}); isErr {
		t.Error("task_journal failed")
	}
	if out, _ := call(t, cs, "pending", nil); !strings.Contains(out, "task-backup-and-restore (Backup and restore) · analysis · waits on me · next: Measure") {
		t.Errorf("pending: %s", out)
	}
	if out, isErr := call(t, cs, "task_park", map[string]any{"id": "task-backup-and-restore", "next": "Write the plan"}); isErr || !strings.Contains(out, "status: parked") {
		t.Errorf("task_park: %s", out)
	}
	if out, _ := call(t, cs, "task_get", map[string]any{"id": "task-backup-and-restore"}); !strings.Contains(out, "- 2026-10-15 — Measured: 2 MB.") {
		t.Errorf("task_get: %s", out)
	}
	if out, isErr := call(t, cs, "task_close", map[string]any{"id": "task-backup-and-restore", "result": "Nightly backups."}); isErr || !strings.Contains(out, "status: closed") {
		t.Errorf("task_close: %s", out)
	}
	if out, _ := call(t, cs, "pending", nil); !strings.Contains(out, "No live tasks.") {
		t.Errorf("pending after close: %s", out)
	}

	if _, isErr := call(t, cs, "area_new", map[string]any{"title": "Platform"}); isErr {
		t.Error("area_new failed")
	}
	if _, isErr := call(t, cs, "knowledge_new", map[string]any{"kind": "lesson", "title": "Filters need smudge", "area": "area-platform"}); isErr {
		t.Error("knowledge_new failed")
	}
	if _, isErr := call(t, cs, "capture", map[string]any{"text": "look at the runner"}); isErr {
		t.Error("capture failed")
	}
	if out, _ := call(t, cs, "where_am_i", nil); !strings.Contains(out, "space: work") || !strings.Contains(out, "area-platform: Platform") {
		t.Errorf("where_am_i: %s", out)
	}

	// Mistakes come back as tool errors the assistant can read, not as protocol errors.
	if out, isErr := call(t, cs, "task_get", map[string]any{"id": "task-nope"}); !isErr || !strings.Contains(out, `no task "task-nope"`) {
		t.Errorf("unknown task: %v %s", isErr, out)
	}
	if out, isErr := call(t, cs, "task_update", map[string]any{"id": "task-backup-and-restore", "status": "doing"}); !isErr || !strings.Contains(out, "status") {
		t.Errorf("invalid status: %v %s", isErr, out)
	}
}

func TestConcurrentCalls(t *testing.T) {
	cs := connect(t, space(t))
	if _, isErr := call(t, cs, "task_new", map[string]any{"title": "Busy"}); isErr {
		t.Fatal("task_new failed")
	}
	// Twenty journal entries at once on the same task: each call reads the task, adds a
	// line and saves it, so without one-at-a-time calls some entries would be lost.
	const n = 20
	var wg sync.WaitGroup
	errs := make(chan string, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "task_journal",
				Arguments: map[string]any{"id": "task-busy", "entry": fmt.Sprintf("entry %02d", i)}})
			if err != nil || res.IsError {
				errs <- fmt.Sprint(err, res)
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Error(e)
	}
	out, _ := call(t, cs, "task_get", map[string]any{"id": "task-busy"})
	if got := strings.Count(out, " — entry "); got != n {
		t.Errorf("the journal has %d of %d entries", got, n)
	}
}

func TestNoSpace(t *testing.T) {
	cs := connect(t, func() (*brain.Brain, error) {
		return nil, errors.New("this folder belongs to no space; run `augmented-brain space add`")
	})
	if out, isErr := call(t, cs, "pending", nil); !isErr || !strings.Contains(out, "space add") {
		t.Errorf("no space: %v %s", isErr, out)
	}
}

// The first session saves the profile through the tool; it keeps its first date and is
// no longer blank afterwards.
func TestProfileSave(t *testing.T) {
	open := space(t)
	cfgDir := filepath.Join(os.Getenv("HOME"), "cfg")
	if !brain.ProfileEmpty(cfgDir) {
		t.Fatal("a new profile must count as blank")
	}
	cs := connect(t, open)
	if out, isErr := call(t, cs, "profile_save", map[string]any{"name": ""}); !isErr {
		t.Errorf("a profile without a name was saved: %s", out)
	}
	out, isErr := call(t, cs, "profile_save", map[string]any{
		"name": "Ana", "about": "- Builds the shop's website", "work_style": "- Short answers"})
	if isErr {
		t.Fatal(out)
	}
	data, err := os.ReadFile(filepath.Join(cfgDir, "profile.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"- Name: Ana\n- Builds the shop's website\n", "## How I like to work\n- Short answers\n", "updated: \"2026-10-15\""} {
		if !strings.Contains(string(data), want) {
			t.Errorf("profile lacks %q:\n%s", want, data)
		}
	}
	if brain.ProfileEmpty(cfgDir) {
		t.Error("a saved profile still counts as blank")
	}
}

// People: a first name leads to the right person through the index; saving the same
// name again updates them.
func TestPersonSave(t *testing.T) {
	cs := connect(t, space(t))
	if out, isErr := call(t, cs, "person_save", map[string]any{
		"name": "Marcos García", "aliases": []string{"Marcos", "Marquitos"}, "role": "my manager"}); isErr {
		t.Fatal(out)
	}
	if out, isErr := call(t, cs, "person_save", map[string]any{
		"name": "Marcos García", "aliases": []string{"Marcos"}, "role": "my manager, product team"}); isErr {
		t.Fatal(out)
	}
	index, err := os.ReadFile(filepath.Join(os.Getenv("HOME"), "brain", "MEMORY.md"))
	if err != nil {
		t.Fatal(err)
	}
	want := "## People\n- [Marcos García](people/person-marcos-garcia.md) — my manager, product team · also: Marcos\n"
	if !strings.Contains(string(index), want) {
		t.Errorf("index lacks %q:\n%s", want, index)
	}
	if _, isErr := call(t, cs, "person_save", map[string]any{"name": " "}); !isErr {
		t.Error("a person without a name was saved")
	}
}

// Safety settings go to the configuration; a list left out stays as it was.
func TestSafetySave(t *testing.T) {
	open := space(t)
	cs := connect(t, open)
	if out, isErr := call(t, cs, "safety_save", map[string]any{
		"production_aws_profiles": []string{"prod"}, "untouchable_hosts": []string{"db01"}}); isErr {
		t.Fatal(out)
	}
	if out, isErr := call(t, cs, "safety_save", map[string]any{"untouchable_hosts": []string{"db01", "db02"}}); isErr {
		t.Fatal(out)
	}
	c, err := config.Load(filepath.Join(os.Getenv("HOME"), "cfg"))
	if err != nil {
		t.Fatal(err)
	}
	w := c.Spaces["work"]
	if strings.Join(w.ProductionAWSProfiles, ",") != "prod" || strings.Join(w.UntouchableHosts, ",") != "db01,db02" {
		t.Errorf("settings: %+v", w)
	}
}

// backup_setup with a repository given: connected, first copy pushed, and asking again
// says where it backs up.
func TestBackupSetup(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	cs := connect(t, space(t))
	remote := filepath.Join(t.TempDir(), "remote.git")
	if out, err := exec.Command("git", "init", "-q", "--bare", "-b", "main", remote).CombinedOutput(); err != nil {
		t.Fatal(err, string(out))
	}
	out, isErr := call(t, cs, "backup_setup", map[string]any{"repo_url": remote})
	if isErr || !strings.Contains(out, "Backup of space work set up") {
		t.Fatalf("setup: %s", out)
	}
	if log, err := exec.Command("git", "--git-dir", remote, "log", "--oneline", "main").CombinedOutput(); err != nil || len(log) == 0 {
		t.Errorf("nothing pushed: %v %s", err, log)
	}
	if out, _ := call(t, cs, "backup_setup", map[string]any{}); !strings.Contains(out, "already backs up to "+remote) {
		t.Errorf("second call: %s", out)
	}
}
