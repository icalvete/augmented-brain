package memory

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const goodTask = "---\nid: task-ok\ntype: task\ntitle: ok\nsummary: s\ncreated: \"2026-01-01\"\nupdated: \"2026-01-01\"\nstatus: plan\nwaiting_on: me\nnext: n\n---\nbody\n"

// spaceWithGoodTask is a memory folder with one valid task.
func spaceWithGoodTask(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "tasks"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "tasks", "task-ok.md"), []byte(goodTask), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

// One file or folder that cannot be read never keeps the rest of the space from
// loading: it is a problem of its own, and the good note is there.
func TestLoadUnreadableFiles(t *testing.T) {
	for _, c := range []struct {
		name, path string
		perms      bool // needs permission bits, which Windows and root do not honor
		make       func(t *testing.T, dir string)
	}{
		{"chmod 000 file", "tasks/task-locked.md", true, func(t *testing.T, dir string) {
			p := filepath.Join(dir, "tasks", "task-locked.md")
			if err := os.WriteFile(p, []byte(goodTask), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(p, 0); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { os.Chmod(p, 0o600) })
		}},
		{"dangling link", "tasks/task-link.md", false, func(t *testing.T, dir string) {
			symlink(t, filepath.Join(dir, "nowhere.md"), filepath.Join(dir, "tasks", "task-link.md"))
		}},
		{"link loop", "tasks/task-loop.md", false, func(t *testing.T, dir string) {
			p := filepath.Join(dir, "tasks", "task-loop.md")
			symlink(t, p, p)
		}},
		{"chmod 000 subfolder", "knowledge/private", true, func(t *testing.T, dir string) {
			p := filepath.Join(dir, "knowledge", "private")
			if err := os.MkdirAll(p, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(p, 0); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { os.Chmod(p, 0o700) })
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			if c.perms && (runtime.GOOS == "windows" || os.Geteuid() == 0) {
				t.Skip("permission bits are not honored here")
			}
			dir := spaceWithGoodTask(t)
			c.make(t, dir)
			m, problems, err := Load(dir)
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if _, ok := m.Notes["task-ok"]; !ok {
				t.Errorf("the good note is missing: %v", m.IDs())
			}
			if len(problems) != 1 || problems[0].Path != c.path {
				t.Fatalf("want one problem for %s, got %v", c.path, problems)
			}
		})
	}
}

// Emacs's lock file (.#note.md, a link to nowhere) and other hidden files are not
// notes: no error and no problem.
func TestLoadSkipsHiddenFiles(t *testing.T) {
	dir := spaceWithGoodTask(t)
	symlink(t, "someone@host.12345:1700000000", filepath.Join(dir, "tasks", ".#task-ok.md"))
	if err := os.WriteFile(filepath.Join(dir, "tasks", ".draft.md"), []byte("not a note"), 0o600); err != nil {
		t.Fatal(err)
	}
	m, problems, err := Load(dir)
	if err != nil || len(problems) != 0 || len(m.Notes) != 1 {
		t.Fatalf("err=%v problems=%v notes=%v", err, problems, m.IDs())
	}
}

// A folder of a type of note that is a link is said, not silently empty.
func TestLoadLinkedTypeFolder(t *testing.T) {
	dir := spaceWithGoodTask(t)
	elsewhere := t.TempDir()
	symlink(t, elsewhere, filepath.Join(dir, "knowledge"))
	_, problems, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) != 1 || problems[0].Path != "knowledge" || !strings.Contains(problems[0].Err.Error(), "link") {
		t.Fatalf("problems: %v", problems)
	}
}

// Only a memory folder that cannot be read at all is an error.
func TestLoadMissingFolder(t *testing.T) {
	if _, _, err := Load(filepath.Join(t.TempDir(), "none")); err == nil {
		t.Fatal("want an error")
	}
}

// A date in a header that is not YYYY-MM-DD is a problem; the note is still kept.
func TestLoadBadDates(t *testing.T) {
	dir := spaceWithGoodTask(t)
	bad := strings.Replace(goodTask, "id: task-ok", "id: task-bad", 1)
	bad = strings.Replace(bad, `updated: "2026-01-01"`, "updated: 2026-13-45\nwatch:\n  - date: 2026-11-3\n    what: look", 1)
	if err := os.WriteFile(filepath.Join(dir, "tasks", "task-bad.md"), []byte(bad), 0o600); err != nil {
		t.Fatal(err)
	}
	m, problems, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := m.Notes["task-bad"]; !ok {
		t.Fatal("a note with a bad date is still kept")
	}
	var got []string
	for _, p := range problems {
		got = append(got, p.Error())
	}
	text := strings.Join(got, "\n")
	if len(problems) != 2 || !strings.Contains(text, `updated "2026-13-45"`) || !strings.Contains(text, `watch date "2026-11-3" is not a date: use YYYY-MM-DD`) {
		t.Fatalf("problems:\n%s", text)
	}
}

func symlink(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		if runtime.GOOS == "windows" {
			t.Skipf("no links here: %v", err)
		}
		t.Fatal(err)
	}
}
