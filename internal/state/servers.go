package state

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/icalvete/augmented-brain/internal/paths"
)

// The MCP server of each open Claude session runs from the program as it was when the
// session started: after an update, those sessions keep the old one until restarted,
// and its tools may fail on what the new one writes. Each server notes its version
// here when it starts, so the new program can say which sessions to restart.

// ServersDir holds one file per running server, named by its process id.
const ServersDir = "servers"

// Server is an MCP server running on this machine.
type Server struct {
	PID     int
	Version string // "" when it did not say (a version before 0.6.1)
}

// RegisterServer notes that this process serves MCP with version. It returns a
// function that takes the note away, for when the server ends.
func RegisterServer(cfgDir, version string) func() {
	dir := filepath.Join(cfgDir, ServersDir)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return func() {}
	}
	f := filepath.Join(dir, strconv.Itoa(os.Getpid()))
	_ = os.WriteFile(f, []byte(version+"\n"), 0o600)
	return func() { _ = os.Remove(f) }
}

// ScanProc says whether to look through /proc for servers that kept no note. Tests
// turn it off, so the sessions open on the machine that runs them do not count.
var ScanProc = runtime.GOOS == "linux"

// OldServers returns the servers still running a version other than current. Notes
// of servers that ended are removed. On Linux it also finds the servers of versions
// that kept no note: an augmented-brain mcp process whose program was replaced on disk.
func OldServers(cfgDir, current string) []Server {
	out, seen := notedServers(cfgDir, current)
	if ScanProc {
		out = append(out, scanProc(seen)...)
	}
	return out
}

// notedServers are the old servers among those that kept a note, and every live
// server with a note (seen). Notes of servers that ended are removed.
func notedServers(cfgDir, current string) (out []Server, seen map[int]bool) {
	seen = map[int]bool{}
	dir := filepath.Join(cfgDir, ServersDir)
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		if pid == os.Getpid() {
			seen[pid] = true
			continue
		}
		if !alive(pid) {
			_ = os.Remove(filepath.Join(dir, e.Name()))
			continue
		}
		seen[pid] = true
		data, _ := os.ReadFile(filepath.Join(dir, e.Name()))
		if v := strings.TrimSpace(string(data)); v != current {
			out = append(out, Server{pid, v})
		}
	}
	return out, seen
}

// procDir is where the processes are read from; tests make one of their own.
var procDir = "/proc"

// scanProc finds, through /proc, the augmented-brain mcp processes not in seen whose
// program was replaced on disk: servers of versions that kept no note.
func scanProc(seen map[int]bool) []Server {
	var out []Server
	procs, _ := os.ReadDir(procDir)
	for _, p := range procs {
		pid, err := strconv.Atoi(p.Name())
		if err != nil || seen[pid] || pid == os.Getpid() {
			continue
		}
		if replacedServer(pid) {
			out = append(out, Server{PID: pid})
		}
	}
	return out
}

// replacedServer says whether process pid is an augmented-brain mcp whose program was
// replaced on disk.
func replacedServer(pid int) bool {
	p := filepath.Join(procDir, strconv.Itoa(pid))
	cmd, err := os.ReadFile(filepath.Join(p, "cmdline"))
	if err != nil {
		return false
	}
	args := strings.Split(strings.TrimRight(string(cmd), "\x00"), "\x00")
	if len(args) < 2 || filepath.Base(args[0]) != "augmented-brain" || args[1] != "mcp" {
		return false
	}
	exe, err := os.Readlink(filepath.Join(p, "exe"))
	return err == nil && strings.HasSuffix(exe, " (deleted)")
}

// ProcScanFile, in the state folder, remembers the last look through /proc at the
// session start: the program it was made with, the day, and the servers it found.
const ProcScanFile = "proc-scan"

// OldServersAtStart is OldServers for the session start, which runs often and must be
// quick. Reading every process of the machine (/proc) only finds servers of versions
// that kept no note, which can only appear when the program on disk is replaced: so it
// looks again only when the program changed since the last look, or on a new day, and
// otherwise checks that the servers found then still run.
func OldServersAtStart(cfgDir, current string, now time.Time) []Server {
	out, seen := notedServers(cfgDir, current)
	if !ScanProc {
		return out
	}
	stamp := programStamp()
	day := now.Format("2006-01-02")
	file := filepath.Join(cfgDir, Dir, ProcScanFile)
	if last, err := os.ReadFile(file); err == nil {
		lines := strings.Split(strings.TrimSpace(string(last)), "\n")
		if len(lines) >= 2 && stamp != "" && lines[0] == stamp && lines[1] == day {
			for _, l := range lines[2:] {
				if pid, err := strconv.Atoi(l); err == nil && !seen[pid] && pid != os.Getpid() && replacedServer(pid) {
					out = append(out, Server{PID: pid})
				}
			}
			return out
		}
	}
	found := scanProc(seen)
	text := stamp + "\n" + day + "\n"
	for _, s := range found {
		text += strconv.Itoa(s.PID) + "\n"
	}
	if err := os.MkdirAll(filepath.Dir(file), 0o700); err == nil {
		_ = paths.WriteAtomic(file, []byte(text), 0o600)
	}
	return slices.Concat(out, found)
}

// programStamp names the program on disk as it is now: its path, time and size. ""
// when it cannot be read, and then the session start always looks.
func programStamp() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	fi, err := os.Stat(exe)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%s %d %d", exe, fi.ModTime().UnixNano(), fi.Size())
}
