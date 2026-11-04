package hostmem

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// fakeProc writes a /proc tree: stat, cmdline and status of each process.
func fakeProc(t *testing.T, procs map[int]Proc, rssKB map[int]int) procFS {
	t.Helper()
	root := t.TempDir()
	for pid, p := range procs {
		dir := filepath.Join(root, strconv.Itoa(pid))
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		// Fields after the name: state ppid pgrp session tty tpgid flags minflt cminflt
		// majflt cmajflt utime stime cutime cstime priority nice threads itrealvalue
		// starttime vsize rss …
		stat := strconv.Itoa(pid) + " (" + p.Name + ") S " + strconv.Itoa(p.PPID) + " 1 1 0 -1 0 0 0 0 0 0 0 0 0 20 0 1 0 " + p.Start + " 1000 50 0\n"
		cmd := ""
		for _, a := range p.Args {
			cmd += a + "\x00"
		}
		status := "Name:\t" + p.Name + "\nVmPeak:\t 999 kB\n"
		if kb, ok := rssKB[pid]; ok {
			status += "VmRSS:\t  " + strconv.Itoa(kb) + " kB\n"
		}
		for name, text := range map[string]string{"stat": stat, "cmdline": cmd, "status": status} {
			if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o600); err != nil {
				t.Fatal(err)
			}
		}
	}
	return procFS{root: root}
}

func TestProcFS(t *testing.T) {
	fs := fakeProc(t, map[int]Proc{
		33: {PPID: 1, Name: "code", Args: []string{"/proc/self/exe", "--type=utility", "--utility-sub-type=node.mojom.NodeService"}, Start: "777"},
		34: {PPID: 33, Name: "claude", Args: []string{"/x/native-binary/claude", "--verbose"}, Start: "800"},
		35: {PPID: 34, Name: "a (b) c", Args: []string{"augmented-brain", "hook", "stop"}, Start: "900"},
	}, map[int]int{33: 9400 * 1024, 34: 300 * 1024})

	p, ok := fs.Proc(35)
	if !ok || p.PPID != 34 || p.Name != "a (b) c" || p.Start != "900" || len(p.Args) != 3 {
		t.Errorf("proc: %+v %v", p, ok)
	}
	h, ok := Find(fs, 35)
	if !ok || h.PID != 33 || h.Kind != VSCode || h.RSS != 9400*mb || h.Key() != "33:777" {
		t.Errorf("host: %+v %v", h, ok)
	}
	if _, ok := fs.RSS(35); ok {
		t.Error("a size without VmRSS")
	}
	if _, ok := fs.Proc(99); ok {
		t.Error("a process that is not there")
	}
	if got := fs.PIDs(); len(got) != 3 {
		t.Errorf("pids: %v", got)
	}
}

// The real /proc: this test's own process reads, with a size.
func TestProcFSReal(t *testing.T) {
	fs := procFS{root: "/proc"}
	p, ok := fs.Proc(os.Getpid())
	if !ok || p.PPID != os.Getppid() || p.Start == "" || len(p.Args) == 0 {
		t.Errorf("self: %+v %v", p, ok)
	}
	if n, ok := fs.RSS(os.Getpid()); !ok || n < mb {
		t.Errorf("own size: %d %v", n, ok)
	}
}
