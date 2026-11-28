// Package telegramprog finds, installs and runs augmented brain's Telegram program
// (augmented-brain-telegram). It is a program of its own because Telegram's library
// triples a program's size: only who turns Telegram on downloads it, from the same
// release as augmented-brain, checked against the release's checksums.txt, and puts it
// next to augmented-brain. install.sh keeps it up to date from then on.
//
// The program itself is in cmd/augmented-brain-telegram; this package never imports
// Telegram's library.
package telegramprog

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/icalvete/augmented-brain/internal/keyring"
	"github.com/icalvete/augmented-brain/internal/runlimit"
	"github.com/icalvete/augmented-brain/internal/update"
	"github.com/icalvete/augmented-brain/internal/version"
)

// Name is the program's name.
const Name = "augmented-brain-telegram"

// Supported says whether this system can run it: it needs a keychain augmented brain
// can use, Linux's or macOS's.
func Supported() bool {
	switch runtime.GOOS {
	case "linux", "darwin":
		return true
	}
	return false
}

// selfDir is the folder of the running augmented-brain. A variable, for tests.
var selfDir = func() (string, error) {
	self, err := os.Executable()
	if err != nil {
		return "", err
	}
	if self, err = filepath.EvalSymlinks(self); err != nil {
		return "", err
	}
	return filepath.Dir(self), nil
}

// Path is where the program is, or "" when it is not installed: next to augmented-brain,
// or else in PATH (a build of one's own).
func Path() string {
	if dir, err := selfDir(); err == nil {
		p := filepath.Join(dir, Name)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	if p, err := exec.LookPath(Name); err == nil {
		return p
	}
	return ""
}

// KeychainService and KeychainSession name the Telegram session in the keychain: the
// same as internal/telegram's Service and NameSession, which this package cannot
// import (it would bring Telegram's library into augmented-brain).
const (
	KeychainService = "augmented-brain"
	KeychainSession = "telegram-session"
)

// SessionSaved says whether the keychain holds a Telegram session, without reading it.
// Without the program, it tells "connected, the program is missing" from "never
// connected". A variable, for tests.
var SessionSaved = func() bool {
	return keyring.Has(KeychainService, KeychainSession) == keyring.Present
}

// Archive is the name of the program's archive in a release. Not .tar.gz: installers
// that pick augmented-brain's own archive by "_<os>_<arch>.tar.gz" must never take it.
func Archive(ver, goos, goarch string) string {
	return Name + "_" + strings.TrimPrefix(ver, "v") + "_" + goos + "_" + goarch + ".tgz"
}

// ErrDevBuild means this augmented-brain was not built for a release: there is nothing
// to download for it.
var ErrDevBuild = errors.New("this augmented-brain is a development build: build the Telegram program next to it with go build -o <its folder>/augmented-brain-telegram ./cmd/augmented-brain-telegram")

// Base, when not "", is where release files are fetched from instead of GitHub (tests,
// the rehearsal): <Base>/<file>.
var Base = os.Getenv("AB_TELEGRAM_FROM")

// Install downloads the program of this augmented-brain's version, checks it against
// the release's checksums.txt, makes sure it starts, and puts it next to augmented-brain.
// It returns its path.
func Install() (string, error) {
	if !Supported() {
		return "", fmt.Errorf("no Telegram on %s yet", runtime.GOOS)
	}
	ver := version.Version
	if strings.Contains(ver, "-dev") {
		return "", ErrDevBuild
	}
	dir, err := selfDir()
	if err != nil {
		return "", err
	}
	tmp, err := os.MkdirTemp("", "ab-telegram-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)
	client := &http.Client{Timeout: 5 * time.Minute}
	sums := filepath.Join(tmp, "checksums.txt")
	if err := update.Download(client, Base, ver, "checksums.txt", sums); err != nil {
		return "", fmt.Errorf("checksums.txt: %w", err)
	}
	name := Archive(ver, runtime.GOOS, runtime.GOARCH)
	want, err := sumOf(sums, name)
	if err != nil {
		return "", err
	}
	arch := filepath.Join(tmp, name)
	if err := update.Download(client, Base, ver, name, arch); err != nil {
		return "", fmt.Errorf("%s: %w", name, err)
	}
	data, err := os.ReadFile(arch)
	if err != nil {
		return "", err
	}
	if got := sha256.Sum256(data); hex.EncodeToString(got[:]) != want {
		return "", errors.New("the download does not match its checksum: not installed")
	}
	prog, err := extract(data, Name)
	if err != nil {
		return "", err
	}
	dst := filepath.Join(dir, Name)
	tmpDst := dst + ".new"
	if err := os.WriteFile(tmpDst, prog, 0o755); err != nil {
		return "", err
	}
	if out, err := runlimit.Output(30*time.Second, tmpDst, "version"); err != nil || strings.TrimSpace(string(out)) == "" {
		_ = os.Remove(tmpDst)
		return "", fmt.Errorf("the Telegram program does not start on this computer: not installed (%v)", err)
	}
	if err := os.Rename(tmpDst, dst); err != nil {
		_ = os.Remove(tmpDst)
		return "", err
	}
	return dst, nil
}

// sumOf is the SHA-256 checksums.txt gives name.
func sumOf(path, name string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) == 2 && strings.TrimPrefix(fields[1], "*") == name {
			return fields[0], nil
		}
	}
	return "", fmt.Errorf("this release has no %s", name)
}

// extract takes the file called name out of a .tgz, wherever it is inside.
func extract(data []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return nil, fmt.Errorf("the archive has no %s", name)
		}
		if err != nil {
			return nil, err
		}
		if h.Typeflag == tar.TypeReg && filepath.Base(h.Name) == name {
			return io.ReadAll(io.LimitReader(tr, 128<<20))
		}
	}
}

// Wait bounds one call of the program that is not a login.
const Wait = 90 * time.Second

// ErrNotConnected is the program's "not connected" (or "no application yet").
var ErrNotConnected = errors.New("not connected to Telegram")

// Call runs the program at path with args and --json, stdin on its standard input, and
// decodes its one JSON line into v.
func Call(ctx context.Context, path string, args []string, stdin string, v any) error {
	ctx, cancel := context.WithTimeout(ctx, Wait)
	defer cancel()
	cmd := runlimit.Command(ctx, path, append(args, "--json")...)
	var out, errOut bytes.Buffer
	cmd.Stdin, cmd.Stdout, cmd.Stderr = strings.NewReader(stdin), &out, &errOut
	runErr := cmd.Run()
	var e struct {
		Error string `json:"error"`
		Code  string `json:"code"`
	}
	if json.Unmarshal(out.Bytes(), &e) == nil && e.Error != "" {
		if e.Code == "not_connected" || e.Code == "no_app" {
			return ErrNotConnected
		}
		return errors.New(e.Error)
	}
	if runErr != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("no answer from Telegram in %s", Wait)
		}
		return fmt.Errorf("%s: %v %s", Name, runErr, strings.TrimSpace(errOut.String()))
	}
	return json.Unmarshal(out.Bytes(), v)
}
