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
	return fakeReleaseOf(t, []byte("#!/bin/sh\necho 9.9.9\n"))
}

// fakeReleaseOf is a release whose program is prog.
func fakeReleaseOf(t *testing.T, prog []byte) (archive string, files map[string][]byte) {
	t.Helper()
	dir := fmt.Sprintf("augmented-brain_9.9.9_%s_%s", runtime.GOOS, runtime.GOARCH)
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

// installSh runs install.sh, only its download and install (AB_NO_INIT), from a release
// in a folder (file://) whose program is prog, into bin; extra goes into its
// environment, after the rest.
func installSh(t *testing.T, prog []byte, bin string, extra ...string) (string, error) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("install.sh is the installer for Linux and macOS")
	}
	_, files := fakeReleaseOf(t, prog)
	rel := t.TempDir()
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(rel, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cmd := exec.Command("sh", filepath.Join("..", "..", "install.sh"))
	cmd.Env = append(os.Environ(), "HOME="+t.TempDir(), "SHELL=/bin/sh", "AB_NO_INIT=1", "AB_NO_DESKTOP=1",
		"AB_LANGUAGE=en", "AB_INSTALL_DIR="+bin, "AB_RELEASE_URL=file://"+rel, "AB_RETRY_DELAY=0",
		"AB_ALLOW_ROOT=")
	cmd.Env = append(cmd.Env, extra...)
	o, err := cmd.CombinedOutput()
	return string(o), err
}

// A new program that does not start never replaces the one that works: the installer
// ends with an error that says so, and the old program is there and answers.
func TestInstallerKeepsWorkingProgram(t *testing.T) {
	bin := t.TempDir()
	old := filepath.Join(bin, "augmented-brain")
	if err := os.WriteFile(old, []byte("#!/bin/sh\necho 1.0.0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	out, err := installSh(t, []byte("#!/bin/sh\nexit 1\n"), bin)
	if err == nil || !strings.Contains(out, "does not start") || !strings.Contains(out, "still there") {
		t.Fatalf("a program that does not start was installed: %v\n%s", err, out)
	}
	if v, err := exec.Command(old).Output(); err != nil || strings.TrimSpace(string(v)) != "1.0.0" {
		t.Errorf("the old program is gone: %q %v", v, err)
	}
	for _, left := range []string{"augmented-brain.new", "augmented-brain.old"} {
		if _, err := os.Stat(filepath.Join(bin, left)); !os.IsNotExist(err) {
			t.Errorf("%s left behind: %v", left, err)
		}
	}

	// One that works replaces it, and nothing is left beside it.
	if out, err := installSh(t, []byte("#!/bin/sh\necho 9.9.9\n"), bin); err != nil || !strings.Contains(out, "9.9.9 installed") {
		t.Fatalf("%v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(bin, "augmented-brain.old")); !os.IsNotExist(err) {
		t.Errorf("the old program left behind: %v", err)
	}
}

// As root the installer stops before downloading anything, unless AB_ALLOW_ROOT=1.
func TestInstallerRefusesRoot(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("install.sh is the installer for Linux and macOS")
	}
	// An id that says root, first in the PATH: nothing here runs as root.
	fake := t.TempDir()
	if err := os.WriteFile(filepath.Join(fake, "id"), []byte("#!/bin/sh\necho 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	path := "PATH=" + fake + string(os.PathListSeparator) + os.Getenv("PATH")
	bin := filepath.Join(t.TempDir(), "bin")
	out, err := installSh(t, []byte("#!/bin/sh\necho 9.9.9\n"), bin, path)
	if err == nil || !strings.Contains(out, "as root") || !strings.Contains(out, "Nothing was installed") {
		t.Fatalf("it ran as root: %v\n%s", err, out)
	}
	if _, err := os.Stat(bin); !os.IsNotExist(err) {
		t.Errorf("something was installed: %v", err)
	}
	if out, err := installSh(t, []byte("#!/bin/sh\necho 9.9.9\n"), bin, path, "AB_ALLOW_ROOT=1"); err != nil {
		t.Errorf("AB_ALLOW_ROOT=1: %v\n%s", err, out)
	}
}
