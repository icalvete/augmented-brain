# Memory format

**Status: v0.1, the format the 0.1 release implements.** This document defines how augmented brain stores what you and your assistant know: what files exist, what goes in each, and which fields the program reads. Everything is plain Markdown that you can open, edit and search with any tool.

## Principles

1. **Plain files, yours.** One folder of Markdown per space. No database, no hosted service.
2. **The program reads fields, never prose.** Anything the program needs (a task's status, who it waits on, its next step) lives in the file's header. The body is free text, in any language.
3. **Indexes are generated.** You never edit an index by hand; the program rebuilds it after every change. Hand-kept indexes drift.
4. **Nothing is deleted.** Closed tasks stay, marked closed. Old journal entries move to `archive/`.
5. **Short at the top, detail below.** The file loaded at the start of every session stays small; detail lives one link away.

## Spaces

A **space** keeps one part of your life apart from the others, for example `work` and `personal`. Each space has:

- its own memory folder (everything below);
- its own rules, on top of the common ones;
- its own backup repository.

The space is chosen by **the folder you open the assistant in**. Spaces never read each other's memory. They share the program, the common rules, the shared notes (below), the safety hooks and the risk settings (which cloud profiles are production, which hosts must not be touched): those are for the whole machine, so a production profile saved from one space is guarded in all of them.

Spaces are declared in the configuration file (see "Configuration" below), not inside the memory.

By default, `augmented-brain init` creates two spaces, `work` and `personal` (in Spanish, `trabajo` and `personal`). Each one is a folder in `~/augmented-brain/`, which is its work folder, and its memory folder sits inside it as `notes/` (`notas/`). Any other layout can be set with `init --custom` or by editing the configuration.

A session that runs outside every work folder (for example, the scratch folder the Claude desktop app opens a new session in) uses the **default space**, `default_space` in the configuration; `init` sets it to `personal`, and `space add` sets it when there are two spaces or more and none is set (to `personal` if a space has that name, or else to the first space). Its backup is the one that carries the profile. Without one, no memory is loaded there and the assistant tells the user which folders to open.

## Layout of a space

```
<memory folder>/
├── MEMORY.md          # the index loaded at the start of every session (generated)
├── profile.md         # optional: what only applies in this space (your role at work…)
├── inbox.md           # quick captures, processed in the weekly review
├── areas/             # ongoing responsibilities (one file each)
├── tasks/             # projects with a beginning and an end (one file each)
├── knowledge/         # how things work (reference) and what you learned (lesson)
├── people/            # who the user deals with, so a first name finds the right person
├── rules/             # this space's own rules (the common rules ship with the program; shared ones live in the configuration folder)
├── reviews/           # one file per weekly review
├── docs/              # optional: long documents of a task (plans, analyses, reports); never loaded
│   └── <task>/
└── archive/           # old journals and merged files; never loaded, never deleted
    └── sessions/      # summaries Claude Code wrote when it compacted a conversation
```

`docs/` keeps the long documents a task produces, so they stay with the notes and out of the repositories the work touches (a repository only gets what its team needs). Each one sits under the id of its task, is plain Markdown or any other file, has no header, and is linked from the task. The program does not read it.

A summary in `archive/sessions/` is plain Markdown with no header, named `<date>-<time>-<session>.md`; the program writes it, with any secret taken out, and adds a journal line to the active task pointing at it.

`MEMORY.md` is the name Claude Code loads automatically from the memory folder; only its first 200 lines (or 25 KB) are read, so the program keeps it well under that.

**Your profile** (who you are, your language, how you like answers) is the same person in every space, so it lives once, next to the configuration: `profile.md` in the configuration folder. A space's own `profile.md` only adds what applies there.

The shared profile has two sections the program knows by their headings, in English or
Spanish: the first ("About me" / "Sobre mí", with the name line) and the work style
("How I like to work" / "Cómo me gusta trabajar"). Other sections a person adds by hand
are kept: the program reads them as part of the first one.

**Shared notes** are the rules, references and lessons that hold in every space (how git
behaves, how the person likes answers). They live once, next to the configuration, in
`shared/` inside the configuration folder, laid out like a space but with only
`rules/` and `knowledge/` (plus the generated `MEMORY.md` and `.augmented-brain/format`).
A shared note has no area, since areas belong to one space; a task, area, person or
review there is a problem, reported and left out. Every space sees them: a note of a
space may name a shared note in `related`, and the session start loads the shared rules
like the space's (see Rules) and lists the shared references and lessons. A shared note never links into a
space. They are migrated like a space's notes, and kept in the backup of the default
space, under `.augmented-brain/shared/`, like the profile.

## File header

Every file except `MEMORY.md`, `inbox.md` and what is in `archive/` starts with a YAML header between `---` lines (a byte order mark before it is fine). Keys and enumerated values are always English, whatever language you write in.

```yaml
---
id: task-backup-restore          # unique in the space; also the file name without .md
type: task                       # profile | area | task | reference | lesson | rule | person
title: Backup and restore        # shown in indexes; any language
summary: One line on what it is  # shown in indexes; any language
area: area-platform              # tasks, references and lessons: the area they belong to
repos: [infra/terraform]         # optional: repositories it touches, relative to the space's work folder
related: [lesson-git-filters]    # optional: other ids; the program keeps links two-way
created: 2026-10-15
updated: 2026-10-15
---
```

A repository in `repos:` that is in none of the space's work folders (it was moved,
renamed or deleted, or this computer never had it) is reported, never fixed by itself:
`augmented-brain check` and the panel's Health page list them, and reading the task
tells the assistant to ask the user where it is now. A closed task is not checked. An
absolute path, or one starting with `~`, is taken as it is.

### Task fields

Tasks add the fields that drive the whole workflow:

```yaml
status: plan                     # see "Task lifecycle"
waiting_on: me                   # me | nobody | a person's name | another task id
next: Approve the plan           # the single next action, one line
watch:                           # optional: dates to look at something again
  - date: 2026-11-16
    what: Certificates renewed?
    done: false
```

A watch date that is due today or tomorrow, or already past and not `done`, is brought up
at the start of the session.

### Subtasks

A task may be part of a bigger one, with one more field:

```yaml
parent: task-new-website         # optional: the id of the task this one is a part of
```

A subtask is a whole task (its own status, waiting_on, next step and journal). Make one
for a part that waits on someone else or goes at its own pace; anything smaller is a
step (below). The rules:

- The parent is an existing task that is not itself a subtask: one level only. The
  program refuses to open a subtask under a missing task, a note that is not a task, a
  subtask, or a closed task.
- A parent cannot be closed while any of its subtasks is not closed. Parking the parent
  leaves its subtasks as they are.
- The index lists a live subtask under its parent, indented. A subtask whose parent is
  closed, someday or missing is listed on its own, so it is never hidden. A `parent`
  that does not exist, is not a task, is the task itself or is itself a subtask is
  reported as a problem (like an `area` that does not exist), and so is a `parent` on a
  note that is not a task; the note is still read and can still be written, and is
  listed on its own, once.

### Steps

A step is a Markdown checkbox in the task's **Plan** section, the level-2 heading
`## Plan` (numbered or not, `## 3. Plan`, any case):

```markdown
## 3. Plan
- [x] Buy the domain
- [ ] Point it to the server
```

Steps need no field: the program finds the Plan heading and reads its checkboxes
(`- [ ]` or `- [x]`, also with `*`), outside code blocks, numbered from 1 in order. The
index and `pending` show how many are done ("1/2 steps") when there are any. Adding a
step puts it, unticked, at the end of the Plan; ticking or unticking one changes only its
`[ ]`, and writes one line in the journal.

## Task lifecycle

A task moves through six phases. You approve each change of phase; inside a phase the assistant works without asking at every step.

| `status` | Meaning |
|---|---|
| `definition` | What we want, what is out, and how we will know it worked |
| `analysis` | How things really are, measured, not assumed |
| `plan` | Ordered steps, each with how to undo it |
| `execution` | Doing it, writing a journal entry as we go |
| `validation` | Proving the "done" criterion |
| `closed` | Finished; the closing checklist is done |

Two more statuses pause a task without losing anything:

| `status` | Meaning |
|---|---|
| `parked` | Paused, resumed soon; `next` says exactly how |
| `someday` | Not for weeks; leaves the session index, stays in its area |

### Task body

The body uses these sections, in your language (the templates write the headings for you):

1. **Essentials**: three to six lines that are enough to resume the task without reading anything else. Rewritten every time the task is parked.
2. **Definition**, **Analysis**, **Plan**, **Validation**, **Closing**: one per phase.
3. **Journal**: dated lines, `- YYYY-MM-DD — what happened`. Always the last section.

The program never searches the prose. It relies on two positions only: the essentials are the **first** section, and the journal is the **last**, so a new entry is simply appended at the end of the file. The one exception is the checkboxes of the Plan section (see "Steps"). When a task file passes about 10 KB, its oldest journal entries move to `archive/` and the task links to them.

## Areas

An area is something you look after with no end date (your platform, your observability, your side project). Its body says what you look after, how it is going and what to watch. The program lists its tasks and knowledge in the area automatically, from their `area` field.

## Knowledge

- **reference**: how something works (a system, a deployment, a tool).
- **lesson**: something learned, with a title that already states the conclusion.

Both belong to an area and can relate to tasks.

## People

A person is someone the user deals with: `people/person-<name>.md`. The title is the name,
and the summary says who they are to the user and the other names the user calls them
("my manager · also: Marcos"), so the index alone tells the assistant who "Marcos" is. The
body keeps anything else worth remembering. People belong to a space: work contacts stay
in the work space.

## Inbox

`inbox.md` has no header. Each capture is one line, with the date it was captured:

```
- 2026-10-17 — Look at the new CI runner
- 2026-10-17 — Call the insurance (remind: 2026-10-22)
```

A capture ending in `(remind: YYYY-MM-DD)` is brought up at the start of the session on
that day. Any other line (the heading, a paragraph) is left alone.

## Reviews

The weekly review goes through the inbox (each capture becomes a task, goes to an area or
knowledge, or is dropped), the live tasks untouched for 14 days or more, and the dates due
in the coming week. Processed captures leave the inbox, and the review is kept as
`reviews/review-<date>.md` (`type: review`), with what was decided as its body and the
first line as its summary. When the last review is 7 days old or more and there is
something to go through, the session start offers a new one.

## Rules

Rules tell the assistant how to work. There are three levels:

- **common rules**, shipped with the program (answer style, verify before advising, never commit or push unless asked, never escalate to root, never write secrets…);
- **shared rules**, the person's own for every space, in the shared notes (for example, "answer in short sentences, one idea each");
- **space rules**, in `rules/`, for what only applies to that space (for example, "hosts db01–db03 are never restarted without explicit permission").

All of them are loaded at the start of every session. The shared and space rules go
whole when everything fits in what a session can take at the start (about 9,500
characters); when not, each goes as one line, its title and summary, with the path of its
file. So a rule's summary should say what to do.

## Format version

The notes of each space say which version of this format they follow, in
`.augmented-brain/format` inside the memory folder: one line with a number. The
configuration folder says its own, for the configuration and the profile, as
`format = N` at the top of `config.toml`. Neither saying it means version 1, which is
also never written to `config.toml`, because releases from before versions were
written refuse settings they do not know. This document describes version 1.

When a release writes something in a new way, it raises the version and brings a step
that moves version N to N+1. At the start of a session (and in `augmented-brain check`)
the program copies the whole folder to `migrations/` in the configuration folder, runs
the missing steps one at a time and writes the new version after each, so a step that
stops half way runs again from there. Notes of a newer version than the program knows
are never written: the program asks to be updated.

## Generated files

The program rebuilds these after every change; editing them by hand is pointless:

- `MEMORY.md`: your profile line, the rules, the areas, and the live tasks grouped by area, one line each, with status, steps done when the Plan has any, who it waits on and the next step; subtasks indented under their parent.
- The "Tasks" and "Knowledge" lists inside each area.

## Configuration

`~/.config/augmented-brain/config.toml` (or the platform equivalent) declares the spaces:

```toml
language = "en"                  # en | es: the language the program speaks to you
default_space = "personal"       # used outside every work folder (optional)
production_aws_profiles = ["prod", "prod-eu"]   # every space: the user confirms first
untouchable_hosts = ["db01", "db02", "db03"]    # ssh, scp, sftp, mosh to them: the user confirms first

[spaces.work]
memory = "~/augmented-brain/work/notes"
work_folders = ["~/augmented-brain/work", "~/code"]   # opening the assistant here selects this space
backup = "git@github.com:me/brain-work-private.git"
git_allowed_repos = ["~/code/website", "~/code/website-*"]   # the assistant may commit and push here, with git -C
git_ask_repos = ["~/code/infra"]                # the same, once the user confirms each time

[spaces.personal]
memory = "~/augmented-brain/personal/notes"
work_folders = ["~/augmented-brain/personal"]
backup = "git@github.com:me/brain-personal-private.git"
```

In `git_allowed_repos` and `git_ask_repos` the last part of a path may hold a `*`, which
matches one folder (`~/code/website-*` is every `website-<name>` next to `website`), and a
git worktree of a listed repository counts as that repository, wherever it is. The guard
joins the lists of every space, as it does with the production profiles: a repository
belongs to the person, not to the space a session runs in. A repository in both lists
asks. Even in an allowed one, a push that rewrites or deletes on the remote (a force, a
delete, a mirror, or a refspec starting with `+` or `:`) asks.

`production_aws_profiles` and `untouchable_hosts` may also appear inside a space, as older
versions wrote them: the guard applies the top-level lists and every space's together.
Host names match ignoring case, and a short name matches the full one either way (`db01`
and `db01.example.com`).

## Backup

Each space backs up to its own private git repository: the memory folder itself is the
repository. Every backup also copies this space's part of the configuration into
`.augmented-brain/` inside the memory folder, so restoring on a new machine needs nothing
else. The default space's backup (or the only space's) also carries the profile; no
other backup does, so the personal profile never reaches a work repository. The
machine-wide guard lists are in no backup (they may name work machines): after a restore
that leaves them empty, the next session offers to ask for them again. Paths are kept with `~`, so they point to the new machine's home, and
the notes go back to the same place inside the work folder. A restore also installs the
Claude Code plugin when it is missing.

A backup is refused while any file of the memory holds a secret (the made-up keys of
AWS's documentation are not secrets). Files over 50 MB are left out, and the user is told.
The repository gets a `.gitattributes` (`inbox.md merge=union`: captures from two machines
are both kept, so one processed on one machine may come back if the other captured at the
same time) and a `.gitignore` (`*.tmp`, `.DS_Store`, `Thumbs.db`, `desktop.ini`); lines
the user adds to them stay. The generated index is rebuilt after a restore.

## State of this machine

`state/<space>.json` in the configuration folder keeps, for this machine only (never
backed up, no text of the conversation): the active task and when it was last written,
when the assistant was last reminded to write, and the sessions that started and ended,
with the path of Claude Code's transcript of each. The Stop reminder, the request after a
compaction and the offer to recover a session that did not end come from it.

`panel.json`, also in the configuration folder and readable only by the user, says where
the panel listens while it runs (its process, port and key); it goes away when the panel
stops. `panel.lock`, next to it, lets one `augmented-brain panel` (or `panel stop`) at a
time start or stop the panel, so several at once start only one. The panel changes notes
only through the same code as the MCP tools.

## Decisions

- **A folder per type**, not one flat folder with prefixes: a newcomer sees at a glance what there is and where each thing goes.
- **One profile for the person**, next to the configuration; each space may add its own `profile.md`.
- **Shared notes, not spaces that read each other.** What holds everywhere lives once, in the shared notes, rather than copied into each space (copies drift) or by letting a space read another's memory (which would mix work and personal). Only rules and knowledge can be shared; tasks, areas and people stay in their space.
- **Importing an existing memory** of Claude Code (`augmented-brain import`) reads the header only: a field this format has is kept, at the top or under `metadata`; a person's own conventions are turned into those fields by a step of their own before importing, not by the program. It plans first and writes only with `--apply`, never over an existing note, and never changes the source.
