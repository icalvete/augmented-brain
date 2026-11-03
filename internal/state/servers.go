package state

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
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
	var out []Server
	seen := map[int]bool{}
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
	if ScanProc {
		procs, _ := os.ReadDir("/proc")
		for _, p := range procs {
			pid, err := strconv.Atoi(p.Name())
			if err != nil || seen[pid] || pid == os.Getpid() {
				continue
			}
			cmd, err := os.ReadFile("/proc/" + p.Name() + "/cmdline")
			if err != nil {
				continue
			}
			args := strings.Split(strings.TrimRight(string(cmd), "\x00"), "\x00")
			if len(args) < 2 || filepath.Base(args[0]) != "augmented-brain" || args[1] != "mcp" {
				continue
			}
			if exe, err := os.Readlink("/proc/" + p.Name() + "/exe"); err == nil && strings.HasSuffix(exe, " (deleted)") {
				out = append(out, Server{PID: pid})
			}
		}
	}
	return out
}
