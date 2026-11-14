package panel

import (
	"testing"
	"time"
)

func TestHumanDate(t *testing.T) {
	now := time.Date(2026, 3, 29, 9, 0, 0, 0, time.FixedZone("CET", 3600)) // the day summer time starts
	for _, c := range []struct{ lang, in, want string }{
		{"en", "2026-03-29", "today"},
		{"es", "2026-03-29", "hoy"},
		{"en", "2026-03-28", "yesterday"},
		{"es", "2026-03-30", "mañana"},
		{"en", "2026-11-13", "13 Nov"},
		{"es", "2026-11-13", "13 nov"},
		{"en", "2025-01-02", "2 Jan 2025"},
		{"es", "2026-03-29 10:04", "hoy, 10:04"},
		{"en", "2026-02-01 23:59", "1 Feb, 23:59"},
		{"en", "not a date", "not a date"},
		{"en", "2026-02-01 later", "2026-02-01 later"},
		{"en", "", ""},
	} {
		if got := humanDate(c.lang, c.in, now); got != c.want {
			t.Errorf("humanDate(%q, %q) = %q, want %q", c.lang, c.in, got, c.want)
		}
	}
	// Across a change of summer time, a day is still a day.
	madrid, err := time.LoadLocation("Europe/Madrid")
	if err == nil {
		after := time.Date(2026, 3, 30, 0, 30, 0, 0, madrid)
		if got := humanDate("en", "2026-03-29", after); got != "yesterday" {
			t.Errorf("across summer time: %q, want yesterday", got)
		}
	}
}
