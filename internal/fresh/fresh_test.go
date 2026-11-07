package fresh

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// task writes a task note with the given header fields and journal text.
func task(t *testing.T, dir, id, status, next, journal string) {
	t.Helper()
	body := fmt.Sprintf("---\nid: %s\ntype: task\ntitle: Title of %s\nsummary: s\ncreated: \"2026-11-07\"\nupdated: \"2026-11-07\"\nstatus: %s\nwaiting_on: me\nnext: %s\n---\n## Diario\n%s\n", id, id, status, next, journal)
	if err := os.WriteFile(filepath.Join(dir, id+".md"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// age moves a file's time back, as if written long before the check.
func age(t *testing.T, dir, id string, d time.Duration) {
	t.Helper()
	old := time.Now().Add(-d)
	if err := os.Chtimes(filepath.Join(dir, id+".md"), old, old); err != nil {
		t.Fatal(err)
	}
}

type env struct {
	cfg string
	sp  Space
	key string
	now time.Time
}

func newEnv(t *testing.T) *env {
	t.Helper()
	root := t.TempDir()
	tasks := filepath.Join(root, "brain", "tasks")
	if err := os.MkdirAll(tasks, 0o700); err != nil {
		t.Fatal(err)
	}
	return &env{cfg: filepath.Join(root, "cfg"), sp: Space{Name: "work", Tasks: tasks, Format: 2}, key: "pid-4242", now: time.Now()}
}

func (e *env) check(t *testing.T) Report {
	t.Helper()
	e.now = e.now.Add(time.Second)
	r, err := Check(e.cfg, e.key, e.sp, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func (e *env) baseline(t *testing.T, keep bool) {
	t.Helper()
	if err := Baseline(e.cfg, e.key, "sess-1", e.sp, time.Now(), keep); err != nil {
		t.Fatal(err)
	}
}

func ids(r Report) string {
	var out []string
	for _, c := range r.Changes {
		out = append(out, c.Kind+":"+c.ID)
	}
	return strings.Join(out, ",")
}

func TestKeys(t *testing.T) {
	for pid, want := range map[string]string{"123": "pid-123", " 77 ": "pid-77", "": "", "abc": "", "../x": "", "1": "", "0": "", "-5": ""} {
		if got := Key(pid); got != want {
			t.Errorf("Key(%q) = %q, want %q", pid, got, want)
		}
	}
	env := map[string]string{"CLAUDE_PID": "900"}
	if got := HookKey(func(k string) string { return env[k] }); got != "pid-900" {
		t.Errorf("HookKey = %q", got)
	}
	if got := ServerKey(func(string) string { return "" }, 900); got != "" {
		t.Errorf("ServerKey outside Claude Code = %q, want nothing", got)
	}
	if got := ServerKey(func(k string) string { return map[string]string{"CLAUDECODE": "1"}[k] }, 900); got != "pid-900" {
		t.Errorf("ServerKey = %q", got)
	}
}

func TestNoKeyDoesNothing(t *testing.T) {
	e := newEnv(t)
	task(t, e.sp.Tasks, "task-a", "plan", "x", "")
	if err := Baseline(e.cfg, "", "s", e.sp, time.Now(), false); err != nil {
		t.Fatal(err)
	}
	if r, err := Check(e.cfg, "", e.sp, time.Now()); err != nil || !r.Empty() {
		t.Fatalf("Check without a key = %v, %v", r, err)
	}
	if _, err := os.Stat(filepath.Join(e.cfg, Dir)); !os.IsNotExist(err) {
		t.Fatal("a seen folder was made without a key")
	}
}

func TestNothingChanged(t *testing.T) {
	e := newEnv(t)
	task(t, e.sp.Tasks, "task-a", "plan", "x", "")
	age(t, e.sp.Tasks, "task-a", time.Hour)
	e.baseline(t, false)
	if r := e.check(t); !r.Empty() {
		t.Fatalf("nothing changed, got %s", ids(r))
	}
}

// The case of 7-nov: another session closed a task this one had only seen in the index.
func TestStatusChangedOutside(t *testing.T) {
	e := newEnv(t)
	task(t, e.sp.Tasks, "task-logo", "validation", "check it", "")
	e.baseline(t, false)
	task(t, e.sp.Tasks, "task-logo", "closed", "", "- closed")
	r := e.check(t)
	if ids(r) != "changed:task-logo" || r.Changes[0].Status != "closed" {
		t.Fatalf("got %+v", r)
	}
	if again := e.check(t); !again.Empty() {
		t.Fatalf("reported twice: %s", ids(again))
	}
}

// A journal entry of another session, in a task this one never touched, is not news.
func TestForeignJournalOnUntouchedTaskIsQuiet(t *testing.T) {
	e := newEnv(t)
	task(t, e.sp.Tasks, "task-b", "execution", "go on", "")
	e.baseline(t, false)
	task(t, e.sp.Tasks, "task-b", "execution", "go on", "- 2026-11-07 — another session wrote this")
	if r := e.check(t); !r.Empty() {
		t.Fatalf("journal-only change reported: %s", ids(r))
	}
}

// Any change of a task this session read or wrote is news.
func TestForeignJournalOnTouchedTaskIsReported(t *testing.T) {
	e := newEnv(t)
	task(t, e.sp.Tasks, "task-b", "execution", "go on", "")
	e.baseline(t, false)
	if err := Absorb(e.cfg, e.key, e.sp, time.Now(), []string{"task-b"}); err != nil {
		t.Fatal(err)
	}
	task(t, e.sp.Tasks, "task-b", "execution", "go on", "- 2026-11-07 — another session wrote this")
	if r := e.check(t); ids(r) != "changed:task-b" {
		t.Fatalf("got %s", ids(r))
	}
}

// What the session's own tool wrote is never reported back to it, and the task it
// wrote counts as touched from then on.
func TestOwnWritesAreNotReported(t *testing.T) {
	e := newEnv(t)
	task(t, e.sp.Tasks, "task-c", "plan", "a", "")
	e.baseline(t, false)
	// the MCP server: Check before the tool, the tool writes, Absorb after it
	if r := e.check(t); !r.Empty() {
		t.Fatal(ids(r))
	}
	task(t, e.sp.Tasks, "task-c", "execution", "b", "- my own entry")
	task(t, e.sp.Tasks, "task-new", "definition", "start", "")
	if err := Absorb(e.cfg, e.key, e.sp, time.Now(), nil); err != nil {
		t.Fatal(err)
	}
	if r := e.check(t); !r.Empty() {
		t.Fatalf("own writes reported: %s", ids(r))
	}
	// and now another session writes in the task this one created
	task(t, e.sp.Tasks, "task-new", "definition", "start", "- someone else")
	if r := e.check(t); ids(r) != "changed:task-new" {
		t.Fatalf("got %s", ids(r))
	}
}

func TestNewAndGoneTasks(t *testing.T) {
	e := newEnv(t)
	task(t, e.sp.Tasks, "task-old", "plan", "x", "")
	e.baseline(t, false)
	if err := os.Remove(filepath.Join(e.sp.Tasks, "task-old.md")); err != nil {
		t.Fatal(err)
	}
	task(t, e.sp.Tasks, "task-fresh", "definition", "y", "")
	r := e.check(t)
	if ids(r) != "new:task-fresh,gone:task-old" {
		t.Fatalf("got %s", ids(r))
	}
	if r.Changes[0].Title != "Title of task-fresh" || r.Changes[0].Status != "definition" {
		t.Fatalf("new task without its header: %+v", r.Changes[0])
	}
}

// Files that are not tasks are left alone.
func TestOnlyTaskFiles(t *testing.T) {
	e := newEnv(t)
	e.baseline(t, false)
	for _, name := range []string{".task-x.md.tmp", "notes.txt", ".hidden.md"} {
		if err := os.WriteFile(filepath.Join(e.sp.Tasks, name), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(e.sp.Tasks, "sub.md"), 0o700); err != nil {
		t.Fatal(err)
	}
	if r := e.check(t); !r.Empty() {
		t.Fatalf("got %s", ids(r))
	}
}

// A file system that keeps whole seconds can leave time and size alike after a change;
// a file written within Racy of the record is hashed again.
func TestSameTimeAndSizeStillSeen(t *testing.T) {
	e := newEnv(t)
	task(t, e.sp.Tasks, "task-r", "plan", "aaaa", "")
	fi, _ := os.Stat(filepath.Join(e.sp.Tasks, "task-r.md"))
	e.baseline(t, false)
	task(t, e.sp.Tasks, "task-r", "plan", "bbbb", "") // same size
	if err := os.Chtimes(filepath.Join(e.sp.Tasks, "task-r.md"), fi.ModTime(), fi.ModTime()); err != nil {
		t.Fatal(err)
	}
	if r := e.check(t); ids(r) != "changed:task-r" {
		t.Fatalf("a change with the same time and size was missed: %s", ids(r))
	}
}

// A file long untouched, whose time and size match, is not read again.
func TestOldFilesAreNotRead(t *testing.T) {
	e := newEnv(t)
	task(t, e.sp.Tasks, "task-o", "plan", "x", "")
	age(t, e.sp.Tasks, "task-o", time.Hour)
	e.baseline(t, false)
	s, _ := Load(e.cfg, e.key)
	v := s.Tasks["task-o"]
	v.Hash = "recorded"
	s.Tasks["task-o"] = v
	if err := save(e.cfg, e.key, s); err != nil {
		t.Fatal(err)
	}
	if r := e.check(t); !r.Empty() {
		t.Fatalf("got %s", ids(r))
	}
	if s, _ := Load(e.cfg, e.key); s.Tasks["task-o"].Hash != "recorded" {
		t.Fatal("an old file was read again")
	}
}

// Moving to another space takes a new baseline, silently: its tasks are not "new".
func TestSpaceChangeIsSilent(t *testing.T) {
	e := newEnv(t)
	task(t, e.sp.Tasks, "task-a", "plan", "x", "")
	e.baseline(t, false)
	other := e.sp
	other.Name, other.Tasks = "personal", t.TempDir()
	task(t, other.Tasks, "task-p", "plan", "x", "")
	r, err := Check(e.cfg, e.key, other, time.Now())
	if err != nil || !r.Empty() {
		t.Fatalf("got %s, %v", ids(r), err)
	}
	task(t, other.Tasks, "task-p", "closed", "", "")
	if r, _ := Check(e.cfg, e.key, other, time.Now()); ids(r) != "changed:task-p" {
		t.Fatalf("after the move: %s", ids(r))
	}
}

// After a migration every task may have been rewritten: one line, not a list.
func TestFormatChange(t *testing.T) {
	e := newEnv(t)
	task(t, e.sp.Tasks, "task-a", "plan", "x", "")
	e.baseline(t, false)
	e.sp.Format = 3
	task(t, e.sp.Tasks, "task-a", "closed", "", "")
	r := e.check(t)
	if !r.Format || len(r.Changes) != 0 {
		t.Fatalf("got %+v", r)
	}
	if !strings.Contains(Text("en", r), "new format") {
		t.Fatal(Text("en", r))
	}
	if r := e.check(t); !r.Empty() {
		t.Fatal("format change reported twice")
	}
}

// With no seen file (none taken, or removed), the first check takes one and says nothing.
func TestNoBaselineIsSilent(t *testing.T) {
	e := newEnv(t)
	task(t, e.sp.Tasks, "task-a", "plan", "x", "")
	if r := e.check(t); !r.Empty() {
		t.Fatal(ids(r))
	}
	task(t, e.sp.Tasks, "task-a", "closed", "", "")
	if r := e.check(t); ids(r) != "changed:task-a" {
		t.Fatal(ids(r))
	}
}

// Absorb without a baseline writes nothing: the next Check takes it.
func TestAbsorbWithoutBaseline(t *testing.T) {
	e := newEnv(t)
	task(t, e.sp.Tasks, "task-a", "plan", "x", "")
	if err := Absorb(e.cfg, e.key, e.sp, time.Now(), []string{"task-a"}); err != nil {
		t.Fatal(err)
	}
	if _, ok := Load(e.cfg, e.key); ok {
		t.Fatal("Absorb took a baseline")
	}
}

// resume and compact keep the touched marks; startup, clear and fork do not.
func TestBaselineKeepsTouched(t *testing.T) {
	e := newEnv(t)
	task(t, e.sp.Tasks, "task-a", "plan", "x", "")
	e.baseline(t, false)
	_ = Absorb(e.cfg, e.key, e.sp, time.Now(), []string{"task-a"})
	e.baseline(t, true)
	if s, _ := Load(e.cfg, e.key); !s.Tasks["task-a"].Touched {
		t.Fatal("resume lost the touched mark")
	}
	e.baseline(t, false)
	if s, _ := Load(e.cfg, e.key); s.Tasks["task-a"].Touched {
		t.Fatal("a fresh start kept the touched mark")
	}
}

func TestHeaderFieldsThatCount(t *testing.T) {
	e := newEnv(t)
	write := func(extra string) {
		body := "---\nid: task-h\ntype: task\ntitle: T\nsummary: s\ncreated: \"2026-11-07\"\nupdated: \"2026-11-07\"\nstatus: plan\nnext: n\n" + extra + "---\nbody\n"
		if err := os.WriteFile(filepath.Join(e.sp.Tasks, "task-h.md"), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("")
	e.baseline(t, false)
	write("updated2: x\n") // not a header field the index shows... and not valid: still the same index
	if r := e.check(t); !r.Empty() {
		t.Fatalf("an unknown field counted: %s", ids(r))
	}
	write("area: area-x\n")
	if r := e.check(t); ids(r) != "changed:task-h" {
		t.Fatalf("area change missed: %s", ids(r))
	}
	write("area: area-x\nparent: task-p\n")
	if r := e.check(t); ids(r) != "changed:task-h" {
		t.Fatalf("parent change missed: %s", ids(r))
	}
}

func TestText(t *testing.T) {
	if Text("en", Report{}) != "" {
		t.Fatal("empty report gave text")
	}
	var r Report
	for i := 0; i < MaxLines+3; i++ {
		r.Changes = append(r.Changes, Change{ID: fmt.Sprintf("task-%02d", i), Kind: "changed", Title: "T", Status: "closed", WaitingOn: "me", Next: "line one\nline two"})
	}
	r.Changes = append(r.Changes[:1], append([]Change{{ID: "task-gone", Kind: "gone"}}, r.Changes[1:]...)...)
	en, es := Text("en", r), Text("es", r)
	if !strings.Contains(en, "task_get") || !strings.Contains(en, "and 4 more") || strings.Contains(en, "line one\nline two") {
		t.Fatalf("en:\n%s", en)
	}
	if !strings.Contains(es, "y 4 más") || !strings.Contains(es, "ya no existe") {
		t.Fatalf("es:\n%s", es)
	}
	long := Change{ID: "task-l", Kind: "new", Title: strings.Repeat("x", 500), Status: "plan"}
	if got := Text("en", Report{Changes: []Change{long}}); len(got) > 1500 {
		t.Fatalf("a long title was not cut: %d bytes", len(got))
	}
}

func TestRemoveAndPrune(t *testing.T) {
	e := newEnv(t)
	e.baseline(t, false)
	Remove(e.cfg, e.key)
	if _, ok := Load(e.cfg, e.key); ok {
		t.Fatal("Remove left the file")
	}
	Remove(e.cfg, "") // nothing to do, no panic
	e.baseline(t, false)
	old := time.Now().Add(-40 * 24 * time.Hour)
	_ = os.Chtimes(file(e.cfg, e.key), old, old)
	other := "pid-99"
	_ = Baseline(e.cfg, other, "s", e.sp, time.Now(), false)
	Prune(e.cfg, time.Now())
	if _, ok := Load(e.cfg, e.key); ok {
		t.Fatal("Prune kept a 40-day-old file")
	}
	if _, ok := Load(e.cfg, other); !ok {
		t.Fatal("Prune removed a fresh file")
	}
}

// A broken seen file is as good as none.
func TestBrokenSeenFile(t *testing.T) {
	e := newEnv(t)
	task(t, e.sp.Tasks, "task-a", "plan", "x", "")
	if err := os.MkdirAll(filepath.Dir(file(e.cfg, e.key)), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file(e.cfg, e.key), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if r := e.check(t); !r.Empty() {
		t.Fatal(ids(r))
	}
	if _, ok := Load(e.cfg, e.key); !ok {
		t.Fatal("a broken file was not replaced")
	}
}

// A hundred tasks: well within a hook's budget.
func TestSpeed(t *testing.T) {
	e := newEnv(t)
	for i := 0; i < 100; i++ {
		id := fmt.Sprintf("task-%03d", i)
		task(t, e.sp.Tasks, id, "plan", "x", strings.Repeat("- entry\n", 200))
		age(t, e.sp.Tasks, id, time.Hour)
	}
	e.baseline(t, false)
	start := time.Now()
	for i := 0; i < 10; i++ {
		e.check(t)
	}
	if per := time.Since(start) / 10; per > 50*time.Millisecond {
		t.Fatalf("a check of 100 tasks took %v", per)
	}
}
