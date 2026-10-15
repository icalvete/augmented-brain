package brain

import (
	"fmt"
	"strings"

	"github.com/icalvete/augmented-brain/internal/config"
)

// SafetyInput is what the guard must ask the user about in this space. A nil list
// leaves that setting as it is; an empty one clears it.
type SafetyInput struct {
	ProductionAWSProfiles []string
	UntouchableHosts      []string
}

// SaveSafety writes the space's risk settings in the configuration file, changing only
// those lines. The guard reads them on the next command.
func (b *Brain) SaveSafety(in SafetyInput) (string, error) {
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
		var clean []string
		for _, v := range s.values {
			if v = strings.TrimSpace(v); v != "" {
				clean = append(clean, v)
			}
		}
		if err := config.SetSpaceList(b.ConfigDir, b.Space.Name, s.key, clean); err != nil {
			return "", err
		}
		done = append(done, fmt.Sprintf("%s = [%s]", s.key, strings.Join(clean, ", ")))
	}
	if len(done) == 0 {
		return "nothing to change", nil
	}
	return fmt.Sprintf("space %s: %s", b.Space.Name, strings.Join(done, "; ")), nil
}
