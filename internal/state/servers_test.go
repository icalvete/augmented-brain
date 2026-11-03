package state

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
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
