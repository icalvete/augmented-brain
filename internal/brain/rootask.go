package brain

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/state"
)

// SaveRootException adds a temporary root exception, host until the date until, to
// root_ask in the configuration (config.NewRootAsk checks it: it has not ended and ends
// within config.RootAskMaxDays). It replaces any exception the same host had: one
// machine, one date. Until then the guard asks the user about each command as root
// there instead of denying it, and after it denies them again by itself.
//
// It loosens the guard, so the tool that calls it is always asked about first (the
// hook, trust.go); the user is also told through the guard's notice, out of the
// assistant's hands. When the lists were the ones the user had accepted, they stay
// accepted with this one change, which the user just confirmed: an exception on lists
// that are not sealed counts only once the user accepts them (guard.root.changed).
func (b *Brain) SaveRootException(host, until string, now time.Time) (string, error) {
	c, err := config.Load(b.ConfigDir)
	if err != nil {
		return "", err
	}
	entry, r, err := config.NewRootAsk(b.Language, host, until, now)
	if err != nil {
		return "", err
	}
	var out []string
	for _, e := range c.RootAsk {
		if old, err := config.ParseRootAsk(b.Language, e); err == nil && strings.EqualFold(old.Host, r.Host) {
			continue
		}
		out = append(out, e)
	}
	out = append(out, entry)
	if slices.Equal(out, c.RootAsk) {
		return fmt.Sprintf("nothing to change: root_ask already has %q", entry), nil
	}
	if err := b.writeRootAsk(c, out); err != nil {
		return "", err
	}
	_ = state.AddGuardNotice(b.ConfigDir, i18n.T(b.Language, "hook.guardnotice.rootadded", r.Host, r.Until.Format("2006-01-02 15:04")), now)
	return fmt.Sprintf("saved in root_ask: %q, for this whole machine's guard. On %s, until %s UTC, a command as root asks the user instead of being denied; after that it is denied again by itself.",
		entry, r.Host, r.Until.Format("2006-01-02 15:04")), nil
}

// RemoveRootException takes out of root_ask every exception for host, and any entry
// written exactly as host. Taking one out only makes the guard stricter, so it is not
// asked about.
func (b *Brain) RemoveRootException(host string, now time.Time) (string, error) {
	host = strings.TrimSpace(host)
	if host == "" {
		return "", fmt.Errorf("host is required")
	}
	c, err := config.Load(b.ConfigDir)
	if err != nil {
		return "", err
	}
	var out, gone []string
	for _, e := range c.RootAsk {
		r, err := config.ParseRootAsk(b.Language, e)
		if strings.TrimSpace(e) == host || err == nil && strings.EqualFold(r.Host, host) {
			gone = append(gone, e)
			continue
		}
		out = append(out, e)
	}
	if len(gone) == 0 {
		return fmt.Sprintf("nothing to change: root_ask has no exception for %s", host), nil
	}
	if err := b.writeRootAsk(c, out); err != nil {
		return "", err
	}
	_ = state.AddGuardNotice(b.ConfigDir, i18n.T(b.Language, "hook.guardnotice.rootremoved", host), now)
	return fmt.Sprintf("taken out of root_ask: %s. Root on %s is denied again.", strings.Join(gone, "; "), host), nil
}

// writeRootAsk writes root_ask as out, and seals the lists again when the user had
// accepted them just before and the only change is this one (nothing else wrote the
// file meanwhile).
func (b *Brain) writeRootAsk(c *config.Config, out []string) error {
	expected := ""
	if g, err := config.LoadGuard(b.ConfigDir); err == nil {
		if seal, err := state.ReadSeal(b.ConfigDir); err == nil && seal != "" && seal == c.GuardFingerprint(g) {
			plus := *c
			plus.RootAsk = out
			expected = plus.GuardFingerprint(g)
		}
	}
	if err := config.SetTopList(b.ConfigDir, "root_ask", out); err != nil {
		return err
	}
	rootWritten()
	resealAdded(b.ConfigDir, expected)
	return nil
}

// rootWritten runs between writing root_ask and sealing it; tests change it.
var rootWritten = func() {}
