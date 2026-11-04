//go:build linux

package hostmem

import (
	"os"
	"strconv"
	"strings"
)

func local() System { return procFS{root: "/proc"} }

// procFS reads processes from a /proc tree; tests point root at a fake one.
type procFS struct{ root string }

func (f procFS) dir(pid int) string { return f.root + "/" + strconv.Itoa(pid) }

// Proc reads /proc/<pid>/stat (parent, name, start) and /proc/<pid>/cmdline.
func (f procFS) Proc(pid int) (Proc, bool) {
	data, err := os.ReadFile(f.dir(pid) + "/stat")
	if err != nil {
		return Proc{}, false
	}
	// "pid (name) state ppid … starttime …": the name may hold spaces and parentheses,
	// so the fields are counted from the last ")".
	s := string(data)
	open, end := strings.IndexByte(s, '('), strings.LastIndexByte(s, ')')
	if open < 0 || end < open {
		return Proc{}, false
	}
	fields := strings.Fields(s[end+1:])
	if len(fields) < 20 {
		return Proc{}, false
	}
	ppid, err := strconv.Atoi(fields[1])
	if err != nil {
		return Proc{}, false
	}
	p := Proc{PID: pid, PPID: ppid, Name: s[open+1 : end], Start: fields[19]}
	if cmd, err := os.ReadFile(f.dir(pid) + "/cmdline"); err == nil && len(cmd) > 0 {
		p.Args = strings.Split(strings.TrimRight(string(cmd), "\x00"), "\x00")
	}
	return p, true
}

// RSS reads VmRSS from /proc/<pid>/status.
func (f procFS) RSS(pid int) (int64, bool) {
	data, err := os.ReadFile(f.dir(pid) + "/status")
	if err != nil {
		return 0, false
	}
	for line := range strings.SplitSeq(string(data), "\n") {
		if rest, ok := strings.CutPrefix(line, "VmRSS:"); ok {
			kb, err := strconv.ParseInt(strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(rest), "kB")), 10, 64)
			return kb << 10, err == nil
		}
	}
	return 0, false
}

func (f procFS) PIDs() []int {
	entries, _ := os.ReadDir(f.root)
	var out []int
	for _, e := range entries {
		if pid, err := strconv.Atoi(e.Name()); err == nil {
			out = append(out, pid)
		}
	}
	return out
}
