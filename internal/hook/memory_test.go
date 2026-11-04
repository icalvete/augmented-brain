package hook

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/hostmem"
	"github.com/icalvete/augmented-brain/internal/state"
)

// fakeProcs is a process tree: pid → process, and sizes in megabytes.
type fakeProcs struct {
	procs map[int]hostmem.Proc
	mb    map[int]int64
}

func (f *fakeProcs) Proc(pid int) (hostmem.Proc, bool) { p, ok := f.procs[pid]; return p, ok }
func (f *fakeProcs) RSS(pid int) (int64, bool)         { n, ok := f.mb[pid]; return n << 20, ok }
func (f *fakeProcs) PIDs() []int                       { return nil }

// vscodeTree: extension host 100 → claude 200 → the hook's shell 300; and a terminal
// claude 500 → shell 600.
func vscodeTree() *fakeProcs {
	return &fakeProcs{
		procs: map[int]hostmem.Proc{
			100: {PID: 100, PPID: 1, Name: "code", Args: []string{"/usr/share/code/code", "--type=utility", "--utility-sub-type=node.mojom.NodeService"}, Start: "10"},
			200: {PID: 200, PPID: 100, Name: "claude", Start: "20"},
			300: {PID: 300, PPID: 200, Name: "sh", Start: "30"},
			400: {PID: 400, PPID: 1, Name: "bash", Start: "40"},
			500: {PID: 500, PPID: 400, Name: "claude", Start: "50"},
			600: {PID: 600, PPID: 500, Name: "sh", Start: "60"},
		},
		mb: map[int]int64{100: 500, 200: 300, 500: 400},
	}
}

func useProcs(t *testing.T, f *fakeProcs, parent int) *int {
	t.Helper()
	oldP, oldPID := Processes, ParentPID
	pid := parent
	Processes, ParentPID = f, func() int { return pid }
	t.Cleanup(func() { Processes, ParentPID = oldP, oldPID })
	return &pid
}

func stopText(t *testing.T, cfgDir, work string) string {
	t.Helper()
	o := run(t, func(in *bytes.Reader, out *bytes.Buffer) error { return Stop(in, out, cfgDir) },
		map[string]any{"cwd": work, "hook_event_name": "Stop"})
	if o == nil {
		return ""
	}
	return o["additionalContext"].(string)
}

// The host of a VS Code session is its extension host: warned at 2, 4 and 8 GB, once
// each; a pid reused by a new host starts clean.
func TestStopMemoryWarning(t *testing.T) {
	cfgDir, work := setupSpace(t)
	f := vscodeTree()
	useProcs(t, f, 300)

	if got := stopText(t, cfgDir, work); got != "" {
		t.Errorf("a host of normal size: %s", got)
	}
	f.mb[100] = 2100
	got := stopText(t, cfgDir, work)
	if !strings.Contains(got, "extension host") || !strings.Contains(got, "2.1 GB") || !strings.Contains(got, "task_park") || !strings.Contains(got, "new VS Code window") {
		t.Errorf("at 2 GB: %s", got)
	}
	f.mb[100] = 3900
	if got := stopText(t, cfgDir, work); got != "" {
		t.Errorf("warned twice for one step: %s", got)
	}
	f.mb[100] = 9000
	if got := stopText(t, cfgDir, work); !strings.Contains(got, "8.8 GB") {
		t.Errorf("past 4 and 8 GB at once: %s", got)
	}
	f.mb[100] = 12000
	if got := stopText(t, cfgDir, work); got != "" {
		t.Errorf("past the last step: %s", got)
	}

	// The window closed; another one got pid 100.
	f.procs[100] = hostmem.Proc{PID: 100, PPID: 1, Name: "code", Args: f.procs[100].Args, Start: "99"}
	f.mb[100] = 2500
	if got := stopText(t, cfgDir, work); !strings.Contains(got, "2.4 GB") {
		t.Errorf("a reused pid inherited the old host's marks: %q", got)
	}
	// The old host's mark was dropped when the new one was written.
	if state.MemoryWarned(cfgDir, hostmem.Key(100, "10")) != 0 || state.MemoryWarned(cfgDir, hostmem.Key(100, "99")) != 2048 {
		t.Error("marks not kept per host")
	}

	// While a Stop hook already continues, nothing.
	f.mb[100] = 4500
	o := run(t, func(in *bytes.Reader, out *bytes.Buffer) error { return Stop(in, out, cfgDir) },
		map[string]any{"cwd": work, "hook_event_name": "Stop", "stop_hook_active": true})
	if o != nil {
		t.Errorf("while continuing: %v", o)
	}
}

// In a terminal the host is claude itself; outside any space the warning comes all the
// same; and in Spanish when the configuration is.
func TestStopMemoryTerminal(t *testing.T) {
	cfgDir, _ := setupSpace(t)
	f := vscodeTree()
	useProcs(t, f, 600)
	f.mb[100] = 9000 // another window's host: not this session's
	f.mb[500] = 2200
	c, _ := config.Load(cfgDir)
	c.Language = "es"
	if err := c.Save(); err != nil {
		t.Fatal(err)
	}
	got := stopText(t, cfgDir, t.TempDir())
	if !strings.Contains(got, "el proceso de Claude de esta sesión usa 2,1 GB") || !strings.Contains(got, "sesión nueva") {
		t.Errorf("terminal, in Spanish: %s", got)
	}
	if got := stopText(t, cfgDir, t.TempDir()); got != "" {
		t.Errorf("twice: %s", got)
	}
}

// No claude above the hook (run by hand), or no processes to read: nothing.
func TestStopMemoryNoClaude(t *testing.T) {
	cfgDir, work := setupSpace(t)
	f := vscodeTree()
	f.mb[100] = 9000
	useProcs(t, f, 400)
	if got := stopText(t, cfgDir, work); got != "" {
		t.Errorf("no claude: %s", got)
	}
	Processes = nil
	if got := stopText(t, cfgDir, work); got != "" {
		t.Errorf("no processes: %s", got)
	}
}

// The journal reminder and the warning together: one text, and the assistant is not
// asked to end its turn in silence.
func TestStopMemoryWithJournal(t *testing.T) {
	cfgDir, work := setupSpace(t)
	start := time.Date(2026, 10, 20, 9, 0, 0, 0, time.Local)
	now := clock(t, start)
	id := openTask(t, cfgDir)
	if err := state.Touched(cfgDir, "work", id, start, true); err != nil {
		t.Fatal(err)
	}
	*now = start.Add(30 * time.Minute)
	f := vscodeTree()
	f.mb[100] = 2100
	useProcs(t, f, 300)
	got := stopText(t, cfgDir, work)
	if !strings.Contains(got, "task_journal") || !strings.Contains(got, "2.1 GB") || strings.Contains(got, "no text at all") ||
		strings.Index(got, "task_journal") > strings.Index(got, "2.1 GB") {
		t.Errorf("both: %s", got)
	}
}

// AB_MEMORY_WARN_MB, a test aid, lowers the steps.
func TestStopMemoryOverride(t *testing.T) {
	cfgDir, work := setupSpace(t)
	f := vscodeTree()
	useProcs(t, f, 300)
	old := Getenv
	Getenv = func(k string) string {
		if k == "AB_MEMORY_WARN_MB" {
			return "400"
		}
		return ""
	}
	t.Cleanup(func() { Getenv = old })
	if got := stopText(t, cfgDir, work); !strings.Contains(got, "0.5 GB of memory") {
		t.Errorf("override: %s", got)
	}
}

// What the check adds to every Stop on this machine: the walk up to claude, the host's
// size and the marks file (no warning due).
func BenchmarkMemoryNote(b *testing.B) {
	if hostmem.Local == nil {
		b.Skip("memory is not watched here")
	}
	old := Processes
	Processes = hostmem.Local
	defer func() { Processes = old }()
	dir := b.TempDir()
	for b.Loop() {
		_ = memoryNote("en", dir)
	}
}
