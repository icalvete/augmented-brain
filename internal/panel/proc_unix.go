//go:build unix

package panel

import (
	"errors"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

// pidAlive reports whether process pid is running. On Linux a process that has ended
// but was not yet collected by its parent (a zombie) does not count.
func pidAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	if runtime.GOOS == "linux" {
		data, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
		if err != nil {
			return false
		}
		if i := strings.LastIndexByte(string(data), ')'); i >= 0 && i+2 < len(data) && data[i+2] == 'Z' {
			return false
		}
		return true
	}
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}

// exePath is the program process pid runs: exact on Linux (/proc), from ps elsewhere.
func exePath(pid int) (string, error) {
	if runtime.GOOS == "linux" {
		p, err := os.Readlink("/proc/" + strconv.Itoa(pid) + "/exe")
		// A program replaced on disk (an update) still runs from the old file.
		return strings.TrimSuffix(p, " (deleted)"), err
	}
	out, err := exec.Command("ps", "-o", "comm=", "-p", strconv.Itoa(pid)).Output()
	return strings.TrimSpace(string(out)), err
}

// exactExe says whether exePath gives the whole path (and not only a name).
const exactExe = runtime.GOOS == "linux"

func terminate(pid int) error { return syscall.Kill(pid, syscall.SIGTERM) }
func kill(pid int) error      { return syscall.Kill(pid, syscall.SIGKILL) }
