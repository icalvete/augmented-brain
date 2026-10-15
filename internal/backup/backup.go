// Package backup keeps each space in its own private git repository, and restores a
// space from it on a new machine.
//
// The memory folder itself becomes the repository. A copy of the configuration and the
// profile travels with it, in .augmented-brain/, so a restore needs nothing else.
package backup

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/guard"
	"github.com/icalvete/augmented-brain/internal/memory"
	"github.com/icalvete/augmented-brain/internal/paths"
	"github.com/icalvete/augmented-brain/internal/setup"
)

// MetaDir holds, inside the memory folder, the copy of the configuration and profile.
const MetaDir = ".augmented-brain"

// git runs git in dir and returns its output.
func git(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

// Setup makes the space's memory folder a git repository that backs up to url, and
// saves url in the configuration.
func Setup(c *config.Config, s *config.Space, url string) error {
	if strings.TrimSpace(url) == "" {
		return errors.New("the backup needs the URL of a private repository")
	}
	if _, err := os.Stat(filepath.Join(s.Memory, ".git")); errors.Is(err, os.ErrNotExist) {
		if _, err := git(s.Memory, "init", "-q", "-b", "main"); err != nil {
			return err
		}
	}
	if out, err := git(s.Memory, "remote", "get-url", "origin"); err == nil {
		// Only replace an origin augmented brain set itself: the memory folder may already
		// be a repository of the user's, pointing somewhere else on purpose.
		if cur := strings.TrimSpace(out); cur != url && cur != s.Backup {
			return fmt.Errorf("%s is already a git repository whose origin is %s; augmented brain does not change it. Set its backup by hand, or remove that remote first", s.Memory, cur)
		}
		if _, err := git(s.Memory, "remote", "set-url", "origin", url); err != nil {
			return err
		}
	} else if _, err := git(s.Memory, "remote", "add", "origin", url); err != nil {
		return err
	}
	s.Backup = url
	return config.SetSpaceKey(c.Dir, s.Name, "backup", url)
}

// Result says what a backup did.
type Result struct {
	Committed bool
	Pushed    bool
}

// Run copies the configuration and profile into the memory, refuses if any file holds
// a secret, commits what changed and pushes it.
func Run(c *config.Config, s *config.Space, host string) (*Result, error) {
	if s.Backup == "" {
		return nil, fmt.Errorf("space %s has no backup yet: run `augmented-brain backup setup <url of a private repository>`", s.Name)
	}
	if err := writeMeta(c, s); err != nil {
		return nil, err
	}
	if err := checkSecrets(s.Memory); err != nil {
		return nil, err
	}
	if _, err := git(s.Memory, "add", "-A"); err != nil {
		return nil, err
	}
	res := &Result{}
	if out, _ := git(s.Memory, "status", "--porcelain"); strings.TrimSpace(out) != "" {
		msg := fmt.Sprintf("backup %s %s", host, time.Now().Format("2006-01-02 15:04"))
		if _, err := git(s.Memory, "-c", "user.name=augmented brain", "-c", "user.email=backup@augmented-brain.invalid",
			"commit", "-q", "-m", msg); err != nil {
			return nil, err
		}
		res.Committed = true
	}
	if err := catchUp(c, s); err != nil {
		return res, err
	}
	if _, err := git(s.Memory, "push", "-q", "-u", "origin", "HEAD:main"); err != nil {
		return res, err
	}
	res.Pushed = true
	return res, nil
}

var ident = []string{"-c", "user.name=augmented brain", "-c", "user.email=backup@augmented-brain.invalid"}

// catchUp merges what another machine pushed to the backup since this one last did,
// so the push that follows is accepted. The generated files (the index and the copy of
// the configuration) are rebuilt when they are all that collides; a real collision on
// a note stops the backup without touching anything.
func catchUp(c *config.Config, s *config.Space) error {
	if _, err := git(s.Memory, "fetch", "-q", "origin"); err != nil {
		return nil // nothing there yet, or offline: the push will say
	}
	if _, err := git(s.Memory, "rev-parse", "-q", "--verify", "origin/main"); err != nil {
		return nil
	}
	if _, err := git(s.Memory, "merge-base", "--is-ancestor", "origin/main", "HEAD"); err == nil {
		return nil // nothing new over there
	}
	if _, err := git(s.Memory, append(ident, "merge", "-q", "--no-edit", "origin/main")...); err == nil {
		return nil
	}
	out, _ := git(s.Memory, "diff", "--name-only", "--diff-filter=U")
	var notes []string
	for _, f := range strings.Fields(out) {
		if f != memory.IndexFile && !strings.HasPrefix(f, MetaDir+"/") {
			notes = append(notes, f)
		}
	}
	if len(notes) > 0 {
		_, _ = git(s.Memory, "merge", "--abort")
		return fmt.Errorf("nothing was backed up: another machine changed the same notes (%s). Open %s, run `git merge origin/main`, keep the right version of each, commit, and back up again",
			strings.Join(notes, ", "), s.Memory)
	}
	// Only generated files collide: rebuild them and finish the merge.
	m, problems, err := memory.Load(s.Memory)
	if err != nil || len(problems) > 0 {
		_, _ = git(s.Memory, "merge", "--abort")
		return fmt.Errorf("merging the backup of another machine left notes that are not valid; nothing was backed up")
	}
	if _, err := m.WriteIndex(c.Language); err != nil {
		return err
	}
	if err := writeMeta(c, s); err != nil {
		return err
	}
	if _, err := git(s.Memory, "add", "-A"); err != nil {
		return err
	}
	_, err = git(s.Memory, append(ident, "commit", "-q", "--no-edit")...)
	return err
}

// writeMeta copies config.toml (this space only) and the profile into MetaDir.
func writeMeta(c *config.Config, s *config.Space) error {
	dir := filepath.Join(s.Memory, MetaDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	one := &config.Config{Language: c.Language, Spaces: map[string]*config.Space{s.Name: s}, Dir: dir}
	if err := one.Save(); err != nil {
		return err
	}
	if data, err := os.ReadFile(filepath.Join(c.Dir, "profile.md")); err == nil {
		return os.WriteFile(filepath.Join(dir, "profile.md"), data, 0o644)
	}
	return nil
}

// checkSecrets refuses the backup when a file of the memory holds a secret.
func checkSecrets(dir string) error {
	var found []string
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if what := guard.FindSecret(string(data)); what != "" {
			rel, _ := filepath.Rel(dir, p)
			found = append(found, fmt.Sprintf("%s (%s)", rel, what))
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(found) > 0 {
		return fmt.Errorf("nothing was backed up: these files hold secrets, remove them first: %s", strings.Join(found, ", "))
	}
	return nil
}

// RestoreOptions say where a restored space goes. Empty fields use what the backup says.
type RestoreOptions struct {
	URL        string
	Memory     string // default: ~/augmented-brain/<space>
	WorkFolder string // default: the work folder saved in the backup
}

// Restore clones a space's backup and adds the space to the configuration in cfgDir,
// creating the configuration and the profile when there are none. It returns the space.
func Restore(cfgDir string, o RestoreOptions) (*config.Space, error) {
	tmp, err := os.MkdirTemp("", "augmented-brain-restore-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)
	clone := filepath.Join(tmp, "space")
	if out, err := exec.Command("git", "clone", "-q", o.URL, clone).CombinedOutput(); err != nil {
		return nil, fmt.Errorf("cloning %s: %v: %s", o.URL, err, strings.TrimSpace(string(out)))
	}
	var saved config.Config
	if _, err := toml.DecodeFile(filepath.Join(clone, MetaDir, config.FileName), &saved); err != nil {
		return nil, fmt.Errorf("%s is not an augmented brain backup: %v", o.URL, err)
	}
	if len(saved.Spaces) != 1 {
		return nil, fmt.Errorf("the backup in %s should hold one space, it holds %d", o.URL, len(saved.Spaces))
	}
	var name string
	var s *config.Space
	for name, s = range saved.Spaces {
	}
	s.Name = name

	c, err := config.Load(cfgDir)
	fresh := false
	switch {
	case errors.Is(err, config.ErrNotFound):
		fresh = true
		c = &config.Config{Language: saved.Language, Spaces: map[string]*config.Space{}, Dir: cfgDir}
		if err := os.MkdirAll(cfgDir, 0o700); err != nil {
			return nil, err
		}
		profile := filepath.Join(cfgDir, "profile.md")
		if data, err := os.ReadFile(filepath.Join(clone, MetaDir, "profile.md")); err == nil {
			if _, err := os.Stat(profile); errors.Is(err, os.ErrNotExist) {
				if err := os.WriteFile(profile, data, 0o644); err != nil {
					return nil, err
				}
			}
		}
	case err != nil:
		return nil, err
	}
	if _, dup := c.Spaces[name]; dup {
		return nil, fmt.Errorf("there is already a space called %q here", name)
	}

	mem := o.Memory
	if mem == "" {
		mem = defaultMemory(name, s, o.WorkFolder)
	}
	if mem, err = expand(mem); err != nil {
		return nil, err
	}
	if info, err := os.Stat(mem); err == nil && !info.IsDir() {
		return nil, fmt.Errorf("%s is a file: choose a folder for the memory", mem)
	}
	if entries, err := os.ReadDir(mem); err == nil && len(entries) > 0 {
		return nil, fmt.Errorf("%s already exists and is not empty: choose another memory folder", mem)
	}
	if err := os.MkdirAll(filepath.Dir(mem), 0o755); err != nil {
		return nil, err
	}
	_ = os.Remove(mem) // an empty folder is fine
	// Clone again, from the first clone, straight into place (a rename could cross
	// filesystems), then point origin back at the real backup.
	if out, err := exec.Command("git", "clone", "-q", clone, mem).CombinedOutput(); err != nil {
		return nil, fmt.Errorf("copying the backup into %s: %v: %s", mem, err, strings.TrimSpace(string(out)))
	}
	if _, err := git(mem, "remote", "set-url", "origin", o.URL); err != nil {
		return nil, err
	}
	if err := setup.EnsureFolders(mem); err != nil {
		return nil, err
	}
	s.Memory = mem
	s.Backup = o.URL
	if o.WorkFolder != "" {
		s.WorkFolders = []string{o.WorkFolder}
	}
	for i, f := range s.WorkFolders {
		if s.WorkFolders[i], err = expand(f); err != nil {
			return nil, err
		}
		if err := os.MkdirAll(s.WorkFolders[i], 0o755); err != nil {
			return nil, err
		}
	}
	c.Spaces[name] = s
	if fresh {
		err = c.Save()
	} else {
		err = config.AddSpace(cfgDir, s) // keep the rest of the file, comments included
	}
	if err != nil {
		return nil, err
	}
	// The index is generated: rebuild it for this machine.
	if m, problems, err := memory.Load(mem); err == nil && len(problems) == 0 {
		_, _ = m.WriteIndex(c.Language)
	}
	return s, nil
}

// defaultMemory says where a restored space's notes go when the user does not say: in
// the same place inside its work folder as on the machine that made the backup (the
// usual ~/augmented-brain/<space>/notes), or ~/augmented-brain/<space> otherwise.
func defaultMemory(name string, saved *config.Space, work string) string {
	if len(saved.WorkFolders) > 0 {
		if rel, err := filepath.Rel(saved.WorkFolders[0], saved.Memory); err == nil && rel != "." && !strings.HasPrefix(rel, "..") {
			if work == "" {
				work = saved.WorkFolders[0]
			}
			return filepath.Join(work, rel)
		}
	}
	return "~/augmented-brain/" + name
}

func expand(p string) (string, error) { return paths.Resolve(p) }
