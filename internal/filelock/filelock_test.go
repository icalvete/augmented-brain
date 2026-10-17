//go:build unix || windows

package filelock

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestHelperProcess is not a real test: the test below starts it as a second process
// that takes the lock, appends to a file and waits a little before releasing it.
func TestHelperProcess(t *testing.T) {
	lockPath, out := os.Getenv("FILELOCK_PATH"), os.Getenv("FILELOCK_OUT")
	if lockPath == "" {
		return
	}
	release, err := Lock(lockPath)
	if err != nil {
		os.Exit(2)
	}
	f, _ := os.OpenFile(out, os.O_APPEND|os.O_WRONLY, 0o600)
	f.WriteString("start " + os.Getenv("FILELOCK_ID") + "\n")
	time.Sleep(50 * time.Millisecond)
	f.WriteString("end " + os.Getenv("FILELOCK_ID") + "\n")
	f.Close()
	release()
	os.Exit(0)
}

// Two processes holding the lock never overlap: every "start" is followed by its "end".
func TestAcrossProcesses(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out")
	if err := os.WriteFile(out, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	var procs []*exec.Cmd
	for i := 0; i < 4; i++ {
		cmd := exec.Command(os.Args[0], "-test.run=TestHelperProcess")
		cmd.Env = append(os.Environ(), "FILELOCK_PATH="+filepath.Join(dir, "lock"),
			"FILELOCK_OUT="+out, "FILELOCK_ID="+strconv.Itoa(i))
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		procs = append(procs, cmd)
	}
	for _, p := range procs {
		if err := p.Wait(); err != nil {
			t.Fatal(err)
		}
	}
	data, _ := os.ReadFile(out)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 8 {
		t.Fatalf("lines:\n%s", data)
	}
	for i := 0; i < len(lines); i += 2 {
		id := strings.TrimPrefix(lines[i], "start ")
		if lines[i+1] != "end "+id {
			t.Fatalf("two processes overlapped:\n%s", data)
		}
	}
}
