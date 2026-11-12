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
	"github.com/icalvete/augmented-brain/internal/filelock"
	"github.com/icalvete/augmented-brain/internal/i18n"
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
	if n <= 0 || o.max <= 0 || o.every <= 0 {
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
	fmt.Fprintln(out, i18n.T(l, "watch.start", n, minutes(l, o.max)))
	r := watch.PR(watch.Options{
		PR: n, Repo: o.repo, Every: o.every, Max: o.max, Grace: 5 * time.Minute,
		Run: watchGh, Sleep: watchSleep, Now: watchNow,
	})
	line := watchLine(l, r)
	fmt.Fprintln(out, line)
	if o.task != "" {
		if err := watchNote(c, s, o.task, r.PR, line); err != nil {
			fmt.Fprintln(errOut, i18n.T(l, "watch.unsaved", o.task, err))
		} else {
			fmt.Fprintln(out, i18n.T(l, "watch.saved", o.task))
		}
	}
	fmt.Fprintln(out, i18n.T(l, "watch.tell"))
	if r.Kind == watch.Green || r.Kind == watch.Merged {
		return 0
	}
	return 1
}

// watchLine is what happened, in one line.
func watchLine(l string, r watch.Result) string {
	took := minutes(l, r.Elapsed)
	var line string
	switch r.Kind {
	case watch.Green:
		line = i18n.T(l, "watch.green", r.PR, took)
	case watch.Failed:
		line = i18n.T(l, "watch.failed", r.PR, strings.Join(r.Failed, ", "), took)
	case watch.Merged:
		line = i18n.T(l, "watch.merged", r.PR, took)
	case watch.Closed:
		line = i18n.T(l, "watch.closed", r.PR, took)
	case watch.TimedOut:
		pending := strings.Join(r.Pending, ", ")
		if pending == "" {
			pending = r.Problem
		}
		line = i18n.T(l, "watch.timeout", r.PR, took, pending)
	default:
		line = i18n.T(l, "watch.broken", r.PR, r.Problem, took)
	}
	if len(r.Others) > 0 {
		line += i18n.T(l, "watch.others", strings.Join(r.Others, ", "))
	}
	return line
}

// watchNote writes the result in the task: a line in its journal, which keeps the
// next step it had, and the result as its next step, which every other session hears
// about before its next answer (package fresh) and Today shows.
func watchNote(c *config.Config, s *config.Space, task string, pr int, line string) error {
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
	_, err = b.Update(brain.UpdateInput{ID: task, Next: line})
	return err
}

func minutes(l string, d time.Duration) string {
	return i18n.T(l, "watch.minutes", int(d.Round(time.Minute)/time.Minute))
}
