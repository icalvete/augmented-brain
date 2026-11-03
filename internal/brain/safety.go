package brain

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/icalvete/augmented-brain/internal/config"
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
func (b *Brain) SaveSafety(in SafetyInput) (string, error) {
	var oldProfiles, oldHosts []string
	if c, err := config.Load(b.ConfigDir); err == nil {
		oldProfiles, oldHosts = c.GuardLists()
	}
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
	return fmt.Sprintf("saved for every space on this machine: %s", strings.Join(done, "; ")), nil
}
