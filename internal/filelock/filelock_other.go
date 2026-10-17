//go:build !unix && !windows

package filelock

import "os"

// Elsewhere (not Unix, not Windows) writers in one process are still serialized by the
// MCP server, but two processes are not.
func lock(*os.File) error   { return nil }
func unlock(*os.File) error { return nil }
