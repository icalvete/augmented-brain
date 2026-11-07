package remote

import (
	"strings"

	"github.com/icalvete/augmented-brain/internal/i18n"
)

func yesNo(l string, b bool) string {
	if b {
		return i18n.T(l, "remote.yes")
	}
	return i18n.T(l, "remote.no")
}

// LingerWord says linger in words.
func LingerWord(l string, g Linger) string {
	switch g {
	case LingerOn:
		return i18n.T(l, "remote.yes")
	case LingerOff:
		return i18n.T(l, "remote.no")
	}
	return i18n.T(l, "remote.unknown")
}

// ServiceWord says whether the service runs, in words.
func (st SpaceStatus) ServiceWord(l string) string {
	if !st.Installed {
		return i18n.T(l, "remote.state.missing")
	}
	switch st.Active {
	case "active", "inactive", "failed", "activating":
		return i18n.T(l, "remote.state."+st.Active)
	}
	return i18n.T(l, "remote.state.other", st.Active)
}

func (st SpaceStatus) enabledWord(l string) string {
	if st.Enabled == "enabled" {
		return i18n.T(l, "remote.enabled.enabled")
	}
	return i18n.T(l, "remote.enabled.other")
}

func (st SpaceStatus) trustWord(l string) string {
	if st.Trusted {
		return i18n.T(l, "remote.trust.yes")
	}
	return i18n.T(l, "remote.trust.no")
}

// Advice is what is left to do for one space, one sentence each; empty when all is
// well. sys gives the linger command.
func (s System) Advice(l string, st SpaceStatus, g Linger) []string {
	var out []string
	switch {
	case !st.Installed:
		out = append(out, i18n.T(l, "remote.fix.install"))
	case !st.Trusted:
		out = append(out, i18n.T(l, "remote.fix.trust", st.Folder))
	case !st.Running():
		out = append(out, i18n.T(l, "remote.fix.restart", st.Unit))
	}
	if st.Installed && st.Outdated {
		out = append(out, i18n.T(l, "remote.fix.update"))
	}
	if g == LingerOff {
		out = append(out, i18n.T(l, "remote.fix.linger", s.LingerCommand()))
	}
	return out
}

// StatusLines are what `remote status` prints.
func (s System) StatusLines(l string, st *Status) []string {
	var out []string
	for _, x := range st.Spaces {
		out = append(out, i18n.T(l, "remote.status.space", x.Space, x.Name),
			i18n.T(l, "remote.status.service", x.ServiceWord(l), x.enabledWord(l)),
			i18n.T(l, "remote.status.folder", x.Folder, x.trustWord(l)))
		for _, a := range s.Advice(l, x, LingerOn) { // linger is said once, below
			out = append(out, i18n.T(l, "remote.status.fix", a))
		}
	}
	out = append(out, i18n.T(l, "remote.status.linger", LingerWord(l, st.Linger)))
	if st.Linger == LingerOff {
		out = append(out, i18n.T(l, "remote.status.fix", i18n.T(l, "remote.fix.linger", s.LingerCommand())))
	}
	return out
}

// DoctorLine is the line `doctor` gives one space: what it found, whether all is well,
// and how to fix it when not.
func (s System) DoctorLine(l string, x SpaceStatus, g Linger) (what, fix string, ok bool) {
	what = i18n.T(l, "remote.doctor", x.Space, x.Name, x.ServiceWord(l), yesNo(l, x.Trusted), LingerWord(l, g))
	adv := s.Advice(l, x, g)
	return what, strings.Join(adv, "; "), len(adv) == 0
}
