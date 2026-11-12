package remote

import (
	"strings"
	"time"

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
	case !st.Trusted && s.GOOS == "windows":
		out = append(out, i18n.T(l, "remote.fix.trust.win", st.Folder))
	case !st.Trusted:
		out = append(out, i18n.T(l, "remote.fix.trust", st.Folder))
	case !st.Running() && st.Log != "":
		out = append(out, i18n.T(l, "remote.fix.restart.log", st.Log))
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
	if w := st.Watch; w != nil {
		out = append(out, i18n.T(l, "remote.status.watch", strings.TrimSuffix(WatchEvery, "min"), s.WatchWord(l, *w)))
		if fix := WatchAdvice(l, *w); fix != "" {
			out = append(out, i18n.T(l, "remote.status.fix", fix))
		}
	}
	if st.Linger == LingerNone {
		return append(out, i18n.T(l, "remote.status.atlogin"))
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
	if g == LingerNone {
		what = i18n.T(l, "remote.doctor.atlogin", x.Space, x.Name, x.ServiceWord(l), yesNo(l, x.Trusted))
	}
	adv := s.Advice(l, x, g)
	return what, strings.Join(adv, "; "), len(adv) == 0
}

// WatchWord says how the watch stands, in words: whether it runs and, when it ran, how
// long ago it last looked.
func (s System) WatchWord(l string, w WatchStatus) string {
	var word string
	switch {
	case !w.Installed:
		return i18n.T(l, "remote.state.missing")
	case w.Active != "active":
		word = i18n.T(l, "remote.state.inactive")
	case w.Enabled != "enabled":
		word = i18n.T(l, "remote.state.active") + ", " + i18n.T(l, "remote.enabled.other")
	default:
		word = i18n.T(l, "remote.state.active")
	}
	if !w.Last.IsZero() {
		now := time.Now
		if s.Now != nil {
			now = s.Now
		}
		word += ", " + i18n.T(l, "remote.watch.last", ago(l, now().Sub(w.Last)))
	}
	if w.Failed {
		word += ", " + i18n.T(l, "remote.watch.failed")
	}
	return word
}

// WatchAdvice is what to do about the watch, in one sentence; "" when all is well.
func WatchAdvice(l string, w WatchStatus) string {
	switch {
	case !w.Installed:
		return i18n.T(l, "remote.fix.install")
	case w.Active != "active" || w.Enabled != "enabled":
		return i18n.T(l, "remote.fix.watch.start")
	case w.Failed:
		return i18n.T(l, "remote.fix.watch.failed", WatchName+".service")
	}
	return ""
}

// WatchDoctorLine is the line `doctor` gives the watch when something is wrong with it;
// ok when there is nothing to say. A missing watch is already said by each space's line
// (its service is out of date).
func (s System) WatchDoctorLine(l string, w WatchStatus) (what, fix string, ok bool) {
	if !w.Installed || w.Healthy() {
		return "", "", true
	}
	return i18n.T(l, "remote.doctor.watch", s.WatchWord(l, w)), WatchAdvice(l, w), false
}

// ago says a span of time in the largest whole unit: seconds, minutes, hours or days.
func ago(l string, d time.Duration) string {
	switch {
	case d < time.Minute:
		return i18n.T(l, "remote.ago.s", max(int(d/time.Second), 0))
	case d < time.Hour:
		return i18n.T(l, "remote.ago.min", int(d/time.Minute))
	case d < 48*time.Hour:
		return i18n.T(l, "remote.ago.h", int(d/time.Hour))
	}
	return i18n.T(l, "remote.ago.d", int(d/(24*time.Hour)))
}
