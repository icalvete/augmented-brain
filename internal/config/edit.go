package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/icalvete/augmented-brain/internal/paths"
)

// The functions below change config.toml one line at a time, so the user's comments and
// layout survive. They share lineEditor, and every change is loaded back before it stays.

// AddSpace appends a space to the configuration file in dir, leaving everything else
// in the file (comments included) as it is.
func AddSpace(dir string, s *Space) error {
	f, err := openLines(dir)
	if err != nil {
		return err
	}
	cp := s.forFile()
	var b strings.Builder
	enc := toml.NewEncoder(&b)
	enc.Indent = ""
	if err := enc.Encode(struct {
		Spaces map[string]*Space `toml:"spaces"`
	}{map[string]*Space{s.Name: &cp}}); err != nil {
		return err
	}
	block := strings.TrimSpace(strings.Replace(b.String(), "[spaces]\n", "", 1))
	return f.write(strings.TrimRight(f.old, "\n") + "\n\n" + block + "\n")
}

// SetSpaceKey sets one string setting of a space in the configuration file in dir,
// changing only that line (or adding it under the space's header).
func SetSpaceKey(dir, space, key, value string) error {
	return setSpaceLine(dir, space, key, fmt.Sprintf("%s = %q", key, value))
}

// SetSpaceList sets one list setting of a space (such as untouchable_hosts) the same way.
func SetSpaceList(dir, space, key string, values []string) error {
	return setSpaceLine(dir, space, key, listLine(key, values))
}

// RemoveSpaceKey takes one setting out of a space in the configuration file in dir,
// changing nothing else. A setting that is not there is fine.
func RemoveSpaceKey(dir, space, key string) error {
	return setSpaceLine(dir, space, key, "")
}

// setSpaceLine puts line in place of the key's line under the space's header, or adds
// it there. An empty line removes the key.
func setSpaceLine(dir, space, key, line string) error {
	f, err := openLines(dir)
	if err != nil {
		return err
	}
	start, end, ok := f.table(space)
	if !ok {
		return fmt.Errorf("%s has no space %q", f.path, space)
	}
	if i := f.find(start, end, key); i >= 0 {
		if line == "" {
			f.lines = append(f.lines[:i], f.lines[i+1:]...)
		} else {
			f.lines[i] = line
		}
		return f.save()
	}
	if line == "" {
		return nil
	}
	f.insert(start, line)
	return f.save()
}

// SetTopList sets one machine-wide list setting (production_aws_profiles,
// untouchable_hosts) in the configuration file in dir: it changes only that line, or
// adds it before the first table, and keeps everything else, comments included.
func SetTopList(dir, key string, values []string) error {
	f, err := openLines(dir)
	if err != nil {
		return err
	}
	line := listLine(key, values)
	_, end, _ := f.table("")
	if i := f.find(0, end, key); i >= 0 {
		f.lines[i] = line
		return f.save()
	}
	// Keep a blank line between the top settings and the first table.
	if end > 0 && strings.TrimSpace(f.lines[end-1]) == "" {
		f.insert(end-1, line)
	} else if end < len(f.lines) {
		f.insert(end, line, "")
	} else {
		f.insert(end, line)
	}
	return f.save()
}

// SetDefaultSpace writes default_space in the configuration file in dir, right after
// the language, changing nothing else.
func SetDefaultSpace(dir, space string) error {
	f, err := openLines(dir)
	if err != nil {
		return err
	}
	line := fmt.Sprintf("default_space = %q", space)
	_, end, _ := f.table("")
	if i := f.find(0, end, "default_space"); i >= 0 {
		f.lines[i] = line
		return f.save()
	}
	f.insert(f.find(0, end, "language")+1, line) // the top of the file when no language
	return f.save()
}

// SetFormat records the version of the configuration folder, as the first line of the
// file, keeping everything else as it is. Version 1 is never written: it is what a
// file that says none means, and releases before versions were written refuse a
// setting they do not know.
func SetFormat(dir string, v int) error {
	if v <= 1 {
		return nil
	}
	f, err := openLines(dir)
	if err != nil {
		return err
	}
	line := fmt.Sprintf("format = %d", v)
	_, end, _ := f.table("")
	if i := f.find(0, end, "format"); i >= 0 {
		if strings.TrimSpace(f.lines[i]) == line {
			return nil
		}
		f.lines[i] = line
		return f.save()
	}
	f.insert(0, line)
	return f.save()
}

// listLine writes a list setting the way TOML reads it back: key = ["a", "b"].
func listLine(key string, values []string) string {
	quoted := make([]string, len(values))
	for i, v := range values {
		quoted[i] = fmt.Sprintf("%q", v)
	}
	return fmt.Sprintf("%s = [%s]", key, strings.Join(quoted, ", "))
}

// lineEditor holds config.toml as lines, so a setting can be changed without
// re-encoding (and losing) the rest of the file.
type lineEditor struct {
	dir, path string
	old       string // the text as read, put back if the change does not load
	lines     []string
}

// openLines reads the configuration file in dir for editing.
func openLines(dir string) (*lineEditor, error) {
	path := filepath.Join(dir, FileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return &lineEditor{dir: dir, path: path, old: string(data), lines: strings.Split(string(data), "\n")}, nil
}

// table returns the lines [start, end) that hold the settings of a space's table, or of
// the top of the file (before any table) when space is "". ok is false when the space
// has no table.
func (f *lineEditor) table(space string) (start, end int, ok bool) {
	start = 0
	if space != "" {
		start = -1
		for i, l := range f.lines {
			if t := strings.TrimSpace(l); t == "[spaces."+space+"]" || t == `[spaces."`+space+`"]` {
				start = i + 1
				break
			}
		}
		if start < 0 {
			return 0, 0, false
		}
	}
	for end = start; end < len(f.lines); end++ {
		if strings.HasPrefix(strings.TrimSpace(f.lines[end]), "[") {
			break
		}
	}
	return start, end, true
}

// find returns the line in [start, end) that sets key, or -1.
func (f *lineEditor) find(start, end int, key string) int {
	for i := start; i < end; i++ {
		t := strings.TrimSpace(f.lines[i])
		if rest, ok := strings.CutPrefix(t, key); ok && strings.HasPrefix(strings.TrimLeft(rest, " \t"), "=") {
			return i
		}
	}
	return -1
}

// insert puts lines before line at.
func (f *lineEditor) insert(at int, lines ...string) {
	f.lines = append(f.lines[:at], append(lines, f.lines[at:]...)...)
}

// save writes the edited lines back (see write).
func (f *lineEditor) save() error { return f.write(strings.Join(f.lines, "\n")) }

// write replaces the file with text and loads it back; if it does not load, the old
// text is put back and the error returned, so a bad edit never leaves a broken file.
func (f *lineEditor) write(text string) error {
	if err := writeAtomic(f.path, text); err != nil {
		return err
	}
	if _, err := Load(f.dir); err != nil {
		_ = writeAtomic(f.path, f.old)
		return fmt.Errorf("the change would break %s, so it was not made: %w", f.path, err)
	}
	return nil
}

// writeAtomic replaces a file in one step (write, then rename). When the path is a
// symlink, the file it points to is replaced and the link is kept.
func writeAtomic(path, text string) error {
	return paths.WriteAtomic(path, []byte(text), 0o600)
}
