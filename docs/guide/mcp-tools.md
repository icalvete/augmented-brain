# MCP tools

[Leer en español](mcp-tools.es.md)

You say "park it". Claude calls the tool `task_park`, with the task's id and the next
step. The program writes the task file. You never call these tools yourself: Claude does,
behind your words.

**MCP** (Model Context Protocol) is an open standard to give tools to an assistant. The
program serves these tools with `augmented-brain mcp`, which Claude Code starts through
the plugin. In Claude Code they show up as `mcp__plugin_brain_augmented-brain__<tool>`.

A few things hold for every tool:

- It works on **the space of the folder Claude is in**, found again on every call.
- Tools run one at a time, and wait for the panel, so two never write at once.
- **No secrets.** A tool refuses to write a password, a token or a key into a note.
- **Nothing is deleted.** Tasks are closed, not removed.
- Dates are written `YYYY-MM-DD`.
- A parameter is required unless it says optional.

## Tools by purpose

- **Where things are**: [`where_am_i`](#where_am_i), [`pending`](#pending).
- **Tasks**: [`task_new`](#task_new), [`task_get`](#task_get),
  [`task_update`](#task_update), [`task_step`](#task_step),
  [`task_journal`](#task_journal), [`task_park`](#task_park), [`task_close`](#task_close).
- **Inbox and the weekly review**: [`capture`](#capture),
  [`capture_done`](#capture_done), [`weekly_review`](#weekly_review), [`review_done`](#review_done).
- **Notes that last**: [`area_new`](#area_new), [`knowledge_new`](#knowledge_new),
  [`rule_new`](#rule_new), [`note_retire`](#note_retire), [`person_save`](#person_save),
  [`repo_describe`](#repo_describe).
- **You and this machine**: [`profile_save`](#profile_save),
  [`safety_save`](#safety_save), [`secret_name_save`](#secret_name_save),
  [`backup_setup`](#backup_setup), [`session_recover`](#session_recover).

## `where_am_i`

Which space this session works in, its notes folder, its areas, and how many tasks are
live. It takes no parameters.

## `pending`

The live tasks, those waiting on you first, each with its status, steps done, who it
waits on and its next step. Then the watch dates that are due. Claude uses it for "where
were we?".

- `within_days` (optional): also list watch dates due within this many days. Default 7.

## `task_new`

Opens a new task, in the definition phase and waiting on you. With a parent, it is a
subtask.

- `title`: a short name for the task.
- `summary` (optional): one line on what it is.
- `area` (optional): the id of its area. Default: the parent's area, or `area-general`.
- `parent` (optional): the id of the task this one is a part of. One level only: the
  parent cannot be a subtask, nor closed.
- `goal` (optional): one line on what the task is for. It goes into the Definition.
- `done_when` (optional): one line on how we will know it worked. It goes into the
  Definition.

## `task_get`

Reads a task: its header, its whole body with the journal, and its subtasks. It also says
which task it is part of, and whether a repository it names is missing from this
computer. Claude reads a task before resuming it.

- `id`: the task's id, such as `task-move-the-blog`.

## `task_update`

Changes a task. Only the fields given change. A change of status is written in the
journal, and when it skips phases (straight from `definition` to `execution`, say), the
line names the ones skipped: skipping is allowed, and it is seen. A closed task opens
again only with `reopen`.

- `id`: the task's id.
- `status` (optional): `definition`, `analysis`, `plan`, `execution`, `validation`,
  `closed`, `parked` or `someday`.
- `waiting_on` (optional): `me`, `nobody`, a person, or another task's id.
- `next` (optional): the single next action, in one line.
- `summary` (optional): one line on what it is.
- `area` (optional): the id of its area.
- `watch_date` (optional): add a date to look at something again. Needs `watch_what`.
- `watch_what` (optional): what to look at on `watch_date`, in one line.
- `watch_done` (optional): mark the watch date of this day as done.
- `reopen` (optional): only to open a closed task again, with a new `status`: why, in a
  few words. It goes to the journal. Claude asks you first.

At least one field besides `id` is needed.

## `task_step`

Steps are the small parts of a task: checkboxes (`- [ ] text`) in its Plan. This tool adds
one, or ticks or unticks one. Each change writes a short journal line, and the answer
lists the steps.

- `id`: the task's id.
- `add` (optional): add this step, unticked, at the end of the Plan, instead of ticking
  one.
- `number` (optional): the step to tick or untick, 1 for the first checkbox.
- `text` (optional): the step to tick or untick, by its text (or a part only it holds),
  when `number` is not given.
- `done` (optional): `true` to tick the step, `false` to untick it. Left out, it unticks.

## `task_journal`

Adds a dated entry to a task's journal. Claude writes one at the moment something is
done or decided.

- `id`: the task's id.
- `entry`: what was done, what came out, what was decided.

## `task_park`

Pauses a task without losing anything, before switching to other work.

- `id`: the task's id.
- `next`: exactly how to resume: a command, a file or a decision.
- `waiting_on` (optional): who it waits on now.
- `essentials` (optional, but Claude should pass it): three to six lines that are enough to
  resume without the journal. They replace the task's first section. Without it, only
  "where it is" and "next" are updated there.
- `someday` (optional): `true` when it will not be touched for weeks. The task leaves
  the daily list.

## `task_close`

Closes a finished task with its result. It is kept, never deleted. A task with open
subtasks cannot close, and a task already closed is refused, so its result is never
written over. If the task has watch dates not marked done, it says so: they
still come up at the start of the session on their day, since closing is often when you
write down what to check later.

- `id`: the task's id.
- `result`: what was achieved, in one or two lines.

## `capture`

Notes something in the inbox without opening a task, optionally with a day to be
reminded. The weekly review processes it.

- `text`: the idea, link or loose end, in one line.
- `remind_on` (optional): a date. It comes up at the start of the session from a week
  before, until it is dealt with.
- `space` (optional): another of your spaces, to send it there instead. It lands in
  that space's inbox marked with where it came from, `(from personal)`, and comes up at
  the start of every session there until it is done. For something decided in one space
  that has to be done in another. Claude only does it when you ask: what goes to a
  space is seen there and goes into its backup.

## `capture_done`

Takes one capture out of the inbox once it is done, or once it has become a task,
without waiting for the weekly review. Above all for what another space sent, which
otherwise comes up at every session start.

- `date`: the day it was captured, as listed.
- `text`: its text exactly as listed, with or without its origin. Another capture with
  the same text stays.

## `weekly_review`

Lists what the weekly review goes through: the inbox, live tasks untouched for 14 days
or more, and the dates due in the next 7 days. Claude then goes through it with you, one
item at a time. It takes no parameters.

## `review_done`

Finishes the weekly review. It saves what was decided in `reviews/`, and takes the
processed captures out of the inbox.

- `summary`: what was decided, in a few lines.
- `processed` (optional): the text of each capture dealt with, exactly as `weekly_review`
  listed it. They leave the inbox. One that is not found is left as it was, and named.

## `area_new`

Creates an area: a responsibility with no end date that groups tasks and knowledge.

- `title`: the area's name.
- `summary` (optional): one line on what it looks after.

## `knowledge_new`

Saves a reference or a lesson worth keeping beyond one task: in this space, or shared by
every space. A title that already exists creates nothing: Claude is told its id, to change
that one or choose another title. Nothing is ever saved twice without anyone knowing.

- `id` (optional): to change a reference or lesson that exists. What is sent replaces
  what it had, what is left out stays, and the version it replaces is kept in
  `archive/`.
- `kind`: `reference` (how something works) or `lesson` (what was learned).
- `title`: for a lesson, a title that already states the conclusion.
- `summary` (optional): one line.
- `area` (optional): the id of its area.
- `body` (optional): the text, in Markdown.
- `shared` (optional): `true` when it holds in every space. It then has no area.

## `rule_new`

Saves a rule you give on how to work. Every session loads it. A title that already exists
creates nothing: Claude is told its id.

- `id` (optional): to change a rule that exists. What is sent replaces what it had, what
  is left out stays, and the version it replaces is kept in `archive/`.
- `title`: the rule in a few words.
- `summary` (optional): one line that says what to do.
- `body`: what to do, and why, in Markdown.
- `shared` (optional): `true` when it applies in every space, not only this one.

## `note_retire`

Retires a rule, reference or lesson that no longer holds, without deleting it. It moves
to `archive/`, which is never loaded, so a retired rule stops reaching the session start.
A line at its end says when, why and what replaces it. For a duplicate or an old
version; Claude asks you first.

- `id`: the rule, reference or lesson.
- `reason`: why, in a few words. It stays with the note.
- `replaced_by` (optional): the id of the note that replaces it.
- `shared` (optional): `true` when the note is in the shared layer.

## `person_save`

Remembers someone you work or deal with, so a first name leads to the right person.
Saving a name that exists updates that person.

A person can be in several spaces: the same person, with the role and notes each space
may show. Claude asks you where it should be visible (for instance, a note about a friend
should not land in your work space) and passes your answer in `spaces`. With only one
space, there is nothing to ask. Without `spaces` and with several, nothing is saved: the
answer lists your spaces and asks. What is saved in a space is seen in that space only.

- `name`: the full name, or how you know them.
- `aliases` (optional): other names you use for them, such as a nickname.
- `role` (optional): who they are to you, in a few words: "my manager", "runs the
  hosting company".
- `notes` (optional): anything else worth remembering.
- `spaces` (optional with one space, needed with several): the names of the spaces where
  this is visible. The role, other names and notes are written in those spaces only.

## `repo_describe`

Writes down, in one line, what one of your git repositories is, for the panel's
Repositories page. Only repositories inside this space's work folders. The line goes to
the reference `ref-repositories`, one line per repository; a new one replaces the old.

- `repo`: the repository: its path (`~/…` or absolute), or its name as a task's `repos:`
  writes it, relative to a work folder.
- `text`: what it is, in one line, in your words. Empty takes the line out.

## `profile_save`

Saves who you are, how you like to work, and the name and personality you give Claude.
Every space shares it, and every session starts by reading it. Claude sends only the
parts that change: a part left out keeps what it has, and an empty one is cleared.

- `name` (needed the first time): what you want to be called.
- `about` (optional): what you do and work on, in short lines starting with `- `.
- `work_style` (optional): how you like to work with Claude: language, length of
  answers, how much to ask before acting. Short lines starting with `- `.
- `assistant_name` (optional): the name you give Claude. It signs with it the emails it
  writes in your name, always followed by a line that says it is an AI (see
  [Panel, Profile](panel.md#profile)).
- `assistant_personality` (optional): how you want Claude to be, in your words. Style
  only: it never changes the rules, the guard or its honesty.

## `safety_save`

Adds to what the guard asks you about. The lists hold on this whole machine, in every
space. It only adds: to take one out, you edit the configuration yourself, and the
answer says where.

- `production_aws_profiles` (optional): AWS CLI profiles that reach production. The guard
  asks before commands that use them.
- `untouchable_hosts` (optional): machines not to touch without asking, as you type them
  for ssh.

## `secret_name_save`

Writes down the name of a secret you keep in the system keychain, and what it is for.
Never its value: a value is refused. The list is the same in every space; the panel
shows it. Saving a name that is listed changes what it is for.

- `service`: the service the secret is kept under in the keychain, such as `telegram`.
- `name`: its name under that service, such as `alerts-bot`.
- `what` (optional): what it is for, in a few words.

## `backup_setup`

Sets up the backup of this space: a private git repository that keeps its notes, pushed
by itself every time a session ends.

- `repo_url` (optional): a private git repository you can push to. Without it, the tool
  creates `augmented-brain-<space>` in your GitHub account with the `gh` command line.
  When `gh` is missing or not logged in, it returns the steps to guide you through.

It makes the first copy right away. If the space already has a backup, it says where it
goes and how the last one went; it never moves it elsewhere: only you do, with
`augmented-brain backup setup <url>`.

## `session_recover`

Reads what you and Claude said in an earlier session of this space that did not end
well, after the last journal entry of its task. Only the messages, no tool output, at
most the last 20,000 characters, from Claude Code's record on this computer (kept 30
days). Claude then writes in the journal what is missing.

- `session_id` (optional): the session to recover. Without it, the one the session start
  offered.
