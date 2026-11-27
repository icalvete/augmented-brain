package telegramprog

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/update"
	"github.com/icalvete/augmented-brain/internal/version"
)

func tgz(t *testing.T, name string, body []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, f := range []struct {
		name string
		body []byte
	}{{"x/LICENSE", []byte("MIT")}, {"x/" + name, body}} {
		if err := tw.WriteHeader(&tar.Header{Name: f.name, Mode: 0o755, Size: int64(len(f.body)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(f.body); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// Install takes the program of this version from the release, checks its checksum,
// and puts it next to augmented-brain; a wrong checksum installs nothing.
func TestInstall(t *testing.T) {
	if !Supported() {
		t.Skip("Telegram is not supported here")
	}
	release, self := t.TempDir(), t.TempDir()
	oldDir, oldVer, oldBase, oldTries := selfDir, version.Version, Base, update.Tries
	t.Cleanup(func() { selfDir, version.Version, Base, update.Tries = oldDir, oldVer, oldBase, oldTries })
	selfDir = func() (string, error) { return self, nil }
	version.Version = "1.2.3"
	Base = "file://" + release
	update.Tries = 1

	name := Archive("1.2.3", runtime.GOOS, runtime.GOARCH)
	if !strings.HasSuffix(name, ".tgz") || strings.HasSuffix(name, ".tar.gz") {
		t.Fatalf("archive %s could be taken for augmented-brain's own", name)
	}
	data := tgz(t, Name, []byte("#!/bin/sh\necho 1.2.3\n"))
	write := func(sum string) {
		if err := os.WriteFile(filepath.Join(release, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(release, "checksums.txt"), []byte(sum+"  "+name+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write(strings.Repeat("0", 64))
	if _, err := Install(); err == nil || !strings.Contains(err.Error(), "checksum") {
		t.Fatalf("wrong checksum: %v", err)
	}
	if _, err := os.Stat(filepath.Join(self, Name)); err == nil {
		t.Fatal("installed despite the checksum")
	}

	sum := sha256.Sum256(data)
	write(hex.EncodeToString(sum[:]))
	p, err := Install()
	if err != nil {
		t.Fatal(err)
	}
	if p != filepath.Join(self, Name) || Path() != p {
		t.Errorf("installed at %s, Path says %s", p, Path())
	}

	version.Version = "0.0.1-dev"
	if _, err := Install(); !errors.Is(err, ErrDevBuild) {
		t.Errorf("dev build: %v", err)
	}
}
