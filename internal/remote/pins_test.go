package remote

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
	"time"
)

func readPins(t *testing.T, f *fake) []string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(f.sys.ClaudeDir, "jobs", pinsFile))
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	if err := json.Unmarshal(data, &ids); err != nil {
		t.Fatalf("pins.json: %q: %v", data, err)
	}
	return ids
}

func writePins(t *testing.T, f *fake, text string) {
	t.Helper()
	dir := filepath.Join(f.sys.ClaudeDir, "jobs")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, pinsFile)
	if err := os.WriteFile(path, []byte(text), 0o664); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o664); err != nil { // past the umask
		t.Fatal(err)
	}
}

// A task's session is pinned while it lives, so Claude does not retire it; the user's
// own pins stay; stopping it, or a new session replacing it, takes its pin out.
func TestSessionPinned(t *testing.T) {
	f, repo := launchFake(t)
	c, set := f.load()
	f.trust(repo)
	writePins(t, f, `["mine0001"]`)
	l := Launch{Task: "task-site", Name: "x", Folder: repo}
	if err := f.sys.Start(c, set, l, "go"); err != nil {
		t.Fatal(err)
	}
	if got := readPins(t, f); !slices.Equal(got, []string{"mine0001", "ab12cd34"}) {
		t.Fatalf("after start: %v", got)
	}
	fi, _ := os.Stat(filepath.Join(f.sys.ClaudeDir, "jobs", pinsFile))
	if runtime.GOOS != "windows" && fi.Mode().Perm() != 0o664 {
		t.Errorf("mode changed: %v", fi.Mode())
	}
	// Retired anyway (an earlier version did not pin): woken, it is pinned again.
	writePins(t, f, `["mine0001"]`)
	f.retire("ab12cd34")
	if woke, err := f.sys.Wake(c.Dir, l); !woke || err != nil {
		t.Fatalf("wake: %v %v", woke, err)
	}
	if got := readPins(t, f); !slices.Equal(got, []string{"mine0001", "ab12cd34"}) {
		t.Fatalf("after wake: %v", got)
	}
	// A new session replaces a retired one: the old pin goes, the new one comes.
	f.retire("ab12cd34")
	f.bgOut = "backgrounded · ef56ab78 · x\n"
	if err := f.sys.Start(c, set, l, "go"); err != nil {
		t.Fatal(err)
	}
	if got := readPins(t, f); !slices.Equal(got, []string{"mine0001", "ef56ab78"}) {
		t.Fatalf("after replacing: %v", got)
	}
	if err := f.sys.Stop(c.Dir, "task-site"); err != nil {
		t.Fatal(err)
	}
	if got := readPins(t, f); !slices.Equal(got, []string{"mine0001"}) {
		t.Fatalf("after stop: %v", got)
	}
	if _, err := os.Stat(filepath.Join(f.sys.ClaudeDir, "jobs", pinsFile+".lock")); !os.IsNotExist(err) {
		t.Errorf("the lock stayed: %v", err)
	}
}

// Pinning is help: a pins.json of another shape is left as it is, a claude with no
// jobs folder gets none, and the session starts anyway.
func TestPinsLeftAlone(t *testing.T) {
	f, repo := launchFake(t)
	c, set := f.load()
	f.trust(repo)
	if err := f.sys.Start(c, set, Launch{Task: "task-site", Name: "x", Folder: repo}, "go"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(f.sys.ClaudeDir, "jobs")); !os.IsNotExist(err) {
		t.Errorf("a jobs folder was made: %v", err)
	}
	writePins(t, f, `{"pins": []}`)
	if err := f.sys.pin("ab12cd34"); err != errPinsFormat {
		t.Errorf("another shape: %v", err)
	}
	if data, _ := os.ReadFile(filepath.Join(f.sys.ClaudeDir, "jobs", pinsFile)); string(data) != `{"pins": []}` {
		t.Errorf("rewritten: %s", data)
	}
	if err := f.sys.Stop(c.Dir, "task-site"); err != nil {
		t.Errorf("stop with pins.json of another shape: %v", err)
	}
}

// Claude's lock is waited for; one left behind longer than proper-lockfile's stale
// time is taken over.
func TestPinsLock(t *testing.T) {
	f, _ := launchFake(t)
	writePins(t, f, `[]`)
	lock := filepath.Join(f.sys.ClaudeDir, "jobs", pinsFile+".lock")
	if err := os.Mkdir(lock, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := f.sys.pin("ab12cd34"); err == nil {
		t.Error("pinned through a held lock")
	}
	old := time.Now().Add(-time.Minute)
	if err := os.Chtimes(lock, old, old); err != nil {
		t.Fatal(err)
	}
	if err := f.sys.pin("ab12cd34"); err != nil {
		t.Fatalf("a stale lock: %v", err)
	}
	if got := readPins(t, f); !slices.Equal(got, []string{"ab12cd34"}) {
		t.Errorf("pins: %v", got)
	}
}
