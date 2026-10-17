//go:build windows

package main

import (
	"os"

	"golang.org/x/sys/windows"
)

// A Windows console shows text in its own code page (437, 850…), not UTF-8, so ✓ and
// the accents of the Spanish messages came out as âœ“ or Ã³. When the program writes
// to a console, it switches that console to UTF-8 first. Hooks and the MCP server
// write to pipes, which are not consoles: nothing changes for them.
func init() {
	var mode uint32
	if windows.GetConsoleMode(windows.Handle(os.Stdout.Fd()), &mode) == nil {
		windows.SetConsoleOutputCP(65001) // UTF-8
	}
}
