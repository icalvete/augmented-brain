package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/icalvete/augmented-brain/internal/backup"
	"github.com/icalvete/augmented-brain/internal/brain"
	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/detach"
	"github.com/icalvete/augmented-brain/internal/filelock"
	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/memory"
	"github.com/icalvete/augmented-brain/internal/paths"
	"github.com/icalvete/augmented-brain/internal/remote"
	"github.com/icalvete/augmented-brain/internal/state"
	"github.com/icalvete/augmented-brain/internal/watch"
)

type watchOpts struct {
	task, repo string
	max, every time.Duration
}

func watchFlags(errOut io.Writer) (*flag.FlagSet, *watchOpts) {
	o := &watchOpts{}
	fs := flag.NewFlagSet("watch pr", flag.ContinueOnError)
	fs.SetOutput(errOut)
	fs.StringVar(&o.task, "task", "", "the task to write the result in: its journal and its next step")
	fs.StringVar(&o.repo, "repo", "", "the repository, OWNER/NAME (default: the one of this folder)")
	fs.DurationVar(&o.max, "max", 3*time.Hour, "the longest it watches")
	fs.DurationVar(&o.every, "every", 30*time.Second, "how often it looks")
	return fs, o
}

// watchGh is how watch runs gh: never a question on the terminal, and a minute at most
// for each call, so a gh that hangs is a failed look, not a watch that never ends.
var watchGh = func(args ...string) (string, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "gh", args...)
	cmd.Env = append(os.Environ(), "GH_PROMPT_DISABLED=1", "GH_NO_UPDATE_NOTIFIER=1", "NO_COLOR=1")
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	err := cmd.Run()
	return out.String(), errOut.String(), err
}

// watchSleep, watchNow and watchLookPath are there for the tests.
var (
	watchSleep    = time.Sleep
	watchNow      = time.Now
	watchLookPath = exec.LookPath
)

// cmdWatch is `watch pr NUMBER`: it follows a pull request until it is merged or
// closed, a required check fails, every required check has finished, or --max passes,
// and then says so: on its output, with a line that has the assistant tell the user,
// and, with --task, in the task itself, so the news does not depend on the session
// that started it being alive or remembering it.
func cmdWatch(args []string, out, errOut io.Writer) int {
	l := userLanguage()
	fs, o := watchFlags(errOut)
	rest, ok := parseArgs(fs, args, 2, errOut)
	if !ok {
		return 2
	}
	n := 0
	if len(rest) == 2 && rest[0] == "pr" {
		n, _ = strconv.Atoi(strings.TrimPrefix(rest[1], "#"))
	}
	if n <= 0 || o.max <= 0 || o.every < minWatchEvery {
		fmt.Fprintln(errOut, i18n.T(l, "usage.watch"))
		return 2
	}
	var c *config.Config
	var s *config.Space
	if o.task != "" {
		// The task is checked before the wait: a typo found three hours later is no use.
		if c, s, ok = currentSpace(errOut); !ok {
			return 1
		}
		l = c.Language
		b, err := brain.Open(c, s)
		if err != nil {
			fmt.Fprintln(errOut, err)
			return 1
		}
		if _, err := b.Task(o.task); err != nil {
			fmt.Fprintln(errOut, i18n.T(l, "session.notask", o.task, s.Name))
			return 1
		}
	}
	if _, err := watchLookPath("gh"); err != nil {
		fmt.Fprintln(errOut, i18n.T(l, "watch.nogh"))
		return 1
	}
	if result := os.Getenv(watchKeeperEnv); result != "" && o.task != "" {
		return watchKeep(c, s, l, o, n, result, out)
	}
	fmt.Fprintln(out, i18n.T(l, "watch.start", n, minutes(l, o.max)))
	if o.task != "" {
		return watchThroughKeeper(c, l, o, n, append([]string{"watch"}, args...), out, errOut)
	}
	r := watchRun(o, n)
	fmt.Fprintln(out, watchLine(l, r))
	fmt.Fprintln(out, i18n.T(l, "watch.tell"))
	return watchCode(r)
}

func watchRun(o *watchOpts, n int) watch.Result {
	return watch.PR(watch.Options{
		PR: n, Repo: o.repo, Every: o.every, Max: o.max, Grace: 5 * time.Minute,
		Run: watchGh, Sleep: watchSleep, Now: watchNow,
	})
}

func watchCode(r watch.Result) int {
	if r.Kind == watch.Green || r.Kind == watch.Merged {
		return 0
	}
	return 1
}

// With --task the watching is done by a keeper: the same command, started detached
// and, on Linux, in a scope of its own, so it outlives the session that asked for it
// (a session closed, restarted or retired by Claude took its background commands with
// it: half the watches lost from 8 to 12-nov). The keeper writes the result in the task
// and in a result file; the command the session runs only waits for that file and
// prints it, so a session still alive is woken as before.
const watchKeeperEnv = "AB_WATCH_RESULT"

// watchesDir holds, per task and pull request, the keeper's pid, result and log.
const watchesDir = "watches"

// watchStart starts the keeper and returns a channel closed when it ends. Tests replace it.
var watchStart = func(args []string, env []string, log string) (int, <-chan struct{}, error) {
	self, err := os.Executable()
	if err != nil {
		return 0, nil, err
	}
	name, rest := remote.Scoped(self, args...)
	cmd := exec.Command(name, rest...)
	cmd.Env = append(os.Environ(), env...)
	cmd.Stdin = nil
	if f, err := os.OpenFile(log, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600); err == nil {
		defer f.Close() // the child keeps its own copy
		cmd.Stdout, cmd.Stderr = f, f
	}
	detach.Cmd(cmd)
	started, err := detach.Start(cmd)
	if err != nil {
		return 0, nil, err
	}
	done := make(chan struct{})
	go func() { _ = started.Wait(); close(done) }()
	return started.Process.Pid, done, nil
}

// watchKeep is the keeper: it watches, writes the task, then the result file (first
// line the exit code, then what to print).
func watchKeep(c *config.Config, s *config.Space, l string, o *watchOpts, n int, result string, out io.Writer) int {
	r := watchRun(o, n)
	line := watchLine(l, r)
	lines := []string{strconv.Itoa(watchCode(r)), line}
	if err := watchNote(c, s, o.task, r.PR, line, watchNext(l, r)); err != nil {
		lines = append(lines, i18n.T(l, "watch.unsaved", o.task, err))
	} else {
		lines = append(lines, i18n.T(l, "watch.saved", o.task))
	}
	fmt.Fprintln(out, strings.Join(lines[1:], "\n"))
	// The pid goes first: once the result is there, nobody should wait on this process.
	_ = os.Remove(strings.TrimSuffix(result, ".result") + ".pid")
	if err := paths.WriteAtomic(result, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		fmt.Fprintln(out, err)
		return 1
	}
	return watchCode(r)
}

// watchThroughKeeper starts the keeper, or joins the one already watching this pull
// request for this task, and waits for its result.
func watchThroughKeeper(c *config.Config, l string, o *watchOpts, n int, args []string, out, errOut io.Writer) int {
	dir := filepath.Join(c.Dir, state.Dir, watchesDir)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	base := filepath.Join(dir, fmt.Sprintf("%s-pr%d", o.task, n))
	result, pidFile, log := base+".result", base+".pid", base+".log"
	var done <-chan struct{}
	if pid, err := readPid(pidFile); err == nil && processAlive(pid) {
		fmt.Fprintln(out, i18n.T(l, "watch.joined", pid))
	} else {
		_ = os.Remove(result)
		pid, d, err := watchStart(args, []string{watchKeeperEnv + "=" + result}, log)
		if err != nil {
			fmt.Fprintln(errOut, err)
			return 1
		}
		done = d
		_ = paths.WriteAtomic(pidFile, []byte(strconv.Itoa(pid)+"\n"), 0o600)
	}
	// The keeper ends by --max; a little more covers its last look and the note.
	deadline := watchNow().Add(o.max + 10*time.Minute)
	for {
		if data, err := os.ReadFile(result); err == nil {
			// Done: a later watch must not take this pid, maybe reused, for a keeper.
			_ = os.Remove(pidFile)
			code, text, _ := strings.Cut(strings.TrimRight(string(data), "\n"), "\n")
			fmt.Fprintln(out, text)
			fmt.Fprintln(out, i18n.T(l, "watch.tell"))
			if v, err := strconv.Atoi(code); err == nil {
				return v
			}
			return 1
		}
		ended := false
		if done != nil {
			select {
			case <-done:
				ended = true
			default:
			}
		} else if pid, err := readPid(pidFile); err != nil || !processAlive(pid) {
			ended = true
		}
		if ended {
			if _, err := os.Stat(result); err == nil {
				continue // it wrote the result right before ending
			}
			fmt.Fprintln(out, i18n.T(l, "watch.lost", tilde(log)))
			fmt.Fprintln(out, i18n.T(l, "watch.tell"))
			return 1
		}
		if watchNow().After(deadline) {
			fmt.Fprintln(out, i18n.T(l, "watch.lost", tilde(log)))
			fmt.Fprintln(out, i18n.T(l, "watch.tell"))
			return 1
		}
		watchSleep(2 * time.Second)
	}
}

func readPid(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(data)))
}

// minWatchEvery is how often watch may look at most: gh counts against the user's
// GitHub quota, and a check takes minutes anyway.
const minWatchEvery = 10 * time.Second

// What watch quotes from GitHub: the names of the checks, which whoever writes the
// pull request's workflows chooses, and what gh says when it fails.
const (
	maxCheckName = 60
	maxChecks    = 5
	maxProblem   = 200
)

// quoted is text that came from GitHub, as watch writes it: one line, cut short, and
// between quotes, so it reads as a name and never as an instruction of its own.
func quoted(s string, max int) string { return "\"" + memory.Clip(memory.OneLine(s), max) + "\"" }

// checkNames lists the names of some checks, at most maxChecks of them.
func checkNames(l string, names []string) string {
	var q []string
	for i, n := range names {
		if i == maxChecks {
			q = append(q, i18n.T(l, "watch.andmore", len(names)-maxChecks))
			break
		}
		q = append(q, quoted(n, maxCheckName))
	}
	return strings.Join(q, ", ")
}

// watchLine is what happened, in one line: the whole of it, with the names of the
// checks, for the output and the task's journal.
func watchLine(l string, r watch.Result) string { return watchText(l, r, true) }

// watchNext is what happened as the task's next step: a sentence of the program's own,
// with no names of checks and nothing gh said, which every session reads at the start;
// those are in the journal.
func watchNext(l string, r watch.Result) string { return watchText(l, r, false) }

func watchText(l string, r watch.Result, names bool) string {
	took := minutes(l, r.Elapsed)
	var line string
	switch r.Kind {
	case watch.Green:
		line = i18n.T(l, "watch.green", r.PR, took)
	case watch.Failed:
		if !names {
			return i18n.T(l, "watch.failed.next", r.PR, took)
		}
		line = i18n.T(l, "watch.failed", r.PR, checkNames(l, r.Failed), took)
	case watch.Merged:
		line = i18n.T(l, "watch.merged", r.PR, took)
	case watch.Closed:
		line = i18n.T(l, "watch.closed", r.PR, took)
	case watch.Conflict:
		line = i18n.T(l, "watch.conflict", r.PR, took)
	case watch.TimedOut:
		if !names {
			return i18n.T(l, "watch.timeout.next", r.PR, took)
		}
		pending := checkNames(l, r.Pending)
		if pending == "" {
			pending = quoted(r.Problem, maxProblem)
		}
		line = i18n.T(l, "watch.timeout", r.PR, took, pending)
	default:
		if !names {
			return i18n.T(l, "watch.broken.next", r.PR, took)
		}
		line = i18n.T(l, "watch.broken", r.PR, quoted(r.Problem, maxProblem), took)
	}
	if len(r.Others) > 0 && names {
		line += i18n.T(l, "watch.others", checkNames(l, r.Others))
	}
	return line
}

// watchNote writes the result in the task: a line in its journal (line, the whole of
// it), which keeps the next step it had, and the result as its next step (next, the
// program's own words), which every other session hears about before its next answer
// (package fresh) and Today shows.
func watchNote(c *config.Config, s *config.Space, task string, pr int, line, next string) error {
	unlock, err := filelock.Lock(filepath.Join(c.Dir, backup.NotesLock))
	if err != nil {
		return err
	}
	defer unlock()
	b, err := brain.Open(c, s)
	if err != nil {
		return err
	}
	t, err := b.Task(task)
	if err != nil {
		return err
	}
	if _, err := b.Journal(task, i18n.T(c.Language, "watch.journal", pr, line, t.Header.Next)); err != nil {
		return err
	}
	_, err = b.Update(brain.UpdateInput{ID: task, Next: next})
	return err
}

func minutes(l string, d time.Duration) string {
	return i18n.T(l, "watch.minutes", int(d.Round(time.Minute)/time.Minute))
}
