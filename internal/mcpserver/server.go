// Package mcpserver serves the method to any assistant that speaks MCP.
//
// Each tool is a thin layer over package brain. The space is resolved on every call
// from the folder the assistant runs in, and the memory is read again, so edits made
// by hand between calls are never lost.
package mcpserver

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/icalvete/augmented-brain/internal/backup"
	"github.com/icalvete/augmented-brain/internal/brain"
	"github.com/icalvete/augmented-brain/internal/filelock"
	"github.com/icalvete/augmented-brain/internal/memory"
	"github.com/icalvete/augmented-brain/internal/state"
	"github.com/icalvete/augmented-brain/internal/version"
)

// Instructions tell the assistant how to use the tools. They are sent once, when it
// connects.
const Instructions = `augmented brain keeps the user's memory: tasks, areas, knowledge and an inbox,
as Markdown files. Use these tools instead of editing those files by hand.

- Every piece of work is a task. It moves through definition, analysis, plan, execution,
  validation and closed, and the user approves each change of phase.
- Keep the task current as you work: write what was done, what came out and what was
  decided in its journal (task_journal) at the moment it happens, not only when parking:
  a session can end at any time, and what is not in the note is lost. Keep its next
  step and who it waits on up to date (task_update).
- Before switching to other work, park the task (task_park) with the exact next step
  and rewritten essentials, so another session can resume it without re-reading
  everything.
- To resume a task, read it first (task_get), then check the real state of what it
  touches before trusting the note.
- Ideas and loose ends that are not a task go to the inbox (capture). "Remind me on
  Thursday" is a capture with remind_on; a date to check on a task is a watch date
  (task_update with watch_date and watch_what). Both come up at the start of the
  session on their day.
- The weekly review empties the inbox and looks at stalled tasks: weekly_review lists
  what to go through; go through it with the user one item at a time (each capture
  becomes a task, goes to an area or knowledge, or is dropped), then call review_done
  with what was decided and the captures processed.
- What is worth keeping beyond a task becomes a reference or a lesson (knowledge_new).
- Nothing is ever deleted: tasks are closed, not removed.
- Who the user is and how they like to work is their profile (profile_save).
- A space without a backup can lose everything with the machine: offer backup_setup.
- When the user names someone, look for them under People in the index. If they are
  not there and seem to matter, offer to remember them (person_save).`

// Resolver opens the brain of the space the assistant is working in.
type Resolver func() (*brain.Brain, error)

// opener opens the brain and returns the function that releases it.
type opener func() (*brain.Brain, func(), error)

// New builds the server with every tool. lockFile, when not empty, is shared by every
// augmented brain process of the user, so two assistant sessions never write at once.
func New(open Resolver, lockFile string) *mcp.Server {
	// The SDK serves requests concurrently. Tools read and write the same files, so they
	// run one at a time: each call sees every change made by the calls before it.
	var mu sync.Mutex
	locked := func() (*brain.Brain, func(), error) {
		mu.Lock()
		release := mu.Unlock
		if lockFile != "" {
			unlock, err := filelock.Lock(lockFile)
			if err != nil {
				mu.Unlock()
				return nil, nil, err
			}
			release = func() { unlock(); mu.Unlock() }
		}
		b, err := open()
		if err != nil {
			release()
			return nil, nil, err
		}
		return b, release, nil
	}
	// backup_setup runs a backup, which takes the notes lock itself (after the backup
	// lock, never before): it runs one at a time in this process, without the file lock.
	unlockedFile := func() (*brain.Brain, func(), error) {
		mu.Lock()
		b, err := open()
		if err != nil {
			mu.Unlock()
			return nil, nil, err
		}
		return b, mu.Unlock, nil
	}
	s := mcp.NewServer(&mcp.Implementation{Name: "augmented-brain", Version: version.Version},
		&mcp.ServerOptions{Instructions: Instructions})

	tool(s, locked, "where_am_i", "Which space this session works in, its memory folder, its areas and how many tasks are live.",
		func(b *brain.Brain, _ struct{}) (string, error) {
			items, _ := b.Pending(0)
			var out strings.Builder
			fmt.Fprintf(&out, "space: %s\nmemory: %s\nlive tasks: %d\nareas:\n", b.Space.Name, b.Mem.Dir, len(items))
			for _, a := range b.Mem.OfType("area") {
				fmt.Fprintf(&out, "- %s: %s — %s\n", a.Header.ID, a.Header.Title, a.Header.Summary)
			}
			return out.String(), nil
		})

	type pendingIn struct {
		WithinDays int `json:"within_days,omitempty" jsonschema:"also list watch dates due within this many days (default 7)"`
	}
	tool(s, locked, "pending", "The live tasks, those waiting on the user first, with status, who each waits on and its next step; and the watch dates that are due.",
		func(b *brain.Brain, in pendingIn) (string, error) {
			if in.WithinDays == 0 {
				in.WithinDays = 7
			}
			items, due := b.Pending(in.WithinDays)
			var out strings.Builder
			if len(items) == 0 {
				out.WriteString("No live tasks.\n")
			}
			for _, i := range items {
				fmt.Fprintf(&out, "- %s (%s) · %s · waits on %s · next: %s\n", i.ID, i.Title, i.Status, i.WaitingOn, i.Next)
			}
			for _, d := range due {
				fmt.Fprintf(&out, "! watch %s (%d days): %s [%s]\n", d.Date, d.Days, d.What, d.TaskID)
			}
			return out.String(), nil
		})

	type newTaskIn struct {
		Title   string `json:"title" jsonschema:"short name of the task"`
		Summary string `json:"summary,omitempty" jsonschema:"one line: what it is"`
		Area    string `json:"area,omitempty" jsonschema:"id of its area; default area-general"`
	}
	tool(s, locked, "task_new", "Open a new task, in the definition phase and waiting on the user. Returns the new task.",
		func(b *brain.Brain, in newTaskIn) (string, error) {
			n, err := b.NewTask(brain.NewTaskInput{Title: in.Title, Summary: in.Summary, Area: in.Area})
			if err == nil {
				touched(b, n.Header.ID, true)
			}
			return render(n, err)
		})

	type idIn struct {
		ID string `json:"id" jsonschema:"task id, e.g. task-backup"`
	}
	tool(s, locked, "task_get", "Read a task: its header and its whole body, journal included.",
		func(b *brain.Brain, in idIn) (string, error) {
			n, err := b.Task(in.ID)
			if err == nil {
				touched(b, in.ID, false)
			}
			return render(n, err)
		})

	type updateIn struct {
		ID        string `json:"id"`
		Status    string `json:"status,omitempty" jsonschema:"definition, analysis, plan, execution, validation, closed, parked or someday"`
		WaitingOn string `json:"waiting_on,omitempty" jsonschema:"me, nobody, a person, or another task id"`
		Next      string `json:"next,omitempty" jsonschema:"the single next action, one line"`
		Summary   string `json:"summary,omitempty"`
		Area      string `json:"area,omitempty" jsonschema:"id of the area"`
		WatchDate string `json:"watch_date,omitempty" jsonschema:"add a date (YYYY-MM-DD) to look at something again; needs watch_what"`
		WatchWhat string `json:"watch_what,omitempty" jsonschema:"what to look at on watch_date, one line"`
		WatchDone string `json:"watch_done,omitempty" jsonschema:"mark the watch date on this day (YYYY-MM-DD) as done"`
	}
	tool(s, locked, "task_update", "Change a task's status, who it waits on, its next step, summary or area, or add or close a watch date. Only the fields given change; a status change is written in the journal.",
		func(b *brain.Brain, in updateIn) (string, error) {
			var n *memory.Note
			var err error
			if in.Status != "" || in.WaitingOn != "" || in.Next != "" || in.Summary != "" || in.Area != "" {
				if n, err = b.Update(brain.UpdateInput{ID: in.ID, Status: in.Status, WaitingOn: in.WaitingOn,
					Next: in.Next, Summary: in.Summary, Area: in.Area}); err != nil {
					return "", err
				}
			}
			if in.WatchDate != "" {
				if n, err = b.AddWatch(in.ID, in.WatchDate, in.WatchWhat); err != nil {
					return "", err
				}
			}
			if in.WatchDone != "" {
				if n, err = b.WatchDone(in.ID, in.WatchDone); err != nil {
					return "", err
				}
			}
			if n == nil {
				return "", errors.New("nothing to change: give at least one field")
			}
			if n.Header.Status == "closed" || n.Header.Status == "parked" || n.Header.Status == "someday" {
				released(b, in.ID)
			} else {
				touched(b, in.ID, true)
			}
			return header(n, nil)
		})

	type journalIn struct {
		ID    string `json:"id"`
		Entry string `json:"entry" jsonschema:"what was done, what came out, what was decided"`
	}
	tool(s, locked, "task_journal", "Add a dated entry to a task's journal.",
		func(b *brain.Brain, in journalIn) (string, error) {
			n, err := b.Journal(in.ID, in.Entry)
			if err == nil {
				touched(b, in.ID, true)
			}
			return header(n, err)
		})

	type parkIn struct {
		ID         string `json:"id"`
		Next       string `json:"next" jsonschema:"exactly how to resume: a command, a file or a decision"`
		WaitingOn  string `json:"waiting_on,omitempty"`
		Essentials string `json:"essentials,omitempty" jsonschema:"3 to 6 lines that are enough to resume the task; replaces its first section"`
		Someday    bool   `json:"someday,omitempty" jsonschema:"true when it will not be touched for weeks"`
	}
	tool(s, locked, "task_park", "Pause a task without losing anything, before switching to other work.",
		func(b *brain.Brain, in parkIn) (string, error) {
			n, err := b.Park(brain.ParkInput{ID: in.ID, Next: in.Next, WaitingOn: in.WaitingOn,
				Essentials: in.Essentials, Someday: in.Someday})
			if err == nil {
				released(b, in.ID)
			}
			return header(n, err)
		})

	type closeIn struct {
		ID     string `json:"id"`
		Result string `json:"result" jsonschema:"what was achieved, in one or two lines"`
	}
	tool(s, locked, "task_close", "Close a finished task with its result. It is kept, never deleted.",
		func(b *brain.Brain, in closeIn) (string, error) {
			n, err := b.Close(in.ID, in.Result)
			if err == nil {
				released(b, in.ID)
			}
			return header(n, err)
		})

	type captureIn struct {
		Text     string `json:"text" jsonschema:"the idea, link or loose end, in one line"`
		RemindOn string `json:"remind_on,omitempty" jsonschema:"YYYY-MM-DD: bring it up again at the start of the session on that day"`
	}
	tool(s, locked, "capture", "Note something in the inbox without opening a task, optionally with a day to be reminded. The weekly review processes it.",
		func(b *brain.Brain, in captureIn) (string, error) {
			if err := b.Capture(in.Text, in.RemindOn); err != nil {
				return "", err
			}
			if in.RemindOn != "" {
				return "Captured; it will come up on " + in.RemindOn + ".", nil
			}
			return "Captured.", nil
		})

	tool(s, locked, "weekly_review", "What the weekly review goes through: the inbox, live tasks untouched for two weeks or more, and the dates due this week.",
		func(b *brain.Brain, _ struct{}) (string, error) {
			r := b.Review()
			var out strings.Builder
			if r.Last == "" {
				out.WriteString("No review yet in this space.\n")
			} else {
				fmt.Fprintf(&out, "Last review: %s.\n", r.Last)
			}
			fmt.Fprintf(&out, "\nInbox (%d):\n", len(r.Inbox))
			for _, e := range r.Inbox {
				fmt.Fprintf(&out, "- [%s] %s\n", e.Date, e.Text)
			}
			fmt.Fprintf(&out, "\nTasks untouched for %d days or more (%d):\n", brain.StaleDays, len(r.Stale))
			for _, t := range r.Stale {
				fmt.Fprintf(&out, "- %s (%s) · %s · waits on %s · next: %s\n", t.ID, t.Title, t.Status, t.WaitingOn, t.Next)
			}
			fmt.Fprintf(&out, "\nDue within %d days (%d):\n", brain.ReviewEvery, len(r.Reminders))
			for _, d := range r.Reminders {
				fmt.Fprintf(&out, "- %s (%d days): %s%s\n", d.Date, d.Days, d.What, taskRef(d.TaskID))
			}
			out.WriteString("\nGo through it with the user one item at a time, then call review_done.")
			return out.String(), nil
		})

	type reviewDoneIn struct {
		Summary   string   `json:"summary" jsonschema:"what was decided in the review, a few lines"`
		Processed []string `json:"processed,omitempty" jsonschema:"the text of each inbox capture dealt with, exactly as weekly_review listed it; they leave the inbox"`
	}
	tool(s, locked, "review_done", "Finish the weekly review: save what was decided in reviews/ and take the processed captures out of the inbox.",
		func(b *brain.Brain, in reviewDoneIn) (string, error) {
			missing, err := b.InboxDone(in.Processed)
			if err != nil && len(in.Processed) > 0 {
				return "", err
			}
			n, err := b.SaveReview(in.Summary)
			if err != nil {
				return "", err
			}
			out := "Review saved in " + n.Path + "."
			if len(missing) > 0 {
				out += "\nNot found in the inbox (left as they were): " + strings.Join(missing, "; ")
			}
			return out, nil
		})

	type areaIn struct {
		Title   string `json:"title"`
		Summary string `json:"summary,omitempty" jsonschema:"one line: what this area looks after"`
	}
	tool(s, locked, "area_new", "Create an area: a responsibility with no end date that groups tasks and knowledge.",
		func(b *brain.Brain, in areaIn) (string, error) {
			n, err := b.NewArea(in.Title, in.Summary)
			return header(n, err)
		})

	type knowledgeIn struct {
		Kind    string `json:"kind" jsonschema:"reference (how something works) or lesson (what was learned)"`
		Title   string `json:"title" jsonschema:"for a lesson, a title that already states the conclusion"`
		Summary string `json:"summary,omitempty"`
		Area    string `json:"area,omitempty"`
		Body    string `json:"body,omitempty" jsonschema:"Markdown"`
	}
	tool(s, locked, "knowledge_new", "Save a reference or a lesson that is worth keeping beyond one task.",
		func(b *brain.Brain, in knowledgeIn) (string, error) {
			n, err := b.NewKnowledge(in.Kind, in.Title, in.Summary, in.Area, in.Body)
			return header(n, err)
		})

	type personIn struct {
		Name    string   `json:"name" jsonschema:"full name, or how the user knows them"`
		Aliases []string `json:"aliases,omitempty" jsonschema:"other names the user uses for them, e.g. a nickname"`
		Role    string   `json:"role,omitempty" jsonschema:"who they are to the user, in a few words: 'my manager', 'client, runs the shop'"`
		Notes   string   `json:"notes,omitempty" jsonschema:"anything else worth remembering about them"`
	}
	tool(s, locked, "person_save", "Remember someone the user works or deals with, so a first name like 'Marcos' leads to the right person. Saving a name that exists updates it. People live in the space, and the index lists them.",
		func(b *brain.Brain, in personIn) (string, error) {
			n, err := b.SavePerson(brain.PersonInput{Name: in.Name, Aliases: in.Aliases, Role: in.Role, Notes: in.Notes})
			return header(n, err)
		})

	type safetyIn struct {
		ProductionAWSProfiles []string `json:"production_aws_profiles,omitempty" jsonschema:"AWS CLI profiles that reach production; the guard asks before commands that use them. Replaces the list."`
		UntouchableHosts      []string `json:"untouchable_hosts,omitempty" jsonschema:"hosts that must not be touched without asking (ssh names as the user types them). Replaces the list."`
	}
	tool(s, locked, "safety_save", "Set what the guard must ask the user about: production AWS profiles and hosts not to touch. They apply on this whole machine, in every space and outside them. Use it in the first session and when the user names a new one; each list given replaces the old one (the answer shows what was there), so include what should stay.",
		func(b *brain.Brain, in safetyIn) (string, error) {
			return b.SaveSafety(brain.SafetyInput{ProductionAWSProfiles: in.ProductionAWSProfiles, UntouchableHosts: in.UntouchableHosts})
		})

	type backupIn struct {
		RepoURL string `json:"repo_url,omitempty" jsonschema:"a private git repository the user can push to; leave it out to create one in their GitHub account"`
	}
	tool(s, unlockedFile, "backup_setup", "Set up the backup of this space: a private git repository that keeps its notes, pushed by itself every time a session ends. Without repo_url it creates augmented-brain-<space> in the user's GitHub account with the gh command line; when gh is missing or not logged in, it returns the steps to guide the user through.",
		func(b *brain.Brain, in backupIn) (string, error) {
			return setupBackup(b, in.RepoURL, backup.Exec)
		})

	type profileIn struct {
		Name      string `json:"name" jsonschema:"what the user wants to be called"`
		About     string `json:"about,omitempty" jsonschema:"what they do and work on, in their words; short lines starting with '- '"`
		WorkStyle string `json:"work_style,omitempty" jsonschema:"how they like to work with you: language, length of answers, how much to ask before acting; short lines starting with '- '"`
	}
	tool(s, locked, "profile_save", "Save who the user is and how they like to work. Every space shares it, and every session starts by reading it. Use it in the first session, and whenever the user asks to change it; it replaces the whole profile, so include what should stay.",
		func(b *brain.Brain, in profileIn) (string, error) {
			n, err := b.SaveProfile(brain.ProfileInput{Name: in.Name, About: in.About, WorkStyle: in.WorkStyle})
			return render(n, err)
		})

	type recoverIn struct {
		SessionID string `json:"session_id,omitempty" jsonschema:"the session to recover; leave it out for the one the session start offered"`
	}
	tool(s, locked, "session_recover", "Read what the user and you said in an earlier session of this space that did not end normally, after the last journal entry of the task it worked on: only the messages, no tool output, at most the last 20,000 characters, from Claude Code's transcript on this machine. Use it when the session start offers it and the user agrees, then write in the task's journal what is missing.",
		func(b *brain.Brain, in recoverIn) (string, error) {
			return recoverSession(b, in.SessionID, time.Now())
		})

	return s
}

// touched and released keep the space's state on this machine (state package): which
// task is active and when it was last written, for the Stop hook's reminder. A failure
// there never fails the tool.
func touched(b *brain.Brain, id string, write bool) {
	if b.ConfigDir != "" {
		_ = state.Touched(b.ConfigDir, b.Space.Name, id, time.Now(), write)
	}
}

func released(b *brain.Brain, id string) {
	if b.ConfigDir != "" {
		_ = state.Released(b.ConfigDir, b.Space.Name, id, time.Now())
	}
}

// recoverMax is the most text session_recover returns.
const recoverMax = 20000

var journalDate = regexp.MustCompile(`(?m)^- (\d{4}-\d{2}-\d{2}) — `)

// recoverSession returns the conversation of an earlier session that never ended,
// after the last journal entry of its task, and marks it as dealt with.
func recoverSession(b *brain.Brain, id string, now time.Time) (string, error) {
	st := state.Read(b.ConfigDir, b.Space.Name)
	var se *state.Session
	// The session asking is running, so its transcript is fresh and Unclosed leaves it out.
	unclosed := st.Unclosed("", now)
	for k := range unclosed {
		if id != "" && unclosed[k].ID == id || id == "" && unclosed[k].Reported {
			se = &unclosed[k]
			break
		}
	}
	if se == nil {
		if id != "" {
			return "", fmt.Errorf("there is no earlier session %s of this space to recover", id)
		}
		return "No earlier session of this space is waiting to be recovered.", nil
	}
	since := se.Started
	// The last write to the task, when it happened while that session was still alive.
	if st.Task == se.Task && st.Since.After(since) {
		if info, err := os.Stat(se.Transcript); err == nil && st.Since.Before(info.ModTime()) {
			since = st.Since
		}
	}
	if n, ok := b.Mem.Notes[se.Task]; ok {
		dates := journalDate.FindAllStringSubmatch(n.Body, -1)
		if len(dates) > 0 {
			if d, err := time.ParseInLocation("2006-01-02", dates[len(dates)-1][1], time.Local); err == nil && d.After(since) {
				since = d
			}
		}
	}
	text, err := state.Conversation(se.Transcript, since, recoverMax)
	if err != nil {
		return "", fmt.Errorf("the transcript of that session cannot be read (%v); Claude Code deletes them after 30 days", err)
	}
	found := se.ID
	_ = state.Update(b.ConfigDir, b.Space.Name, now, func(x *state.Space) {
		for k := range x.Sessions {
			if x.Sessions[k].ID == found {
				x.Sessions[k].Closed = true
			}
		}
	})
	task := "the task it worked on"
	if se.Task != "" {
		task = "task " + se.Task
	}
	if strings.TrimSpace(text) == "" {
		return fmt.Sprintf("The session started %s has nothing said after the last journal entry of %s.", se.Started.Local().Format("2006-01-02 15:04"), task), nil
	}
	return fmt.Sprintf("Session started %s, from %s on (the user's and your messages only). Write in the journal of %s what was done, decided or found that is not there yet, in a few short entries.\n\n%s",
		se.Started.Local().Format("2006-01-02 15:04"), since.Local().Format("2006-01-02 15:04"), task, text), nil
}

// tool registers a handler that receives the brain of the current space, opened afresh
// for every call, and returns text for the assistant. Errors come back as tool errors,
// which the assistant reads and can act on.
func tool[In any](s *mcp.Server, open opener, name, desc string, h func(*brain.Brain, In) (string, error)) {
	mcp.AddTool(s, &mcp.Tool{Name: name, Description: desc},
		func(_ context.Context, _ *mcp.CallToolRequest, in In) (*mcp.CallToolResult, any, error) {
			b, release, err := open()
			if err != nil {
				return nil, nil, err
			}
			defer release()
			text, err := h(b, in)
			if note := brain.ProblemsNote(b.Problems, 3); note != "" {
				// Files that are not valid notes do not stop the tools; the user hears about them.
				if err != nil {
					err = fmt.Errorf("%w\n\n%s", err, note)
				} else {
					text = strings.TrimRight(text, "\n") + "\n\n" + note
				}
			}
			if err != nil {
				return nil, nil, err
			}
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}, nil, nil
		})
}

func taskRef(id string) string {
	if id == "" {
		return " [inbox]"
	}
	return " [" + id + "]"
}

func render(n *memory.Note, err error) (string, error) {
	if err != nil {
		return "", err
	}
	data, err := n.Render()
	return string(data), err
}

func header(n *memory.Note, err error) (string, error) {
	if err != nil {
		return "", err
	}
	h := n.Header
	out := fmt.Sprintf("%s (%s) saved in %s", h.ID, h.Title, n.Path)
	if h.Type == "task" {
		out += fmt.Sprintf("\nstatus: %s · waits on: %s · next: %s", h.Status, h.WaitingOn, h.Next)
	}
	return out, nil
}
