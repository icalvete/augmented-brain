//go:build darwin

package hostmem

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

func local() System { return darwin{} }

// darwin asks the kernel for processes (sysctl, no program run) and ps for the size.
type darwin struct{}

func (darwin) Proc(pid int) (Proc, bool) {
	k, err := unix.SysctlKinfoProc("kern.proc.pid", pid)
	if err != nil || k == nil || int(k.Proc.P_pid) != pid {
		return Proc{}, false
	}
	name := k.Proc.P_comm[:]
	if i := bytes.IndexByte(name, 0); i >= 0 {
		name = name[:i]
	}
	t := k.Proc.P_starttime
	p := Proc{PID: pid, PPID: int(k.Eproc.Ppid), Name: string(name),
		Start: fmt.Sprintf("%d.%06d", t.Sec, t.Usec)}
	if raw, err := unix.SysctlRaw("kern.procargs2", pid); err == nil {
		p.Args = procArgs(raw)
	}
	return p, true
}

// procArgs reads kern.procargs2: the number of arguments (4 bytes), the program's path,
// NULs, then the arguments, each ending in a NUL.
func procArgs(raw []byte) []string {
	if len(raw) < 4 {
		return nil
	}
	argc := int(raw[0]) | int(raw[1])<<8 | int(raw[2])<<16 | int(raw[3])<<24
	rest := raw[4:]
	i := bytes.IndexByte(rest, 0)
	if i < 0 {
		return nil
	}
	rest = bytes.TrimLeft(rest[i:], "\x00")
	var args []string
	for len(args) < argc && len(rest) > 0 {
		j := bytes.IndexByte(rest, 0)
		if j < 0 {
			j = len(rest)
		}
		args = append(args, string(rest[:j]))
		rest = rest[min(j+1, len(rest)):]
	}
	return args
}

// RSS runs `ps -o rss= -p <pid>` (kilobytes), for half a second at most.
func (darwin) RSS(pid int) (int64, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	out, err := exec.CommandContext(ctx, "/bin/ps", "-o", "rss=", "-p", strconv.Itoa(pid)).Output()
	if err != nil {
		return 0, false
	}
	kb, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
	return kb << 10, err == nil
}

func (darwin) PIDs() []int {
	procs, err := unix.SysctlKinfoProcSlice("kern.proc.uid", os.Getuid())
	if err != nil {
		return nil
	}
	out := make([]int, 0, len(procs))
	for _, k := range procs {
		out = append(out, int(k.Proc.P_pid))
	}
	return out
}
