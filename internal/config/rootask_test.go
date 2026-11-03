package config

import (
	"testing"
	"time"
)

func TestParseRootAsk(t *testing.T) {
	for in, want := range map[string]RootAsk{
		"db01 until 2026-10-31":                {"db01", time.Date(2026, 10, 31, 23, 59, 59, 0, time.UTC)},
		"ubuntu@10.0.0.1 until 2026-10-31T23:59Z": {"ubuntu@10.0.0.1", time.Date(2026, 10, 31, 23, 59, 0, 0, time.UTC)},
		" db01  until  2026-11-01T08:00:00+01:00": {"db01", time.Date(2026, 11, 1, 7, 0, 0, 0, time.UTC)},
	} {
		got, err := ParseRootAsk(in)
		if err != nil || got.Host != want.Host || !got.Until.Equal(want.Until) {
			t.Errorf("%q: %v %v, want %v", in, got, err, want)
		}
	}
	for _, in := range []string{"db01", "db01 until", "db01 until tomorrow", " until 2026-10-31", "a b until 2026-10-31"} {
		if _, err := ParseRootAsk(in); err == nil {
			t.Errorf("%q was read", in)
		}
	}
}
