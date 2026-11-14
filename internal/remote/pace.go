package remote

import "time"

// A pace is a wait measured by the clock, with its looks spaced out. The waits used to
// add up their pauses and never look at the clock: with a `claude agents` that took 25
// s, "30 minutes at most" lasted almost 7 hours, and a normal wait started claude every
// 2 s, 900 times (R-09 of the November 2026 review).
//
// What has passed is the larger of the clock and the pauses slept, so a test whose
// Sleep does not really sleep still ends.
type pace struct {
	s       System
	start   time.Time
	max     time.Duration
	slept   time.Duration
	told    time.Duration // when Waiting was last called
	spacing func(passed time.Duration) time.Duration
}

// WaitNote is how often a long wait calls System.Waiting.
const WaitNote = 5 * time.Minute

// idleSpacing is how far apart the looks of a wait for a session to end its answer
// are: 2 s for the first half minute, then 5 s up to 2 minutes, 10 s up to 5, and 30 s
// after. Thirty minutes take about a hundred looks.
func idleSpacing(passed time.Duration) time.Duration {
	switch {
	case passed < 30*time.Second:
		return 2 * time.Second
	case passed < 2*time.Minute:
		return 5 * time.Second
	case passed < 5*time.Minute:
		return 10 * time.Second
	}
	return 30 * time.Second
}

// startSpacing is for the short waits of a session starting (a minute at most): a
// look every second for the first 10 s, then every 3 s.
func startSpacing(passed time.Duration) time.Duration {
	if passed < 10*time.Second {
		return time.Second
	}
	return 3 * time.Second
}

// everySecond keeps a look a second for the waits of a few seconds.
func everySecond(time.Duration) time.Duration { return time.Second }

// newPace starts a wait of at most max.
func (s System) newPace(max time.Duration, spacing func(time.Duration) time.Duration) *pace {
	return &pace{s: s, start: s.now(), max: max, spacing: spacing}
}

// passed is how long the wait has lasted.
func (p *pace) passed() time.Duration {
	return max(p.s.now().Sub(p.start), p.slept)
}

// over says whether the wait has used up its time.
func (p *pace) over() bool { return p.passed() > p.max }

// sleep pauses until the next look. soon asks for the shortest pause, as when a session
// was just seen idle once and is looked at again to be sure.
func (p *pace) sleep(soon bool) {
	d := p.spacing(p.passed())
	if soon {
		d = p.spacing(0)
	}
	if left := p.max - p.passed(); left > 0 && d > left {
		d = left
	}
	p.s.sleep(d)
	p.slept += d
	if p.s.Waiting != nil && p.passed()-p.told >= WaitNote {
		p.told = p.passed()
		p.s.Waiting(p.told)
	}
}
