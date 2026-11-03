package panel

import (
	"errors"
	"net/http"
	"strings"

	"github.com/icalvete/augmented-brain/internal/brain"
	"github.com/icalvete/augmented-brain/internal/keyring"
)

// secretRow is one secret of the list: its name, what it is for, whether this
// computer's keychain has it (only when asked: asking a locked keychain makes the
// system ask for its password), and the tasks that name it.
type secretRow struct {
	brain.SecretName
	State string // "", "present", "missing", "unknown"
	How   string // when missing: the command that stores it, asking for its value
	Used  []taskLink
}

// secrets lists the secrets the user keeps, by name. Values are never read here: the
// check asks the keychain whether it has each one and throws the answer's text away.
func (s *Server) secrets(v *view, r *http.Request) (string, page, error) {
	check := r.FormValue("check") == "1"
	var rows []secretRow
	names, bad := brain.SecretNames(v.c.Dir)
	// The notes of every space, to find which tasks name each secret.
	type noteText struct {
		link taskLink
		text string
	}
	var notes []noteText
	for _, name := range v.c.SpaceNames() {
		b := v.b
		if name != v.b.Space.Name {
			var err error
			if b, err = brain.Open(v.c, v.c.Spaces[name]); err != nil {
				continue
			}
		}
		for _, n := range b.Mem.OfType("task") {
			h := n.Header
			if h.Status == "closed" {
				continue
			}
			notes = append(notes, noteText{taskLink{name, h.ID, h.Title, h.Status}, h.Summary + "\n" + h.Next + "\n" + n.Body})
		}
	}
	if check {
		v.unlock() // the keychain can take seconds: not with the notes locked
	}
	for _, sn := range names {
		row := secretRow{SecretName: sn}
		if check {
			row.State = map[keyring.State]string{keyring.Present: "present", keyring.Missing: "missing", keyring.Unknown: "unknown"}[keyring.Has(sn.Service, sn.Name)]
			if row.State == "missing" {
				row.How = keyring.HowToStore(sn.Service, sn.Name)
			}
		}
		for _, n := range notes {
			if namesKey(n.text, sn.Key()) {
				row.Used = append(row.Used, n.link)
			}
		}
		rows = append(rows, row)
	}
	p := v.frame("secrets", v.t("secrets"))
	p.D = struct {
		Rows    []secretRow
		Checked bool
		Bad     int
	}{rows, check, bad}
	return "secrets", p, nil
}

func (s *Server) secretSave(v *view, r *http.Request) (string, error) {
	if err := required(v, r, "service", "name"); err != nil {
		return "", err
	}
	_, err := v.b.SaveSecretName(brain.SecretName{Service: field(r, "service"), Name: field(r, "name"), What: field(r, "what")})
	switch {
	case errors.Is(err, brain.ErrSecretValue):
		return "", errors.New(v.t("err.secretvalue"))
	case errors.Is(err, brain.ErrSecretName):
		return "", errors.New(v.t("err.secretname"))
	}
	return "/secrets", err
}

func (s *Server) secretForget(v *view, r *http.Request) (string, error) {
	if field(r, "confirm") != "yes" {
		return "", errors.New(v.t("err.confirm"))
	}
	return "/secrets", v.b.ForgetSecretName(field(r, "service"), field(r, "name"))
}

// namesKey says whether text names the secret service/name as a whole: telegram/bot is
// not named by telegram/bot2.
func namesKey(text, key string) bool {
	for i := 0; ; {
		j := strings.Index(text[i:], key)
		if j < 0 {
			return false
		}
		end := i + j + len(key)
		// A dot right after ends a sentence, not the name.
		if end == len(text) || !isNamePart(rune(text[end])) || text[end] == '.' && (end+1 == len(text) || !isNamePart(rune(text[end+1]))) {
			if start := i + j; start == 0 || !isNamePart(rune(text[start-1])) {
				return true
			}
		}
		i = end
	}
}

func isNamePart(r rune) bool {
	return r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("._@+-/", r)
}
