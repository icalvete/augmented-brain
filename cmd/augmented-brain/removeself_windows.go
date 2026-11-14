//go:build windows

package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/icalvete/augmented-brain/internal/i18n"
	"golang.org/x/sys/windows"
)

// removeSelf deletes the running program. Windows does not let a running .exe be
// deleted, so it leaves a small cmd.exe behind, with no window and on its own, that
// waits two seconds for this program to end and then deletes it (removeSelfLine).
func removeSelf(self, lang string) error {
	exe, line, err := removeSelfLine(self, os.Getenv("SystemRoot"))
	if err != nil {
		return errors.New(i18n.T(lang, "uninstall.selfpath", self))
	}
	cmd := exec.Command(exe)
	// Not the folder uninstall ran from: nothing of it is looked at.
	cmd.Dir = filepath.Dir(exe)
	// cmd.exe reads its command line by its own rules: it is written whole, not through
	// Go's argument quoting.
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CmdLine:       line,
		CreationFlags: windows.DETACHED_PROCESS | windows.CREATE_NEW_PROCESS_GROUP,
		HideWindow:    true,
	}
	return cmd.Start()
}
