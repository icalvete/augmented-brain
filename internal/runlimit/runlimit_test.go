package runlimit

import (
	"context"
	"os/exec"
	"runtime"
	"testing"
	"time"
)

// A program that leaves a child holding its output used to keep the call waiting for
// the child, not for the limit (R-08 of the November 2026 review).
func TestCommandReturnsWithinItsLimit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("needs sh")
	}
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("no sh")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	start := time.Now()
	out, err := Command(ctx, "sh", "-c", "sleep 50 & echo started; wait").Output()
	took := time.Since(start)
	if err == nil {
		t.Fatalf("no error after the limit; output %q", out)
	}
	// Stopping the whole group ends the wait at once: WaitDelay is not even needed.
	if took > 4*time.Second {
		t.Fatalf("returned after %s; the limit is 1 s", took)
	}
}

func TestOutputWhenTheProgramEnds(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("needs sh")
	}
	out, err := Output(10*time.Second, "sh", "-c", "echo hi")
	if err != nil || string(out) != "hi\n" {
		t.Fatalf("Output = %q, %v", out, err)
	}
}
