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
	"github.com/icalvete/augmented-brain/internal/state"
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
	open := space(t)
	cs := connect(t, open)
	b, err := open()
	if err != nil {
		t.Fatal(err)
	}
	shared := filepath.Join(b.ConfigDir, "shared")
	exists := func(rel string) bool { _, err := os.Stat(filepath.Join(shared, rel)); return err == nil }

	list, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, tl := range list.Tools {
		names = append(names, tl.Name)
	}
	sort.Strings(names)
	want := "area_new backup_setup capture knowledge_new pending person_save profile_save repo_describe review_done rule_new safety_save secret_name_save session_recover task_close task_get task_journal task_new task_park task_step task_update weekly_review where_am_i"
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
	// Shared notes go to the configuration folder, for every space; they have no area.
	if out, isErr := call(t, cs, "knowledge_new", map[string]any{"kind": "lesson", "title": "Small commits win", "shared": true}); isErr ||
		!exists("knowledge/lesson-small-commits-win.md") {
		t.Errorf("shared knowledge_new: %s", out)
	}
	if index, _ := os.ReadFile(filepath.Join(shared, "MEMORY.md")); !strings.Contains(string(index), "## Knowledge\n- [Small commits win]") || strings.Contains(string(index), "## Areas") {
		t.Errorf("index of the shared notes:\n%s", index)
	}
	if out, isErr := call(t, cs, "knowledge_new", map[string]any{"kind": "lesson", "title": "x", "area": "area-platform", "shared": true}); !isErr {
		t.Errorf("a shared note with an area was saved: %s", out)
	}
	if out, isErr := call(t, cs, "rule_new", map[string]any{"title": "Answer briefly", "body": "Short answers.", "shared": true}); isErr ||
		!exists("rules/rule-answer-briefly.md") {
		t.Errorf("shared rule_new: %s", out)
	}
	if out, isErr := call(t, cs, "rule_new", map[string]any{"title": "Never restart db01", "body": "Ask first."}); isErr || !strings.Contains(out, "rules/rule-never-restart-db01.md") {
		t.Errorf("rule_new: %s", out)
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
	// A part left out keeps what it has; an empty one is cleared.
	if out, isErr := call(t, cs, "profile_save", map[string]any{"assistant_name": "Argos", "assistant_personality": "- Calm"}); isErr {
		t.Fatal(out)
	}
	if out, isErr := call(t, cs, "profile_save", map[string]any{"work_style": ""}); isErr {
		t.Fatal(out)
	}
	got, err := brain.LoadProfile(cfgDir)
	want := brain.ProfileInput{Name: "Ana", About: "- Builds the shop's website", Assistant: "Argos", Personality: "- Calm"}
	if err != nil || got != want {
		t.Errorf("after partial saves: %+v, %v; want %+v", got, err, want)
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

// Safety settings go to the configuration, machine-wide; a list left out stays as it
// was, and a list a space kept of its own (older configurations) is replaced too.
func TestSafetySave(t *testing.T) {
	open := space(t)
	cfgDir := filepath.Join(os.Getenv("HOME"), "cfg")
	if err := config.SetSpaceList(cfgDir, "work", "untouchable_hosts", []string{"old01"}); err != nil {
		t.Fatal(err)
	}
	cs := connect(t, open)
	if out, isErr := call(t, cs, "safety_save", map[string]any{
		"production_aws_profiles": []string{"prod"}, "untouchable_hosts": []string{"db01"}}); isErr {
		t.Fatal(out)
	} else if !strings.Contains(out, "before: [old01]") {
		t.Errorf("the answer does not say what was there before: %s", out)
	}
	// Found in the review of 2026-11-03: the tool only adds, so an assistant talked into
	// it can never empty what makes the guard ask.
	if out, isErr := call(t, cs, "safety_save", map[string]any{"untouchable_hosts": []string{"db02"}, "production_aws_profiles": []string{}}); isErr {
		t.Fatal(out)
	}
	c, err := config.Load(cfgDir)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(c.ProductionAWSProfiles, ",") != "prod" || strings.Join(c.UntouchableHosts, ",") != "old01,db01,db02" {
		t.Errorf("machine-wide settings: %v %v", c.ProductionAWSProfiles, c.UntouchableHosts)
	}
	if w := c.Spaces["work"]; len(w.ProductionAWSProfiles)+len(w.UntouchableHosts) != 0 {
		t.Errorf("the space kept lists of its own: %+v", w)
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
	// Found in the review of 2026-11-03: the tool never moves a backup somewhere else.
	if out, isErr := call(t, cs, "backup_setup", map[string]any{"repo_url": remote + "-other"}); !isErr || !strings.Contains(out, "is for the user") {
		t.Errorf("moving the backup through the tool: %s", out)
	}
	if out, _ := call(t, cs, "backup_setup", map[string]any{}); !strings.Contains(out, "already backs up to "+remote) {
		t.Errorf("second call: %s", out)
	}
}

// A file that is not a valid note does not lock the tools; every answer says which file
// it is, so the user can fix it.
func TestToolsWithInvalidNote(t *testing.T) {
	open := space(t)
	b, err := open()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(b.Mem.Dir, "tasks", "task-x.md"), []byte("oops\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cs := connect(t, open)
	out, isErr := call(t, cs, "task_new", map[string]any{"title": "Still works"})
	if isErr || !strings.Contains(out, "task-still-works") || !strings.Contains(out, "tasks/task-x.md") {
		t.Errorf("task_new: %v %s", isErr, out)
	}
}

// The tools keep which task is active for the Stop hook, and session_recover returns
// what was said in a session that never ended, once.
func TestActiveTaskAndRecover(t *testing.T) {
	open := space(t)
	cfgDir := filepath.Join(os.Getenv("HOME"), "cfg")
	cs := connect(t, open)
	if out, isErr := call(t, cs, "task_new", map[string]any{"title": "Move the shop"}); isErr {
		t.Fatal(out)
	}
	if st := state.Read(cfgDir, "work"); st.Task != "task-move-the-shop" {
		t.Errorf("active task after task_new: %+v", st)
	}
	if out, isErr := call(t, cs, "task_park", map[string]any{"id": "task-move-the-shop", "next": "call the landlord"}); isErr {
		t.Fatal(out)
	}
	if st := state.Read(cfgDir, "work"); st.Task != "" {
		t.Errorf("a parked task is still active: %+v", st)
	}

	if out, _ := call(t, cs, "session_recover", map[string]any{}); !strings.Contains(out, "No earlier session") {
		t.Errorf("nothing to recover: %s", out)
	}
	transcript := filepath.Join(t.TempDir(), "s1.jsonl")
	line := `{"type":"user","timestamp":"` + time.Now().Add(time.Minute).UTC().Format(time.RFC3339) + `","message":{"content":"Use the blue van."}}`
	if err := os.WriteFile(transcript, []byte(line+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(transcript, time.Now().Add(-time.Hour), time.Now().Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if err := state.Update(cfgDir, "work", now, func(s *state.Space) {
		s.Sessions = append(s.Sessions, state.Session{ID: "s1", Transcript: transcript, Started: now.Add(-time.Hour),
			Task: "task-move-the-shop", Reported: true})
	}); err != nil {
		t.Fatal(err)
	}
	out, isErr := call(t, cs, "session_recover", map[string]any{})
	if isErr || !strings.Contains(out, "user: Use the blue van.") || !strings.Contains(out, "task task-move-the-shop") {
		t.Errorf("recover: %s", out)
	}
	if out, _ := call(t, cs, "session_recover", map[string]any{}); !strings.Contains(out, "No earlier session") {
		t.Errorf("recovered twice: %s", out)
	}
}

// What was already journaled in the dead session is not returned again.
func TestRecoverAfterTheLastWrite(t *testing.T) {
	open := space(t)
	cfgDir := filepath.Join(os.Getenv("HOME"), "cfg")
	now := time.Now()
	transcript := filepath.Join(t.TempDir(), "s1.jsonl")
	msg := func(at time.Time, text string) string {
		return `{"type":"user","timestamp":"` + at.UTC().Format(time.RFC3339) + `","message":{"content":"` + text + `"}}` + "\n"
	}
	if err := os.WriteFile(transcript, []byte(msg(now.Add(-50*time.Minute), "already journaled")+msg(now.Add(-30*time.Minute), "not yet")), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(transcript, now.Add(-20*time.Minute), now.Add(-20*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := state.Update(cfgDir, "work", now, func(s *state.Space) {
		s.Task, s.Since = "task-x", now.Add(-40*time.Minute)
		s.Sessions = []state.Session{{ID: "s1", Transcript: transcript, Started: now.Add(-time.Hour), Task: "task-x", Reported: true}}
	}); err != nil {
		t.Fatal(err)
	}
	cs := connect(t, open)
	out, _ := call(t, cs, "session_recover", map[string]any{})
	if !strings.Contains(out, "not yet") || strings.Contains(out, "already journaled") {
		t.Errorf("recover: %s", out)
	}
}

func TestSubtasksAndSteps(t *testing.T) {
	cs := connect(t, space(t))
	if out, isErr := call(t, cs, "task_new", map[string]any{"title": "New website"}); isErr {
		t.Fatal(out)
	}
	if out, isErr := call(t, cs, "task_new", map[string]any{"title": "Domain name", "parent": "task-new-website"}); isErr || !strings.Contains(out, "parent: task-new-website") {
		t.Fatalf("task_new with parent: %s", out)
	}
	if out, isErr := call(t, cs, "task_new", map[string]any{"title": "Deeper", "parent": "task-domain-name"}); !isErr || !strings.Contains(out, "one level") {
		t.Errorf("grandchild: %s", out)
	}
	if out, isErr := call(t, cs, "task_step", map[string]any{"id": "task-new-website", "add": "Pick a theme"}); isErr || !strings.Contains(out, "1. [ ] Pick a theme") {
		t.Errorf("task_step add: %s", out)
	}
	if out, isErr := call(t, cs, "task_step", map[string]any{"id": "task-new-website", "number": 1, "done": true}); isErr || !strings.Contains(out, "1. [x] Pick a theme") {
		t.Errorf("task_step tick: %s", out)
	}
	out, _ := call(t, cs, "pending", nil)
	if !strings.Contains(out, "- task-new-website (New website) · definition · 1/1 steps · waits on me") ||
		!strings.Contains(out, "\n  - task-domain-name (Domain name) · definition") {
		t.Errorf("pending: %s", out)
	}
	if out, _ := call(t, cs, "task_get", map[string]any{"id": "task-new-website"}); !strings.Contains(out, "Subtasks:\n- task-domain-name (Domain name) · definition") {
		t.Errorf("task_get: %s", out)
	}
	if out, _ := call(t, cs, "task_get", map[string]any{"id": "task-domain-name"}); !strings.Contains(out, "Part of: task-new-website (New website)") {
		t.Errorf("task_get child: %s", out)
	}
	if out, isErr := call(t, cs, "task_close", map[string]any{"id": "task-new-website", "result": "Up"}); !isErr || !strings.Contains(out, "task-domain-name") {
		t.Errorf("close with an open subtask: %s", out)
	}
}

// task_get says when a repository of the task is in none of the work folders, so the
// assistant asks the user instead of working on a folder that is gone.
func TestTaskGetMissingRepo(t *testing.T) {
	cs := connect(t, space(t))
	if _, isErr := call(t, cs, "task_new", map[string]any{"title": "Ship it"}); isErr {
		t.Fatal("task_new failed")
	}
	home := os.Getenv("HOME")
	if err := os.MkdirAll(filepath.Join(home, "work", "app"), 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(home, "brain", "tasks", "task-ship-it.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data = []byte(strings.Replace(string(data), "\ntype: task\n", "\ntype: task\nrepos: [app, gone]\n", 1))
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	out, isErr := call(t, cs, "task_get", map[string]any{"id": "task-ship-it"})
	if isErr || !strings.Contains(out, "Not on this computer: repos gone of this task") || strings.Contains(out, "repos app,") {
		t.Errorf("task_get:\n%s", out)
	}
}

// repo_describe writes what a repository is in the space's notes, by its name as a
// task's repos: writes it or by its path, and refuses one outside the work folders.
func TestRepoDescribe(t *testing.T) {
	open := space(t)
	cs := connect(t, open)
	b, err := open()
	if err != nil {
		t.Fatal(err)
	}
	repo := filepath.Join(b.Space.WorkFolders[0], "code", "site")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	if out, isErr := call(t, cs, "repo_describe", map[string]any{"repo": "code/site", "text": "The public website"}); isErr ||
		!strings.Contains(out, "~/work/code/site described in knowledge/ref-repositories.md") {
		t.Fatalf("repo_describe: %s", out)
	}
	if b, _ = open(); b.RepoDescriptionOf(repo) != "The public website" {
		t.Errorf("not kept: %q", b.RepoDescriptionOf(repo))
	}
	if out, isErr := call(t, cs, "repo_describe", map[string]any{"repo": "~/work/code/site", "text": ""}); isErr || !strings.Contains(out, "no description now") {
		t.Errorf("taking it out: %s", out)
	}
	if out, isErr := call(t, cs, "repo_describe", map[string]any{"repo": filepath.Dir(b.Space.WorkFolders[0]), "text": "home"}); !isErr {
		t.Errorf("a folder outside the work folders was described: %s", out)
	}
}

// With several spaces, person_save does not guess where a person is visible: it asks,
// listing the spaces of the configuration; with spaces it writes in those only.
func TestPersonSaveAsksForSpaces(t *testing.T) {
	open := space(t)
	b, err := open()
	if err != nil {
		t.Fatal(err)
	}
	home := os.Getenv("HOME")
	folder := filepath.Join(home, "other")
	if err := os.MkdirAll(folder, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := setup.AddSpace(b.ConfigDir, setup.Options{Space: "casa", WorkFolder: folder, Memory: "~/brain-casa"}); err != nil {
		t.Fatal(err)
	}
	cs := connect(t, open)
	out, isErr := call(t, cs, "person_save", map[string]any{"name": "Lucia Prado", "role": "friend"})
	if isErr || !strings.Contains(out, "Not saved") || !strings.Contains(out, "casa, work") {
		t.Fatalf("no question: %q %v", out, isErr)
	}
	for _, d := range []string{"brain", "brain-casa"} {
		if _, err := os.Stat(filepath.Join(home, d, "people", "person-lucia-prado.md")); err == nil {
			t.Errorf("saved in %s without being told where", d)
		}
	}
	if out, isErr := call(t, cs, "person_save", map[string]any{"name": "Lucia Prado", "role": "friend", "spaces": []string{"casa"}}); isErr {
		t.Fatal(out)
	}
	if _, err := os.Stat(filepath.Join(home, "brain-casa", "people", "person-lucia-prado.md")); err != nil {
		t.Error("not saved in casa")
	}
	if _, err := os.Stat(filepath.Join(home, "brain", "people", "person-lucia-prado.md")); err == nil {
		t.Error("saved in work, which was not asked for")
	}
	if out, isErr := call(t, cs, "person_save", map[string]any{"name": "Lucia Prado", "spaces": []string{"mars"}}); !isErr || !strings.Contains(out, "casa, work") {
		t.Errorf("unknown space: %q %v", out, isErr)
	}
}
