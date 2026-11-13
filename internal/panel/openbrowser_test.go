package panel

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// The redirect file sends the browser on to the address, escaped, is the owner's alone,
// and those of earlier runs go once their ticket has expired.
func TestRedirectFile(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	old, err := redirectFile(dir, "http://127.0.0.1:1/t/old", now)
	if err != nil {
		t.Fatal(err)
	}
	past := now.Add(-TicketTTL - time.Minute)
	if err := os.Chtimes(old, past, past); err != nil {
		t.Fatal(err)
	}
	f, err := redirectFile(dir, `http://127.0.0.1:1/t/abc?s=a&b="x"`, now)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(f)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `url=http://127.0.0.1:1/t/abc?s=a&amp;b=&#34;x&#34;"`) {
		t.Errorf("page: %s", b)
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Errorf("the expired file is still there: %v", err)
	}
	if runtime.GOOS != "windows" {
		if fi, _ := os.Stat(f); fi.Mode().Perm() != 0o600 {
			t.Errorf("file mode %v", fi.Mode().Perm())
		}
		if fi, _ := os.Stat(filepath.Dir(f)); fi.Mode().Perm() != 0o700 {
			t.Errorf("folder mode %v", fi.Mode().Perm())
		}
	}
}

// The file is used only when HTML files open in the default browser and it is neither
// a snap nor a flatpak; when in doubt, the browser gets the address.
func TestBrowserReadsFiles(t *testing.T) {
	data := t.TempDir()
	t.Setenv("XDG_DATA_HOME", filepath.Join(data, "home"))
	t.Setenv("XDG_DATA_DIRS", filepath.Join(data, "sys"))
	write := func(dir, name, exec string) {
		d := filepath.Join(data, dir, "applications")
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(d, name), []byte("[Desktop Entry]\nExec="+exec+" %U\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("sys", "chrome.desktop", "/usr/bin/google-chrome")
	write("sys", "snapfox.desktop", "env BAMF=1 /snap/bin/firefox")
	write("home", "flat.desktop", "/usr/bin/flatpak run org.mozilla.firefox")
	for _, c := range []struct {
		browser, html string
		want          bool
	}{
		{"chrome.desktop", "chrome.desktop", true},
		{"chrome.desktop", "firefox_firefox.desktop", false}, // HTML opens elsewhere
		{"snapfox.desktop", "snapfox.desktop", false},
		{"flat.desktop", "flat.desktop", false},
		{"gone.desktop", "gone.desktop", false},
		{"", "", false},
	} {
		q := func(name string, args ...string) string {
			if name == "xdg-mime" {
				return c.html
			}
			return c.browser
		}
		if got := browserReadsFiles(q); got != c.want {
			t.Errorf("%s / %s: %v", c.browser, c.html, got)
		}
	}
}
