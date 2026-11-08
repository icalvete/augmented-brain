package update

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// github stands in for github.com and its API: the public address answers with public,
// the API serves the file when api is true. It counts the calls to each.
func github(t *testing.T, public int, api bool) (*int32, *int32) {
	t.Helper()
	var pub, viaAPI int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/download/"):
			atomic.AddInt32(&pub, 1)
			if public != http.StatusOK {
				http.Error(w, "no", public)
				return
			}
			w.Write([]byte("echo public"))
		case r.URL.Path == "/api/releases/tags/v1.2.0":
			atomic.AddInt32(&viaAPI, 1)
			if !api {
				http.Error(w, "no", http.StatusServiceUnavailable)
				return
			}
			w.Write([]byte(`{"assets":[{"name":"other","url":"x"},{"name":"install.sh","url":"http://` + r.Host + `/api/assets/7"}]}`))
		case r.URL.Path == "/api/assets/7":
			if r.Header.Get("Accept") != "application/octet-stream" {
				w.Write([]byte(`{"json":"not the file"}`))
				return
			}
			w.Write([]byte("echo api"))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	oldPub, oldAPI, oldDelay := PublicBase, APIBase, RetryDelay
	PublicBase, APIBase, RetryDelay = srv.URL+"/download", srv.URL+"/api", time.Millisecond
	t.Cleanup(func() { PublicBase, APIBase, RetryDelay = oldPub, oldAPI, oldDelay })
	return &pub, &viaAPI
}

func TestDownload(t *testing.T) {
	client := &http.Client{Timeout: 5 * time.Second}
	read := func(p string) string {
		b, _ := os.ReadFile(p)
		return string(b)
	}

	// The public address works: the API is never asked.
	pub, api := github(t, http.StatusOK, true)
	dst := filepath.Join(t.TempDir(), "install.sh")
	if err := Download(client, "", "v1.2.0", "install.sh", dst); err != nil || read(dst) != "echo public" {
		t.Fatalf("public: %v %q", err, read(dst))
	}
	if *api != 0 {
		t.Errorf("the API was asked %d times", *api)
	}

	// The public address answers 504 (8-nov): the same file comes through the API.
	pub, api = github(t, http.StatusGatewayTimeout, true)
	dst = filepath.Join(t.TempDir(), "install.sh")
	if err := Download(client, "", "1.2.0", "install.sh", dst); err != nil || read(dst) != "echo api" {
		t.Fatalf("504 then API: %v %q", err, read(dst))
	}
	if *pub != 1 || *api != 1 {
		t.Errorf("public %d, API %d calls", *pub, *api)
	}

	// Both fail: it tries Tries times, writes nothing, and says what each answered.
	pub, _ = github(t, http.StatusGatewayTimeout, false)
	dst = filepath.Join(t.TempDir(), "install.sh")
	err := Download(client, "", "1.2.0", "install.sh", dst)
	if err == nil || !strings.Contains(err.Error(), "504") || !strings.Contains(err.Error(), "API") {
		t.Fatalf("both down: %v", err)
	}
	if int(*pub) != Tries {
		t.Errorf("%d tries, want %d", *pub, Tries)
	}
	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		t.Errorf("a failed download left a file: %v", err)
	}

	// A release that lacks the file says so.
	github(t, http.StatusNotFound, true)
	if err := Download(client, "", "1.2.0", "nothing.sh", dst); err == nil || !strings.Contains(err.Error(), "has no nothing.sh") {
		t.Errorf("missing file: %v", err)
	}
}

// With a base address (AB_RELEASE_URL: tests, the rehearsal) only that is asked, also
// as file://.
func TestDownloadFromBase(t *testing.T) {
	client := &http.Client{Timeout: 5 * time.Second}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "install.sh"), []byte("echo local"), 0o600); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(t.TempDir(), "install.sh")
	base := "file://" + filepath.ToSlash(dir)
	if !strings.HasPrefix(filepath.ToSlash(dir), "/") {
		base = "file:///" + filepath.ToSlash(dir) // Windows: file:///C:/…
	}
	if err := Download(client, base+"/", "", "install.sh", dst); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(dst); string(b) != "echo local" {
		t.Errorf("got %q", b)
	}
	old := RetryDelay
	RetryDelay = time.Millisecond
	defer func() { RetryDelay = old }()
	if err := Download(client, base, "", "missing.sh", dst+"2"); err == nil {
		t.Error("a missing file was downloaded")
	}
}
