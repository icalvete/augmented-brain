package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/icalvete/augmented-brain/internal/backup"
	"github.com/icalvete/augmented-brain/internal/brain"
	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/filelock"
	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/replywait"
	"github.com/icalvete/augmented-brain/internal/telegramprog"
)

// maxReplyText bounds each reply written into a journal: it is someone else's text,
// and a task's journal is not an archive of a chat.
const maxReplyText = 1000

// telegramRead reads the messages of a chat after a message; a variable, for tests.
var telegramRead = func(ctx context.Context, path, chat string, after int) ([]replywait.Message, error) {
	var r struct {
		Messages []replywait.Message `json:"messages"`
	}
	err := telegramprog.Call(ctx, path, []string{"read", "--chat", chat, "--after", strconv.Itoa(after), "--limit", "100"}, "", &r)
	return r.Messages, err
}

// telegramCheck looks for the replies tasks wait for, and brings each to its task:
// the timer runs it every few minutes. With no wait it does not connect at all.
func telegramCheck(c *config.Config, out, errOut io.Writer) int {
	l := c.Language
	waits, err := replywait.List(c.Dir)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	if len(waits) == 0 {
		return 0
	}
	path := telegramPath()
	if path == "" {
		return 0 // Telegram is off: the waits stay for when it is on again
	}
	failed := false
	for _, w := range waits {
		if !waitLive(c, w) {
			if _, err := replywait.Remove(c.Dir, w.Same); err != nil {
				fmt.Fprintln(errOut, err)
				failed = true
				continue
			}
			fmt.Fprintln(out, i18n.T(l, "telegram.wait.dropped", w.Task, w.Chat))
			continue
		}
		msgs, err := telegramRead(context.Background(), path, w.Chat, w.After)
		if errors.Is(err, telegramprog.ErrNotConnected) {
			return 0 // logged out: nothing can be read until it connects again
		}
		if err != nil {
			fmt.Fprintf(errOut, "%s (%s): %v\n", w.Task, w.Chat, err)
			failed = true
			continue
		}
		replies := w.Replies(msgs)
		if len(replies) == 0 {
			continue
		}
		who := replyWho(l, w, replies)
		if err := replyNote(c, w, who, replies); err != nil {
			fmt.Fprintf(errOut, "%s: %v\n", w.Task, err)
			failed = true
			continue
		}
		if _, err := replywait.Remove(c.Dir, w.Same); err != nil {
			fmt.Fprintln(errOut, err)
			failed = true
			continue
		}
		fmt.Fprintln(out, i18n.T(l, "telegram.reply.brought", w.Task, who))
	}
	if failed {
		return 1
	}
	return 0
}

// waitLive says whether the task of a wait is still there and not closed. A space that
// cannot be read keeps its waits: that may pass.
func waitLive(c *config.Config, w replywait.Wait) bool {
	s, ok := c.Spaces[w.Space]
	if !ok {
		return false
	}
	b, err := brain.Open(c, s)
	if err != nil {
		return true
	}
	t, err := b.Task(w.Task)
	return err == nil && t.Header.Status != "closed"
}

// replyWho names who answered: the person the wait names, or who wrote the replies.
func replyWho(l string, w replywait.Wait, replies []replywait.Message) string {
	if w.Who != "" {
		return w.Who
	}
	var names []string
	for _, m := range replies {
		if m.From != "" && !slices.Contains(names, m.From) {
			names = append(names, m.From)
		}
	}
	if len(names) == 0 {
		return i18n.T(l, "telegram.reply.someone")
	}
	return strings.Join(names, ", ")
}

// replyText is the replies in one line, each with its time and, in a group, its author,
// cut to maxReplyText.
func replyText(w replywait.Wait, replies []replywait.Message) string {
	var parts []string
	for _, m := range replies {
		t := strings.Join(strings.Fields(m.Text), " ")
		if r := []rune(t); len(r) > maxReplyText {
			t = string(r[:maxReplyText]) + "…"
		}
		if w.Group() && m.From != "" {
			t = m.From + ": " + t
		}
		parts = append(parts, t)
	}
	return strings.Join(parts, " / ")
}

// replyNote writes the replies in the task: a journal line marked as someone else's
// text, waiting on the user again, and the next step to read it. Every other session
// hears of the change before its next answer (package fresh).
func replyNote(c *config.Config, w replywait.Wait, who string, replies []replywait.Message) error {
	unlock, err := filelock.Lock(filepath.Join(c.Dir, backup.NotesLock))
	if err != nil {
		return err
	}
	defer unlock()
	b, err := brain.Open(c, c.Spaces[w.Space])
	if err != nil {
		return err
	}
	when := replies[len(replies)-1].Date.Local().Format("2006-01-02 15:04")
	if _, err := b.Journal(w.Task, i18n.T(c.Language, "telegram.reply.journal", who, when, replyText(w, replies))); err != nil {
		return err
	}
	_, err = b.Update(brain.UpdateInput{ID: w.Task, WaitingOn: "me", Next: i18n.T(c.Language, "telegram.reply.next", who)})
	return err
}

func telegramWaitsFlags(errOut io.Writer) (*flag.FlagSet, *string) {
	fs := flag.NewFlagSet("telegram waits", flag.ContinueOnError)
	fs.SetOutput(errOut)
	drop := fs.String("drop", "", "take out the waits of this task")
	return fs, drop
}

// telegramWaits lists the replies tasks wait for, or takes out a task's.
func telegramWaits(c *config.Config, args []string, out, errOut io.Writer) int {
	l := c.Language
	fs, drop := telegramWaitsFlags(errOut)
	if rest, ok := parseArgs(fs, args, 0, errOut); !ok || len(rest) != 0 {
		fmt.Fprintln(errOut, i18n.T(l, "usage.telegram"))
		return 2
	}
	if *drop != "" {
		n, err := replywait.Remove(c.Dir, func(w replywait.Wait) bool { return w.Task == *drop })
		if err != nil {
			fmt.Fprintln(errOut, err)
			return 1
		}
		fmt.Fprintln(out, i18n.T(l, "telegram.waits.removed", n))
		return 0
	}
	waits, err := replywait.List(c.Dir)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	if len(waits) == 0 {
		fmt.Fprintln(out, i18n.T(l, "telegram.waits.none"))
		return 0
	}
	for _, w := range waits {
		who := w.Who
		if who == "" {
			who = i18n.T(l, "telegram.reply.someone")
		}
		fmt.Fprintln(out, i18n.T(l, "telegram.waits.line", w.Task, w.Space, who, w.Chat, w.Since.Local().Format("2006-01-02 15:04")))
	}
	return 0
}
