# Editing notes by hand

[Leer en español](editing-notes.es.md)

Your notes are plain Markdown files, and they are yours: open any of them with any
editor and change it. Most of the time it is easier to ask Claude ("change the next step
of the backup task"): its tools check everything before writing. By hand is fine for a
typo, a sentence written better, or a note you want to reorganize. This page says what
to keep, so the program keeps reading the note.

## Before and after

- **Before**, look at whether a Claude session is working on that same note. If it
  writes while you edit, one of the two changes is lost. An open session hears of your
  change before its next answer.
- **After**, run `augmented-brain check` in a folder of the space. It checks every note
  and rebuilds the index. If it names a problem, fix it: until then, that note is left
  out, and Claude does not see it.

## What a note looks like

A note has a header between two `---` lines, then the text:

```markdown
---
id: task-new-website
type: task
title: New website
summary: Move the website to the new hosting
area: area-home
status: execution
waiting_on: me
next: Point the domain to the new server
created: 2026-11-02
updated: 2026-11-13
---
## Essentials
- …

## Journal
- 2026-11-02 — Task opened.
- 2026-11-13 — Hosting chosen: the cheapest one with backups.
```

The text below the header is yours: write what you like there. The header is what the
program reads, so it has rules.

## The header

- **Keys in English, always.** `title`, `status`, `next`… whatever language you write in.
  So are the fixed values: `status: parked`, never `status: aparcada`.
- **Only the keys the program knows.** A key it does not know makes the note invalid:
  the program cannot tell a typo from a key of a newer version, so it leaves the note
  out instead of guessing. Do not add keys of your own; write that in the text.
- **One line per field.** A title or a next step written over several lines is read as
  one line.
- **Dates as `YYYY-MM-DD`**: `2026-11-03`, not `3-11-2026` or `2026-11-3`.
- **`id` is the file name** without `.md`. If you change one, change the other, and every
  note that names it in `related` or `parent`. Easier: ask Claude to rename it.
- **The values of `status`**: `definition`, `analysis`, `plan`, `execution`,
  `validation`, `closed`, `parked` or `someday`. `waiting_on` is `me`, `nobody`, a
  person's name or another task's id.

Every key and what it means is in the [memory format](../spec/memory-format.md#file-header).

## Tasks

- **The first section is the essentials, the last one the journal.** The program relies
  on that: it adds journal entries at the end of the file, and reads the essentials to
  resume. Change the headings' words if you like, but not their order.
- **A journal entry is one line**: a dash, the date, a long dash `—` and the text:
  `- 2026-11-13 — What happened`. A short dash `-` in place of the long one is not read as an
  entry.
- **Steps are checkboxes** under the Plan heading: `- [ ]` to do, `- [x]` done.

## The inbox

`inbox.md` has no header: one line per capture, written like a journal entry,
`- 2026-11-13 — Call the insurance`. To have it come up on a day, end it with
`(remind: 2026-11-20)`.

## What not to touch

- **`MEMORY.md`**, the index: the program writes it after every change, and yours would
  be lost.
- **The ending `(from <space>)`** of a capture or a journal entry: it says that another
  space sent it. Do not write it yourself.
- **The `.augmented-brain/` folder**: it holds the version of the notes' format.
- **Deleting a note.** Nothing is ever deleted: close a task (`status: closed`), and ask
  Claude to retire a rule, reference or lesson that no longer holds: it moves to
  `archive/`, which is never loaded.

## Next

- [Tasks](tasks.md): the phases, steps and the journal.
- [Areas, knowledge and the rest](notes.md): the other kinds of note.
- [Troubleshooting](troubleshooting.md#a-note-is-not-valid): when `check` says a note is
  not valid.
