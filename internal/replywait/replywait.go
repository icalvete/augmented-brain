// Package replywait keeps the replies a task waits for on Telegram: a message the
// assistant sent that asks for an answer. A wait is data, not a process: it lasts until
// the answer comes or someone takes it out, however long people take to reply, and a
// timer looks for the answers (augmented-brain telegram check).
//
// The waits live in the configuration folder, never in the notes: what a task shows of
// one is a line of its journal and who it waits on. Losing this machine loses the wait,
// never the task.
package replywait

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/icalvete/augmented-brain/internal/filelock"
	"github.com/icalvete/augmented-brain/internal/paths"
)

// File is the waits' file, in the state folder of the configuration folder.
const File = "telegram-waits.json"

// Message is one message as augmented-brain-telegram's read gives it. This package
// never imports Telegram's library: augmented brain calls that program instead.
type Message struct {
	ID      int       `json:"id"`
	Date    time.Time `json:"date"`
	From    string    `json:"from"`
	Out     bool      `json:"out"`
	Text    string    `json:"text"`
	ReplyTo int       `json:"reply_to,omitempty"`
}

// Wait is one reply a task waits for.
type Wait struct {
	Space string    `json:"space"`
	Task  string    `json:"task"`
	Chat  string    `json:"chat"`
	After int       `json:"after"` // the id of the message sent: the reply comes after it
	Who   string    `json:"who,omitempty"`
	Since time.Time `json:"since"`
}

// Same says whether two waits are the same: one task, one chat.
func (w Wait) Same(o Wait) bool {
	return w.Space == o.Space && w.Task == o.Task && w.Chat == o.Chat
}

// Group says whether the chat is a group or a channel, where a reply is only a message
// that answers the one sent: anyone else's message there is not for this task.
func (w Wait) Group() bool {
	return strings.HasPrefix(w.Chat, "group:") || strings.HasPrefix(w.Chat, "channel:")
}

// Replies picks, among the messages of the chat after the one sent, those that answer
// it: in a chat with one person, every message that came in; in a group, those sent as
// an answer to it or to a message of its thread.
func (w Wait) Replies(msgs []Message) []Message {
	var out []Message
	thread := map[int]bool{w.After: true} // in a group: the message sent and what answers it
	for _, m := range msgs {
		if m.ID <= w.After {
			continue
		}
		if w.Group() {
			if !thread[m.ReplyTo] {
				continue
			}
			thread[m.ID] = true
		}
		if !m.Out {
			out = append(out, m)
		}
	}
	return out
}

func path(dir string) string { return filepath.Join(dir, "state", File) }

// with runs f on the waits under the lock, and writes them back when f says so.
func with(dir string, f func(ws []Wait) ([]Wait, bool)) error {
	unlock, err := filelock.Lock(path(dir) + ".lock")
	if err != nil {
		return err
	}
	defer unlock()
	ws, err := read(dir)
	if err != nil {
		return err
	}
	ws, write := f(ws)
	if !write {
		return nil
	}
	if ws == nil {
		ws = []Wait{}
	}
	b, err := json.MarshalIndent(ws, "", "  ")
	if err != nil {
		return err
	}
	return paths.WriteAtomic(path(dir), append(b, '\n'), 0o600)
}

func read(dir string) ([]Wait, error) {
	b, err := os.ReadFile(path(dir))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var ws []Wait
	if err := json.Unmarshal(b, &ws); err != nil {
		return nil, err
	}
	return ws, nil
}

// List gives the waits, oldest first.
func List(dir string) ([]Wait, error) {
	var out []Wait
	err := with(dir, func(ws []Wait) ([]Wait, bool) {
		out = ws
		return ws, false
	})
	return out, err
}

// Add keeps a wait. A wait of the same task in the same chat is replaced: the reply
// awaited is the one to the last message.
func Add(dir string, w Wait) error {
	if w.Since.IsZero() {
		w.Since = time.Now()
	}
	return with(dir, func(ws []Wait) ([]Wait, bool) {
		ws = slices.DeleteFunc(ws, w.Same)
		return append(ws, w), true
	})
}

// Remove takes out the waits for which drop is true, and says how many went.
func Remove(dir string, drop func(Wait) bool) (int, error) {
	n := 0
	err := with(dir, func(ws []Wait) ([]Wait, bool) {
		before := len(ws)
		ws = slices.DeleteFunc(ws, drop)
		n = before - len(ws)
		return ws, n > 0
	})
	return n, err
}
