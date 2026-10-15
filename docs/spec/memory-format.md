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

## Layout of a space

```
<memory folder>/
├── MEMORY.md          # the index loaded at the start of every session (generated)
├── profile.md         # optional: what only applies in this space (your role at work…)
├── inbox.md           # quick captures, processed in the weekly review
├── areas/             # ongoing responsibilities (one file each)
├── tasks/             # projects with a beginning and an end (one file each)
├── knowledge/         # how things work (reference) and what you learned (lesson)
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
type: task                       # profile | area | task | reference | lesson | rule
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

[spaces.work]
memory = "~/brains/work"
work_folders = ["~/work"]        # opening the assistant here selects this space
backup = "git@github.com:me/brain-work-private.git"
production_aws_profiles = ["prod", "prod-eu"]   # the user confirms first
untouchable_hosts = ["db01", "db02", "db03"]    # ssh to them: the user confirms first
git_allowed_repos = ["~/work/notes"]            # the assistant may commit and push here, with git -C

[spaces.personal]
memory = "~/brains/personal"
work_folders = ["~/personal"]
backup = "git@github.com:me/brain-personal-private.git"
```

## Decisions

- **A folder per type**, not one flat folder with prefixes: a newcomer sees at a glance what there is and where each thing goes.
- **One profile for the person**, next to the configuration; each space may add its own `profile.md`.
- **Importing an existing memory** from other formats is not part of 0.1; it is planned for 0.2.
