package brain

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/BurntSushi/toml"
	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/guard"
	"github.com/icalvete/augmented-brain/internal/paths"
)

// SecretName is a secret the user keeps in the system keychain: where it is, and what
// it is for. Never its value: augmented brain keeps names only, so a person (and a new
// machine) knows which secrets there are.
type SecretName struct {
	Service, Name, What string
}

// Key is how the secret is written in notes and commands: service/name.
func (s SecretName) Key() string { return s.Service + "/" + s.Name }

// secretPart is a service or a name: one word, never starting with "-", so the
// keychain's command can never take it for one of its options.
var secretPart = regexp.MustCompile(`^[A-Za-z0-9._@+][A-Za-z0-9._@+-]{0,99}$`)

// ParseSecretName reads one entry of the configuration's secrets list,
// "service/name: what it is for".
func ParseSecretName(entry string) (SecretName, bool) {
	key, what, _ := strings.Cut(entry, ":")
	service, name, ok := strings.Cut(strings.TrimSpace(key), "/")
	if !ok || !secretPart.MatchString(service) || !secretPart.MatchString(name) {
		return SecretName{}, false
	}
	return SecretName{service, name, strings.TrimSpace(what)}, true
}

// SecretList returns the entries of the secrets list, as written: those of its file,
// then any 0.6.0 left in config.toml that the file does not have yet.
func SecretList(cfgDir string) []string {
	var f struct {
		Secrets []string `toml:"secrets"`
	}
	_, _ = toml.DecodeFile(filepath.Join(cfgDir, config.SecretsFile), &f)
	out := f.Secrets
	if c, err := config.Load(cfgDir); err == nil {
		for _, e := range c.Secrets {
			if !slices.Contains(out, e) {
				out = append(out, e)
			}
		}
	}
	return out
}

// SecretNames returns the secrets the list names, in its order, and how many of its
// entries cannot be read.
func SecretNames(cfgDir string) ([]SecretName, int) {
	var out []SecretName
	list := SecretList(cfgDir)
	for _, e := range list {
		if s, ok := ParseSecretName(e); ok {
			out = append(out, s)
		}
	}
	return out, len(list) - len(out)
}

// writeSecrets writes the list to its file. Every entry is plain text (plainLine), so
// quoting it is enough for TOML.
func writeSecrets(cfgDir string, list []string) error {
	var b strings.Builder
	b.WriteString("# The names of the secrets you keep in the system keychain, and what each is for.\n")
	b.WriteString("# Never their values. augmented brain writes this file; the panel shows it.\n")
	b.WriteString("secrets = [\n")
	for _, e := range list {
		b.WriteString("  \"" + e + "\",\n")
	}
	b.WriteString("]\n")
	return paths.WriteAtomic(filepath.Join(cfgDir, config.SecretsFile), []byte(b.String()), 0o600)
}

// SaveSecretList writes a whole list, as a restore brings it.
func SaveSecretList(cfgDir string, list []string) error {
	if len(list) == 0 {
		return nil
	}
	return writeSecrets(cfgDir, list)
}

// MoveSecrets takes the list 0.6.0 kept in config.toml to its own file, so programs
// before 0.6.0, which refuse a setting they do not know, can read the configuration.
// Safe to repeat; nothing to do when config.toml has none.
func MoveSecrets(cfgDir string) error {
	c, err := config.Load(cfgDir)
	if err != nil || len(c.Secrets) == 0 {
		return err
	}
	if err := writeSecrets(cfgDir, CleanSecretNames(SecretList(cfgDir))); err != nil {
		return err
	}
	return config.RemoveTopKey(cfgDir, "secrets")
}

// ErrSecretName is a service or a name that is not one plain word, or a description
// that is not one plain line.
var ErrSecretName = errors.New("service and name are single words (letters, digits, . _ @ + -, not starting with -), and what it is for one line of plain text")

// ErrSecretValue is a name or a description that looks like a secret itself.
var ErrSecretValue = errors.New("that looks like a secret's value: only its name and what it is for are kept")

// SaveSecretName adds a secret's name to the configuration, or changes what it is for.
// It refuses anything that looks like a value.
func (b *Brain) SaveSecretName(in SecretName) (string, error) {
	in.Service, in.Name, in.What = strings.TrimSpace(in.Service), strings.TrimSpace(in.Name), strings.TrimSpace(in.What)
	if !secretPart.MatchString(in.Service) || !secretPart.MatchString(in.Name) || !plainLine(in.What) {
		return "", ErrSecretName
	}
	if kind := guard.FindSecret(in.Key() + " " + in.What); kind != "" {
		return "", ErrSecretValue
	}
	if err := MoveSecrets(b.ConfigDir); err != nil {
		return "", err
	}
	entry := in.Key()
	if in.What != "" {
		entry += ": " + in.What
	}
	var list []string
	found := false
	for _, e := range CleanSecretNames(SecretList(b.ConfigDir)) {
		if s, ok := ParseSecretName(e); ok && s.Key() == in.Key() {
			if !found {
				list = append(list, entry)
			}
			found = true
			continue
		}
		list = append(list, e)
	}
	if !found {
		list = append(list, entry)
	}
	if err := writeSecrets(b.ConfigDir, list); err != nil {
		return "", err
	}
	return "secrets: " + entry, nil
}

// ForgetSecretName takes a secret's name out of the list. The secret itself, in the
// keychain, is not touched.
func (b *Brain) ForgetSecretName(service, name string) error {
	if err := MoveSecrets(b.ConfigDir); err != nil {
		return err
	}
	key := service + "/" + name
	all := SecretList(b.ConfigDir)
	var list []string
	for _, e := range all {
		if s, ok := ParseSecretName(e); ok && s.Key() == key {
			continue
		}
		list = append(list, e)
	}
	if len(list) == len(all) {
		return fmt.Errorf("there is no secret %s in the list", key)
	}
	return writeSecrets(b.ConfigDir, list)
}

// plainLine says whether text is one line of printable text, which the configuration
// file can always hold: no control characters, no quotes or backslashes, valid UTF-8.
func plainLine(text string) bool {
	if !utf8.ValidString(text) || strings.ContainsAny(text, "\"\\") {
		return false
	}
	for _, r := range text {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	return true
}

// CleanSecretNames keeps the entries of a secrets list that can be read and written
// back: what a restore takes from a backup.
func CleanSecretNames(list []string) []string {
	var out []string
	for _, e := range list {
		if s, ok := ParseSecretName(e); ok && plainLine(s.What) {
			out = append(out, e)
		}
	}
	return out
}
