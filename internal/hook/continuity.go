package hook

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/icalvete/augmented-brain/internal/backup"
	"github.com/icalvete/augmented-brain/internal/brain"
	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/filelock"
	"github.com/icalvete/augmented-brain/internal/guard"
	"github.com/icalvete/augmented-brain/internal/state"
)

// Now gives the time; tests replace it.
var Now = time.Now

// RemindAfter is how long the active task may go without a journal entry before the
// Stop hook reminds the assistant to write one.
var RemindAfter = 20 * time.Minute

// Stop runs when the assistant finishes an answer. When a task is active and nothing
// has been written to it for a while, it asks the assistant to write in its journal
// what was done since, so a session that dies loses little. It never reminds twice
// without a write in between, never while already continuing because of a Stop hook,
// and it reads one small file: it takes milliseconds.
func Stop(in io.Reader, out io.Writer, cfgDir string) error {
	i := read(in)
	if i.StopHookActive {
		return nil
	}
	c, s := ownSpace(cfgDir, i.Cwd)
	if s == nil {
		return nil
	}
	now := Now()
	st := state.Read(cfgDir, s.Name)
	if st.Task == "" || now.Sub(st.Since) < RemindAfter || !st.Reminded.Before(st.Since) {
		return nil
	}
	if err := state.Update(cfgDir, s.Name, now, func(x *state.Space) { x.Reminded = now }); err != nil {
		return err
	}
	text := fmt.Sprintf("augmented brain: nothing has been written in the journal of task %s since %s. "+
		"If anything was done, decided or found since then, add it now in one short entry with task_journal, "+
		"in the language the user writes in (%s if you cannot tell), then stop. If nothing worth keeping happened, just stop, without saying anything.",
		st.Task, st.Since.Local().Format("15:04"), language(c.Language))
	return write(out, output{specific{HookEventName: "Stop", AdditionalContext: text}})
}

// PostCompact keeps the summary Claude Code wrote when it compacted the conversation:
// in archive/sessions/ of the space (never loaded at the start, backed up like any
// note, secrets taken out), with a line in the journal of the active task pointing at it.
func PostCompact(in io.Reader, cfgDir string) error {
	i := read(in)
	if strings.TrimSpace(i.CompactSummary) == "" {
		return nil
	}
	c, s := ownSpace(cfgDir, i.Cwd)
	if s == nil {
		return nil
	}
	now := Now()
	id := i.SessionID
	if len(id) > 8 {
		id = id[:8]
	}
	if id == "" {
		id = "session"
	}
	rel := fmt.Sprintf("archive/sessions/%s-%s-%s.md", now.Format("2006-01-02"), now.Format("150405"), id)
	head := map[string]string{"en": "# Conversation summary, %s\n\nWritten by Claude Code when it compacted the conversation (%s).\n\n",
		"es": "# Resumen de la conversación, %s\n\nLo escribió Claude Code al compactar la conversación (%s).\n\n"}[c.Language]
	if head == "" {
		head = "# Conversation summary, %s\n\nWritten by Claude Code when it compacted the conversation (%s).\n\n"
	}
	text := fmt.Sprintf(head, now.Format("2006-01-02 15:04"), i.Trigger) + guard.RedactSecrets(strings.TrimSpace(i.CompactSummary)) + "\n"

	unlock, err := filelock.Lock(filepath.Join(cfgDir, backup.NotesLock))
	if err != nil {
		return err
	}
	defer unlock()
	path := filepath.Join(s.Memory, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		return err
	}
	task := state.Read(cfgDir, s.Name).Task
	if task == "" {
		return nil
	}
	b, err := brain.Open(c, s)
	if err != nil {
		return err
	}
	b.Now = Now
	if _, err := b.JournalText("compacted", task, rel); err != nil {
		return nil // the task is gone or broken: the summary is kept all the same
	}
	return state.Touched(cfgDir, s.Name, task, now, true)
}

// ownSpace is the space whose work folder holds dir, never the default space: what the
// conversation leaves (summaries, session records) must not land in the personal space
// just because a session ran in some other folder.
func ownSpace(cfgDir, dir string) (*config.Config, *config.Space) {
	c, err := config.Load(cfgDir)
	if err != nil {
		return nil, nil
	}
	return c, c.SpaceFor(dir)
}

// compactNote is added to the session text right after a compaction: the details of
// what came before are gone from the conversation, so what matters goes to the journal
// now, while the summary is still there.
func compactNote(lang, task string) string {
	where := "in the journal of the task you were working on (task_journal), if any"
	if task != "" {
		where = "in the journal of task " + task + " (task_journal)"
	}
	return fmt.Sprintf("\nThe conversation was just compacted: its details are gone, only a summary is left. Before going on, write %s "+
		"what was decided, found or done since its last entry, in one or two short entries, in the language the user writes in (%s if you cannot tell). "+
		"Then carry on.\n", where, language(lang))
}

// recoverNote offers to recover a session of the space that never ended (it was closed
// by force, or the machine went down), from its transcript, still on this machine.
func recoverNote(lang string, se state.Session) string {
	task := ""
	if se.Task != "" {
		task = ", working on task " + se.Task
	}
	return fmt.Sprintf("\nA previous session of this space, started %s%s, did not end normally (it may have been closed by force), "+
		"so what was said after its last journal entry may be missing from the notes. In one short line of your first answer, in the language the user "+
		"writes in (%s if you cannot tell), offer to recover it. If they agree, call session_recover, read it, and write in the task's journal what is missing.\n",
		se.Started.Local().Format("2006-01-02 15:04"), task, language(lang))
}

func language(lang string) string {
	if lang == "es" {
		return "Spanish"
	}
	return "English"
}

// sessionStarted records the session in the space's state and returns what the session
// text should add: after a compaction, the request to journal; on a new start, the offer
// to recover a session that never ended.
func sessionStarted(c *config.Config, s *config.Space, i input, cfgDir string) string {
	if c.SpaceFor(i.Cwd) != s {
		return "" // the default space: not tracked (see ownSpace)
	}
	now := Now()
	st := state.Read(cfgDir, s.Name)
	note := ""
	if i.Source == "compact" {
		note += compactNote(c.Language, st.Task)
	}
	if i.Source == "startup" || i.Source == "" {
		for _, se := range st.Unclosed(i.SessionID, now) {
			if se.Reported {
				continue
			}
			note += recoverNote(c.Language, se)
			id := se.ID
			_ = state.Update(cfgDir, s.Name, now, func(x *state.Space) {
				for k := range x.Sessions {
					if x.Sessions[k].ID == id {
						x.Sessions[k].Reported = true
					}
				}
			})
			break
		}
	}
	_ = state.Started(cfgDir, s.Name, i.SessionID, i.TranscriptPath, now)
	return note
}

// askSafety, after a restore that brought no guard lists (no backup carries them, as
// they may name work machines), asks the assistant once to offer the safety questions.
func askSafety(c *config.Config, cfgDir string) string {
	marker := filepath.Join(cfgDir, config.AskSafetyFile)
	if _, err := os.Stat(marker); err != nil || brain.ProfileEmpty(cfgDir) {
		return "" // a blank profile means the first-session interview, which asks anyway
	}
	_ = os.Remove(marker)
	if profiles, hosts := c.GuardLists(); len(profiles)+len(hosts) > 0 {
		return ""
	}
	return fmt.Sprintf("\nThis machine was set up from a backup, and backups do not carry the guard's lists (production AWS profiles, untouchable hosts). "+
		"In one short line of your first answer, in the language the user writes in (%s if you cannot tell), offer to ask for them again; "+
		"if they agree, ask which AWS profiles reach production and which machines must not be touched, and save them with safety_save.\n", language(c.Language))
}
