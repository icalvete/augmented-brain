package telegram

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/icalvete/augmented-brain/internal/runlimit"
)

// Service is the keychain service every secret of this package is kept under.
const Service = "augmented-brain"

// Names of the secrets in the keychain. The session is the whole account: whoever
// holds it is logged in as the user.
const (
	NameAPIID   = "telegram-api-id"
	NameAPIHash = "telegram-api-hash"
	NameSession = "telegram-session"
)

// ErrNotFound means the keychain has no such secret.
var ErrNotFound = errors.New("not in the keychain")

// ErrNoKeychain means this computer has no keychain this package knows how to use.
var ErrNoKeychain = errors.New("no keychain on this computer")

// Secrets reads and writes the secrets of this package. Keychain is the real one;
// tests use a map.
type Secrets interface {
	Get(name string) (string, error)
	Set(name, value string) error
	Delete(name string) error
}

// Keychain keeps secrets in the system keychain: the Secret Service through secret-tool
// on Linux, the login keychain through security on macOS. Values never travel in a
// command line, where other users of the computer could read them: they go through the
// command's standard input.
type Keychain struct{}

const keychainTimeout = 10 * time.Second

// Supported says whether this computer has a keychain Keychain can use.
func Supported() bool {
	switch runtime.GOOS {
	case "linux", "freebsd", "openbsd":
		_, err := exec.LookPath("secret-tool")
		return err == nil
	case "darwin":
		_, err := exec.LookPath("security")
		return err == nil
	}
	return false
}

func run(stdin string, name string, arg ...string) (string, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), keychainTimeout)
	defer cancel()
	cmd := runlimit.Command(ctx, name, arg...)
	var out, errOut bytes.Buffer
	cmd.Stdin, cmd.Stdout, cmd.Stderr = strings.NewReader(stdin), &out, &errOut
	err := cmd.Run()
	if ctx.Err() != nil {
		return "", "", fmt.Errorf("the keychain did not answer in %s", keychainTimeout)
	}
	return out.String(), errOut.String(), err
}

func exitCode(err error) int {
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return exit.ExitCode()
	}
	return -1
}

// Get returns the secret name.
func (Keychain) Get(name string) (string, error) {
	switch runtime.GOOS {
	case "linux", "freebsd", "openbsd":
		out, stderr, err := run("", "secret-tool", "lookup", "service", Service, "name", name)
		if err != nil {
			if exitCode(err) == 1 && strings.TrimSpace(stderr) == "" {
				return "", ErrNotFound
			}
			return "", fmt.Errorf("secret-tool: %s", strings.TrimSpace(stderr+" "+err.Error()))
		}
		return strings.TrimRight(out, "\n"), nil
	case "darwin":
		out, stderr, err := run("", "security", "find-generic-password", "-s", Service, "-a", name, "-w")
		if err != nil {
			if exitCode(err) == 44 {
				return "", ErrNotFound
			}
			return "", fmt.Errorf("security: %s", strings.TrimSpace(stderr+" "+err.Error()))
		}
		return strings.TrimRight(out, "\n"), nil
	}
	return "", ErrNoKeychain
}

// Set stores the secret name, replacing what was there.
func (Keychain) Set(name, value string) error {
	if strings.ContainsAny(value, "\n\r") {
		return errors.New("a secret cannot hold a line break")
	}
	switch runtime.GOOS {
	case "linux", "freebsd", "openbsd":
		_, stderr, err := run(value, "secret-tool", "store", "--label=augmented brain: "+name,
			"service", Service, "name", name)
		if err != nil {
			return fmt.Errorf("secret-tool: %s", strings.TrimSpace(stderr+" "+err.Error()))
		}
		return nil
	case "darwin":
		// security -i reads its commands from standard input, so the value stays out of
		// the command line. Values here are digits, hex or base64: no quoting needed,
		// and anything else is refused rather than escaped.
		for _, r := range value {
			if r == '"' || r == '\\' || r == ' ' {
				return errors.New("this secret has characters the macOS keychain command cannot take")
			}
		}
		line := "add-generic-password -U -s " + Service + " -a " + name + " -w " + value + "\n"
		_, stderr, err := run(line, "security", "-i")
		if err != nil || strings.Contains(stderr, "error") {
			return fmt.Errorf("security: %s", strings.TrimSpace(stderr))
		}
		return nil
	}
	return ErrNoKeychain
}

// Delete takes the secret name out of the keychain. A secret that is not there is no
// error.
func (Keychain) Delete(name string) error {
	switch runtime.GOOS {
	case "linux", "freebsd", "openbsd":
		_, stderr, err := run("", "secret-tool", "clear", "service", Service, "name", name)
		if err != nil && strings.TrimSpace(stderr) != "" {
			return fmt.Errorf("secret-tool: %s", strings.TrimSpace(stderr))
		}
		return nil
	case "darwin":
		_, stderr, err := run("", "security", "delete-generic-password", "-s", Service, "-a", name)
		if err != nil && exitCode(err) != 44 {
			return fmt.Errorf("security: %s", strings.TrimSpace(stderr))
		}
		return nil
	}
	return ErrNoKeychain
}

// App is the user's own Telegram application, made at my.telegram.org.
type App struct {
	ID   int
	Hash string
}

// LoadApp reads the application from the keychain. ErrNotFound means it was never
// saved.
func LoadApp(s Secrets) (App, error) {
	id, err := s.Get(NameAPIID)
	if err != nil {
		return App{}, err
	}
	hash, err := s.Get(NameAPIHash)
	if err != nil {
		return App{}, err
	}
	return ParseApp(id, hash)
}

// ParseApp checks an api_id and an api_hash as my.telegram.org gives them.
func ParseApp(id, hash string) (App, error) {
	id, hash = strings.TrimSpace(id), strings.TrimSpace(hash)
	n, err := strconv.Atoi(id)
	if err != nil || n <= 0 {
		return App{}, errors.New("api_id is a number, as my.telegram.org shows it")
	}
	if len(hash) != 32 || strings.Trim(strings.ToLower(hash), "0123456789abcdef") != "" {
		return App{}, errors.New("api_hash is 32 letters and digits (0-9, a-f), as my.telegram.org shows it")
	}
	return App{ID: n, Hash: strings.ToLower(hash)}, nil
}

// SaveApp checks and stores the application in the keychain.
func SaveApp(s Secrets, id, hash string) (App, error) {
	app, err := ParseApp(id, hash)
	if err != nil {
		return App{}, err
	}
	if err := s.Set(NameAPIID, strconv.Itoa(app.ID)); err != nil {
		return App{}, err
	}
	if err := s.Set(NameAPIHash, app.Hash); err != nil {
		return App{}, err
	}
	return app, nil
}
