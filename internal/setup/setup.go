// Package setup creates the configuration and the spaces: `init` and `space add`.
//
// It never overwrites a file that already exists, and never touches the assistant's own
// settings: the plugin's session hook finds the space from the folder it is opened in.
package setup

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/memory"
	"github.com/icalvete/augmented-brain/internal/paths"
	"github.com/icalvete/augmented-brain/internal/templates"
)

// Folders every space has.
var spaceFolders = []string{"areas", "tasks", "knowledge", "people", "rules", "reviews", "archive"}

var validName = regexp.MustCompile(`^[a-z][a-z0-9-]{0,30}$`)

// Options describe one space to create.
type Options struct {
	Language   string // en | es; only used by Init
	Space      string // e.g. work
	WorkFolder string // opening the assistant here selects the space
	Memory     string // where the space's notes live
	Today      string // YYYY-MM-DD; empty means today
}

// Result lists what was created, for the user to see.
type Result struct {
	ConfigPath string
	Created    []string
}

// Init creates the configuration, the profile and the first space.
func Init(cfgDir string, o Options) (*Result, error) {
	if _, err := os.Stat(filepath.Join(cfgDir, config.FileName)); err == nil {
		return nil, fmt.Errorf("there is already a configuration in %s; to add a space use `augmented-brain space add`", cfgDir)
	}
	lang := o.Language
	if lang == "" {
		lang = "en"
	}
	c := &config.Config{Language: lang, Spaces: map[string]*config.Space{}, Dir: cfgDir}
	return addSpace(c, o, true)
}

// AddSpace adds a space to an existing configuration.
func AddSpace(cfgDir string, o Options) (*Result, error) {
	c, err := config.Load(cfgDir)
	if err != nil {
		return nil, err
	}
	return addSpace(c, o, false)
}

func addSpace(c *config.Config, o Options, first bool) (*Result, error) {
	if !validName.MatchString(o.Space) {
		return nil, fmt.Errorf("space name %q: use lowercase letters, digits and dashes, starting with a letter", o.Space)
	}
	if _, dup := c.Spaces[o.Space]; dup {
		return nil, fmt.Errorf("there is already a space called %q", o.Space)
	}
	work, err := abs(o.WorkFolder)
	if err != nil {
		return nil, err
	}
	if info, err := os.Stat(work); err == nil && !info.IsDir() {
		return nil, fmt.Errorf("work folder %s is a file", work)
	}
	if s := c.SpaceFor(work); s != nil {
		return nil, fmt.Errorf("work folder %s already belongs to space %q", work, s.Name)
	}
	mem, err := abs(o.Memory)
	if err != nil {
		return nil, err
	}
	for _, s := range c.Spaces {
		if s.Memory == mem {
			return nil, fmt.Errorf("memory folder %s already belongs to space %q", mem, s.Name)
		}
	}
	today := o.Today
	if today == "" {
		today = time.Now().Format("2006-01-02")
	}

	res := &Result{ConfigPath: filepath.Join(c.Dir, config.FileName)}
	if _, err := os.Stat(work); errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(work, 0o755); err != nil {
			return nil, err
		}
		res.Created = append(res.Created, work+string(filepath.Separator))
	}
	if first {
		if err := os.MkdirAll(c.Dir, 0o700); err != nil {
			return nil, err
		}
		created, err := copyTemplate(c.Language, "profile.md", filepath.Join(c.Dir, "profile.md"), today)
		if err != nil {
			return nil, err
		}
		if created {
			res.Created = append(res.Created, filepath.Join(c.Dir, "profile.md"))
		}
	}
	created, err := createSpace(c.Language, mem, today)
	if err != nil {
		return nil, err
	}
	res.Created = append(res.Created, created...)

	sp := &config.Space{Name: o.Space, Memory: mem, WorkFolders: []string{work}}
	c.Spaces[o.Space] = sp
	if first {
		err = c.Save()
	} else {
		err = config.AddSpace(c.Dir, sp) // keep the rest of the file, comments included
	}
	if err != nil {
		return nil, err
	}
	return res, nil
}

// createSpace makes the space's folders, copies the starter files and writes the index.
func createSpace(lang, dir string, today string) ([]string, error) {
	var created []string
	for _, f := range spaceFolders {
		p := filepath.Join(dir, f)
		if _, err := os.Stat(p); errors.Is(err, os.ErrNotExist) {
			if err := os.MkdirAll(p, 0o755); err != nil {
				return nil, err
			}
			created = append(created, p+string(filepath.Separator))
		}
	}
	root := lang + "/space"
	err := fs.WalkDir(templates.FS, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		dst := filepath.Join(dir, filepath.FromSlash(strings.TrimPrefix(path, root+"/")))
		ok, err := copyTemplate(lang, strings.TrimPrefix(path, lang+"/"), dst, today)
		if ok {
			created = append(created, dst)
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	m, problems, err := memory.Load(dir)
	if err != nil {
		return nil, err
	}
	if len(problems) > 0 {
		return nil, fmt.Errorf("the memory folder %s has notes that are not valid: %v", dir, problems[0])
	}
	if changed, err := m.WriteIndex(lang); err != nil {
		return nil, err
	} else if changed {
		created = append(created, filepath.Join(dir, memory.IndexFile))
	}
	return created, nil
}

// EnsureFolders creates the folders every space has, if missing. git does not keep empty
// folders, so a restored space needs them back.
func EnsureFolders(dir string) error {
	for _, f := range spaceFolders {
		if err := os.MkdirAll(filepath.Join(dir, f), 0o755); err != nil {
			return err
		}
	}
	return nil
}

// copyTemplate writes templates/<lang>/<name> to dst unless dst exists. It reports
// whether it wrote the file.
func copyTemplate(lang, name, dst, today string) (bool, error) {
	if _, err := os.Stat(dst); err == nil {
		return false, nil
	}
	data, err := templates.FS.ReadFile(lang + "/" + name)
	if err != nil {
		return false, err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return false, err
	}
	text := strings.ReplaceAll(string(data), "{{DATE}}", today)
	return true, os.WriteFile(dst, []byte(text), 0o644)
}

func abs(p string) (string, error) {
	if p == "" {
		return "", errors.New("a folder is missing")
	}
	return paths.Resolve(p)
}
