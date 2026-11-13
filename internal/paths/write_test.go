package paths

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// WriteAtomic replaces the file a symlink points to and keeps the link.
func TestWriteAtomicKeepsLinks(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "real.toml")
	link := filepath.Join(dir, "link.toml")
	if err := os.WriteFile(real, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, link); err != nil {
		t.Skip("no symlinks here:", err)
	}
	if err := WriteAtomic(link, []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	if fi, err := os.Lstat(link); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Error("the link was replaced by a file")
	}
	if data, _ := os.ReadFile(real); string(data) != "new" {
		t.Errorf("the file the link points to has %q", data)
	}
	if _, err := os.Stat(real + ".tmp"); err == nil {
		t.Error("the temporary file was left behind")
	}
}

// countSyncs counts the file and folder syncs while the test runs.
func countSyncs(t *testing.T) (files, dirs *int) {
	t.Helper()
	f, d := 0, 0
	origFile, origDir := syncFile, syncDir
	t.Cleanup(func() { syncFile, syncDir = origFile, origDir })
	syncFile = func(fh *os.File) error { f++; return origFile(fh) }
	syncDir = func(dir string) error { d++; return origDir(dir) }
	return &f, &d
}

// WriteAtomic and WriteAtomicNoLink wait for the disk: one sync of the file before the
// rename and one of its folder after. WriteAtomicQuick does neither.
func TestWriteAtomicSyncs(t *testing.T) {
	dir := t.TempDir()
	for _, c := range []struct {
		name        string
		write       func(string, []byte, os.FileMode) error
		files, dirs int
	}{
		{"WriteAtomic", WriteAtomic, 1, 1},
		{"WriteAtomicNoLink", WriteAtomicNoLink, 1, 1},
		{"WriteAtomicQuick", WriteAtomicQuick, 0, 0},
	} {
		files, dirs := countSyncs(t)
		path := filepath.Join(dir, c.name+".md")
		if err := c.write(path, []byte("text"), 0o600); err != nil {
			t.Fatal(err)
		}
		if *files != c.files || *dirs != c.dirs {
			t.Errorf("%s: %d file syncs and %d folder syncs, want %d and %d", c.name, *files, *dirs, c.files, c.dirs)
		}
		if data, _ := os.ReadFile(path); string(data) != "text" {
			t.Errorf("%s wrote %q", c.name, data)
		}
		if left, _ := filepath.Glob(path + ".*.tmp"); len(left) > 0 {
			t.Errorf("%s left %v", c.name, left)
		}
	}
}

// A write that fails leaves no temporary file and the old file as it was.
func TestWriteAtomicFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")
	if err := os.WriteFile(path, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	orig := syncFile
	t.Cleanup(func() { syncFile = orig })
	syncFile = func(*os.File) error { return errors.New("disk gone") }
	if err := WriteAtomic(path, []byte("new"), 0o600); err == nil {
		t.Fatal("a failed sync was not an error")
	}
	if data, _ := os.ReadFile(path); string(data) != "old" {
		t.Errorf("the file holds %q", data)
	}
	if left, _ := filepath.Glob(path + ".*.tmp"); len(left) > 0 {
		t.Errorf("left %v", left)
	}
	if err := WriteAtomic(filepath.Join(dir, "missing", "note.md"), nil, 0o600); err == nil {
		t.Error("a write into a missing folder worked")
	}
}

// WriteAtomicNoLink never writes through a link: the link and what it points to stay
// as they were. A plain file it replaces.
func TestWriteAtomicNoLinkRefusesLinks(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.txt")
	link := filepath.Join(dir, "MEMORY.md")
	if err := os.WriteFile(outside, []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, link); err != nil {
		t.Skip("no symlinks here:", err)
	}
	if err := WriteAtomicNoLink(link, []byte("index"), 0o600); !errors.Is(err, ErrLink) {
		t.Fatalf("writing over a link: %v, want ErrLink", err)
	}
	if data, _ := os.ReadFile(outside); string(data) != "mine" {
		t.Errorf("the file outside holds %q", data)
	}
	if fi, err := os.Lstat(link); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Error("the link was replaced")
	}
}

func TestIsTemp(t *testing.T) {
	for name, want := range map[string]bool{
		"task-a.md.123456.tmp": true,
		"MEMORY.md.1.tmp":      true,
		"task-a.md.tmp":        false,
		"task-a.md.12a.tmp":    false,
		"task-a.md..tmp":       false,
		".123.tmp":             false,
		"notes.tmp":            false,
		"task-a.md.123456":     false,
	} {
		if got := isTemp(name); got != want {
			t.Errorf("isTemp(%q) = %v, want %v", name, got, want)
		}
	}
}

// CleanTemps removes the temporary files a write cut short left, once they are old
// enough, in every folder but the ones skipped; anything else stays.
func TestCleanTemps(t *testing.T) {
	dir := t.TempDir()
	old := time.Now().Add(-2 * time.Hour)
	files := map[string]bool{ // name: removed
		"tasks/task-a.md.123.tmp":  true,
		"MEMORY.md.9.tmp":          true,
		"tasks/task-b.md.456.tmp":  false, // new: a write may still be going on
		"tasks/task-a.md":          false,
		"docs/report.txt.789.tmp":  false, // a skipped folder
		"tasks/backup.tmp":         false, // not one of ours
		"archive/x/deep.md.42.tmp": true,
	}
	for name := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		if name != "tasks/task-b.md.456.tmp" {
			if err := os.Chtimes(p, old, old); err != nil {
				t.Fatal(err)
			}
		}
	}
	if n := CleanTemps(dir, time.Hour, "docs"); n != 3 {
		t.Errorf("removed %d, want 3", n)
	}
	for name, removed := range files {
		_, err := os.Stat(filepath.Join(dir, filepath.FromSlash(name)))
		if gone := os.IsNotExist(err); gone != removed {
			t.Errorf("%s: removed %v, want %v", name, gone, removed)
		}
	}
	if n := CleanTemps(filepath.Join(dir, "missing"), time.Hour); n != 0 {
		t.Errorf("a missing folder: %d", n)
	}
}

// CopyTreeExact copies links as links and syncs every file and folder; CopyTree leaves
// links out, as a backup must.
func TestCopyTree(t *testing.T) {
	src := t.TempDir()
	for _, f := range []string{"tasks/task-a.md", "skip/x.md", "top.md"} {
		p := filepath.Join(src, filepath.FromSlash(f))
		_ = os.MkdirAll(filepath.Dir(p), 0o700)
		if err := os.WriteFile(p, []byte(f), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	links := os.Symlink("../top.md", filepath.Join(src, "tasks", "link.md")) == nil

	files, dirs := countSyncs(t)
	exact := filepath.Join(t.TempDir(), "exact")
	if err := CopyTreeExact(src, exact, "skip"); err != nil {
		t.Fatal(err)
	}
	if *files != 2 || *dirs != 2 {
		t.Errorf("%d file syncs and %d folder syncs, want 2 and 2", *files, *dirs)
	}
	if data, _ := os.ReadFile(filepath.Join(exact, "tasks", "task-a.md")); string(data) != "tasks/task-a.md" {
		t.Errorf("copied %q", data)
	}
	if _, err := os.Stat(filepath.Join(exact, "skip")); err == nil {
		t.Error("a skipped folder was copied")
	}
	if links {
		if to, err := os.Readlink(filepath.Join(exact, "tasks", "link.md")); err != nil || to != "../top.md" {
			t.Errorf("the link in the copy: %q %v", to, err)
		}
	}

	plain := filepath.Join(t.TempDir(), "plain")
	if err := CopyTree(src, plain); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(plain, "tasks", "link.md")); err == nil {
		t.Error("CopyTree copied a link")
	}
	if _, err := os.Stat(filepath.Join(plain, "skip", "x.md")); err != nil {
		t.Error("CopyTree left out a folder it was not told to skip")
	}
}

func TestRemove(t *testing.T) {
	p := filepath.Join(t.TempDir(), "f")
	if err := os.WriteFile(p, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Remove(p); err != nil {
		t.Fatal(err)
	}
	if err := Remove(p); err != nil {
		t.Errorf("removing a file already gone: %v", err)
	}
}
