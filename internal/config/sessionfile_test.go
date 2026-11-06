package config

import "testing"

func TestSessionFile(t *testing.T) {
	dir := t.TempDir()
	if s, err := LoadSession(dir); err != nil || s.PreferredModel != "" {
		t.Fatalf("missing file: %+v %v", s, err)
	}
	if err := SaveSession(dir, Session{PreferredModel: " opus "}); err != nil {
		t.Fatal(err)
	}
	if s, err := LoadSession(dir); err != nil || s.PreferredModel != "opus" {
		t.Fatalf("read back: %+v %v", s, err)
	}
}

func TestModelMatches(t *testing.T) {
	for _, c := range []struct {
		pref, model string
		ok          bool
	}{
		{"opus", "claude-opus-5-5", true},
		{"opus", "opus[1m]", true},
		{"Opus", "claude-sonnet-5-5", false},
		{"claude-opus-5-5", "claude-opus-5-5", true},
		{"", "claude-sonnet-5-5", true},
		{"opus", "", true},
	} {
		if got := ModelMatches(c.pref, c.model); got != c.ok {
			t.Errorf("%q %q: %v", c.pref, c.model, got)
		}
	}
}
