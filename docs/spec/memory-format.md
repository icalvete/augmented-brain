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
- its own risk settings (which cloud profiles are production, which hosts must not be touched);
- its own backup repository.

The space is chosen by **the folder you open the assistant in**. Spaces never read each other's memory. They share the program, the common rules and the safety hooks.

Spaces are declared in the configuration file (see "Configuration" below), not inside the memory.

By default, `augmented-brain init` creates two spaces, `work` and `personal` (in Spanish, `trabajo` and `personal`). Each one is a folder in `~/augmented-brain/`, which is its work folder, and its memory folder sits inside it as `notes/` (`notas/`). Any other layout can be set with `init --custom` or by editing the configuration.

A session that runs outside every work folder (for example, the scratch folder the Claude desktop app opens a new session in) uses the **default space**, `default_space` in the configuration; `init` sets it to `personal`. Without one, no memory is loaded there and the assistant tells the user which folders to open.

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
├── rules/             # this space's own rules (the common rules ship with the program)
├── reviews/           # one file per weekly review
└── archive/           # old journals and merged files; never loaded, never deleted
```

`MEMORY.md` is the name Claude Code loads automatically from the memory folder; only its first 200 lines (or 25 KB) are read, so the program keeps it well under that.

**Your profile** (who you are, your language, how you like answers) is the same person in every space, so it lives once, next to the configuration: `profile.md` in the configuration folder. A space's own `profile.md` only adds what applies there.

## File header

Every file except `MEMORY.md` and `inbox.md` starts with a YAML header between `---` lines. Keys and enumerated values are always English, whatever language you write in.

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

The program never searches the prose. It relies on two positions only: the essentials are the **first** section, and the journal is the **last**, so a new entry is simply appended at the end of the file. When a task file passes about 10 KB, its oldest journal entries move to `archive/` and the task links to them.

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

Rules tell the assistant how to work. There are two levels:

- **common rules**, shipped with the program (answer style, verify before advising, never commit or push unless asked, never escalate to root, never write secrets…);
- **space rules**, in `rules/`, for what only applies to that space (for example, "hosts db01–db03 are never restarted without explicit permission").

All of them are loaded at the start of every session.

## Generated files

The program rebuilds these after every change; editing them by hand is pointless:

- `MEMORY.md`: your profile line, the rules, the areas, and the live tasks grouped by area, one line each, with status, who it waits on and the next step.
- The "Tasks" and "Knowledge" lists inside each area.

## Configuration

`~/.config/augmented-brain/config.toml` (or the platform equivalent) declares the spaces:

```toml
language = "en"                  # en | es: the language the program speaks to you
default_space = "personal"       # used outside every work folder (optional)

[spaces.work]
memory = "~/augmented-brain/work/notes"
work_folders = ["~/augmented-brain/work", "~/code"]   # opening the assistant here selects this space
backup = "git@github.com:me/brain-work-private.git"
production_aws_profiles = ["prod", "prod-eu"]   # the user confirms first
untouchable_hosts = ["db01", "db02", "db03"]    # ssh to them: the user confirms first
git_allowed_repos = ["~/code/website"]          # the assistant may commit and push here, with git -C

[spaces.personal]
memory = "~/augmented-brain/personal/notes"
work_folders = ["~/augmented-brain/personal"]
backup = "git@github.com:me/brain-personal-private.git"
```

## Backup

Each space backs up to its own private git repository: the memory folder itself is the
repository. Every backup also copies this space's part of the configuration and the
profile into `.augmented-brain/` inside the memory folder, so restoring on a new machine
needs nothing else. Paths are kept with `~`, so they point to the new machine's home, and
the notes go back to the same place inside the work folder. A restore also installs the
Claude Code plugin when it is missing.

A backup is refused while any file of the memory holds a secret. The generated index is
rebuilt after a restore.

## Decisions

- **A folder per type**, not one flat folder with prefixes: a newcomer sees at a glance what there is and where each thing goes.
- **One profile for the person**, next to the configuration; each space may add its own `profile.md`.
- **Importing an existing memory** from other formats is not part of 0.1; it is planned for 0.2.
