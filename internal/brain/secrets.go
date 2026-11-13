package brain

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"unicode"
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
// then any 0.6.0 left in config.toml that the file does not have yet. A file that
// cannot be read gives what config.toml has; readSecretList says why.
func SecretList(cfgDir string) []string {
	out, _ := readSecretList(cfgDir)
	return out
}

// readSecretList is SecretList with the reason its file could not be read, if so.
// What writes the list refuses to go on then: it would write over the names it could
// not read.
func readSecretList(cfgDir string) ([]string, error) {
	var f struct {
		Secrets []string `toml:"secrets"`
	}
	path := filepath.Join(cfgDir, config.SecretsFile)
	_, ferr := toml.DecodeFile(path, &f)
	if errors.Is(ferr, os.ErrNotExist) {
		ferr = nil
	} else if ferr != nil {
		ferr = fmt.Errorf("%s cannot be read, so it is not written over; fix it or remove the line that is wrong: %w", path, ferr)
	}
	out := f.Secrets
	if c, err := config.Load(cfgDir); err == nil {
		for _, e := range c.Secrets {
			if !slices.Contains(out, e) {
				out = append(out, e)
			}
		}
	}
	return out, ferr
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

// tomlString writes text as a TOML basic string: quotes, backslashes and control
// characters escaped, so an entry written by hand with any of them reads back the same.
func tomlString(text string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range text {
		switch {
		case r == '"' || r == '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		case r < 0x20 || r == 0x7f:
			fmt.Fprintf(&b, "\\u%04X", r)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// writeSecrets writes the list to its file, each entry escaped as TOML wants it.
func writeSecrets(cfgDir string, list []string) error {
	var b strings.Builder
	b.WriteString("# The names of the secrets you keep in the system keychain, and what each is for.\n")
	b.WriteString("# Never their values. augmented brain writes this file; the panel shows it.\n")
	b.WriteString("secrets = [\n")
	for _, e := range list {
		b.WriteString("  " + tomlString(e) + ",\n")
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
	list, err := readSecretList(cfgDir)
	if err != nil {
		return err
	}
	if err := writeSecrets(cfgDir, CleanSecretNames(list)); err != nil {
		return err
	}
	return config.RemoveTopKey(cfgDir, "secrets")
}

// ErrSecretName is a service or a name that is not one plain word, or a description
// that is not one plain line.
var ErrSecretName = errors.New("service and name are single words (letters, digits, . _ @ + -, not starting with -), and what it is for one line of plain text, up to 160 characters")

// secretWhatMax is how long what a secret is for may be: a few words, never a value
// pasted with them.
const secretWhatMax = 160

// ErrSecretValue is a name or a description that looks like a secret itself.
var ErrSecretValue = errors.New("that looks like a secret's value: only its name and what it is for are kept")

// SaveSecretName adds a secret's name to the configuration, or changes what it is for.
// It refuses anything that looks like a value.
func (b *Brain) SaveSecretName(in SecretName) (string, error) {
	in.Service, in.Name, in.What = strings.TrimSpace(in.Service), strings.TrimSpace(in.Name), strings.TrimSpace(in.What)
	if !secretPart.MatchString(in.Service) || !secretPart.MatchString(in.Name) || !plainLine(in.What) ||
		utf8.RuneCountInString(in.What) > secretWhatMax {
		return "", ErrSecretName
	}
	if kind := guard.FindSecret(in.Key() + " " + in.What); kind != "" || looksLikeValue(in.Service) ||
		looksLikeValue(in.Name) || looksLikeValue(in.What) {
		return "", ErrSecretValue
	}
	if err := MoveSecrets(b.ConfigDir); err != nil {
		return "", err
	}
	entry := in.Key()
	if in.What != "" {
		entry += ": " + in.What
	}
	all, err := readSecretList(b.ConfigDir)
	if err != nil {
		return "", err
	}
	var list []string
	found := false
	for _, e := range all {
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
	all, err := readSecretList(b.ConfigDir)
	if err != nil {
		return err
	}
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

// What gives a value away where a name or a few words should be. FindSecret knows the
// tokens of each provider by their prefix; these catch the rest: a value said with its
// label, one inside an address, a JWT, a PIN.
var (
	// labelled is "password=…", "token: …", "the password is …", in English or Spanish.
	labelled = regexp.MustCompile(`(?i)\b(?:pass(?:word|wd|phrase)?|pwd|contraseña|clave|secret[oa]?|token|api[_-]?key|pin|cvv|cvc)\s*(=|:|\bis\b|\bes\b)\s*([^\s,;]+)`)
	// inURL is a password in an address: scheme://user:password@host.
	inURL = regexp.MustCompile(`://[^/\s:@]+:[^/\s@]+@`)
	// jwt is three base64url blocks, the first a JSON header: "{\"" is always "eyJ".
	jwt = regexp.MustCompile(`\beyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.`)
	// pinCode is a card or phone code with its number.
	pinCode = regexp.MustCompile(`(?i)\b(?:pin|cvv|cvc)\b\D{0,3}\d{3,}`)
	// tokenWord is a long run of the characters keys are made of.
	tokenWord = regexp.MustCompile(`[A-Za-z0-9+/=_-]{20,}`)
	// nameWord is a long word that is still a name: pieces no longer than a word, each
	// in one case, such as "github-actions-deploy-2024" or "S3_BACKUP_BUCKET_KEY".
	nameWord = regexp.MustCompile(`^(?:[a-z0-9]{1,12}|[A-Z0-9]{1,12})(?:[-_.](?:[a-z0-9]{1,12}|[A-Z0-9]{1,12}))+$`)
)

// looksLikeValue says whether text holds what looks like a secret's value rather than
// a name or a description.
func looksLikeValue(text string) bool {
	if inURL.MatchString(text) || jwt.MatchString(text) || pinCode.MatchString(text) {
		return true
	}
	for _, m := range labelled.FindAllStringSubmatch(text, -1) {
		// "password=x" gives a value whatever it is; "the token is for the bot" or
		// "API key: deploys" does not, so after "is" or ":" only a word with digits or
		// signs counts.
		if m[1] == "=" || strings.IndexFunc(strings.TrimRight(m[2], ".!?)"), func(r rune) bool {
			return !unicode.IsLetter(r)
		}) >= 0 {
			return true
		}
	}
	for _, w := range tokenWord.FindAllString(text, -1) {
		if strings.ContainsAny(w, "0123456789") && strings.IndexFunc(w, unicode.IsLetter) >= 0 && !nameWord.MatchString(w) {
			return true
		}
	}
	return false
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
