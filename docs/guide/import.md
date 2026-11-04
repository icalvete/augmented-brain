# Bring in a memory you already have

[Leer en español](import.es.md)

If Claude Code has been keeping a memory for you, you do not start from zero. Claude Code
writes it in `~/.claude/projects/<folder>/memory/`. You can bring it into your spaces.

## First, look

```bash
augmented-brain import ~/.claude/projects/<folder>/memory
```

That writes nothing. It shows where each note would go, and what it could not decide:

- files that are not notes (an index, say);
- notes about you: copy what matters into your profile;
- links that would cross from one space to another;
- ids your space already has.

The folder you import from is only read, now and later.

## Then, write

When the list looks right, run it again with `--apply`:

```bash
augmented-brain import ~/.claude/projects/<folder>/memory --apply
```

It never writes over a note that already exists.

## What each file becomes

Each file becomes a note of its kind: by its name (`task-…`, `area-…`, `ref-…`,
`lesson-…`, `rule-…`, `person-…`), or else by what Claude Code says it is. A task with no
status comes in parked, waiting on you, with "review this imported task" as its next
step.

The rest comes along too:

- The captures of your old `inbox.md` (each `- …` line) go to the inbox of the space of
  `*` (below), dated, and never twice.
- `archive/` and `docs/<task>/` are copied as they are, next to their note, also when you
  import again later.
- A file with a secret written in it (a key, a token) is imported, but named in the
  report. The backup will not upload it until the secret is out. `augmented-brain check`
  names it too.
- A reference or lesson that is a list of people (one `## Name` section each, with a word
  like "people" or "team" in its id or title) comes in as it is, and the report names it.
  People are notes of their own, so ask Claude to save each one as a person.

## More than one space

A small file says which space each area goes to, which notes go somewhere other than
their area, and which go to the shared notes:

```text
area-family = personal
task-evening-course = work
lesson-git-small-commits = shared
* = work
```

`*` is for everything else. Pass the file with `--map`:

```bash
augmented-brain import ~/.claude/projects/<folder>/memory --map map.txt
```

If the notes name repositories in `repos:`, `--repos-base <folder>` says what those paths
are relative to. They are then rewritten relative to the space's work folder.

## Turn off the old memory

Has Claude Code been keeping that memory by itself? Then it goes on keeping it for your
spaces' folders: a second memory, next to the space's, that drifts from it.
`augmented-brain doctor` tells you, and how to turn it off: `"autoMemoryEnabled": false`
in Claude Code's settings. What it already has stays, ready to import.

All the options: [Commands](commands.md#augmented-brain-import).
