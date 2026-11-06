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
	"github.com/icalvete/augmented-brain/internal/hostmem"
	"github.com/icalvete/augmented-brain/internal/i18n"
	"github.com/icalvete/augmented-brain/internal/session"
	"github.com/icalvete/augmented-brain/internal/state"
	"github.com/icalvete/augmented-brain/internal/version"
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
//
// It also looks at the memory of the process that hosts the session (memoryNote), and
// has the assistant warn the user when it has grown far past normal.
func Stop(in io.Reader, out io.Writer, cfgDir string) error {
	i := read(in)
	if i.StopHookActive {
		return nil
	}
	c, err := config.Load(cfgDir)
	if err != nil {
		return nil
	}
	mem := memoryNote(c.Language, cfgDir)
	model := modelStopNote(c.Language, cfgDir, i)
	journal, err := journalNote(c, cfgDir, i.Cwd, mem != "" || model != "")
	if err != nil {
		return err
	}
	var notes []string
	for _, n := range []string{journal, model, mem} {
		if n != "" {
			notes = append(notes, n)
		}
	}
	text := strings.Join(notes, "\n\n")
	if text == "" {
		return nil
	}
	return write(out, output{specific{HookEventName: "Stop", AdditionalContext: text}})
}

// journalNote is the reminder to write in the active task's journal, when it is due,
// and records that it was given. With more, another note follows it that the
// assistant must pass on, so it is not asked to end its turn in silence.
func journalNote(c *config.Config, cfgDir, dir string, more bool) (string, error) {
	s := c.SpaceFor(dir)
	if s == nil {
		return "", nil
	}
	now := Now()
	st := state.Read(cfgDir, s.Name)
	if st.Task == "" || now.Sub(st.Since) < RemindAfter || !st.Reminded.Before(st.Since) {
		return "", nil
	}
	if err := state.Update(cfgDir, s.Name, now, func(x *state.Space) { x.Reminded = now }); err != nil {
		return "", err
	}
	end := "Either way, this is not a message from the user and your answer to them is already given: " +
		"end your turn with no text at all, not even to say that nothing changed or that you wrote the entry."
	if more {
		end = "Either way, this is not a message from the user: say nothing about the journal, not even that you wrote the entry; only pass on the note that follows."
	}
	return fmt.Sprintf("augmented brain: nothing has been written in the journal of task %s since %s. "+
		"If anything was done, decided or found since then, add it now in one short entry with task_journal, "+
		"%s. %s", st.Task, st.Since.Local().Format("15:04"), session.InUserLanguage(c.Language), end), nil
}

// Processes is the machine whose processes memoryNote looks at; tests replace it.
var Processes = hostmem.Local

// ParentPID is the hook's parent process, where the walk up to claude starts; tests
// replace it.
var ParentPID = os.Getppid

// memoryNote asks the assistant to warn the user when the process that hosts this
// session (VS Code's extension host, or claude in a terminal) has crossed a step of
// hostmem.Steps it was not warned about yet. A host is named by its pid and start, so
// a reused pid is a new host. The warning is given only once its mark is written, so a
// state that cannot be written never repeats it after every answer. On Linux it reads
// a few small files of /proc: well under a millisecond.
func memoryNote(lang, cfgDir string) string {
	if Processes == nil {
		return ""
	}
	h, ok := hostmem.Find(Processes, ParentPID())
	if !ok {
		return ""
	}
	step := hostmem.Due(h.RSS, state.MemoryWarned(cfgDir, h.Key()), hostmem.Steps(Getenv))
	if step == 0 {
		return ""
	}
	alive := func(key string) bool { return hostmem.Alive(Processes, key) }
	if err := state.MarkMemory(cfgDir, h.Key(), step, alive); err != nil {
		return ""
	}
	return i18n.T(lang, "hook.memory."+h.Kind, hostmem.GB(h.RSS, lang))
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
	head := i18n.T(c.Language, "note.summary.head")

	unlock, err := filelock.Lock(filepath.Join(cfgDir, backup.NotesLock))
	if err != nil {
		return err
	}
	defer unlock()
	b, err := brain.Open(c, s)
	if err != nil {
		return err
	}
	b.Now = Now
	rel, err := b.SaveSession("compacted", i.SessionID, fmt.Sprintf(head, now.Format("2006-01-02 15:04"), i.Trigger), i.CompactSummary)
	if err != nil {
		return err
	}
	task := state.Read(cfgDir, s.Name).Task
	if task == "" {
		return nil
	}
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
// what came before are gone, and the summary left may be wrong. It asks the assistant
// not to take the summary as fact and to tell the user, suggesting to park the task and
// go on in a new session. (PostCompact already keeps the summary and notes it in the
// journal of the active task.)
func compactNote(lang string) string {
	return i18n.T(lang, "hook.compact")
}

// recoverNote offers to recover a session of the space that never ended (it was closed
// by force, or the machine went down), from its transcript, still on this machine.
func recoverNote(lang string, se state.Session) string {
	task := ""
	if se.Task != "" {
		task = ", working on task " + se.Task
	}
	return fmt.Sprintf("\nA previous session of this space, started %s%s, did not end normally (it may have been closed by force), "+
		"so what was said after its last journal entry may be missing from the notes. %soffer to recover it. "+
		"If they agree, call session_recover, read it, and write in the task's journal what is missing.\n",
		se.Started.Local().Format("2006-01-02 15:04"), task, session.FirstAnswerLine(lang))
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
		note += compactNote(c.Language)
	}
	if i.Source == "startup" || i.Source == "" {
		for _, se := range st.Unclosed(i.SessionID, now) {
			if se.Reported {
				continue
			}
			note += recoverNote(c.Language, se)
			_ = state.Reported(cfgDir, s.Name, se.ID, now)
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
	return fmt.Sprintf("\nThis machine was set up from a backup, and backups do not carry the guard's lists (production AWS profiles, untouchable hosts), nor the repositories you may commit in (git_allowed_repos: the user adds those to the configuration). "+
		"%soffer to ask for them again; "+
		"if they agree, ask which AWS profiles reach production and which machines must not be touched, and save them with safety_save.\n", session.FirstAnswerLine(c.Language))
}

// rootNote names the root exceptions in force, so the assistant knows that there,
// and only there, a command as root goes to the user instead of being refused.
func rootNote(c *config.Config) string {
	roots, unread := c.RootExceptions()
	bad := ""
	if len(unread) > 0 {
		bad = "\nThese root exceptions (root_ask) cannot be read, so they grant nothing: " + strings.Join(unread, "; ") +
			`. Each is "<host> until <date>". ` + session.TellUserLine(c.Language) + ".\n"
	}
	var in []string
	for _, r := range roots {
		if session.Now().Before(r.Until) {
			in = append(in, r.Host+" until "+r.Until.Format("2006-01-02 15:04")+" UTC")
		}
	}
	if len(in) == 0 {
		return bad
	}
	return bad + "\nTemporary root exceptions (root_ask): " + strings.Join(in, "; ") + ". On those machines, and only while they last, a command as root (sudo, also through ssh) is not refused: the guard asks the user to confirm each one. Use it only for what the user asked, and say first what the command does.\n"
}

// oldServersNote says when other Claude sessions on this machine still run an earlier
// augmented brain (they started before an update): their tools may fail on what this
// one writes until they are restarted.
func oldServersNote(c *config.Config) string {
	old := state.OldServers(c.Dir, version.Version)
	if len(old) == 0 {
		return ""
	}
	return fmt.Sprintf("\n%d other Claude sessions on this machine still run an earlier augmented brain: they started before it was updated to %s, and their augmented-brain tools may fail until they are restarted. %smention it in one line: close and open those sessions again when it suits them.\n",
		len(old), version.Version, session.FirstAnswerLine(c.Language))
}

// preferredModel is the model the user wants (session.toml), or "" when there is none
// or the file cannot be read: a hook must never break the session.
func preferredModel(cfgDir string) string {
	s, err := config.LoadSession(cfgDir)
	if err != nil {
		return ""
	}
	return s.PreferredModel
}

// modelStartNote, at the start of a session, records whether Claude Code said which
// model it starts with (it may not), and asks the assistant to tell the user when it is
// not the preferred one.
func modelStartNote(lang, cfgDir string, i input) string {
	pref := preferredModel(cfgDir)
	if pref == "" {
		return ""
	}
	now := Now()
	_ = state.RecordStartModel(cfgDir, i.Source, i.Model, now)
	if i.Model == "" || config.ModelMatches(pref, i.Model) || i.SessionID == "" {
		return ""
	}
	if err := state.MarkModel(cfgDir, i.SessionID, i.Model, now); err != nil {
		return ""
	}
	return i18n.T(lang, "hook.model.start", i.Model, pref)
}

// modelStopNote, after an answer, reads which model wrote it (the end of the
// transcript) and asks the assistant to tell the user when it is not the preferred one:
// it covers a start that did not say its model and a change in the middle of a
// session. Once per session and model, and only once its mark is written, so it never
// repeats after every answer.
func modelStopNote(lang, cfgDir string, i input) string {
	pref := preferredModel(cfgDir)
	if pref == "" || i.SessionID == "" {
		return ""
	}
	model := state.LastModel(i.TranscriptPath)
	if config.ModelMatches(pref, model) || state.ModelWarned(cfgDir, i.SessionID) == model {
		return ""
	}
	if err := state.MarkModel(cfgDir, i.SessionID, model, Now()); err != nil {
		return ""
	}
	return i18n.T(lang, "hook.model.stop", model, pref)
}
