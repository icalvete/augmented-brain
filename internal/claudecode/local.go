package claudecode

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	augmentedbrain "github.com/icalvete/augmented-brain"
	"github.com/icalvete/augmented-brain/internal/paths"
)

// LocalDir is the folder, inside the configuration folder, where the program writes the
// plugin it carries. Claude Code reads the plugin from there, in place.
func LocalDir(cfgDir string) string {
	return filepath.Join(cfgDir, "marketplace")
}

// writtenList names, inside LocalDir, the files the program wrote there last time, so
// a file a newer version no longer carries is taken away and nothing else is touched.
const writtenList = ".augmented-brain-files"

// Files are the files of the marketplace this program carries, relative to LocalDir,
// with / between folders, sorted.
func Files() []string {
	var list []string
	fs.WalkDir(augmentedbrain.Marketplace, ".", func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			list = append(list, p)
		}
		return nil
	})
	sort.Strings(list)
	return list
}

// Content is a file of the marketplace as the program writes it for this version and
// system: plugin.json says the version, and on Windows .mcp.json leaves the PATH alone
// (its "${PATH}:…" joins with ":", which there breaks the last folder of the user's
// PATH; the installer puts the program on the PATH itself).
func Content(name, version, goos string) ([]byte, error) {
	b, err := augmentedbrain.Marketplace.ReadFile(name)
	if err != nil {
		return nil, err
	}
	switch name {
	case "plugin/.claude-plugin/plugin.json":
		return editJSON(b, func(m map[string]any) { m["version"] = version })
	case "plugin/.mcp.json":
		if goos != "windows" {
			return b, nil
		}
		return editJSON(b, func(m map[string]any) {
			servers, _ := m["mcpServers"].(map[string]any)
			for _, s := range servers {
				srv, _ := s.(map[string]any)
				env, _ := srv["env"].(map[string]any)
				delete(env, "PATH")
				if env != nil && len(env) == 0 {
					delete(srv, "env")
				}
			}
		})
	}
	return b, nil
}

func editJSON(b []byte, edit func(map[string]any)) ([]byte, error) {
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	edit(m)
	out, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}

// WriteLocal writes the marketplace this program carries to dir, one file at a time and
// each whole (a session reading the plugin meanwhile sees the old file or the new one),
// and takes away the files it wrote last time that this version no longer carries.
func WriteLocal(dir, version, goos string) error {
	files := Files()
	if len(files) == 0 {
		return fmt.Errorf("this program carries no plugin")
	}
	keep := map[string]bool{}
	for _, f := range files {
		keep[f] = true
		b, err := Content(f, version, goos)
		if err != nil {
			return err
		}
		p := filepath.Join(dir, filepath.FromSlash(f))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return err
		}
		if err := paths.WriteAtomic(p, b, 0o644); err != nil {
			return err
		}
	}
	for _, f := range readWritten(dir) {
		if !keep[f] {
			removeFile(dir, f)
		}
	}
	list := strings.Join(files, "\n") + "\n"
	return paths.WriteAtomic(filepath.Join(dir, writtenList), []byte(list), 0o644)
}

// RemoveLocal takes away what WriteLocal wrote, and the folders it leaves empty. A file
// someone else put there stays, and so does its folder.
func RemoveLocal(dir string) error {
	written := readWritten(dir)
	if len(written) == 0 {
		written = Files() // the list itself is gone: what this version writes
	}
	var first error
	for _, f := range written {
		if err := removeFile(dir, f); err != nil && first == nil {
			first = err
		}
	}
	if err := os.Remove(filepath.Join(dir, writtenList)); err != nil && !os.IsNotExist(err) && first == nil {
		first = err
	}
	os.Remove(dir) // only when empty
	return first
}

// readWritten is the list WriteLocal left in dir: only plain relative paths, so a list
// changed by hand cannot point outside the folder.
func readWritten(dir string) []string {
	b, err := os.ReadFile(filepath.Join(dir, writtenList))
	if err != nil {
		return nil
	}
	var list []string
	sc := bufio.NewScanner(bytes.NewReader(b))
	for sc.Scan() {
		f := strings.TrimSpace(sc.Text())
		if f == "" || path.IsAbs(f) || path.Clean(f) != f || f == ".." || strings.HasPrefix(f, "../") ||
			strings.Contains(f, "\\") || strings.Contains(f, ":") {
			continue
		}
		list = append(list, f)
	}
	return list
}

// removeFile removes dir/f and then each of its folders up to dir that is left empty.
func removeFile(dir, f string) error {
	p := filepath.Join(dir, filepath.FromSlash(f))
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return err
	}
	for d := filepath.Dir(p); d != dir && strings.HasPrefix(d, dir); d = filepath.Dir(d) {
		if os.Remove(d) != nil {
			break // not empty, or not there
		}
	}
	return nil
}
