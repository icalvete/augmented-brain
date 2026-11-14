//go:build !linux

package leftovers

// claudeFolders cannot be read here without a program of the system's: only the
// sessions in Claude Code's own folder count.
func claudeFolders() ([]string, bool) { return nil, false }
