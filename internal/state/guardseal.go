package state

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/icalvete/augmented-brain/internal/paths"
)

// sealFile keeps the fingerprint of the guard's lists as the user last knew them: at
// the start of a session, or when the panel saved them. A list that differs from it
// changed some other way, which the guard notices (see config.GuardFingerprint).
const sealFile = "guard.seal"

// ReadSeal returns the saved fingerprint, or "" when there is none. Any other problem
// (the file or the state folder cannot be read) is an error: the guard cannot tell then
// whether the lists changed, and must not take that as "no seal".
func ReadSeal(cfgDir string) (string, error) {
	data, err := os.ReadFile(filepath.Join(cfgDir, Dir, sealFile))
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// WriteSeal saves the fingerprint the user has accepted.
func WriteSeal(cfgDir, fingerprint string) error {
	if err := os.MkdirAll(filepath.Join(cfgDir, Dir), 0o700); err != nil {
		return err
	}
	return paths.WriteAtomic(filepath.Join(cfgDir, Dir, sealFile), []byte(fingerprint+"\n"), 0o600)
}
