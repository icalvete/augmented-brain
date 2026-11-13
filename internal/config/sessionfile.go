package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/icalvete/augmented-brain/internal/paths"
)

// SessionFile, in the configuration folder, holds how the user wants their sessions:
// the model they expect to answer, and whether the assistant asks before it saves what
// every session answers to. A file of its own, not a setting of
// config.toml, for the same reason as GuardFile: earlier programs refuse an unknown
// setting of config.toml. A missing file means no preference: nothing is checked.
const SessionFile = "session.toml"

// Session is the content of session.toml.
type Session struct {
	// PreferredModel is the model the user expects to answer ("opus", or a full name
	// such as "claude-opus-5-5"). When a session answers with another one, the
	// assistant is asked to tell the user. Empty: no check.
	PreferredModel string `toml:"preferred_model"`
	// MemoryTrust is how much the assistant may save on its own of what every session
	// answers to (a rule, the profile, the guard's lists): TrustTell, it saves and the
	// user is told, or TrustAsk, the user is asked first. Read it with Trust.
	MemoryTrust string `toml:"memory_trust"`
}

// The values of memory_trust.
const (
	TrustTell = "tell"
	TrustAsk  = "ask"
)

// Trust is the level in force: TrustTell when nothing is set, and TrustAsk for any
// value this program does not know, so a level a newer program wrote is never read
// as the laxer one.
func (s Session) Trust() string {
	switch strings.ToLower(strings.TrimSpace(s.MemoryTrust)) {
	case "", TrustTell:
		return TrustTell
	}
	return TrustAsk
}

// LoadSession reads session.toml in the configuration folder. A missing file is an
// empty Session; a file that cannot be read is an error.
func LoadSession(cfgDir string) (Session, error) {
	var s Session
	path := filepath.Join(cfgDir, SessionFile)
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if _, err := toml.DecodeFile(path, &s); err != nil {
		return Session{}, fmt.Errorf("%s: %w", path, err)
	}
	s.PreferredModel = strings.TrimSpace(s.PreferredModel)
	s.MemoryTrust = strings.TrimSpace(s.MemoryTrust)
	return s, nil
}

// SaveSession writes session.toml (0600, replacing the file in one step).
func SaveSession(cfgDir string, s Session) error {
	var b strings.Builder
	b.WriteString("# How you want your sessions. The panel's Guard page writes this file.\n")
	b.WriteString("# preferred_model: the model you expect to answer (opus, sonnet, haiku or a full name);\n")
	b.WriteString("# when a session answers with another one, the assistant tells you. Empty: no check.\n")
	b.WriteString("preferred_model = " + fmt.Sprintf("%q", strings.TrimSpace(s.PreferredModel)) + "\n")
	// Written only when set: a file with no level stays as earlier programs wrote it.
	if t := strings.TrimSpace(s.MemoryTrust); t != "" {
		b.WriteString("# memory_trust: \"tell\", the assistant saves a rule, your profile or the guard's lists\n")
		b.WriteString("# and you are told; \"ask\", it asks you first.\n")
		b.WriteString("memory_trust = " + fmt.Sprintf("%q", t) + "\n")
	}
	return paths.WriteAtomic(filepath.Join(cfgDir, SessionFile), []byte(b.String()), 0o600)
}

// ModelMatches says whether model, as Claude Code names it ("claude-opus-5-5",
// "opus[1m]"), is the preferred one: the same name, or one that contains it, so
// "opus" matches every Opus. An empty preference or model matches.
func ModelMatches(preferred, model string) bool {
	p, m := strings.ToLower(strings.TrimSpace(preferred)), strings.ToLower(strings.TrimSpace(model))
	return p == "" || m == "" || strings.Contains(m, p)
}
