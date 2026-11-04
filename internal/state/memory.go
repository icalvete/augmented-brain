package state

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/icalvete/augmented-brain/internal/filelock"
	"github.com/icalvete/augmented-brain/internal/paths"
)

// MemoryFile keeps the memory warnings already given: for each host process (named by
// its pid and when it started, so a reused pid starts clean), the highest step, in
// megabytes, the assistant was asked to warn about. See package hostmem.
const MemoryFile = "memory.json"

type memoryMarks struct {
	Warned map[string]int64 `json:"warned,omitempty"`
}

func memoryFile(cfgDir string) string { return filepath.Join(cfgDir, Dir, MemoryFile) }

func readMemory(cfgDir string) memoryMarks {
	var m memoryMarks
	if data, err := os.ReadFile(memoryFile(cfgDir)); err == nil {
		_ = json.Unmarshal(data, &m)
	}
	return m
}

// MemoryWarned is the highest step already warned about for a host (0 for none). It
// takes no lock: one small file read, after every answer.
func MemoryWarned(cfgDir, key string) int64 {
	return readMemory(cfgDir).Warned[key]
}

// MarkMemory records a warning about a host at step, and forgets the hosts that no
// longer run (alive says, by key), so the file stays as small as the open windows.
func MarkMemory(cfgDir, key string, step int64, alive func(key string) bool) error {
	unlock, err := filelock.Lock(filepath.Join(cfgDir, Dir, ".lock"))
	if err != nil {
		return err
	}
	defer unlock()
	m := readMemory(cfgDir)
	kept := map[string]int64{key: max(step, m.Warned[key])}
	for k, v := range m.Warned {
		if k != key && alive(k) {
			kept[k] = v
		}
	}
	data, err := json.MarshalIndent(memoryMarks{Warned: kept}, "", "  ")
	if err != nil {
		return err
	}
	return paths.WriteAtomic(memoryFile(cfgDir), data, 0o600)
}
