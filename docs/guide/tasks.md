# Tasks

[Leer en español](tasks.es.md)

You say "new task: move the blog to a new host". Claude opens a task. You agree on what
"done" means, then on a plan. In the evening you say "park it". Next week, in a new
session, you say "let's carry on with the blog", and Claude picks it up where you left it.

A **task** is any piece of work with a start and an end. It is one text file in the
space's `tasks/` folder.

## What a task looks like

```markdown
---
id: task-move-the-blog
type: task
title: Move the blog
area: area-blog
status: parked
waiting_on: me
next: Copy the database to the new host
created: "2026-10-15"
updated: "2026-10-16"
---
## Essentials
The blog moves to a cheaper host. Plan agreed: copy, test, switch the domain.
Done: the new host is ready. Next: copy the database.

## 3. Plan
- [x] Open an account on the new host
- [ ] Copy the database
- [ ] Switch the domain

## Journal
- 2026-10-15 — Task opened.
- 2026-10-16 — We go with the cheaper host: same speed, half the price.
- 2026-10-16 — Parked. Next: Copy the database to the new host
```

The part between the `---` lines is the **header**: the data the program reads. Below it
is free text, in any language. The first section is always the **essentials**: a few
lines that are enough to resume the task. The last one is always the **journal**: dated
lines of what happened.

You can open and edit the file with any editor. The full format is in the
[memory format](../spec/memory-format.md).

## The six phases

A task moves through six phases. **You approve each change of phase.** Inside a phase,
Claude works without asking at every step.

1. **Definition**: what we want, what is out, and how we will know it worked.
2. **Analysis**: how things really are, measured, not assumed.
3. **Plan**: ordered steps, each with how to undo it.
4. **Execution**: doing it, writing in the journal as we go.
5. **Validation**: proving the "done" criterion.
6. **Closed**: finished. It is kept, never deleted.

The phase is the task's `status`. Two more statuses pause a task without losing anything:

- **parked**: paused, to resume soon. `next` says exactly how.
- **someday**: not for weeks. It leaves the list Claude reads at the start, and stays in
  its area.

A small task may skip phases (straight from definition to execution): it is allowed, and
the journal says which phases it skipped. A closed task stays closed: closing it again is
refused, so its result is never written over, and opening it again needs a reason: Claude asks
you first, and the reason goes to the journal. In the panel, choosing a new status for a
closed task is that reason.

## Who it waits on, and what comes next

Every task says two more things:

- **waiting_on**: `me` (you), `nobody`, a person's name, or another task.
- **next**: the single next action, in one line.

That is what "where were we?" shows you: each live task, those waiting on you first, with
its next step.

## Park and resume

Before you switch to other work, say "park it". Claude writes down three things: the exact
next step, who it waits on, and the essentials, rewritten so another session can resume
without reading the whole journal.

To resume, say "let's carry on with the blog". Claude reads the task first. Then it checks
the real state of what the task touches (the files, the server) before trusting the note.
Things may have changed since.

## The journal, as you go

Each decision or finished step goes to the journal at the moment, not only when you park
it. A session can end at any time, and what is not in the note is lost.

If the session has a task open and nothing has been written in it for 20 minutes, Claude
gets a reminder at the end of its answer, once. Each session has its own task: with two
windows on different tasks, each is reminded of its own. When a long conversation is compacted (Claude
Code replaces the details with a summary), the summary is kept in `archive/sessions/`,
and the journal points at it. The summary can be wrong, so Claude is asked not to take it
as fact, to check before stating anything from before, and to tell you the session was
compacted and suggest parking the task and going on in a new session, which starts from
the notes.

## Steps and subtasks

A big task can be split in two ways.

A **step** is a small part: a checkbox in the task's Plan, such as `- [ ] Copy the
database`. Claude ticks it when it is done. The list of tasks shows how many are done
("1/3 steps").

A **subtask** is a part with a life of its own, because it waits on someone else or goes
at its own pace. For example, "Ask Diego for the domain password" waits on Diego, while
you go on with the rest. A subtask is a whole task, with one more line in its header:
`parent: task-move-the-blog`.

- Only one level: a subtask has no subtasks.
- The bigger task closes last, when all its subtasks are closed.
- Parking the bigger task leaves its subtasks as they are.

Rule of thumb: if it waits on someone or has its own pace, it is a subtask. Otherwise, it
is a step.

## Dates to look at something again

Some checks belong to a day: "on the 16th, is the certificate renewed?". Tell Claude and
it adds a **watch date** to the task. From 7 days before, and every day after, Claude
tells you at the start of the session, until it is marked done.

This also works on a closed task. Closing is often when you write down what to check
later, so when Claude closes a task that still has one, it tells you it will keep coming
up.

For something that is not a task ("remind me on Thursday to call the insurance"), Claude
uses the inbox instead: see [Areas, knowledge and the rest](notes.md#inbox).

## Closing

Say "close the task". Claude writes the result, and the task stays in its folder, marked
closed. Nothing is ever deleted.

A task that nobody has changed for 14 days comes up in the
[weekly review](weekly-review.md), so nothing stalls in silence.

## Next

- [What to say to Claude](phrases.md): the phrases for all of this.
- [MCP tools](mcp-tools.md): the tools Claude uses behind each phrase.
