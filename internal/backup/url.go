package backup

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/icalvete/augmented-brain/internal/guard"
)

// checkURL refuses what cannot be a repository's address (nothing, or something git
// would read as one of its options) and an address that carries a password or a token.
// The address is kept in config.toml and inside the backup itself, so a password in it
// would travel with every copy; git keeps the login better (gh auth setup-git, ssh, or
// its credential manager).
func checkURL(raw string) error {
	switch {
	case strings.TrimSpace(raw) == "":
		return errors.New("the backup needs the URL of a private repository")
	case strings.HasPrefix(strings.TrimSpace(raw), "-"):
		return fmt.Errorf("%q is not the address of a repository", raw)
	case HasPassword(raw):
		return fmt.Errorf("the address %s holds a password or a token, which would be kept in the configuration and in the backup itself. "+
			"Give the address without it, and let git keep the login: for GitHub, `gh auth setup-git`; or an ssh address (git@host:user/repo.git)",
			WithoutPassword(raw))
	}
	return nil
}

// HasPassword says whether a repository's address carries a password (user:password@)
// or something that looks like a token anywhere in it.
func HasPassword(raw string) bool {
	if guard.FindSecret(raw) != "" {
		return true
	}
	if !strings.Contains(raw, "://") {
		return false
	}
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		// What cannot be read may hold one: only an @ before the path says it might.
		return strings.Contains(beforePath(raw), "@")
	}
	_, set := u.User.Password()
	return set
}

// WithoutPassword is the address with its password, or a user name that is a token,
// taken out: what can be shown and kept. An address it cannot read loses everything
// before its @.
func WithoutPassword(raw string) string {
	raw = strings.TrimSpace(raw)
	if !strings.Contains(raw, "://") {
		// The scp form, user@host:path: only the user could be a token.
		if at := strings.Index(raw, "@"); at > 0 && guard.FindSecret(raw[:at]) != "" {
			return raw[at+1:]
		}
		return raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		scheme, rest, _ := strings.Cut(raw, "://")
		if at := strings.LastIndex(beforePath(rest), "@"); at >= 0 {
			rest = rest[at+1:]
		}
		return scheme + "://" + rest
	}
	if u.User != nil {
		name := u.User.Username()
		if guard.FindSecret(name) != "" {
			u.User = nil
		} else {
			u.User = url.User(name)
		}
	}
	return u.String()
}

// beforePath is the part of an address before its path: scheme, user and host.
func beforePath(raw string) string {
	_, rest, ok := strings.Cut(raw, "://")
	if !ok {
		rest = raw
	}
	if i := strings.Index(rest, "/"); i >= 0 {
		return rest[:i]
	}
	return rest
}
