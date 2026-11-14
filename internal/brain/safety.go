package brain

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/state"
)

// SafetyInput is what the guard must ask the user about, on this whole machine. Its
// entries are added to the lists; nothing is taken out (see SaveSafety).
type SafetyInput struct {
	ProductionAWSProfiles []string
	UntouchableHosts      []string
}

// SaveSafety adds to the risk settings in the configuration file, changing only those
// lines. They are machine-wide: the guard applies them in every space, and outside every
// space, from the next command on. It only adds: what makes the guard ask is never
// taken out by a tool the assistant may be talked into using; the user takes an entry
// out by editing the configuration. A list a space kept of its own (older
// configurations had them per space) moves into the machine-wide one.
//
// Adding only makes the guard ask more, so when the user had accepted the lists just
// before (the seal matched), the new lists are sealed too: the guard does not take its
// own addition for a change the user has not seen. Lists already changed some other way
// stay unsealed, and so do lists that end up different from the old ones plus what was
// added (something else wrote the file meanwhile).
func (b *Brain) SaveSafety(in SafetyInput) (string, error) {
	var oldProfiles, oldHosts []string
	c, err := config.Load(b.ConfigDir)
	if err == nil {
		oldProfiles, oldHosts = c.GuardLists()
	}
	expected := sealedPlus(b.ConfigDir, c, err, in)
	before := map[string][]string{"production_aws_profiles": oldProfiles, "untouchable_hosts": oldHosts}
	var done []string
	for _, s := range []struct {
		key    string
		values []string
	}{
		{"production_aws_profiles", in.ProductionAWSProfiles},
		{"untouchable_hosts", in.UntouchableHosts},
	} {
		if s.values == nil {
			continue
		}
		clean := append([]string{}, before[s.key]...)
		for _, v := range s.values {
			if v = strings.TrimSpace(v); v != "" && !slices.Contains(clean, v) {
				clean = append(clean, v)
			}
		}
		if err := config.SetTopList(b.ConfigDir, s.key, clean); err != nil {
			return "", err
		}
		c, err := config.Load(b.ConfigDir)
		if err != nil {
			return "", err
		}
		for _, name := range c.SpaceNames() {
			if err := config.RemoveSpaceKey(b.ConfigDir, name, s.key); err != nil {
				return "", err
			}
		}
		done = append(done, fmt.Sprintf("%s = [%s] (before: [%s])", s.key, strings.Join(clean, ", "), strings.Join(before[s.key], ", ")))
	}
	if len(done) == 0 {
		return "nothing to change", nil
	}
	_ = os.Remove(filepath.Join(b.ConfigDir, config.AskSafetyFile)) // asked and answered
	safetyWritten()
	resealAdded(b.ConfigDir, expected)
	return fmt.Sprintf("saved for every space on this machine: %s", strings.Join(done, "; ")), nil
}

// safetyWritten runs between writing the lists and sealing them; tests change it.
var safetyWritten = func() {}

// sealedPlus returns the fingerprint the lists will have once in is added, but only
// when the seal matches the lists as they are now; "" otherwise (no seal yet, a seal
// that cannot be read, lists changed since the user last accepted them).
func sealedPlus(cfgDir string, c *config.Config, loadErr error, in SafetyInput) string {
	if loadErr != nil {
		return ""
	}
	g, err := config.LoadGuard(cfgDir)
	if err != nil {
		return ""
	}
	seal, err := state.ReadSeal(cfgDir)
	if err != nil || seal == "" || seal != c.GuardFingerprint(g) {
		return ""
	}
	plus := *c
	for _, v := range in.ProductionAWSProfiles {
		plus.ProductionAWSProfiles = append(slices.Clone(plus.ProductionAWSProfiles), strings.TrimSpace(v))
	}
	for _, v := range in.UntouchableHosts {
		plus.UntouchableHosts = append(slices.Clone(plus.UntouchableHosts), strings.TrimSpace(v))
	}
	return plus.GuardFingerprint(g)
}

// resealAdded seals the lists when they are exactly the expected ones.
func resealAdded(cfgDir, expected string) {
	if expected == "" {
		return
	}
	c, err := config.Load(cfgDir)
	if err != nil {
		return
	}
	g, err := config.LoadGuard(cfgDir)
	if err != nil || c.GuardFingerprint(g) != expected {
		return
	}
	_ = state.WriteSeal(cfgDir, expected)
}
