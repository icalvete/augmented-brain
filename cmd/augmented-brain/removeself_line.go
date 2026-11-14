package main

import (
	"errors"
	"strings"
)

// errSelfPath is a program path the cleanup line of cmd.exe cannot carry.
var errSelfPath = errors.New("the program's path holds a character cmd.exe would change")

// removeSelfLine is what removeSelf runs on Windows: the cmd.exe to start and its whole
// command line, which waits two seconds for this program to end and then deletes it and
// the .old copy install.ps1 keeps when it updates. Every program goes by its full path
// under sysRoot (SystemRoot): cmd.exe looks for a bare name in the current folder first,
// so a ping.bat in the folder uninstall ran from would run (R-15). cmd.exe expands %VAR%
// even inside quotes and ends a quoted path at a quote, so a path with either is
// refused, and the program stays.
func removeSelfLine(self, sysRoot string) (exe, line string, err error) {
	if strings.ContainsAny(self, "%\"\r\n") {
		return "", "", errSelfPath
	}
	if sysRoot == "" || strings.ContainsAny(sysRoot, "%\"\r\n") {
		sysRoot = `C:\Windows`
	}
	sys := strings.TrimRight(sysRoot, `\`) + `\System32\`
	exe = sys + "cmd.exe"
	// /s: cmd.exe takes off only the outer quotes and keeps the rest as written.
	line = `"` + exe + `" /d /s /c ""` + sys + `PING.EXE" -n 3 127.0.0.1 >nul & del /f /q "` + self + `" "` + self + `.old""`
	return exe, line, nil
}
