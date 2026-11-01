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
	"slices"
	"strings"
	"time"

	"github.com/BurntSushi/toml"

	"github.com/icalvete/augmented-brain/internal/brain"
	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/filelock"
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
	if err := ensureRepoFiles(s.Memory); err != nil {
		return err
	}
	s.Backup = url
	return config.SetSpaceKey(c.Dir, s.Name, "backup", url)
}

// Result says what a backup did.
type Result struct {
	Committed bool
	Pushed    bool
	// LeftOut lists the files too big for the backup, as "path (size)".
	LeftOut []string
}

// Warning is what the user should hear about a backup that went through: the files it
// left out. Empty when there is nothing to say.
func (r *Result) Warning() string {
	if r == nil || len(r.LeftOut) == 0 {
		return ""
	}
	return fmt.Sprintf("these files are bigger than %d MB and were left out of the backup: %s. Move them out of the notes folder (or make them smaller); the rest is backed up",
		MaxFileSize>>20, strings.Join(r.LeftOut, ", "))
}

// Run copies the configuration and profile into the memory, refuses if any file holds
// a secret, commits what changed and pushes it. It takes the backup lock, so two
// backups never share a repository, and the notes lock while it reads and merges the
// notes, so it never meets a half-done write of the assistant.
func Run(c *config.Config, s *config.Space, host string) (*Result, error) {
	if s.Backup == "" {
		return nil, fmt.Errorf("space %s has no backup yet: run `augmented-brain backup setup <url of a private repository>`", s.Name)
	}
	unlock, err := filelock.Lock(filepath.Join(c.Dir, backupLock))
	if err != nil {
		return nil, err
	}
	defer unlock()
	res := &Result{}
	if err := commit(c, s, host, res); err != nil {
		return nil, err
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

// commit records what changed in the notes. The slow scans (sizes and secrets of every
// file) run first, without the notes lock; under it, only the copy of the settings, the
// staging and a check of what was staged, so the assistant never waits long.
func commit(c *config.Config, s *config.Space, host string, res *Result) error {
	if err := ensureRepoFiles(s.Memory); err != nil {
		return err
	}
	var err error
	if res.LeftOut, err = leaveOutLarge(s.Memory); err != nil {
		return err
	}
	if err := checkSecrets(s.Memory); err != nil {
		return err
	}
	unlock, err := lockNotes(c.Dir)
	if err != nil {
		return err
	}
	defer unlock()
	if err := writeMeta(c, s); err != nil {
		return err
	}
	if _, err := git(s.Memory, "add", "-A"); err != nil {
		return err
	}
	// A file may have changed since the scans: check what was staged, whatever the
	// patterns in .git/info/exclude matched.
	more, err := checkStaged(s.Memory)
	if err != nil {
		_, _ = git(s.Memory, "reset", "-q")
		return err
	}
	for _, m := range more {
		if !slices.Contains(res.LeftOut, m) {
			res.LeftOut = append(res.LeftOut, m)
		}
	}
	if out, _ := git(s.Memory, "status", "--porcelain"); strings.TrimSpace(out) != "" {
		msg := fmt.Sprintf("backup %s %s", host, time.Now().Format("2006-01-02 15:04"))
		if _, err := git(s.Memory, append(ident, "commit", "-q", "-m", msg)...); err != nil {
			return err
		}
		res.Committed = true
	}
	return nil
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
	// The merge writes notes: not while the assistant writes them too.
	unlock, err := lockNotes(c.Dir)
	if err != nil {
		return err
	}
	defer unlock()
	// Notes that were already not valid do not stop the merge; only new ones do.
	before := map[string]bool{}
	if _, problems, err := memory.Load(s.Memory); err == nil {
		for _, p := range problems {
			before[p.Error()] = true
		}
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
	var added []string
	for _, p := range problems {
		if !before[p.Error()] {
			added = append(added, p.Error())
		}
	}
	if err != nil || len(added) > 0 {
		_, _ = git(s.Memory, "merge", "--abort")
		return fmt.Errorf("merging the backup of another machine left notes that are not valid (%s); nothing was backed up", strings.Join(added, "; "))
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

// ProfileSpace is the space whose backup carries the profile: the default space, or
// the only one. Only one backup carries them, so the
// personal profile never lands in, say, a work repository on a company's git.
func ProfileSpace(c *config.Config) string {
	if c.DefaultSpace != "" {
		return c.DefaultSpace
	}
	if len(c.Spaces) == 1 {
		for name := range c.Spaces {
			return name
		}
	}
	return ""
}

// writeMeta copies config.toml (this space only) into MetaDir and, in the space that
// carries it (ProfileSpace), the profile. Any other space's backup holds no profile: a
// copy left there by an older version is removed. The machine-wide guard lists are in
// no backup: they may name work machines, and a new machine asks for them again.
func writeMeta(c *config.Config, s *config.Space) error {
	dir := filepath.Join(s.Memory, MetaDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	carries := ProfileSpace(c) == s.Name
	one := &config.Config{Language: c.Language, Spaces: map[string]*config.Space{s.Name: s}, Dir: dir}
	if err := one.Save(); err != nil {
		return err
	}
	profile := filepath.Join(dir, "profile.md")
	if !carries {
		if err := os.Remove(profile); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}
	if data, err := os.ReadFile(filepath.Join(c.Dir, "profile.md")); err == nil {
		return os.WriteFile(profile, data, 0o644)
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
		if info, err := d.Info(); err == nil && info.Size() > MaxFileSize {
			return nil // left out of the backup anyway
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
	// A new machine already has the spaces the installer made, empty. One that is still
	// untouched is replaced by its backup; one with anything of the user's is not.
	var old *config.Space
	if sp, dup := c.Spaces[name]; dup {
		if !Pristine(sp.Memory) {
			return nil, fmt.Errorf("there is already a space called %q here, with notes of its own: restore with --memory into another folder, or remove that space first", name)
		}
		old = sp
		if o.Memory == "" {
			o.Memory = old.Memory
		}
		if o.WorkFolder == "" && len(old.WorkFolders) > 0 {
			o.WorkFolder = old.WorkFolders[0]
		}
	}

	mem := o.Memory
	if mem == "" {
		mem = defaultMemory(name, s, o.WorkFolder)
	}
	if mem, err = expand(mem); err != nil {
		return nil, err
	}
	replacing := old != nil && paths.Same(old.Memory, mem)
	if info, err := os.Stat(mem); err == nil && !info.IsDir() {
		return nil, fmt.Errorf("%s is a file: choose a folder for the memory", mem)
	}
	if entries, err := os.ReadDir(mem); err == nil && len(entries) > 0 && !replacing {
		return nil, fmt.Errorf("%s already exists and is not empty: choose another memory folder", mem)
	}
	if err := os.MkdirAll(filepath.Dir(mem), 0o755); err != nil {
		return nil, err
	}
	// Clone again, from the first clone, next to its place (a rename could not cross
	// filesystems), and point origin back at the real backup. Nothing that is there yet
	// is touched until this copy is complete.
	staging := filepath.Join(filepath.Dir(mem), "."+filepath.Base(mem)+".restoring")
	_ = os.RemoveAll(staging)
	if out, err := exec.Command("git", "clone", "-q", clone, staging).CombinedOutput(); err != nil {
		return nil, fmt.Errorf("copying the backup next to %s: %v: %s", mem, err, strings.TrimSpace(string(out)))
	}
	if _, err := git(staging, "remote", "set-url", "origin", o.URL); err != nil {
		os.RemoveAll(staging)
		return nil, err
	}
	if err := setup.EnsureFolders(staging); err != nil {
		os.RemoveAll(staging)
		return nil, err
	}
	// Swap it in: what was there (an empty folder, or the untouched space) goes aside,
	// and comes back if anything after this fails.
	aside := ""
	if _, err := os.Stat(mem); err == nil {
		aside = filepath.Join(filepath.Dir(mem), "."+filepath.Base(mem)+".before-restore")
		_ = os.RemoveAll(aside)
		if err := paths.Rename(mem, aside); err != nil {
			os.RemoveAll(staging)
			return nil, err
		}
	}
	undo := func() {
		os.RemoveAll(mem)
		if aside != "" {
			_ = paths.Rename(aside, mem)
		}
	}
	if err := paths.Rename(staging, mem); err != nil {
		os.RemoveAll(staging)
		undo()
		return nil, err
	}

	s.Memory = mem
	s.Backup = o.URL
	if o.WorkFolder != "" {
		s.WorkFolders = []string{o.WorkFolder}
	}
	for i, f := range s.WorkFolders {
		if s.WorkFolders[i], err = expand(f); err != nil {
			undo()
			return nil, err
		}
		if err := os.MkdirAll(s.WorkFolders[i], 0o755); err != nil {
			undo()
			return nil, err
		}
	}
	c.Spaces[name] = s
	// Several edits of the configuration may follow: if one fails, the file goes back to
	// what it was, so it never points at a folder that is not there.
	cfgFile := filepath.Join(cfgDir, config.FileName)
	oldCfg, readErr := os.ReadFile(cfgFile)
	switch {
	case fresh:
		err = c.Save()
	case old != nil:
		err = editSpace(cfgDir, s) // the space is there: change its settings, keep the rest
	default:
		err = config.AddSpace(cfgDir, s) // keep the rest of the file, comments included
	}
	if err != nil {
		if fresh {
			os.Remove(cfgFile)
		} else if readErr == nil {
			_ = os.WriteFile(cfgFile, oldCfg, 0o600)
		}
		undo()
		return nil, err
	}
	if aside != "" {
		os.RemoveAll(aside)
	}
	// The untouched space now lives elsewhere: remove it, unless the restored notes went
	// inside it.
	if old != nil && !replacing && !paths.Within(mem, old.Memory) {
		os.RemoveAll(old.Memory)
	}
	// The guard's lists are in no backup: the next session offers to ask for them.
	if c2, err := config.Load(cfgDir); err == nil {
		if profiles, hosts := c2.GuardLists(); len(profiles)+len(hosts) == 0 {
			_ = os.WriteFile(filepath.Join(cfgDir, config.AskSafetyFile), nil, 0o600)
		}
	}
	// A blank profile (the one a new install writes) gives way to the backed-up one, when
	// this backup carries it (only the default space's does).
	if !fresh && brain.ProfileEmpty(cfgDir) {
		if data, err := os.ReadFile(filepath.Join(mem, MetaDir, "profile.md")); err == nil {
			if err := os.WriteFile(filepath.Join(cfgDir, "profile.md"), data, 0o644); err != nil {
				return nil, err
			}
		}
	}
	// The index is generated: rebuild it for this machine.
	if m, problems, err := memory.Load(mem); err == nil && len(problems) == 0 {
		_, _ = m.WriteIndex(c.Language)
	}
	return s, nil
}

// Pristine reports whether a space's memory is still as the installer left it: not a
// backup, no notes but the starter area, and an empty inbox. Only such a space may be
// replaced by a restore.
func Pristine(dir string) bool {
	if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
		return false
	}
	m, problems, err := memory.Load(dir)
	if err != nil || len(problems) > 0 || len(m.Notes) > 1 {
		return false
	}
	for id := range m.Notes {
		if id != "area-general" {
			return false
		}
	}
	inbox, _ := os.ReadFile(filepath.Join(dir, "inbox.md"))
	for _, line := range strings.Split(string(inbox), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "- ") {
			return false
		}
	}
	return true
}

// editSpace is updateSpace; tests replace it to fail half way.
var editSpace = updateSpace

// updateSpace writes a restored space's settings over those of the space it replaces.
func updateSpace(cfgDir string, s *config.Space) error {
	for key, value := range map[string]string{"memory": paths.Tilde(s.Memory), "backup": s.Backup} {
		if err := config.SetSpaceKey(cfgDir, s.Name, key, value); err != nil {
			return err
		}
	}
	work := make([]string, len(s.WorkFolders))
	for i, f := range s.WorkFolders {
		work[i] = paths.Tilde(f)
	}
	for key, values := range map[string][]string{
		"work_folders":            work,
		"production_aws_profiles": s.ProductionAWSProfiles,
		"untouchable_hosts":       s.UntouchableHosts,
		"git_allowed_repos":       s.GitAllowedRepos,
	} {
		if len(values) == 0 && key != "work_folders" {
			continue
		}
		if err := config.SetSpaceList(cfgDir, s.Name, key, values); err != nil {
			return err
		}
	}
	return nil
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
