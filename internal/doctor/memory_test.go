package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/hostmem"
	"github.com/icalvete/augmented-brain/internal/setup"
)

type procs struct {
	list map[int]hostmem.Proc
	mb   map[int]int64
}

func (p procs) Proc(pid int) (hostmem.Proc, bool) { x, ok := p.list[pid]; return x, ok }
func (p procs) RSS(pid int) (int64, bool)         { n, ok := p.mb[pid]; return n << 20, ok }
func (p procs) PIDs() []int                       { return []int{10, 11, 12, 20, 30} }

// A Claude window past 2 GB is a warning, once per host, whatever the number of its
// sessions; one of normal size says nothing.
func TestMemoryHogs(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	work := filepath.Join(home, "work")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(home, "cfg")
	if _, err := setup.Init(cfg, setup.Options{Space: "work", WorkFolder: work, Memory: "~/brain"}); err != nil {
		t.Fatal(err)
	}
	ext := []string{"code", "--type=utility", "--utility-sub-type=node.mojom.NodeService"}
	hostmem.Local = procs{
		list: map[int]hostmem.Proc{
			10: {PID: 10, PPID: 1, Name: "code", Args: ext, Start: "1"},
			11: {PID: 11, PPID: 10, Name: "claude", Start: "2"},
			12: {PID: 12, PPID: 10, Name: "claude", Start: "3"},
			20: {PID: 20, PPID: 1, Name: "code", Args: ext, Start: "4"},
			30: {PID: 30, PPID: 20, Name: "claude", Start: "5"},
		},
		mb: map[int]int64{10: 5000, 11: 300, 12: 300, 20: 500, 30: 300},
	}
	t.Cleanup(func() { hostmem.Local = nil })

	checks := Run(Env{ConfigDir: cfg, WorkDir: work, LookPath: allTools, Plugin: yes, Hook: denies})
	var hogs []Check
	for _, c := range checks {
		if strings.Contains(c.What, "GB of memory") {
			hogs = append(hogs, c)
		}
	}
	if len(hogs) != 1 || hogs[0].Status != Warning || !strings.Contains(hogs[0].What, "4.9 GB") ||
		!strings.Contains(hogs[0].What, "VS Code's extension host, process 10") || !strings.Contains(hogs[0].Fix, "park the task") {
		t.Errorf("memory checks: %+v", hogs)
	}
}
