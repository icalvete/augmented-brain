// Package backup keeps each space in its own private git repository, and restores a
// space from it on a new machine.
//
// The memory folder itself becomes the repository. A copy of the configuration and the
// profile travels with it, in .augmented-brain/, so a restore needs nothing else.
package backup

import (
	"errors"
	"fmt"
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
	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/memory"
	"github.com/icalvete/augmented-brain/internal/paths"
	"github.com/icalvete/augmented-brain/internal/setup"
)

// MetaDir holds, inside the memory folder, the copy of the configuration and profile.
const MetaDir = ".augmented-brain"

// runGit runs git and returns its combined output. Every git call of the package goes
// through it: with GIT_TERMINAL_PROMPT=0, a repository that wants a password fails at
// once instead of waiting for an answer nobody will type (the automatic backup has no
// terminal).
func runGit(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	// Never wait for a password; only the usual ways to reach a repository (no ext::);
	// never take a folder above the temporary one for a repository.
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_ALLOW_PROTOCOL=file:git:https:ssh",
		"GIT_CEILING_DIRECTORIES="+os.TempDir())
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// git runs git in dir and returns its output; an error carries the command and what
// git said.
func git(dir string, args ...string) (string, error) {
	out, err := runGit(append([]string{"-C", dir}, args...)...)
	if err != nil {
		return out, fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(out))
	}
	return out, nil
}

// checkURL refuses what cannot be a repository's address: nothing, or something git
// would read as one of its options.
func checkURL(url string) error {
	switch {
	case strings.TrimSpace(url) == "":
		return errors.New("the backup needs the URL of a private repository")
	case strings.HasPrefix(strings.TrimSpace(url), "-"):
		return fmt.Errorf("%q is not the address of a repository", url)
	}
	return nil
}

// links returns the symbolic links in the tree of rev in the repository at dir. A
// backup never holds one: augmented brain writes none, and one in a backup could make
// a write of a note land outside the notes, or a read take a file from elsewhere.
func links(dir, rev string) ([]string, error) {
	out, err := git(dir, "ls-tree", "-r", rev)
	if err != nil {
		return nil, err
	}
	var found []string
	for _, line := range strings.Split(out, "\n") {
		if mode, rest, ok := strings.Cut(line, " "); ok && mode == "120000" {
			if _, path, ok := strings.Cut(rest, "\t"); ok {
				found = append(found, path)
			}
		}
	}
	return found, nil
}

// Setup makes the space's memory folder a git repository that backs up to url, and
// saves url in the configuration.
func Setup(c *config.Config, s *config.Space, url string) error {
	if err := checkURL(url); err != nil {
		return err
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
	// LeftOut lists the files too big for the backup, as "path (size)".
	LeftOut []string
}

// Warning is what the user should hear about a backup that went through, in their
// language: the files it left out. Empty when there is nothing to say.
func (r *Result) Warning(lang string) string {
	if r == nil || len(r.LeftOut) == 0 {
		return ""
	}
	return i18n.T(lang, "backup.warning", MaxFileSize>>20, strings.Join(r.LeftOut, ", "))
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
	return res, nil
}

// commit records what changed in the notes. The slow scan (sizes and secrets of every
// file) runs first, without the notes lock; under it, only the copy of the settings, the
// staging and a check of what was staged, so the assistant never waits long.
func commit(c *config.Config, s *config.Space, host string, res *Result) error {
	if err := ensureRepoFiles(s.Memory); err != nil {
		return err
	}
	sc, err := scanMemory(s.Memory)
	if err != nil {
		return err
	}
	if res.LeftOut, err = leaveOutLarge(s.Memory, sc.big); err != nil {
		return err
	}
	if err := sc.refusal(); err != nil {
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
	if found, err := links(s.Memory, "origin/main"); err != nil {
		return err
	} else if len(found) > 0 {
		return fmt.Errorf("nothing was backed up: the backup in %s holds symbolic links (%s), which augmented brain never writes. Someone changed it by hand: look at it before going on", s.Backup, strings.Join(found, ", "))
	}
	// The merge writes notes: not while the assistant writes them too.
	unlock, err := lockNotes(c.Dir)
	if err != nil {
		return err
	}
	defer unlock()
	// Notes that were already not valid do not stop the merge; only new ones do.
	before := map[string]bool{}
	if _, problems, err := memory.LoadSpace(s.Memory, c.SharedDir()); err == nil {
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
		// A note each machine only added to (a journal entry each) joins by itself.
		if f != memory.IndexFile && !strings.HasPrefix(f, MetaDir+"/") && !joinAdditions(s.Memory, f) {
			notes = append(notes, f)
		}
	}
	if len(notes) > 0 {
		_, _ = git(s.Memory, "merge", "--abort")
		return &ConflictError{Notes: notes, Dir: s.Memory}
	}
	// Only generated files collide: rebuild them and finish the merge.
	m, problems, err := memory.LoadSpace(s.Memory, c.SharedDir())
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
// carries them (ProfileSpace), the profile and the shared notes. Any other space's
// backup holds neither: a copy left there by an older version is removed. The machine-wide guard lists are in
// no backup: they may name work machines, and a new machine asks for them again.
func writeMeta(c *config.Config, s *config.Space) error {
	dir := filepath.Join(s.Memory, MetaDir)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	carries := ProfileSpace(c) == s.Name
	one := &config.Config{Language: c.Language, Spaces: map[string]*config.Space{s.Name: s}, Dir: dir}
	if carries {
		one.DefaultSpace = s.Name // so a restore makes it the default space again
		one.Secrets = c.Secrets   // the names of the user's secrets travel with the profile
	}
	if err := one.Save(); err != nil {
		return err
	}
	profile := filepath.Join(dir, "profile.md")
	shared := filepath.Join(dir, config.SharedFolder)
	if err := os.RemoveAll(shared); err != nil {
		return err
	}
	if !carries {
		if err := os.Remove(profile); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}
	if _, err := os.Stat(c.SharedDir()); err == nil {
		if err := paths.CopyTree(c.SharedDir(), shared); err != nil {
			return err
		}
	}
	if data, err := os.ReadFile(filepath.Join(c.Dir, "profile.md")); err == nil {
		return os.WriteFile(profile, data, 0o600)
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
//
// Nothing that is there yet is touched until the copy of the notes is complete; if a
// later step fails, the notes and the configuration go back to what they were.
func Restore(cfgDir string, o RestoreOptions) (*config.Space, error) {
	tmp, err := os.MkdirTemp("", "augmented-brain-restore-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)
	clone := filepath.Join(tmp, "space")
	lang, s, wasDefault, err := cloneBackup(o.URL, clone)
	if err != nil {
		return nil, err
	}
	c, fresh, err := openConfig(cfgDir, lang, clone)
	if err != nil {
		return nil, err
	}
	old, err := spaceToReplace(c, s.Name, &o)
	if err != nil {
		return nil, err
	}
	mem, replacing, err := chooseMemory(s, o, old)
	if err != nil {
		return nil, err
	}
	sw, err := swapIn(clone, mem, o.URL)
	if err != nil {
		return nil, err
	}
	if err := placeSpace(s, mem, o.URL, o.WorkFolder); err != nil {
		sw.undo()
		return nil, err
	}
	c.Spaces[s.Name] = s
	if err := saveSpace(c, cfgDir, s, fresh, old != nil); err != nil {
		sw.undo()
		return nil, err
	}
	sw.keep()
	// A clone follows the umask: the restored notes become the owner's only, as notes
	// written here are.
	if err := paths.Private(mem); err != nil {
		return nil, err
	}
	// The space that was the default one is the default again, unless this machine
	// already has one: otherwise no backup would carry the profile and the shared notes.
	if wasDefault {
		if c2, err := config.Load(cfgDir); err == nil && c2.DefaultSpace == "" {
			_ = config.SetDefaultSpace(cfgDir, s.Name)
		}
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
		if err := copyProfile(filepath.Join(mem, MetaDir), cfgDir); err != nil {
			return nil, err
		}
	}
	if err := copyShared(filepath.Join(mem, MetaDir), cfgDir); err != nil {
		return nil, err
	}
	// The names of the user's secrets (never their values), when this machine has none.
	// An entry that cannot be read is left out: it never stops the restore.
	if saved, err := config.Load(filepath.Join(mem, MetaDir)); err == nil {
		if list := brain.CleanSecretNames(saved.Secrets); len(list) > 0 {
			if here, err := config.Load(cfgDir); err == nil && len(here.Secrets) == 0 {
				_ = config.SetTopList(cfgDir, "secrets", list)
			}
		}
	}
	// The index is generated: rebuild it for this machine.
	if m, problems, err := memory.LoadSpace(mem, filepath.Join(cfgDir, config.SharedFolder)); err == nil && len(problems) == 0 {
		_, _ = m.WriteIndex(c.Language)
	}
	return s, nil
}

// cloneBackup clones the backup at url into dir and reads the configuration saved in
// it: the language, the one space it holds, and whether that was the default space.
func cloneBackup(url, dir string) (string, *config.Space, bool, error) {
	if err := checkURL(url); err != nil {
		return "", nil, false, err
	}
	if out, err := runGit("-c", "core.symlinks=false", "clone", "-q", "--", url, dir); err != nil {
		return "", nil, false, fmt.Errorf("cloning %s: %v: %s", url, err, strings.TrimSpace(out))
	}
	if found, err := links(dir, "HEAD"); err != nil {
		return "", nil, false, err
	} else if len(found) > 0 {
		return "", nil, false, fmt.Errorf("the backup in %s holds symbolic links (%s), which augmented brain never writes; it is not restored", url, strings.Join(found, ", "))
	}
	var saved config.Config
	if _, err := toml.DecodeFile(filepath.Join(dir, MetaDir, config.FileName), &saved); err != nil {
		return "", nil, false, fmt.Errorf("%s is not an augmented brain backup: %v", url, err)
	}
	if len(saved.Spaces) != 1 {
		return "", nil, false, fmt.Errorf("the backup in %s should hold one space, it holds %d", url, len(saved.Spaces))
	}
	var s *config.Space
	for name, sp := range saved.Spaces {
		s = sp
		s.Name = name
	}
	return saved.Language, s, saved.DefaultSpace == s.Name, nil
}

// openConfig loads the configuration in cfgDir. A machine with none gets a new one, in
// the backup's language, with the profile of the backup in clone; fresh says so.
func openConfig(cfgDir, lang, clone string) (c *config.Config, fresh bool, err error) {
	c, err = config.Load(cfgDir)
	if !errors.Is(err, config.ErrNotFound) {
		return c, false, err
	}
	c = &config.Config{Language: lang, Spaces: map[string]*config.Space{}, Dir: cfgDir}
	if err := os.MkdirAll(cfgDir, 0o700); err != nil {
		return nil, false, err
	}
	if _, err := os.Stat(filepath.Join(cfgDir, "profile.md")); errors.Is(err, os.ErrNotExist) {
		if err := copyProfile(filepath.Join(clone, MetaDir), cfgDir); err != nil {
			return nil, false, err
		}
	}
	return c, true, nil
}

// copyProfile copies the profile kept in a backup's MetaDir meta into cfgDir. A backup
// without one copies nothing.
func copyProfile(meta, cfgDir string) error {
	data, err := os.ReadFile(filepath.Join(meta, "profile.md"))
	if err != nil {
		return nil
	}
	return os.WriteFile(filepath.Join(cfgDir, "profile.md"), data, 0o600)
}

// copyShared brings the shared notes kept in a backup's MetaDir meta to cfgDir, when
// this machine has none yet: the ones here are never written over.
func copyShared(meta, cfgDir string) error {
	src, dst := filepath.Join(meta, config.SharedFolder), filepath.Join(cfgDir, config.SharedFolder)
	if _, err := os.Stat(src); err != nil {
		return nil
	}
	if _, err := os.Stat(dst); err == nil {
		return nil
	}
	return paths.CopyTree(src, dst)
}

// spaceToReplace returns the space called name that is already here, if any. A new
// machine already has the spaces the installer made, empty: one that is still untouched
// is replaced by its backup, and its folders fill what o leaves empty; one with
// anything of the user's is not replaced.
func spaceToReplace(c *config.Config, name string, o *RestoreOptions) (*config.Space, error) {
	old, dup := c.Spaces[name]
	if !dup {
		return nil, nil
	}
	if !Pristine(old.Memory) {
		return nil, fmt.Errorf("there is already a space called %q here, with notes of its own: restore with --memory into another folder, or remove that space first", name)
	}
	if o.Memory == "" {
		o.Memory = old.Memory
	}
	if o.WorkFolder == "" && len(old.WorkFolders) > 0 {
		o.WorkFolder = old.WorkFolders[0]
	}
	return old, nil
}

// chooseMemory says where the restored notes go, and refuses a file or a folder with
// anything in it, unless that folder is the untouched space being replaced (replacing).
func chooseMemory(s *config.Space, o RestoreOptions, old *config.Space) (mem string, replacing bool, err error) {
	mem = o.Memory
	if mem == "" {
		mem = defaultMemory(s.Name, s, o.WorkFolder)
	}
	if mem, err = paths.Resolve(mem); err != nil {
		return "", false, err
	}
	replacing = old != nil && paths.Same(old.Memory, mem)
	if info, err := os.Stat(mem); err == nil && !info.IsDir() {
		return "", false, fmt.Errorf("%s is a file: choose a folder for the memory", mem)
	}
	if entries, err := os.ReadDir(mem); err == nil && len(entries) > 0 && !replacing {
		return "", false, fmt.Errorf("%s already exists and is not empty: choose another memory folder", mem)
	}
	return mem, replacing, nil
}

// swap is a restored copy of the notes put at mem; what was there before waits aside
// until the restore is kept or undone.
type swap struct{ mem, aside string }

// undo removes the restored copy and puts back what was there.
func (sw *swap) undo() {
	os.RemoveAll(sw.mem)
	if sw.aside != "" {
		_ = paths.Rename(sw.aside, sw.mem)
	}
}

// keep drops what was there before.
func (sw *swap) keep() {
	if sw.aside != "" {
		os.RemoveAll(sw.aside)
	}
}

// swapIn copies the clone to mem with origin pointing back at url. It clones again,
// from the first clone, next to mem (a rename could not cross filesystems), and only
// when that copy is complete moves what was at mem (an empty folder, or the untouched
// space) aside and the copy in.
func swapIn(clone, mem, url string) (*swap, error) {
	if err := os.MkdirAll(filepath.Dir(mem), 0o700); err != nil {
		return nil, err
	}
	staging := filepath.Join(filepath.Dir(mem), "."+filepath.Base(mem)+".restoring")
	_ = os.RemoveAll(staging)
	if out, err := runGit("-c", "core.symlinks=false", "clone", "-q", "--", clone, staging); err != nil {
		return nil, fmt.Errorf("copying the backup next to %s: %v: %s", mem, err, strings.TrimSpace(out))
	}
	if _, err := git(staging, "remote", "set-url", "origin", url); err != nil {
		os.RemoveAll(staging)
		return nil, err
	}
	if err := setup.EnsureFolders(staging); err != nil {
		os.RemoveAll(staging)
		return nil, err
	}
	sw := &swap{mem: mem}
	if _, err := os.Stat(mem); err == nil {
		sw.aside = filepath.Join(filepath.Dir(mem), "."+filepath.Base(mem)+".before-restore")
		_ = os.RemoveAll(sw.aside)
		if err := paths.Rename(mem, sw.aside); err != nil {
			os.RemoveAll(staging)
			return nil, err
		}
	}
	if err := paths.Rename(staging, mem); err != nil {
		os.RemoveAll(staging)
		sw.undo()
		return nil, err
	}
	return sw, nil
}

// placeSpace points the restored space at its new home: its notes at mem, its backup
// at url, and its work folders (work, when given) resolved and created.
func placeSpace(s *config.Space, mem, url, work string) error {
	s.Memory = mem
	s.Backup = url
	// Lists that let the assistant do more never come from a backup, which someone
	// else may have changed: the user gives them again on this machine. Lists that make
	// it ask (production profiles, untouchable hosts) do come.
	s.NotRestored = append(append([]string{}, s.GitAllowedRepos...), s.GitAskRepos...)
	s.GitAllowedRepos, s.GitAskRepos = nil, nil
	if work != "" {
		s.WorkFolders = []string{work}
	}
	for i, f := range s.WorkFolders {
		dir, err := paths.Resolve(f)
		if err != nil {
			return err
		}
		s.WorkFolders[i] = dir
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return nil
}

// saveSpace writes the restored space into the configuration: the whole file when it is
// fresh, the settings of the space it replaces, or a new space added to the rest.
// Several edits may be needed: if one fails, the file goes back to what it was, so it
// never points at a folder that is not there.
func saveSpace(c *config.Config, cfgDir string, s *config.Space, fresh, replace bool) error {
	cfgFile := filepath.Join(cfgDir, config.FileName)
	oldCfg, readErr := os.ReadFile(cfgFile)
	var err error
	switch {
	case fresh:
		err = c.Save()
	case replace:
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
	}
	return err
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
