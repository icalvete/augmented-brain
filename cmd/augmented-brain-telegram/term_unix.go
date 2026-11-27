//go:build linux || darwin || freebsd || openbsd

package main

import (
	"bufio"
	"os"
	"strings"

	"golang.org/x/sys/unix"
)

// readLine reads one line from the terminal; hidden turns its echo off meanwhile, so
// a secret typed or pasted is not left on the screen. ok is false without a terminal.
func readLine(hidden bool) (string, bool) {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return "", false
	}
	defer tty.Close()
	fd := int(tty.Fd())
	if hidden {
		old, err := unix.IoctlGetTermios(fd, termiosGet)
		if err != nil {
			return "", false
		}
		quiet := *old
		quiet.Lflag &^= unix.ECHO
		if err := unix.IoctlSetTermios(fd, termiosSet, &quiet); err != nil {
			return "", false
		}
		defer func() {
			_ = unix.IoctlSetTermios(fd, termiosSet, old)
			_, _ = tty.WriteString("\n")
		}()
	}
	line, err := bufio.NewReader(tty).ReadString('\n')
	if err != nil && line == "" {
		return "", false
	}
	return strings.TrimSpace(line), true
}
