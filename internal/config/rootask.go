package config

import (
	"fmt"
	"strings"
	"time"
)

// RootAsk is a temporary exception for root on one machine, written in root_ask as
// "<host> until <date>": until then, a command that would run as root there asks the
// user instead of being denied. The date is a day (the exception lasts to its end, in
// UTC) or a time, such as 2026-10-31T23:59Z.
type RootAsk struct {
	Host  string
	Until time.Time
}

// ParseRootAsk reads one entry of root_ask.
func ParseRootAsk(entry string) (RootAsk, error) {
	host, date, ok := strings.Cut(strings.TrimSpace(entry), " until ")
	host, date = strings.TrimSpace(host), strings.TrimSpace(date)
	if !ok || host == "" || strings.ContainsAny(host, " \t") {
		return RootAsk{}, fmt.Errorf("%q is not \"<host> until <date>\"", entry)
	}
	if d, err := time.Parse("2006-01-02", date); err == nil {
		return RootAsk{host, d.Add(24*time.Hour - time.Second)}, nil
	}
	for _, layout := range []string{"2006-01-02T15:04Z07:00", time.RFC3339} {
		if t, err := time.Parse(layout, date); err == nil {
			return RootAsk{host, t.UTC()}, nil
		}
	}
	return RootAsk{}, fmt.Errorf("%q: the date %q is neither a day (2026-10-31) nor a time (2026-10-31T23:59Z)", entry, date)
}

// RootExceptions returns the entries of root_ask that can be read, and those that
// cannot (they grant nothing).
func (c *Config) RootExceptions() (valid []RootAsk, unread []string) {
	for _, e := range c.RootAsk {
		r, err := ParseRootAsk(e)
		if err != nil {
			unread = append(unread, e)
			continue
		}
		valid = append(valid, r)
	}
	return valid, unread
}
