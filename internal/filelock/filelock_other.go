//go:build !unix

package filelock

import "os"

// Windows is not supported yet: writers in one process are still serialized by the
// MCP server, but two processes are not.
func lock(*os.File) error   { return nil }
func unlock(*os.File) error { return nil }
