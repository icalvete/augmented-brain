//go:build windows

package main

import (
	"os"
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

// removeSelf deletes the running program. Windows does not let a running .exe be
// deleted, so it leaves a small cmd.exe behind, with no window and on its own, that
// waits two seconds for this program to end and then deletes it, and the .old copy
// install.ps1 keeps when it updates.
func removeSelf(self string) error {
	// cmd.exe reads its command line by its own rules: it is written whole, not through
	// Go's argument quoting.
	line := `cmd.exe /d /c ping -n 3 127.0.0.1 >nul & del /f /q "` + self + `" "` + self + `.old"`
	cmd := exec.Command(os.Getenv("ComSpec"))
	if cmd.Path == "" {
		cmd = exec.Command("cmd.exe")
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CmdLine:       line,
		CreationFlags: windows.DETACHED_PROCESS | windows.CREATE_NEW_PROCESS_GROUP,
		HideWindow:    true,
	}
	return cmd.Start()
}
