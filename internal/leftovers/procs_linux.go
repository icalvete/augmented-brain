package leftovers

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// claudeFolders are the working folders of every claude process running now, read from
// /proc: whatever Claude folder (CLAUDE_CONFIG_DIR) each one uses.
func claudeFolders() ([]string, bool) {
	dirs, err := os.ReadDir("/proc")
	if err != nil {
		return nil, false
	}
	var out []string
	for _, d := range dirs {
		if _, err := strconv.Atoi(d.Name()); err != nil {
			continue
		}
		p := filepath.Join("/proc", d.Name())
		comm, _ := os.ReadFile(filepath.Join(p, "comm"))
		claude := strings.TrimSpace(string(comm)) == "claude"
		if !claude {
			cmdline, _ := os.ReadFile(filepath.Join(p, "cmdline"))
			first, _, _ := strings.Cut(string(cmdline), "\x00")
			claude = filepath.Base(first) == "claude"
		}
		if !claude {
			continue
		}
		if cwd, err := os.Readlink(filepath.Join(p, "cwd")); err == nil {
			out = append(out, strings.TrimSuffix(cwd, " (deleted)"))
		}
	}
	return out, true
}
