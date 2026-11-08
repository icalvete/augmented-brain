package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// fakeRelease is a release of a program that only says its version: the archive for
// this system and its checksums.txt.
func fakeRelease(t *testing.T) (archive string, files map[string][]byte) {
	t.Helper()
	dir := fmt.Sprintf("augmented-brain_9.9.9_%s_%s", runtime.GOOS, runtime.GOARCH)
	prog := []byte("#!/bin/sh\necho 9.9.9\n")
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	tw.WriteHeader(&tar.Header{Name: dir + "/", Typeflag: tar.TypeDir, Mode: 0o755})
	tw.WriteHeader(&tar.Header{Name: dir + "/augmented-brain", Mode: 0o755, Size: int64(len(prog))})
	tw.Write(prog)
	tw.Close()
	gz.Close()
	archive = dir + ".tar.gz"
	sum := sha256.Sum256(buf.Bytes())
	return archive, map[string][]byte{
		archive:         buf.Bytes(),
		"checksums.txt": []byte(fmt.Sprintf("%x  %s\n", sum, archive)),
	}
}

// runInstall runs install.sh, only its download and install (AB_NO_INIT), against a
// server that stands in for GitHub: the public address under /rel, the API under /api.
// public says whether the public address serves the files (otherwise it answers 504);
// api, whether the API does.
func runInstall(t *testing.T, public, api bool) (out string, err error, bin string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("install.sh is the installer for Linux and macOS")
	}
	srv := fakeGitHub(t, public, api)
	home := t.TempDir()
	bin = filepath.Join(home, "bin")
	cmd := exec.Command("sh", filepath.Join("..", "..", "install.sh"))
	cmd.Env = append(os.Environ(), "HOME="+home, "SHELL=/bin/sh", "AB_NO_INIT=1", "AB_NO_DESKTOP=1",
		"AB_LANGUAGE=en", "AB_INSTALL_DIR="+bin, "AB_RELEASE_URL="+srv.URL+"/rel",
		"AB_API_URL="+srv.URL+"/api", "AB_RETRY_DELAY=0")
	o, err := cmd.CombinedOutput()
	return string(o), err, bin
}

// fakeGitHub stands in for GitHub: the public address of the release's files under
// /rel, the API under /api. public says whether the public address serves the files
// (otherwise it answers 504); api, whether the API does.
func fakeGitHub(t *testing.T, public, api bool) *httptest.Server {
	t.Helper()
	archive, files := fakeRelease(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := filepath.Base(r.URL.Path)
		switch {
		case strings.HasPrefix(r.URL.Path, "/rel/") && public && files[name] != nil:
			w.Write(files[name])
		case strings.HasPrefix(r.URL.Path, "/rel/"):
			http.Error(w, "gateway timeout", http.StatusGatewayTimeout)
		case !api:
			http.Error(w, "unavailable", http.StatusServiceUnavailable)
		case r.URL.Path == "/api/releases/latest":
			fmt.Fprintf(w, `{"tag_name": "v9.9.9", "assets": [
  {"url": "http://%[1]s/api/releases/assets/1", "id": 1, "name": "checksums.txt", "uploader": {"url": "http://%[1]s/users/x"}},
  {"url": "http://%[1]s/api/releases/assets/2", "id": 2, "name": "%[2]s"}]}`, r.Host, archive)
		case strings.HasPrefix(r.URL.Path, "/api/releases/assets/") && r.Header.Get("Accept") == "application/octet-stream":
			if name == "1" {
				w.Write(files["checksums.txt"])
			} else {
				w.Write(files[archive])
			}
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// install.ps1 with both ways down: the message says nothing was installed, nothing is,
// and with AB_EXIT_CODE=1 (how `update` runs it) it ends with an error code. Only the
// failure: an install that works would add a folder to the user's PATH for good.
func TestInstallPs1DownloadFails(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("install.ps1 is the installer for Windows")
	}
	srv := fakeGitHub(t, false, false)
	bin := filepath.Join(t.TempDir(), "bin")
	for _, exitCode := range []string{"", "1"} {
		cmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", filepath.Join("..", "..", "install.ps1"))
		cmd.Env = append(os.Environ(), "AB_NO_INIT=1", "AB_NO_DESKTOP=1", "AB_LANGUAGE=en",
			"AB_INSTALL_DIR="+bin, "AB_RELEASE_URL="+srv.URL+"/rel", "AB_API_URL="+srv.URL+"/api",
			"AB_RETRY_DELAY=0", "AB_EXIT_CODE="+exitCode)
		o, err := cmd.CombinedOutput()
		if !strings.Contains(string(o), "cannot download checksums.txt") || !strings.Contains(string(o), "nothing was installed") {
			t.Errorf("AB_EXIT_CODE=%q: the message does not say what happened:\n%s", exitCode, o)
		}
		if (err != nil) != (exitCode == "1") {
			t.Errorf("AB_EXIT_CODE=%q: exit %v", exitCode, err)
		}
	}
	if _, err := os.Stat(filepath.Join(bin, "augmented-brain.exe")); !os.IsNotExist(err) {
		t.Errorf("something was installed: %v", err)
	}
}

// The public address answers 504 (as on 8-nov): the installer gets the same files
// through GitHub's API and installs.
func TestInstallerFallsBackToAPI(t *testing.T) {
	out, err, bin := runInstall(t, false, true)
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if v, err := exec.Command(filepath.Join(bin, "augmented-brain")).Output(); err != nil || strings.TrimSpace(string(v)) != "9.9.9" {
		t.Errorf("not installed: %q %v\n%s", v, err, out)
	}
}

// Both ways down: the installer stops with an error that says what happened and that
// nothing was installed, and nothing is.
func TestInstallerDownloadFails(t *testing.T) {
	out, err, bin := runInstall(t, false, false)
	if err == nil {
		t.Fatalf("a failed download was taken for an install:\n%s", out)
	}
	if !strings.Contains(out, "cannot download checksums.txt") || !strings.Contains(out, "nothing was installed") {
		t.Errorf("the message does not say what happened:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(bin, "augmented-brain")); !os.IsNotExist(err) {
		t.Errorf("something was installed: %v", err)
	}
}

// The public address works: nothing else is asked.
func TestInstallerPublic(t *testing.T) {
	out, err, bin := runInstall(t, true, false)
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(bin, "augmented-brain")); err != nil {
		t.Errorf("not installed: %v\n%s", err, out)
	}
}
