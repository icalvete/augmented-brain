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
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/icalvete/augmented-brain/internal/backup"
	"github.com/icalvete/augmented-brain/internal/brain"
	"github.com/icalvete/augmented-brain/internal/config"
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
- A part of a task that waits on someone else or goes at its own pace is a subtask
  (task_new with parent; one level only). Any other part is a step: a checkbox in the
  task's Plan (task_step).
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
  A way of working the user asks for becomes a rule (rule_new). When it holds in every
  part of their life (a lesson about a tool, how they like answers), save it as shared:
  every space sees it.
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
	w := filelock.NewWriter(lockFile)
	locked := func() (*brain.Brain, func(), error) {
		release, err := w.Lock()
		if err != nil {
			return nil, nil, err
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
		release := w.LockProcess()
		b, err := open()
		if err != nil {
			release()
			return nil, nil, err
		}
		return b, release, nil
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
				indent, part, steps := "", "", ""
				if i.Sub {
					indent = "  "
				} else if i.ParentTitle != "" {
					part = " (part of " + i.ParentTitle + ")"
				}
				if i.StepsTotal > 0 {
					steps = fmt.Sprintf(" · %d/%d steps", i.StepsDone, i.StepsTotal)
				}
				fmt.Fprintf(&out, "%s- %s (%s)%s · %s%s · waits on %s · next: %s\n", indent, i.ID, i.Title, part, i.Status, steps, i.WaitingOn, i.Next)
			}
			for _, d := range due {
				fmt.Fprintf(&out, "! watch %s (%d days): %s [%s]\n", d.Date, d.Days, d.What, d.TaskID)
			}
			return out.String(), nil
		})

	type newTaskIn struct {
		Title    string `json:"title" jsonschema:"short name of the task"`
		Summary  string `json:"summary,omitempty" jsonschema:"one line: what it is"`
		Area     string `json:"area,omitempty" jsonschema:"id of its area; default the parent's area, or area-general"`
		Parent   string `json:"parent,omitempty" jsonschema:"id of the task this one is a part of, making it a subtask; one level only"`
		Goal     string `json:"goal,omitempty" jsonschema:"one line: what the task is for; goes into its Definition"`
		DoneWhen string `json:"done_when,omitempty" jsonschema:"one line: how we will know it worked; goes into its Definition"`
	}
	tool(s, locked, "task_new", "Open a new task, in the definition phase and waiting on the user. Returns the new task. With parent it is a subtask: a part with its own status and next step, for a part that waits on someone else or goes at its own pace; the parent cannot close while it is open.",
		func(b *brain.Brain, in newTaskIn) (string, error) {
			n, err := b.NewTask(brain.NewTaskInput{Title: in.Title, Summary: in.Summary, Area: in.Area, Parent: in.Parent,
				Goal: in.Goal, DoneWhen: in.DoneWhen})
			if err == nil {
				b.Touched(n.Header.ID, true)
			}
			return render(n, err)
		})

	type idIn struct {
		ID string `json:"id" jsonschema:"task id, e.g. task-backup"`
	}
	tool(s, locked, "task_get", "Read a task: its header and its whole body, journal included, and its subtasks.",
		func(b *brain.Brain, in idIn) (string, error) {
			n, err := b.Task(in.ID)
			if err == nil {
				b.Touched(in.ID, false)
			}
			out, err := render(n, err)
			if err != nil {
				return "", err
			}
			if p := b.Mem.ParentOf(n); p != nil {
				out += fmt.Sprintf("\nPart of: %s (%s) · %s\n", p.Header.ID, p.Header.Title, p.Header.Status)
			}
			if miss := b.MissingRepos(n); len(miss) > 0 {
				out += fmt.Sprintf("\nNot on this computer: repos %s of this task is in none of the space's work folders (%s); it was moved, renamed or deleted. "+
					"Tell the user in one line and ask where it is now; do not change repos unless they say so.\n",
					strings.Join(miss, ", "), strings.Join(b.Space.WorkFolders, ", "))
			}
			if kids := b.Mem.Children(n.Header.ID); len(kids) > 0 {
				out += "\nSubtasks:\n"
				for _, c := range kids {
					out += fmt.Sprintf("- %s (%s) · %s · waits on %s · next: %s\n", c.Header.ID, c.Header.Title, c.Header.Status, c.Header.WaitingOn, c.Header.Next)
				}
			}
			return out, nil
		})

	type stepIn struct {
		ID     string `json:"id"`
		Number int    `json:"number,omitempty" jsonschema:"the step to tick or untick, 1 for the first checkbox of the Plan"`
		Text   string `json:"text,omitempty" jsonschema:"the step to tick or untick, by its text (or a part only it contains), when number is not given"`
		Done   bool   `json:"done,omitempty" jsonschema:"true to tick the step, false to untick it"`
		Add    string `json:"add,omitempty" jsonschema:"add this step, unticked, at the end of the Plan, instead of ticking one"`
	}
	tool(s, locked, "task_step", "Steps are the small parts of a task: checkboxes (- [ ] text) in its Plan section. Add one (add), or tick or untick one by number or text (done). Each change writes a short journal line.",
		func(b *brain.Brain, in stepIn) (string, error) {
			var n *memory.Note
			var err error
			if in.Add != "" {
				n, err = b.AddStep(in.ID, in.Add)
			} else {
				n, err = b.Step(brain.StepInput{ID: in.ID, Number: in.Number, Text: in.Text, Done: in.Done})
			}
			if err != nil {
				return "", err
			}
			b.Touched(in.ID, true)
			var out strings.Builder
			out.WriteString("Steps of " + n.Header.ID + ":\n")
			for _, st := range memory.Steps(n.Body) {
				mark := " "
				if st.Done {
					mark = "x"
				}
				fmt.Fprintf(&out, "%d. [%s] %s\n", st.N, mark, st.Text)
			}
			return out.String(), nil
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
			u := brain.UpdateInput{ID: in.ID, Status: in.Status, WaitingOn: in.WaitingOn, Next: in.Next,
				Summary: in.Summary, Area: in.Area, WatchDate: in.WatchDate, WatchWhat: in.WatchWhat, WatchDone: in.WatchDone}
			if in.Status+in.WaitingOn+in.Next+in.Summary+in.Area+in.WatchDate+in.WatchDone == "" {
				return "", errors.New("nothing to change: give at least one field")
			}
			n, err := b.Update(u)
			if err != nil {
				return "", err
			}
			if n.Header.Status == "closed" || n.Header.Status == "parked" || n.Header.Status == "someday" {
				b.Released(in.ID)
			} else {
				b.Touched(in.ID, true)
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
				b.Touched(in.ID, true)
			}
			return header(n, err)
		})

	type parkIn struct {
		ID         string `json:"id"`
		Next       string `json:"next" jsonschema:"exactly how to resume: a command, a file or a decision"`
		WaitingOn  string `json:"waiting_on,omitempty"`
		Essentials string `json:"essentials,omitempty" jsonschema:"pass it: 3 to 6 lines that are enough to resume the task without its journal (what it is, where it is, what was decided, the next step); replaces its first section. Without it only 'where it is' and 'next' are updated"`
		Someday    bool   `json:"someday,omitempty" jsonschema:"true when it will not be touched for weeks"`
	}
	tool(s, locked, "task_park", "Pause a task without losing anything, before switching to other work.",
		func(b *brain.Brain, in parkIn) (string, error) {
			n, err := b.Park(brain.ParkInput{ID: in.ID, Next: in.Next, WaitingOn: in.WaitingOn,
				Essentials: in.Essentials, Someday: in.Someday})
			if err == nil {
				b.Released(in.ID)
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
				b.Released(in.ID)
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
		Shared  bool   `json:"shared,omitempty" jsonschema:"true when it holds in every space, whatever part of the user's life (it then has no area)"`
	}
	tool(s, locked, "knowledge_new", "Save a reference or a lesson that is worth keeping beyond one task: in this space, or shared by every space.",
		func(b *brain.Brain, in knowledgeIn) (string, error) {
			b, err := where(b, in.Shared)
			if err != nil {
				return "", err
			}
			return header(b.NewKnowledge(in.Kind, in.Title, in.Summary, in.Area, in.Body))
		})

	type ruleIn struct {
		Title   string `json:"title" jsonschema:"the rule in a few words"`
		Summary string `json:"summary,omitempty"`
		Body    string `json:"body" jsonschema:"Markdown: what to do, and why"`
		Shared  bool   `json:"shared,omitempty" jsonschema:"true when it applies in every space, not only this one"`
	}
	tool(s, locked, "rule_new", "Save a rule the user gives on how to work, loaded at the start of every session: in this space, or shared by every space.",
		func(b *brain.Brain, in ruleIn) (string, error) {
			b, err := where(b, in.Shared)
			if err != nil {
				return "", err
			}
			return header(b.NewRule(in.Title, in.Summary, in.Body))
		})

	type personIn struct {
		Name    string   `json:"name" jsonschema:"full name, or how the user knows them"`
		Aliases []string `json:"aliases,omitempty" jsonschema:"other names the user uses for them, e.g. a nickname"`
		Role    string   `json:"role,omitempty" jsonschema:"who they are to the user, in a few words: 'my manager', 'client, runs the shop'"`
		Notes   string   `json:"notes,omitempty" jsonschema:"anything else worth remembering about them"`
		Spaces  []string `json:"spaces,omitempty" jsonschema:"the spaces where this is visible; ask the user which, never guess: without it nothing is saved and the answer is the question to ask"`
	}
	tool(s, locked, "person_save", "Remember someone the user works or deals with, so a first name like 'Marcos' leads to the right person. Saving a name that exists updates it. A person can be in several spaces (the same person, with the role and notes each space is allowed to show): pass spaces, the ones the user says. Without spaces nothing is saved and the answer asks which spaces to use; ask the user and call again. The index of each space lists its people.",
		func(b *brain.Brain, in personIn) (string, error) {
			c, err := config.Load(b.ConfigDir)
			if err != nil {
				return "", err
			}
			names := c.SpaceNames()
			spaces := in.Spaces
			if len(spaces) == 0 {
				if len(names) != 1 {
					return fmt.Sprintf("Not saved yet: where should %q be visible? Ask the user which of these spaces, and call person_save again with spaces: %s. "+
						"What is saved in a space is seen in that space only (a personal note about someone should not be in a work space).",
						strings.TrimSpace(in.Name), strings.Join(names, ", ")), nil
				}
				spaces = names // one space: nothing to ask
			}
			id, written, err := brain.SavePersonIn(c, spaces, brain.PersonInput{Name: in.Name, Aliases: in.Aliases, Role: in.Role, Notes: in.Notes})
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("%s (%s) saved in: %s", id, strings.TrimSpace(in.Name), strings.Join(written, ", ")), nil
		})

	type safetyIn struct {
		ProductionAWSProfiles []string `json:"production_aws_profiles,omitempty" jsonschema:"AWS CLI profiles that reach production, to add; the guard asks before commands that use them"`
		UntouchableHosts      []string `json:"untouchable_hosts,omitempty" jsonschema:"hosts that must not be touched without asking (ssh names as the user types them), to add"`
	}
	tool(s, locked, "safety_save", "Add to what the guard must ask the user about: production AWS profiles and hosts not to touch. They apply on this whole machine, in every space and outside them. Use it in the first session and when the user names a new one. It only adds: to take one out, the user edits the configuration file themselves (the answer says where).",
		func(b *brain.Brain, in safetyIn) (string, error) {
			return b.SaveSafety(brain.SafetyInput{ProductionAWSProfiles: in.ProductionAWSProfiles, UntouchableHosts: in.UntouchableHosts})
		})

	type secretIn struct {
		Service string `json:"service" jsonschema:"the service the secret is kept under in the system keychain, e.g. telegram"`
		Name    string `json:"name" jsonschema:"its name under that service, e.g. alerts-bot"`
		What    string `json:"what,omitempty" jsonschema:"what it is for, in a few words"`
	}
	tool(s, locked, "secret_name_save", "Write down the name of a secret the user keeps in the system keychain (service and name) and what it is for, so they and a new machine know which secrets there are. Never its value: a value is refused. Use it when the user stores a new secret. Saving a name that is listed changes what it is for. The list is the same in every space; the panel shows it.",
		func(b *brain.Brain, in secretIn) (string, error) {
			return b.SaveSecretName(brain.SecretName{Service: in.Service, Name: in.Name, What: in.What})
		})

	type repoIn struct {
		Repo string `json:"repo" jsonschema:"the repository: its path (~/… or absolute), or its name as a task's repos: writes it, relative to a work folder"`
		Text string `json:"text" jsonschema:"what it is, in one line, in the user's words; empty takes the description out"`
	}
	tool(s, locked, "repo_describe", "Write down, in one line, what one of the user's git repositories is, so the panel's Repositories page (their map of repositories) says it. Only repositories inside this space's work folders. It is kept in the space's notes, in the reference ref-repositories, one line per repository; saving one that has a line replaces it.",
		func(b *brain.Brain, in repoIn) (string, error) {
			where, err := b.DescribeRepo(in.Repo, in.Text)
			if err != nil {
				return "", err
			}
			if strings.TrimSpace(in.Text) == "" {
				return fmt.Sprintf("%s has no description now (%s).", where, memory.PathFor("reference", brain.RepoNoteID)), nil
			}
			return fmt.Sprintf("%s described in %s.", where, memory.PathFor("reference", brain.RepoNoteID)), nil
		})

	type backupIn struct {
		RepoURL string `json:"repo_url,omitempty" jsonschema:"a private git repository the user can push to; leave it out to create one in their GitHub account"`
	}
	tool(s, unlockedFile, "backup_setup", "Set up the backup of this space: a private git repository that keeps its notes, pushed by itself every time a session ends. Without repo_url it creates augmented-brain-<space> in the user's GitHub account with the gh command line; when gh is missing or not logged in, it returns the steps to guide the user through.",
		func(b *brain.Brain, in backupIn) (string, error) {
			return setupBackup(b, in.RepoURL, backup.Exec)
		})

	type profileIn struct {
		Name        *string `json:"name,omitempty" jsonschema:"what the user wants to be called"`
		About       *string `json:"about,omitempty" jsonschema:"what they do and work on, in their words; short lines starting with '- '"`
		WorkStyle   *string `json:"work_style,omitempty" jsonschema:"how they like to work with you: language, length of answers, how much to ask before acting; short lines starting with '- '"`
		Assistant   *string `json:"assistant_name,omitempty" jsonschema:"the name the user gives you, their assistant"`
		Personality *string `json:"assistant_personality,omitempty" jsonschema:"the personality the user wants you to have, in their words: style only, never above the rules"`
	}
	tool(s, locked, "profile_save", "Save who the user is, how they like to work, and the name and personality they give you. Every space shares it, and every session starts by reading it. Use it in the first session, and whenever the user asks to change it. Send only the parts that change: a part left out keeps what it has, and an empty one is cleared. The name is needed the first time.",
		func(b *brain.Brain, in profileIn) (string, error) {
			p, err := brain.LoadProfile(b.ConfigDir)
			if err != nil {
				return "", err
			}
			for _, f := range []struct {
				from *string
				to   *string
			}{{in.Name, &p.Name}, {in.About, &p.About}, {in.WorkStyle, &p.WorkStyle}, {in.Assistant, &p.Assistant}, {in.Personality, &p.Personality}} {
				if f.from != nil {
					*f.to = *f.from
				}
			}
			n, err := b.SaveProfile(p)
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

// recoverSession returns the conversation of an earlier session that never ended,
// after the last journal entry of its task, and marks it as dealt with. Without id it
// is the one the session start offered.
func recoverSession(b *brain.Brain, id string, now time.Time) (string, error) {
	if id == "" {
		if id = b.Offered(now); id == "" {
			return "No earlier session of this space is waiting to be recovered.", nil
		}
	}
	rec, err := b.Recover(id, brain.RecoverForAssistant, now)
	if errors.Is(err, brain.ErrNoDeadSession) {
		return "", fmt.Errorf("there is no earlier session %s of this space to recover", id)
	}
	if err != nil {
		return "", fmt.Errorf("the transcript of that session cannot be read (%v); Claude Code deletes them after 30 days", err)
	}
	se := rec.Session
	_ = state.Dismissed(b.ConfigDir, b.Space.Name, se.ID, now)
	task := "the task it worked on"
	if se.Task != "" {
		task = "task " + se.Task
	}
	if strings.TrimSpace(rec.Text) == "" {
		return fmt.Sprintf("The session started %s has nothing said after the last journal entry of %s.", se.Started.Local().Format("2006-01-02 15:04"), task), nil
	}
	return fmt.Sprintf("Session started %s, from %s on (the user's and your messages only). Write in the journal of %s what was done, decided or found that is not there yet, in a few short entries.\n\n%s",
		se.Started.Local().Format("2006-01-02 15:04"), rec.From.Local().Format("2006-01-02 15:04"), task, rec.Text), nil
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

// where is the brain a new note goes to: this space's, or the shared notes'.
func where(b *brain.Brain, shared bool) (*brain.Brain, error) {
	if !shared {
		return b, nil
	}
	return b.Shared()
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
