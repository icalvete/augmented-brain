package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/icalvete/augmented-brain/internal/backup"
	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/state"
	"github.com/icalvete/augmented-brain/internal/update"
)

// A download that fails is an update that failed. On 8-nov GitHub's public download
// address answered 504 for over an hour: `update` must say it did not update, and exit
// with an error, never report the update done.
func TestUpdateDownloadFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "gateway timeout", http.StatusGatewayTimeout)
	}))
	defer srv.Close()
	t.Setenv("AB_CONFIG_DIR", t.TempDir())
	t.Setenv("AB_LANGUAGE", "en")
	quickRetries(t)
	var out, errOut bytes.Buffer
	if code := cmdUpdate([]string{"--from", srv.URL}, &out, &errOut); code == 0 {
		t.Fatalf("a failed download was taken for an update:\n%s%s", out.String(), errOut.String())
	}
	if !strings.Contains(errOut.String(), "did not finish") || !strings.Contains(errOut.String(), "504") {
		t.Errorf("the error does not say what happened:\n%s", errOut.String())
	}
	if strings.Contains(out.String(), "No Claude session") {
		t.Errorf("it went on as if updated:\n%s", out.String())
	}
}

// quickRetries makes the downloads' retries quick, for tests.
func quickRetries(t *testing.T) {
	t.Helper()
	old := update.RetryDelay
	update.RetryDelay = time.Millisecond
	t.Cleanup(func() { update.RetryDelay = old })
}

// Before a new version: the health check (shown, never a reason to stop), then a backup
// of every space that has one. A backup that fails stops the update, unless --force.
func TestBeforeUpdate(t *testing.T) {
	git, err := exec.LookPath("git") // before home, which empties PATH
	if err != nil {
		t.Skip("git is needed")
	}
	h, cfg := home(t)
	var out, errOut bytes.Buffer
	// home leaves no claude and no git in PATH: the check says so, and the update goes on.
	if code := beforeUpdate("es", false, &out, &errOut); code != 0 {
		t.Fatalf("problems of the check stopped it: %d\n%s%s", code, out.String(), errOut.String())
	}
	if !strings.Contains(errOut.String(), "Claude Code") || strings.Contains(out.String(), "Copia de seguridad") {
		t.Errorf("out:\n%s\nerr:\n%s", out.String(), errOut.String())
	}
	t.Setenv("PATH", filepath.Dir(git))

	// A space with a backup that works is backed up.
	bare := filepath.Join(h, "copia.git")
	if err := exec.Command("git", "init", "-q", "--bare", "-b", "main", bare).Run(); err != nil {
		t.Fatal(err)
	}
	c, err := config.Load(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := backup.Setup(c, c.Spaces["personal"], "file://"+filepath.ToSlash(bare)); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	errOut.Reset()
	if code := beforeUpdate("es", false, &out, &errOut); code != 0 {
		t.Fatalf("with a working backup: %d\n%s%s", code, out.String(), errOut.String())
	}
	if !strings.Contains(out.String(), "personal") {
		t.Errorf("the backup is not told:\n%s", out.String())
	}

	// A backup that fails stops the update; --force goes on.
	if err := os.RemoveAll(bare); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	errOut.Reset()
	if code := beforeUpdate("es", false, &out, &errOut); code != 1 || !strings.Contains(errOut.String(), "no se ha actualizado nada") {
		t.Errorf("a failed backup: %d\n%s%s", code, out.String(), errOut.String())
	}
	if code := beforeUpdate("es", true, &out, &errOut); code != 0 {
		t.Errorf("--force: %d", code)
	}
}

// After: the sessions still on the old version, and the active task of each space.
func TestAfterUpdate(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("needs a process to stand for an open session")
	}
	sleep, err := exec.LookPath("sleep")
	if err != nil {
		t.Skip("needs sleep")
	}
	_, cfg := home(t)
	old := state.ScanProc
	state.ScanProc = false // the sessions open on this machine are not the test's
	t.Cleanup(func() { state.ScanProc = old })
	var out bytes.Buffer
	afterUpdate("es", "0.10.0-alpha", true, &out)
	if !strings.Contains(out.String(), "Ninguna sesión") {
		t.Errorf("no old sessions:\n%s", out.String())
	}

	// An open session on the old version: a live process that noted 0.9.0.
	p := exec.Command(sleep, "30")
	if err := p.Start(); err != nil {
		t.Skip(err)
	}
	defer p.Process.Kill()
	dir := filepath.Join(cfg, state.ServersDir)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, strconv.Itoa(p.Process.Pid)), []byte("0.9.0-alpha\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := state.Started(cfg, "trabajo", "s1", "", "", time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := state.Touched(cfg, "trabajo", state.Who{ID: "s1"}, "task-mudanza", time.Now(), true); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	afterUpdate("es", "0.10.0-alpha", false, &out) // --no-restart
	for _, want := range []string{"1 sesiones", "trabajo", "task-mudanza", "augmented-brain doctor"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("missing %q:\n%s", want, out.String())
		}
	}

	// With the restart, the new program does it: when it leaves none, nothing more is
	// said; when it leaves some, the tasks to park and how to restart Claude follow.
	ran := 0
	left := error(nil)
	orig := restartOldWith
	t.Cleanup(func() { restartOldWith = orig })
	restartOldWith = func(w io.Writer) error {
		ran++
		fmt.Fprintln(w, "(restart --old)")
		return left
	}
	out.Reset()
	afterUpdate("es", "0.10.0-alpha", true, &out)
	if ran != 1 || strings.Contains(out.String(), "task-mudanza") {
		t.Errorf("all restarted (%d runs):\n%s", ran, out.String())
	}
	left = errors.New("exit status 1")
	out.Reset()
	afterUpdate("es", "0.10.0-alpha", true, &out)
	if !strings.Contains(out.String(), "task-mudanza") || strings.Contains(out.String(), "1 sesiones") {
		t.Errorf("some left:\n%s", out.String())
	}
}

// AB_RELEASE_URL (and AB_MARKETPLACE) never decide where an update comes from: a
// project's settings can put them in a Claude session's environment. `update` says it
// ignores them and asks GitHub as always; nothing is asked of the address they name.
func TestUpdateIgnoresReleaseURL(t *testing.T) {
	asked := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked = true
		fmt.Fprintln(w, "echo pwned")
	}))
	defer srv.Close()
	home(t)
	t.Setenv("AB_LANGUAGE", "en")
	t.Setenv("AB_RELEASE_URL", srv.URL)
	t.Setenv("AB_MARKETPLACE", srv.URL)
	old := latestRelease
	latestRelease = func(*http.Client) (string, error) { return "0.0.1", nil } // nothing newer
	t.Cleanup(func() { latestRelease = old })
	var out, errOut bytes.Buffer
	if code := cmdUpdate(nil, &out, &errOut); code != 0 {
		t.Fatalf("exit %d\n%s%s", code, out.String(), errOut.String())
	}
	for _, v := range []string{"AB_RELEASE_URL", "AB_MARKETPLACE"} {
		if !strings.Contains(errOut.String(), v+" está puesta, pero `update` no le hace caso") {
			t.Errorf("it does not say it ignores %s:\n%s", v, errOut.String())
		}
	}
	if !strings.Contains(out.String(), "es la última versión") {
		t.Errorf("it did not ask GitHub:\n%s", out.String())
	}
	if asked {
		t.Error("the address of AB_RELEASE_URL was asked")
	}
}

// --from takes https://, file:// and http:// only to this computer or a test VM's host.
func TestUpdateFrom(t *testing.T) {
	for addr, ok := range map[string]bool{
		"https://example.com/r":     true,
		"file:///tmp/release":       true,
		"http://127.0.0.1:8000":     true,
		"http://localhost:8000/rel": true,
		"http://10.0.2.2:8000":      true,
		"http://example.com/r":      false,
		"http://127.0.0.1.evil.com": false,
		"ftp://127.0.0.1/r":         false,
		"127.0.0.1:8000":            false,
	} {
		if err := checkFrom(addr); (err == nil) != ok {
			t.Errorf("%s: %v", addr, err)
		}
	}
	t.Setenv("AB_CONFIG_DIR", t.TempDir())
	t.Setenv("AB_LANGUAGE", "en")
	var out, errOut bytes.Buffer
	if code := cmdUpdate([]string{"--from", "http://example.com/r"}, &out, &errOut); code != 2 || !strings.Contains(errOut.String(), "Nothing was updated") {
		t.Errorf("exit %d\n%s%s", code, out.String(), errOut.String())
	}
}

// The installer gets no AB_ variable but those of installerEnv: the system's own go.
func TestInstallerEnviron(t *testing.T) {
	got := installerEnviron([]string{"PATH=/bin", "HOME=/h", "AB_CONFIG_DIR=/c", "AB_INSTALL_DIR=/b",
		"AB_RELEASE_URL=http://x", "AB_MARKETPLACE=/m", "AB_DESKTOP_URL=x", "ab_api_url=x", "LANG=es_ES.UTF-8"})
	want := "PATH=/bin HOME=/h AB_CONFIG_DIR=/c AB_INSTALL_DIR=/b LANG=es_ES.UTF-8"
	if strings.Join(got, " ") != want {
		t.Errorf("got %q", got)
	}
}
