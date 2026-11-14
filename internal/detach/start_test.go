package detach

import (
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

// A process Start started is waited for, so it leaves no zombie behind in a program
// that lives long (R-17 of the November 2026 review).
func TestStartLeavesNoZombie(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("reads /proc")
	}
	cmd := exec.Command("true")
	Cmd(cmd)
	p, err := Start(cmd)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-p.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("never ended")
	}
	if p.Err() != nil {
		t.Errorf("true ended with %v", p.Err())
	}
	time.Sleep(300 * time.Millisecond)
	stat, err := os.ReadFile("/proc/" + strconv.Itoa(p.Pid) + "/stat")
	if err == nil && strings.Contains(string(stat), ") Z ") {
		t.Errorf("pid %d is a zombie: %s", p.Pid, stat)
	}
}
