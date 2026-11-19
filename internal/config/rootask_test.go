package config

import (
	"strings"
	"testing"
	"time"
)

func TestParseRootAsk(t *testing.T) {
	for in, want := range map[string]RootAsk{
		"db01 until 2026-10-31":                   {"db01", time.Date(2026, 10, 31, 23, 59, 59, 0, time.UTC)},
		"ubuntu@10.0.0.1 until 2026-10-31T23:59Z": {"ubuntu@10.0.0.1", time.Date(2026, 10, 31, 23, 59, 0, 0, time.UTC)},
		" db01  until  2026-11-01T08:00:00+01:00": {"db01", time.Date(2026, 11, 1, 7, 0, 0, 0, time.UTC)},
	} {
		got, err := ParseRootAsk("en", in)
		if err != nil || got.Host != want.Host || !got.Until.Equal(want.Until) {
			t.Errorf("%q: %v %v, want %v", in, got, err, want)
		}
	}
	for _, in := range []string{"db01", "db01 until", "db01 until tomorrow", " until 2026-10-31", "a b until 2026-10-31"} {
		if _, err := ParseRootAsk("en", in); err == nil {
			t.Errorf("%q was read", in)
		}
	}
	if _, err := ParseRootAsk("es", "db01 until tomorrow"); err == nil || !strings.Contains(err.Error(), `la fecha "tomorrow" no es ni un día`) {
		t.Errorf("in Spanish: %v", err)
	}
}

func TestNewRootAsk(t *testing.T) {
	now := time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)
	entry, r, err := NewRootAsk("en", " db01 ", "2026-11-20", now)
	if err != nil || entry != "db01 until 2026-11-20" || r.Host != "db01" {
		t.Fatalf("%q %v %v", entry, r, err)
	}
	if _, _, err := NewRootAsk("en", "db01", "2026-11-19", now); err != nil {
		t.Errorf("today, ending at its end: %v", err)
	}
	if _, _, err := NewRootAsk("en", "db01", "2026-12-19T12:00Z", now); err != nil {
		t.Errorf("exactly 30 days: %v", err)
	}
	for until, want := range map[string]string{
		"2026-11-18":        "already passed",
		"2026-11-19T11:00Z": "already passed",
		"2026-12-20":        "too far",
		"tomorrow":          "neither a day",
	} {
		if _, _, err := NewRootAsk("en", "db01", until, now); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: %v, want %q", until, err, want)
		}
	}
	if _, _, err := NewRootAsk("es", "db01", "2026-12-20", now); err == nil || !strings.Contains(err.Error(), "demasiado lejos") {
		t.Errorf("in Spanish: %v", err)
	}
}
