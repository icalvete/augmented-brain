package state

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestOldServers(t *testing.T) {
	ScanProc = false
	defer func() { ScanProc = runtime.GOOS == "linux" }()
	dir := t.TempDir()
	done := RegisterServer(dir, "0.6.1")
	if old := OldServers(dir, "0.6.1"); len(old) != 0 {
		t.Errorf("this server is not old: %+v", old)
	}
	// Another live process (the test's parent) that said an older version, and one
	// that ended.
	sd := filepath.Join(dir, ServersDir)
	parent := os.Getppid()
	if err := os.WriteFile(filepath.Join(sd, strconv.Itoa(parent)), []byte("0.6.0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sd, "999999999"), []byte("0.5.0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	old := OldServers(dir, "0.6.1")
	found := false
	for _, s := range old {
		if s.PID == parent && s.Version == "0.6.0" {
			found = true
		}
		if s.PID == 999999999 {
			t.Error("a server that ended is listed")
		}
	}
	if !found {
		t.Errorf("the old server is missing: %+v", old)
	}
	if _, err := os.Stat(filepath.Join(sd, "999999999")); !os.IsNotExist(err) {
		t.Error("the note of a server that ended stays")
	}
	done()
	if _, err := os.Stat(filepath.Join(sd, strconv.Itoa(os.Getpid()))); !os.IsNotExist(err) {
		t.Error("the server's note outlived it")
	}
}

// fakeServer makes, in a /proc of the test's own, an augmented-brain mcp process whose
// program was replaced on disk.
func fakeServer(t *testing.T, proc string, pid int) {
	t.Helper()
	p := filepath.Join(proc, strconv.Itoa(pid))
	if err := os.MkdirAll(p, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(p, "cmdline"), []byte("/usr/bin/augmented-brain\x00mcp\x00"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/usr/bin/augmented-brain (deleted)", filepath.Join(p, "exe")); err != nil {
		t.Skip("no symbolic links here:", err)
	}
}

func pids(servers []Server) []int {
	var out []int
	for _, s := range servers {
		out = append(out, s.PID)
	}
	slices.Sort(out)
	return out
}

// The session start reads every process only when the program changed or on a new
// day; in between it checks again only the servers it found.
func TestOldServersAtStartLooksOnce(t *testing.T) {
	ScanProc = true
	defer func() { ScanProc = runtime.GOOS == "linux" }()
	proc := t.TempDir()
	procDir = proc
	defer func() { procDir = "/proc" }()
	dir := t.TempDir()
	day := time.Date(2026, 11, 14, 9, 0, 0, 0, time.UTC)

	fakeServer(t, proc, 4001)
	if got := pids(OldServersAtStart(dir, "1.0.0", day)); !slices.Equal(got, []int{4001}) {
		t.Fatalf("first look found %v, want [4001]", got)
	}
	fakeServer(t, proc, 4002) // appears after the look, the same day
	if got := pids(OldServersAtStart(dir, "1.0.0", day)); !slices.Equal(got, []int{4001}) {
		t.Errorf("same day, same program: found %v, want only the one found before", got)
	}
	if err := os.RemoveAll(filepath.Join(proc, "4001")); err != nil { // that server ended
		t.Fatal(err)
	}
	if got := pids(OldServersAtStart(dir, "1.0.0", day)); len(got) != 0 {
		t.Errorf("a server that ended is still listed: %v", got)
	}
	if got := pids(OldServersAtStart(dir, "1.0.0", day.AddDate(0, 0, 1))); !slices.Equal(got, []int{4002}) {
		t.Errorf("next day: found %v, want [4002]", got)
	}
	// A program replaced on disk looks again at once.
	file := filepath.Join(dir, Dir, ProcScanFile)
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	fakeServer(t, proc, 4003)
	if err := os.WriteFile(file, []byte("another program\n"+string(data[strings.Index(string(data), "\n")+1:])), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := pids(OldServersAtStart(dir, "1.0.0", day.AddDate(0, 0, 1))); !slices.Equal(got, []int{4002, 4003}) {
		t.Errorf("after the program changed: found %v, want [4002 4003]", got)
	}
}
