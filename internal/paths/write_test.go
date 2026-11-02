package paths

import (
	"os"
	"path/filepath"
	"testing"
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
