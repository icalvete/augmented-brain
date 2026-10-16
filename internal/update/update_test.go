package update

import (
	"errors"
	"testing"
	"time"
)

func TestNewer(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"0.2.1-alpha", "0.2.0-alpha", true},
		{"0.2.0-alpha", "0.2.1-alpha", false},
		{"0.3.0", "0.2.9", true},
		{"0.10.0", "0.9.0", true},
		{"1.0", "0.99.99", true},
		{"0.2.1", "0.2.1-alpha", false}, // same numbers: not newer
		{"v0.2.1-alpha", "0.2.0", true},
		{"0.2.1", "0.0.1-dev", true},
		{"0.2.1", "dev", false}, // a build without a version never asks to update
		{"", "0.2.0", false},
	}
	for _, c := range cases {
		if got := Newer(c.a, c.b); got != c.want {
			t.Errorf("Newer(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

// The latest version is asked for at most once a day; no network is no news.
func TestAvailable(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("AB_NO_UPDATE_CHECK", "")
	now := time.Date(2026, 10, 16, 10, 0, 0, 0, time.UTC)
	calls := 0
	fetch := func(v string) func() (string, error) {
		return func() (string, error) { calls++; return v, nil }
	}
	if got := Available(dir, "0.2.0-alpha", now, fetch("0.2.1-alpha")); got != "0.2.1-alpha" {
		t.Errorf("first check: %q", got)
	}
	// An hour later the answer comes from the cache, even if a newer one is out.
	if got := Available(dir, "0.2.0-alpha", now.Add(time.Hour), fetch("0.3.0")); got != "0.2.1-alpha" || calls != 1 {
		t.Errorf("cached check: %q after %d calls", got, calls)
	}
	// A day later it asks again.
	if got := Available(dir, "0.2.0-alpha", now.Add(25*time.Hour), fetch("0.3.0")); got != "0.3.0" || calls != 2 {
		t.Errorf("next day: %q after %d calls", got, calls)
	}
	// Up to date: nothing to say.
	if got := Available(dir, "0.3.0", now.Add(26*time.Hour), fetch("0.3.0")); got != "" {
		t.Errorf("up to date: %q", got)
	}
	// No network and no cache: nothing to say, no error.
	other := t.TempDir()
	if got := Available(other, "0.2.0", now, func() (string, error) { return "", errors.New("offline") }); got != "" {
		t.Errorf("offline: %q", got)
	}
	// Turned off.
	t.Setenv("AB_NO_UPDATE_CHECK", "1")
	if got := Available(t.TempDir(), "0.2.0", now, fetch("9.9.9")); got != "" {
		t.Errorf("turned off: %q", got)
	}
}
