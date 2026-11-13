package mcpserver

import (
	"slices"
	"strings"
	"time"

	"github.com/icalvete/augmented-brain/internal/brain"
	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/state"
)

// toolNotice leaves for the user, through Claude Code and not through the assistant,
// a line about something a tool saved that every later session loads: a rule, the
// profile, the guard's safety lists. The assistant calls these tools on its own
// judgement, and a hostile text it read could talk it into one; the user hears of it
// once, from the next hook (state.TakeToolNotice). A failure to write it never fails
// the tool: what was saved stays saved.
func toolNotice(b *brain.Brain, key string, args ...any) {
	_ = state.AddToolNotice(b.ConfigDir, i18n.T(b.Language, key, args...), time.Now())
}

// noticeWhere names where a rule lives, for toolNotice.
func noticeWhere(b *brain.Brain, shared bool) string {
	if shared {
		return i18n.T(b.Language, "hook.toolnotice.where.shared")
	}
	return i18n.T(b.Language, "hook.toolnotice.where.space", b.Space.Name)
}

// profileChanges names the parts of the profile a profile_save changes: in holds what
// it sends, a nil part left as it was.
func profileChanges(b *brain.Brain, before brain.ProfileInput, in [5]*string) []string {
	var out []string
	for _, f := range []struct {
		key  string
		from *string
		was  string
	}{{"name", in[0], before.Name}, {"about", in[1], before.About}, {"work", in[2], before.WorkStyle},
		{"assistant", in[3], before.Assistant}, {"personality", in[4], before.Personality}} {
		if f.from != nil && strings.TrimSpace(*f.from) != strings.TrimSpace(f.was) {
			out = append(out, i18n.T(b.Language, "hook.toolnotice.part."+f.key))
		}
	}
	return out
}

// safetyAdded is what a safety_save adds to the guard's lists that was not there.
func safetyAdded(cfgDir string, addProfiles, addHosts []string) []string {
	var profiles, hosts []string
	if c, err := config.Load(cfgDir); err == nil {
		profiles, hosts = c.GuardLists()
	}
	var out []string
	for _, l := range []struct {
		have, add []string
	}{{profiles, addProfiles}, {hosts, addHosts}} {
		for _, v := range l.add {
			if v = strings.TrimSpace(v); v != "" && !slices.Contains(l.have, v) && !slices.Contains(out, v) {
				out = append(out, v)
			}
		}
	}
	return out
}
