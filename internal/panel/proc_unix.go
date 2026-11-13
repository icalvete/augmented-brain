//go:build unix

package panel

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
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

// procInfo is process pid's command line and when it started: from /proc on Linux,
// from ps elsewhere (macOS).
func procInfo(pid int) (args []string, started time.Time, err error) {
	if runtime.GOOS == "linux" {
		return linuxProc(pid)
	}
	p := strconv.Itoa(pid)
	cmd := exec.Command("ps", "-o", "lstart=", "-p", p)
	cmd.Env = append(os.Environ(), "LC_ALL=C", "LANG=C")
	out, err := cmd.Output()
	if err != nil {
		return nil, time.Time{}, err
	}
	started, err = time.ParseInLocation("Mon Jan _2 15:04:05 2006", strings.Join(strings.Fields(string(out)), " "), time.Local)
	if err != nil {
		// lstart pads the day with a space; Fields took it out.
		started, err = time.ParseInLocation("Mon Jan 2 15:04:05 2006", strings.Join(strings.Fields(string(out)), " "), time.Local)
	}
	if err != nil {
		return nil, time.Time{}, err
	}
	out, err = exec.Command("ps", "-o", "args=", "-p", p).Output()
	if err != nil {
		return nil, time.Time{}, err
	}
	return strings.Fields(string(out)), started, nil
}

// clockTicks is Linux's USER_HZ, in which /proc/<pid>/stat counts: 100 on every
// architecture Go runs on.
const clockTicks = 100

func linuxProc(pid int) ([]string, time.Time, error) {
	dir := "/proc/" + strconv.Itoa(pid)
	cmdline, err := os.ReadFile(dir + "/cmdline")
	if err != nil {
		return nil, time.Time{}, err
	}
	args := strings.Split(strings.TrimRight(string(cmdline), "\x00"), "\x00")
	stat, err := os.ReadFile(dir + "/stat")
	if err != nil {
		return nil, time.Time{}, err
	}
	// The fields after the command's name, which may hold spaces and parentheses: the
	// 22nd field of the line, starttime, is the 20th of these.
	i := strings.LastIndexByte(string(stat), ')')
	if i < 0 {
		return nil, time.Time{}, errors.New("bad /proc stat")
	}
	f := strings.Fields(string(stat[i+1:]))
	if len(f) < 20 {
		return nil, time.Time{}, errors.New("bad /proc stat")
	}
	ticks, err := strconv.ParseInt(f[19], 10, 64)
	if err != nil {
		return nil, time.Time{}, err
	}
	boot, err := bootTime()
	if err != nil {
		return nil, time.Time{}, err
	}
	return args, boot.Add(time.Duration(ticks) * time.Second / clockTicks), nil
}

// bootTime is when the machine started, from /proc/stat.
func bootTime() (time.Time, error) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return time.Time{}, err
	}
	for _, line := range strings.Split(string(data), "\n") {
		if v, ok := strings.CutPrefix(line, "btime "); ok {
			n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
			if err != nil {
				return time.Time{}, err
			}
			return time.Unix(n, 0), nil
		}
	}
	return time.Time{}, fmt.Errorf("no btime in /proc/stat")
}

// exactExe says whether exePath gives the whole path (and not only a name).
const exactExe = runtime.GOOS == "linux"

func terminate(pid int) error { return syscall.Kill(pid, syscall.SIGTERM) }
func kill(pid int) error      { return syscall.Kill(pid, syscall.SIGKILL) }
