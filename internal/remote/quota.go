package remote

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/paths"
	"github.com/icalvete/augmented-brain/internal/state"
)

// Waking the sessions the usage limit stopped (Linux only, from the watch).
//
// Measured on 12 and 13-nov with claude 2.1.322. When the usage limit of the
// subscription refuses a turn, Claude Code writes the refusal as the session's last
// answer, with the hour the limit resets (state.QuotaStop), and the turn ends. An
// interactive session then waits and goes on by itself at that hour (Claude Code's
// autoContinueAtUsageLimit); a background one does not: it keeps its process and does
// nothing until someone writes to it (one stood eleven hours on 12-nov).
//
// Claude has no command that writes to a background session that runs. What works:
// `claude stop <id>`, wait until Claude lists it with no process, and `claude --bg
// --resume <session id> <message>`: it comes back under the same id, with its
// conversation and its saved options (--remote-control among them, so the same thread
// in Claude Desktop and claude.ai), and the message is its next turn. Resumed before
// Claude has let the process go (a fraction of a second after `stop` returns), Claude
// starts a copy under another id instead, and says so.
//
// Only a task's own session is woken (one on record, launched by `session start`), one
// on each look of the watch: the sessions of a stopped batch come back a look apart
// and do not spend the new quota all at once.

// quotaFile keeps the sessions that wait for the usage limit, in the state folder.
const quotaFile = "quota-wait.json"

// quotaMargin is how long after the reset a session is woken: clocks differ.
const quotaMargin = time.Minute

// quotaStopWait is how long a stopped session may take to lose its process (10 seconds
// measured on 13-nov).
const quotaStopWait = 40 * time.Second

// quotaTries is how many times a session is tried before it is left to the user.
const quotaTries = 3

type quotaEntry struct {
	ID     string    `json:"id"`
	Resets time.Time `json:"resets"`
	// Woken is when it was woken; zero while it waits.
	Woken time.Time `json:"woken,omitempty"`
	Tries int       `json:"tries,omitempty"`
}

// quotaState is what quotaFile holds: the sessions whose last answer is the stop, by
// task, and how many were woken since the wait began (a woken session leaves Sessions
// as soon as its transcript shows the new turn).
type quotaState struct {
	Sessions map[string]quotaEntry `json:"sessions"`
	Woken    int                   `json:"woken,omitempty"`
}

func (e quotaEntry) waits() bool { return e.Woken.IsZero() && e.Tries < quotaTries }

// QuotaReport is what one look did about the sessions the usage limit stopped.
type QuotaReport struct {
	// Began: this look found the first session waiting; Until is the latest reset among
	// those waiting. The user hears of it once, not once per session.
	Began bool
	Until time.Time
	// Woken is the task whose session this look woke, "" when none.
	Woken string
	// Ended: this look woke the last one waiting; Count is how many came back since
	// the wait began.
	Ended bool
	Count int
	// Problem is why Task's session could not be woken on this look; GaveUp, that it
	// will not be tried again.
	Task    string
	Problem string
	GaveUp  bool
}

func (s System) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// QuotaResets is when the usage limit that stopped the session with this id resets, and
// whether its last answer is that stop (state.QuotaStop).
func (s System) QuotaResets(sessionID string) (time.Time, bool) {
	return state.QuotaStop(s.transcript(sessionID))
}

// WakeQuota looks once at the tasks' sessions on record: it notes those the usage limit
// stopped and, once the limit has reset, wakes one with message(task) as its next turn.
func (s System) WakeQuota(c *config.Config, message func(task string) string) (QuotaReport, error) {
	var rep QuotaReport
	if s.GOOS != "linux" {
		return rep, ErrUnsupported
	}
	statePath := filepath.Join(c.Dir, state.Dir, quotaFile)
	var st quotaState
	if data, err := os.ReadFile(statePath); err == nil {
		_ = json.Unmarshal(data, &st)
	}
	seen := st.Sessions
	entries, _ := os.ReadDir(launchRecords(c.Dir))
	if len(entries) == 0 && len(seen) == 0 {
		return rep, nil
	}
	claude, err := s.FindClaude()
	if err != nil {
		return rep, errors.New("claude is not installed or not in the PATH")
	}
	list, err := s.agents(claude)
	if err != nil {
		return rep, err
	}
	next := map[string]quotaEntry{}
	records := map[string]launchRecord{}
	agents := map[string]claudeAgent{}
	for _, e := range entries {
		task, ok := strings.CutSuffix(e.Name(), ".json")
		if !ok {
			continue
		}
		r, ok := readLaunch(c.Dir, task)
		if !ok || !sessionID.MatchString(r.ID) {
			continue
		}
		a, listed := findAgent(list, r.ID)
		if !listed {
			continue
		}
		resets, stopped := s.QuotaResets(a.SessionID)
		if !stopped {
			continue // it never stopped, or it goes on: forgotten
		}
		old, had := seen[r.Task]
		if !had || old.ID != r.ID || !old.Resets.Equal(resets) {
			old = quotaEntry{ID: r.ID, Resets: resets}
		}
		next[r.Task], records[r.Task], agents[r.Task] = old, r, a
	}
	waitedBefore := false
	for _, e := range seen {
		waitedBefore = waitedBefore || e.waits()
	}
	var due []string
	for task, e := range next {
		if !e.waits() {
			continue
		}
		if e.Resets.After(rep.Until) {
			rep.Until = e.Resets
		}
		if !s.now().Before(e.Resets.Add(quotaMargin)) {
			due = append(due, task)
		}
	}
	rep.Began = !waitedBefore && !rep.Until.IsZero()
	if rep.Began {
		st.Woken = 0
	}
	sort.Strings(due)
	for _, task := range due {
		if agents[task].Status == "busy" {
			continue // someone is at it
		}
		e := next[task]
		e.Tries++
		if err := s.wakeStopped(claude, records[task], agents[task], message(task)); err != nil {
			rep.Task, rep.Problem, rep.GaveUp = task, err.Error(), e.Tries >= quotaTries
		} else {
			e.Woken, rep.Woken = s.now(), task
			st.Woken++
		}
		next[task] = e
		break // one on each look
	}
	left := 0
	for _, e := range next {
		if e.waits() {
			left++
		}
	}
	if rep.Ended = rep.Woken != "" && left == 0; rep.Ended {
		rep.Count = st.Woken
	}
	st.Sessions = next
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return rep, err
	}
	if err := os.MkdirAll(filepath.Dir(statePath), 0o700); err != nil {
		return rep, err
	}
	return rep, paths.WriteAtomic(statePath, append(data, '\n'), 0o600)
}

// wakeStopped gives the session of record r a new turn with message: stopped first when
// it has a process, then resumed in the background under its own id.
func (s System) wakeStopped(claude string, r launchRecord, a claudeAgent, message string) error {
	if !s.loggedIn(claude) {
		return ErrLoggedOut
	}
	if a.SessionID == "" || strings.HasPrefix(a.SessionID, "-") {
		return errors.New("claude gives no session id for " + r.ID)
	}
	s.wakeDaemon(claude)
	if a.Pid > 0 {
		if _, err := s.Run(claude, "stop", r.ID); err != nil {
			return fmt.Errorf("claude stop %s: %w", r.ID, err)
		}
		gone := false
		for waited := time.Duration(0); waited <= quotaStopWait && !gone; waited += time.Second {
			s.sleep(time.Second)
			list, err := s.agents(claude)
			if err != nil {
				continue
			}
			now, listed := findAgent(list, r.ID)
			gone = listed && now.Pid == 0
		}
		if !gone {
			// Resumed now, Claude would start a copy. It is stopped: the next look resumes it.
			return fmt.Errorf("claude stop %s: the session still has a process after %d seconds", r.ID, int(quotaStopWait.Seconds()))
		}
	}
	bg, bgArgs := s.scoped(claude, "--bg", "--resume", a.SessionID, message)
	out, err := s.RunIn(r.Folder, bg, bgArgs...)
	m := backgrounded.FindStringSubmatch(out)
	switch {
	case err != nil || m == nil:
		msg := tail(cleanTerminal(out), 300)
		if err != nil {
			msg = strings.TrimSpace(msg + " " + err.Error())
		}
		return fmt.Errorf("the session did not come back: %s", msg)
	case m[1] != r.ID:
		// A copy: it would work in the task's folder beside the one on record.
		_, _ = s.Run(claude, "stop", m[1])
		_, _ = s.Run(claude, "rm", m[1])
		return fmt.Errorf("claude started a copy (%s) instead of resuming %s; the copy was removed", m[1], r.ID)
	}
	return nil
}
