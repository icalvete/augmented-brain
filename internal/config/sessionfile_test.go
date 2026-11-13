package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

func TestMemoryTrust(t *testing.T) {
	dir := t.TempDir()
	if s, _ := LoadSession(dir); s.Trust() != TrustTell {
		t.Fatalf("no file: %q", s.Trust())
	}
	// A file with no level stays as earlier programs wrote it.
	if err := SaveSession(dir, Session{PreferredModel: "opus"}); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(filepath.Join(dir, SessionFile)); strings.Contains(string(data), "memory_trust") {
		t.Errorf("a level nobody set was written:\n%s", data)
	}
	if err := SaveSession(dir, Session{PreferredModel: "opus", MemoryTrust: TrustAsk}); err != nil {
		t.Fatal(err)
	}
	if s, err := LoadSession(dir); err != nil || s.Trust() != TrustAsk || s.PreferredModel != "opus" {
		t.Fatalf("read back: %+v %v", s, err)
	}
	for value, want := range map[string]string{"": TrustTell, "tell": TrustTell, " Tell ": TrustTell,
		"ask": TrustAsk, "ASK": TrustAsk, "silent": TrustAsk, "paranoid": TrustAsk} {
		if got := (Session{MemoryTrust: value}).Trust(); got != want {
			t.Errorf("%q: %q, want %q", value, got, want)
		}
	}
}
