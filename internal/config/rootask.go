package config

import (
	"errors"
	"strings"
	"time"

	"github.com/icalvete/augmented-brain/internal/i18n"
)

// RootAsk is a temporary exception for root on one machine, written in root_ask as
// "<host> until <date>": until then, a command that would run as root there asks the
// user instead of being denied. The date is a day (the exception lasts to its end, in
// UTC) or a time, such as 2026-10-31T23:59Z.
type RootAsk struct {
	Host  string
	Until time.Time
}

// ParseRootAsk reads one entry of root_ask; lang is the language of what it says when
// it cannot.
func ParseRootAsk(lang, entry string) (RootAsk, error) {
	host, date, ok := strings.Cut(strings.TrimSpace(entry), " until ")
	host, date = strings.TrimSpace(host), strings.TrimSpace(date)
	if !ok || host == "" || strings.ContainsAny(host, " \t") {
		return RootAsk{}, errors.New(i18n.T(lang, "config.rootask.form", entry))
	}
	if d, err := time.Parse("2006-01-02", date); err == nil {
		return RootAsk{host, d.Add(24*time.Hour - time.Second)}, nil
	}
	for _, layout := range []string{"2006-01-02T15:04Z07:00", time.RFC3339} {
		if t, err := time.Parse(layout, date); err == nil {
			return RootAsk{host, t.UTC()}, nil
		}
	}
	return RootAsk{}, errors.New(i18n.T(lang, "config.rootask.date", entry, date))
}

// RootAskMaxDays is how far ahead a new root exception may end: an exception is for a
// job at hand, and one that lasts longer is written again when it ends.
const RootAskMaxDays = 30

// NewRootAsk checks a root exception about to be added, for host until the date until:
// that it reads, that it has not ended and that it ends within RootAskMaxDays of now. It
// returns the entry to write in root_ask. Entries already written are not held to this:
// they are read as they are (ParseRootAsk).
func NewRootAsk(lang, host, until string, now time.Time) (string, RootAsk, error) {
	entry := strings.TrimSpace(host) + " until " + strings.TrimSpace(until)
	r, err := ParseRootAsk(lang, entry)
	if err != nil {
		return "", RootAsk{}, err
	}
	if !r.Until.After(now) {
		return "", RootAsk{}, errors.New(i18n.T(lang, "config.rootask.past", strings.TrimSpace(until)))
	}
	if r.Until.After(now.Add(RootAskMaxDays * 24 * time.Hour)) {
		return "", RootAsk{}, errors.New(i18n.T(lang, "config.rootask.far", strings.TrimSpace(until), RootAskMaxDays))
	}
	return entry, r, nil
}

// RootExceptions returns the entries of root_ask that can be read, and those that
// cannot (they grant nothing).
func (c *Config) RootExceptions() (valid []RootAsk, unread []string) {
	for _, e := range c.RootAsk {
		r, err := ParseRootAsk(c.Language, e)
		if err != nil {
			unread = append(unread, e)
			continue
		}
		valid = append(valid, r)
	}
	return valid, unread
}
