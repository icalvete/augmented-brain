package panel

import (
	"strings"
	"time"

	"github.com/icalvete/augmented-brain/internal/i18n"
)

// humanDate says a date the way a person reads it: "today", "yesterday", "tomorrow",
// "13 Nov", or "13 Nov 2025" in another year; with a time ("2026-11-13 10:04") it
// follows the day: "today, 10:04". What is not a date comes back as it was.
func humanDate(lang, s string, now time.Time) string {
	day, clock, _ := strings.Cut(strings.TrimSpace(s), " ")
	// Both days in UTC, so a change of summer time between them never makes a day 23 hours.
	d, err := time.Parse("2006-01-02", day)
	if err != nil {
		return s
	}
	if clock != "" {
		if _, err := time.Parse("15:04", clock); err != nil {
			return s
		}
	}
	y, m, dd := now.Date()
	today := time.Date(y, m, dd, 0, 0, 0, 0, time.UTC)
	var out string
	switch d.Sub(today) / (24 * time.Hour) {
	case 0:
		out = i18n.T(lang, "panel.date.today")
	case -1:
		out = i18n.T(lang, "panel.date.yesterday")
	case 1:
		out = i18n.T(lang, "panel.date.tomorrow")
	default:
		month := strings.Fields(i18n.T(lang, "panel.date.months"))[d.Month()-1]
		if d.Year() == y {
			out = i18n.T(lang, "panel.date.day", d.Day(), month)
		} else {
			out = i18n.T(lang, "panel.date.year", d.Day(), month, d.Year())
		}
	}
	if clock != "" {
		out = i18n.T(lang, "panel.date.time", out, clock)
	}
	return out
}
