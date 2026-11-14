package remote

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A service's log on macOS that grew past LogCap is cut to its last lines (R-20 of the
// November 2026 review: launchd never rotates it).
func TestTrimLog(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.log")
	line := strings.Repeat("x", 99) + "\n"
	big := strings.Repeat(line, 10<<20/len(line)) + "the last line\n"
	if err := os.WriteFile(path, []byte(big), 0o600); err != nil {
		t.Fatal(err)
	}
	trimLog(path)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) > LogKeep || !strings.HasSuffix(string(data), "the last line\n") || !strings.HasPrefix(string(data), "xxx") {
		t.Errorf("%d bytes left, ending %q", len(data), data[max(len(data)-20, 0):])
	}
	small := []byte("a short log\n")
	if err := os.WriteFile(path, small, 0o600); err != nil {
		t.Fatal(err)
	}
	trimLog(path)
	if data, _ := os.ReadFile(path); string(data) != string(small) {
		t.Errorf("a short log changed: %q", data)
	}
}
