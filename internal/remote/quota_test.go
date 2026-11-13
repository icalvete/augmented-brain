package remote

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// quotaFake is a claude with background sessions the usage limit may stop: stop takes
// a session's process away (after lag looks at the list), --bg --resume gives it back
// and writes the message in its transcript.
type quotaFake struct {
	*fake
	pid    map[string]int // id -> its process; 0 is stopped
	lag    int
	copyAs string // --resume starts a copy under this id instead
	now    time.Time
	sent   []string
	slept  int // how many times it waited
}

func newQuotaFake(t *testing.T) *quotaFake {
	q := &quotaFake{fake: newFake(t), pid: map[string]int{}, now: time.Unix(1794526000, 0)}
	q.sys.Now = func() time.Time { return q.now }
	q.sys.Sleep = func(time.Duration) { q.slept++ }
	q.agentList = func() []map[string]any {
		var list []map[string]any
		for id, pid := range q.pid {
			a := map[string]any{"id": id, "kind": "background", "state": "done", "sessionId": "s-" + id}
			if pid > 0 && q.lag > 0 && q.live[id+"/stopping"] {
				q.lag--
			} else if q.live[id+"/stopping"] {
				delete(q.live, id+"/stopping")
				q.pid[id], pid = 0, 0
			}
			if pid > 0 {
				a["pid"], a["status"] = pid, "idle"
				if q.busy[id] > 0 {
					a["status"] = "busy"
				}
			}
			list = append(list, a)
		}
		list = append(list, map[string]any{"pid": 7, "kind": "interactive", "status": "idle"})
		return list
	}
	run := q.sys.Run
	q.sys.Run = func(name string, args ...string) (string, error) {
		if filepath.Base(name) == "claude" && len(args) == 2 && args[0] == "stop" {
			q.calls = append(q.calls, name+" stop "+args[1])
			q.live[args[1]+"/stopping"] = true
			return "stopped " + args[1] + "\n", nil
		}
		return run(name, args...)
	}
	q.sys.RunIn = func(dir string, name string, args ...string) (string, error) {
		q.ranIn = append(q.ranIn, dir+" | "+filepath.Base(name)+" "+strings.Join(args, " "))
		if len(args) != 4 || args[0] != "--bg" || args[1] != "--resume" {
			return "", errors.New("unexpected command")
		}
		id := strings.TrimPrefix(args[2], "s-")
		if q.pid[id] > 0 || q.copyAs != "" {
			other := q.copyAs
			if other == "" {
				other = "c0ffee00"
			}
			q.pid[other] = 99
			return fmt.Sprintf("note: session %s is already running in the background, so this started a copy as %s.\nbackgrounded · %s\n", id, other, other), nil
		}
		q.pid[id] = 500
		q.sent = append(q.sent, id+": "+args[3])
		q.transcript(id, `{"type":"user","message":{"role":"user","content":"woken"}}`)
		return "note: woke session " + id + " with its saved options (--remote-control, -n).\nbackgrounded · " + id + " · x\n", nil
	}
	return q
}

// session puts a task's session on record, running, with a transcript that ends in an
// ordinary answer.
func (q *quotaFake) session(task, id string) {
	q.t.Helper()
	q.pid[id] = 100 + len(q.pid)
	dir := launchRecords(q.cfgDir)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		q.t.Fatal(err)
	}
	data, _ := json.Marshal(launchRecord{ID: id, Task: task, Name: task, Folder: filepath.Join(q.root, "p")})
	if err := os.WriteFile(launchRecordPath(q.cfgDir, task), data, 0o600); err != nil {
		q.t.Fatal(err)
	}
	q.transcript(id, `{"type":"assistant","message":{"model":"claude-opus-5-5","content":[{"type":"text","text":"On it."}]}}`)
}

func (q *quotaFake) transcript(id, line string) {
	q.t.Helper()
	dir := filepath.Join(q.sys.ClaudeDir, "projects", "-p")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		q.t.Fatal(err)
	}
	f, err := os.OpenFile(filepath.Join(dir, "s-"+id+".jsonl"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		q.t.Fatal(err)
	}
	defer f.Close()
	if _, err := f.WriteString(line + "\n"); err != nil {
		q.t.Fatal(err)
	}
}

// limit ends the session's transcript with the usage limit's refusal.
func (q *quotaFake) limit(id string, resets time.Time) {
	q.transcript(id, fmt.Sprintf(`{"type":"assistant","message":{"model":"<synthetic>","content":[{"type":"text","text":"You've hit your 5-hour limit"}]},"apiError":"usage_limit_reached","quotaLimits":{"resetsAt":%d},"error":"rate_limit","isApiErrorMessage":true}`, resets.Unix()))
}

func (q *quotaFake) look() QuotaReport {
	q.t.Helper()
	q.calls = nil
	c, _ := q.load()
	rep, err := q.sys.WakeQuota(c, func(task string) string { return "go on with " + task })
	if err != nil {
		q.t.Fatal(err)
	}
	return rep
}

func TestWakeQuota(t *testing.T) {
	q := newQuotaFake(t)
	q.session("task-a", "aaaa1111")
	q.session("task-b", "bbbb2222")
	q.session("task-c", "cccc3333")
	if rep := q.look(); rep != (QuotaReport{}) {
		t.Fatalf("nothing stopped, yet %+v", rep)
	}
	resets := q.now.Add(10 * time.Minute)
	q.limit("aaaa1111", resets)
	rep := q.look()
	if !rep.Began || !rep.Until.Equal(resets) || rep.Woken != "" {
		t.Fatalf("first stop: %+v", rep)
	}
	q.limit("bbbb2222", resets)
	if rep := q.look(); rep.Began || rep.Woken != "" || len(q.sent) != 0 {
		t.Fatalf("a second stop is no news, and nothing is woken before the reset: %+v %v", rep, q.sent)
	}
	q.now = resets.Add(30 * time.Second)
	if rep := q.look(); rep.Woken != "" {
		t.Fatalf("woken inside the margin: %+v", rep)
	}

	q.now = resets.Add(2 * time.Minute)
	q.lag = 2
	rep = q.look()
	if rep.Woken != "task-a" || rep.Ended || rep.Problem != "" {
		t.Fatalf("first wake: %+v", rep)
	}
	if len(q.sent) != 1 || q.sent[0] != "aaaa1111: go on with task-a" {
		t.Fatalf("sent %v", q.sent)
	}
	if got := q.ranIn[len(q.ranIn)-1]; !strings.HasPrefix(got, filepath.Join(q.root, "p")+" | claude --bg --resume s-aaaa1111 ") {
		t.Fatalf("resumed with %q", got)
	}
	rep = q.look()
	if rep.Woken != "task-b" || !rep.Ended || rep.Count != 2 {
		t.Fatalf("second wake: %+v", rep)
	}
	if rep := q.look(); rep != (QuotaReport{}) {
		t.Fatalf("all woken, yet %+v", rep)
	}
	if q.pid["cccc3333"] == 0 || len(q.sent) != 2 {
		t.Fatalf("a session that never stopped was touched: %v %v", q.pid, q.sent)
	}

	// It stops again, with another reset: a new wait, told again.
	again := q.now.Add(time.Hour)
	q.limit("aaaa1111", again)
	if rep := q.look(); !rep.Began || !rep.Until.Equal(again) {
		t.Fatalf("second wait: %+v", rep)
	}
	q.now = again.Add(time.Hour)
	if rep := q.look(); rep.Woken != "task-a" || !rep.Ended || rep.Count != 1 {
		t.Fatalf("second wait's wake: %+v", rep)
	}
}

func TestWakeQuotaLeavesOthers(t *testing.T) {
	q := newQuotaFake(t)
	q.session("task-a", "aaaa1111")
	resets := q.now.Add(-time.Hour)

	// A session with no record, stopped by the limit too: never touched.
	q.pid["dddd4444"] = 300
	q.limit("dddd4444", resets)
	if rep := q.look(); rep != (QuotaReport{}) {
		t.Fatalf("a session of no task: %+v", rep)
	}

	// The user wrote to it after the stop: it goes on by itself.
	q.limit("aaaa1111", resets)
	q.transcript("aaaa1111", `{"type":"user","message":{"role":"user","content":"go on"}}`)
	if rep := q.look(); rep != (QuotaReport{}) || len(q.sent) != 0 {
		t.Fatalf("written to after the stop: %+v", rep)
	}

	// Someone is at it: left for the next look.
	q.limit("aaaa1111", resets)
	q.busy["aaaa1111"] = 1
	if rep := q.look(); rep.Woken != "" || len(q.sent) != 0 {
		t.Fatalf("a busy session: %+v", rep)
	}
	q.busy["aaaa1111"] = 0

	// Retired by Claude: no stop, resumed as it is.
	q.pid["aaaa1111"] = 0
	rep := q.look()
	if rep.Woken != "task-a" || !rep.Ended {
		t.Fatalf("a retired one: %+v", rep)
	}
	for _, call := range q.calls {
		if strings.Contains(call, " stop ") {
			t.Fatalf("stopped a session with no process: %v", q.calls)
		}
	}
}

func TestWakeQuotaProblems(t *testing.T) {
	q := newQuotaFake(t)
	q.session("task-a", "aaaa1111")
	q.limit("aaaa1111", q.now.Add(-time.Hour))

	// The process never goes: not resumed (Claude would start a copy), tried again.
	q.lag = 1000
	rep := q.look()
	if rep.Woken != "" || rep.Task != "task-a" || rep.Problem == "" || rep.GaveUp || len(q.ranIn) != 0 {
		t.Fatalf("a process that stays: %+v %v", rep, q.ranIn)
	}
	q.lag = 0

	// Claude starts a copy: it is removed, and that is said.
	q.copyAs = "c0ffee00"
	rep = q.look()
	if rep.Woken != "" || !strings.Contains(rep.Problem, "c0ffee00") {
		t.Fatalf("a copy: %+v", rep)
	}
	if !strings.Contains(strings.Join(q.calls, "\n"), "rm c0ffee00") {
		t.Fatalf("the copy stays: %v", q.calls)
	}

	// Logged out: the third try, and the last.
	q.copyAs = ""
	q.auth = `{"loggedIn": false}`
	rep = q.look()
	if rep.Problem == "" || !rep.GaveUp {
		t.Fatalf("logged out: %+v", rep)
	}
	q.auth = ""
	if rep := q.look(); rep != (QuotaReport{}) || len(q.sent) != 0 {
		t.Fatalf("tried a fourth time: %+v", rep)
	}
}

func TestWakeQuotaOnlyOnLinux(t *testing.T) {
	q := newQuotaFake(t)
	q.sys.GOOS = "darwin"
	c, _ := q.load()
	if _, err := q.sys.WakeQuota(c, func(string) string { return "" }); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("err = %v", err)
	}
}

// After the list shows the session with no process, the daemon's own word is waited
// for: resumed before it, Claude starts a copy (14-nov).
func TestWakeQuotaWaitsForTheDaemon(t *testing.T) {
	q := newQuotaFake(t)
	q.session("task-a", "aaaa1111")
	q.limit("aaaa1111", q.now.Add(-time.Hour))
	if rep := q.look(); rep.Woken != "task-a" {
		t.Fatalf("no log: %+v", rep)
	}
	if want := 1 + int(quotaSettleWait/time.Second); q.slept != want {
		t.Fatalf("with no log it waited %d times, want %d", q.slept, want)
	}

	q.session("task-b", "bbbb2222")
	q.limit("bbbb2222", q.now.Add(-time.Hour))
	log := "[2020-01-01T00:00:00.000Z] [bg] bg settled bbbb2222 (killed)\n" + // an earlier stop
		"[" + time.Now().UTC().Format("2006-01-02T15:04:05.000Z") + "] [bg] bg settled bbbb2222 (killed)\n"
	if err := os.WriteFile(filepath.Join(q.sys.ClaudeDir, daemonLog), []byte(log), 0o600); err != nil {
		t.Fatal(err)
	}
	q.slept = 0
	if rep := q.look(); rep.Woken != "task-b" {
		t.Fatalf("with the log: %+v", rep)
	}
	if q.slept != 1 {
		t.Fatalf("the daemon had said so, yet it waited %d times", q.slept)
	}
	if q.sys.daemonSettled("bbbb2222", time.Now().Add(time.Hour)) || q.sys.daemonSettled("cccc3333", time.Time{}) {
		t.Fatal("an earlier line, or another session's, counts")
	}
}
