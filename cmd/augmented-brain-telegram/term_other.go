//go:build !(linux || darwin || freebsd || openbsd)

package main

// readLine has no terminal to read from here: Telegram is not supported on this system.
func readLine(bool) (string, bool) { return "", false }
