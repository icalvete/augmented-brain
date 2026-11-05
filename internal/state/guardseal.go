package state

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/icalvete/augmented-brain/internal/paths"
)

// sealFile keeps the fingerprint of the guard's lists as the user last knew them: at
// the start of a session, or when the panel saved them. A list that differs from it
// changed some other way, which the guard notices (see config.GuardFingerprint).
const sealFile = "guard.seal"

// ReadSeal returns the saved fingerprint, or "" when there is none.
func ReadSeal(cfgDir string) string {
	data, err := os.ReadFile(filepath.Join(cfgDir, Dir, sealFile))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// WriteSeal saves the fingerprint the user has accepted.
func WriteSeal(cfgDir, fingerprint string) error {
	if err := os.MkdirAll(filepath.Join(cfgDir, Dir), 0o700); err != nil {
		return err
	}
	return paths.WriteAtomic(filepath.Join(cfgDir, Dir, sealFile), []byte(fingerprint+"\n"), 0o600)
}
