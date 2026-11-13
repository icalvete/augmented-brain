package hook

import (
	"io"
	"path/filepath"
	"time"

	"github.com/icalvete/augmented-brain/internal/backup"
	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/filelock"
	"github.com/icalvete/augmented-brain/internal/fresh"
)

// FreshWait is the most UserPromptSubmit waits for the notes lock before telling the
// session what changed: the user is waiting for the answer. When a tool or a backup
// holds it for longer, the check is left for the next message.
var FreshWait = 500 * time.Millisecond

// freshBaseline records every task of the space as seen by this Claude process, right
// before the session is given the index (package fresh). A resumed or compacted
// session keeps the tasks it had touched; a new, cleared or forked one starts afresh.
func freshBaseline(c *config.Config, s *config.Space, i input, cfgDir string) {
	key := fresh.HookKey(Getenv)
	if key == "" || s == nil {
		return
	}
	unlock, err := filelock.LockWithin(filepath.Join(cfgDir, backup.NotesLock), LockWait)
	if err != nil {
		return // the next check takes one
	}
	defer unlock()
	now := Now()
	fresh.Prune(cfgDir, now)
	keep := i.Source == "resume" || i.Source == "compact"
	_ = fresh.Baseline(cfgDir, key, i.SessionID, fresh.SpaceOf(s.Name, s.Memory), now, keep)
}

// UserPromptSubmit runs before the assistant answers a message (the user's, another
// session's, a background task's): it tells the assistant which tasks changed outside
// this session since it last looked, so it does not act on an index read hours ago.
// It never blocks the message: any problem means nothing is said.
func UserPromptSubmit(in io.Reader, out io.Writer, cfgDir string) error {
	i := read(in)
	c, s, err := load(cfgDir, i.Cwd)
	if err != nil {
		return nil
	}
	notice := userNotice(c.Language, cfgDir)
	say := func(text string) error {
		if text == "" && notice == "" {
			return nil
		}
		o := output{SystemMessage: notice}
		if text != "" {
			o.HookSpecificOutput = specific{HookEventName: "UserPromptSubmit", AdditionalContext: text}
		}
		return write(out, o)
	}
	key := fresh.HookKey(Getenv)
	if key == "" || s == nil {
		return say("")
	}
	unlock, err := filelock.LockWithin(filepath.Join(cfgDir, backup.NotesLock), FreshWait)
	if err != nil {
		return say("")
	}
	r, err := fresh.Check(cfgDir, key, fresh.SpaceOf(s.Name, s.Memory), Now())
	unlock()
	if err != nil {
		return say("")
	}
	return say(fresh.Text(c.Language, r))
}
