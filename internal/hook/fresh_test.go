package hook

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/icalvete/augmented-brain/internal/backup"
	"github.com/icalvete/augmented-brain/internal/brain"
	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/filelock"
	"github.com/icalvete/augmented-brain/internal/fresh"
	"github.com/icalvete/augmented-brain/internal/setup"
)

// withClaudePID makes the hooks run as under a Claude process (CLAUDE_PID), or none.
func withClaudePID(t *testing.T, pid string) {
	t.Helper()
	old := Getenv
	t.Cleanup(func() { Getenv = old })
	Getenv = func(k string) string {
		if k == "CLAUDE_PID" {
			return pid
		}
		return ""
	}
}

// otherSession writes like another session would: through brain, outside this process.
func otherSession(t *testing.T, cfgDir string, change func(b *brain.Brain)) {
	t.Helper()
	c, err := config.Load(cfgDir)
	if err != nil {
		t.Fatal(err)
	}
	b, err := brain.Open(c, c.Spaces["work"])
	if err != nil {
		t.Fatal(err)
	}
	change(b)
}

func prompt(t *testing.T, cfgDir, cwd string) string {
	t.Helper()
	o := run(t, func(in *bytes.Reader, out *bytes.Buffer) error { return UserPromptSubmit(in, out, cfgDir) },
		map[string]any{"cwd": cwd, "session_id": "s1", "prompt": "hello"})
	if o == nil {
		return ""
	}
	if o["hookEventName"] != "UserPromptSubmit" {
		t.Fatalf("event: %v", o)
	}
	return o["additionalContext"].(string)
}

func start(t *testing.T, cfgDir, cwd, source string) {
	t.Helper()
	run(t, func(in *bytes.Reader, out *bytes.Buffer) error { return SessionStartPart(in, out, cfgDir, 1) },
		map[string]any{"cwd": cwd, "session_id": "s1", "source": source})
}

// The whole story of 7-nov through the hooks: the session starts, another session
// closes a task, and the next message carries the news.
func TestUserPromptSubmitTellsWhatChanged(t *testing.T) {
	cfgDir, work := setupSpace(t)
	withClaudePID(t, "5555")
	otherSession(t, cfgDir, func(b *brain.Brain) {
		if _, err := b.NewTask(brain.NewTaskInput{Title: "Logo"}); err != nil {
			t.Fatal(err)
		}
	})
	start(t, cfgDir, work, "startup")
	if _, ok := fresh.Load(cfgDir, "pid-5555"); !ok {
		t.Fatal("SessionStart took no baseline")
	}
	if got := prompt(t, cfgDir, work); got != "" {
		t.Fatalf("news with nothing changed: %s", got)
	}
	otherSession(t, cfgDir, func(b *brain.Brain) {
		if _, err := b.Close("task-logo", "done elsewhere"); err != nil {
			t.Fatal(err)
		}
	})
	got := prompt(t, cfgDir, work)
	if !strings.Contains(got, "task-logo") || !strings.Contains(got, "closed") || !strings.Contains(got, "task_get") {
		t.Fatalf("got: %q", got)
	}
	if again := prompt(t, cfgDir, work); again != "" {
		t.Fatalf("told twice: %s", again)
	}
}

// Spanish configuration, Spanish note.
func TestUserPromptSubmitInSpanish(t *testing.T) {
	cfgDir, work := setupSpace(t)
	withClaudePID(t, "5556")
	c, _ := config.Load(cfgDir)
	c.Language = "es"
	if err := c.Save(); err != nil {
		t.Fatal(err)
	}
	start(t, cfgDir, work, "startup")
	otherSession(t, cfgDir, func(b *brain.Brain) { _, _ = b.NewTask(brain.NewTaskInput{Title: "Nueva"}) })
	if got := prompt(t, cfgDir, work); !strings.Contains(got, "tarea nueva") {
		t.Fatalf("got: %q", got)
	}
}

// Without CLAUDE_PID nothing is tracked and nothing is said; outside a space, the same.
func TestUserPromptSubmitQuietWithout(t *testing.T) {
	cfgDir, work := setupSpace(t)
	withClaudePID(t, "")
	start(t, cfgDir, work, "startup")
	otherSession(t, cfgDir, func(b *brain.Brain) { _, _ = b.NewTask(brain.NewTaskInput{Title: "X"}) })
	if got := prompt(t, cfgDir, work); got != "" {
		t.Fatalf("without CLAUDE_PID: %q", got)
	}
	if _, err := os.Stat(filepath.Join(cfgDir, fresh.Dir)); !os.IsNotExist(err) {
		t.Fatal("a seen file without CLAUDE_PID")
	}
	withClaudePID(t, "5557")
	if got := prompt(t, cfgDir, "/"); got != "" {
		t.Fatalf("outside a space: %q", got)
	}
}

// When the notes lock stays held (a backup, a long tool), the message is not held up:
// the news waits for the next one.
func TestUserPromptSubmitDoesNotWaitLong(t *testing.T) {
	cfgDir, work := setupSpace(t)
	withClaudePID(t, "5558")
	start(t, cfgDir, work, "startup")
	otherSession(t, cfgDir, func(b *brain.Brain) { _, _ = b.NewTask(brain.NewTaskInput{Title: "Held"}) })
	unlock, err := filelock.Lock(filepath.Join(cfgDir, backup.NotesLock))
	if err != nil {
		t.Fatal(err)
	}
	began := time.Now()
	got := prompt(t, cfgDir, work)
	if took := time.Since(began); took > FreshWait+time.Second {
		t.Fatalf("waited %v", took)
	}
	unlock()
	if got != "" {
		t.Fatalf("said something without the lock: %q", got)
	}
	if got := prompt(t, cfgDir, work); !strings.Contains(got, "task-held") {
		t.Fatalf("the news was lost: %q", got)
	}
}

// A broken input never blocks the message: no output, no error.
func TestUserPromptSubmitBadInput(t *testing.T) {
	cfgDir, _ := setupSpace(t)
	withClaudePID(t, "5559")
	var out bytes.Buffer
	if err := UserPromptSubmit(strings.NewReader("not json"), &out, cfgDir); err != nil || out.Len() != 0 {
		t.Fatalf("err %v, out %q", err, out.String())
	}
}

// /clear keeps the process (and the seen file); any other end removes it.
func TestSessionEndRemovesSeen(t *testing.T) {
	cfgDir, work := setupSpace(t)
	t.Setenv("AB_NO_AUTO_BACKUP", "1")
	withClaudePID(t, "5560")
	start(t, cfgDir, work, "startup")
	end := func(reason string) {
		data, _ := json.Marshal(map[string]any{"cwd": work, "reason": reason})
		if err := SessionEnd(bytes.NewReader(data), cfgDir); err != nil {
			t.Fatal(err)
		}
	}
	end("clear")
	if _, ok := fresh.Load(cfgDir, "pid-5560"); !ok {
		t.Fatal("/clear removed the seen file")
	}
	end("prompt_input_exit")
	if _, ok := fresh.Load(cfgDir, "pid-5560"); ok {
		t.Fatal("the end of the session left the seen file")
	}
}

// After compact and resume the tasks the session worked on stay its own: another
// session's journal entry in them is still news. A new start forgets them.
func TestStartSourcesKeepOrDropTouched(t *testing.T) {
	cfgDir, work := setupSpace(t)
	withClaudePID(t, "5561")
	otherSession(t, cfgDir, func(b *brain.Brain) { _, _ = b.NewTask(brain.NewTaskInput{Title: "Mine"}) })
	start(t, cfgDir, work, "startup")
	sp := fresh.SpaceOf("work", filepath.Join(os.Getenv("HOME"), "brain"))
	if err := fresh.Absorb(cfgDir, "pid-5561", sp, time.Now(), []string{"task-mine"}); err != nil {
		t.Fatal(err)
	}
	journal := func(text string) {
		otherSession(t, cfgDir, func(b *brain.Brain) {
			if _, err := b.Journal("task-mine", text); err != nil {
				t.Fatal(err)
			}
		})
	}
	start(t, cfgDir, work, "compact")
	journal("after compact")
	if got := prompt(t, cfgDir, work); !strings.Contains(got, "task-mine") {
		t.Fatalf("after compact: %q", got)
	}
	start(t, cfgDir, work, "clear")
	journal("after clear")
	if got := prompt(t, cfgDir, work); got != "" {
		t.Fatalf("after clear a journal entry of an untouched task was news: %q", got)
	}
}

// What another space sent comes up at the start of a session here, until it is done.
func TestSessionStartShowsWhatOtherSpacesSent(t *testing.T) {
	cfgDir, work := setupSpace(t)
	home := os.Getenv("HOME")
	other := filepath.Join(home, "me")
	if err := os.MkdirAll(other, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := setup.AddSpace(cfgDir, setup.Options{Space: "personal", WorkFolder: other, Memory: "~/brain-personal"}); err != nil {
		t.Fatal(err)
	}
	c, _ := config.Load(cfgDir)
	p, err := brain.Open(c, c.Spaces["personal"])
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.SaveProfile(brain.ProfileInput{Name: "Lucía"}); err != nil { // past the first session
		t.Fatal(err)
	}
	if _, err := p.CaptureTo("work", "Purge php8.0 on the old server", ""); err != nil {
		t.Fatal(err)
	}
	text := func() string {
		o := run(t, func(in *bytes.Reader, out *bytes.Buffer) error { return SessionStartPart(in, out, cfgDir, 1) },
			map[string]any{"cwd": work, "source": "startup"})
		return o["additionalContext"].(string)
	}
	got := text()
	if !strings.Contains(got, "Sent from other spaces") || !strings.Contains(got, "Purge php8.0 on the old server [from personal]") || !strings.Contains(got, "capture_done") {
		t.Fatalf("session start:\n%s", got)
	}
	w, _ := brain.Open(c, c.Spaces["work"])
	if found, err := w.InboxDoneEntry(w.Now().Format("2006-01-02"), "Purge php8.0 on the old server"); !found || err != nil {
		t.Fatal(found, err)
	}
	if got := text(); strings.Contains(got, "Sent from other spaces") {
		t.Fatalf("still listed after it was done:\n%s", got)
	}
}
