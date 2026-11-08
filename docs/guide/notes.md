# Areas, knowledge and the rest

[Leer en español](notes.es.md)

Tasks come and go. Around them, a space keeps a few other kinds of notes. Each one is a
Markdown file in its own folder, and you never have to create them by hand: you say it,
and Claude writes it.

## Areas

"The blog", "the servers", "the house". An **area** is something you look after with no
end date. It groups the tasks and the knowledge that belong to it.

You say: "the newsletter is a new area". Claude creates `areas/area-newsletter.md`. Its
body says what you look after, how it is going and what to watch. The program lists the
area's tasks and knowledge inside it by itself.

The first-session interview creates your first areas, from what you said you look after.

## Knowledge

What is worth keeping beyond one task. There are two kinds, in `knowledge/`:

- A **reference** says how something works: "how the blog is deployed".
- A **lesson** says what you learned, with a title that already states the conclusion:
  "Test the backup restore, not only the backup".

You say: "save that as a lesson". Both belong to an area, and Claude opens them when a
task needs them.

## People

You say: "Diego runs the hosting company; I call him Dieguito". Claude saves
`people/person-diego.md`. The next time you say "tell Dieguito it is done", it knows who
he is.

A **person** has a name, who they are to you, the other names you use for them, and
anything worth remembering. A task can wait on a person (`waiting_on: Diego`).

**The same person can be in more than one space.** Say Diego is your client and also
your friend. Each space keeps its own note for him, with the same id, and that note holds
only what you want to see there: his role, his other names and his details. The panel
joins them into one card, so you see everything about him in one place, and for each
detail you tick the spaces where it shows. A detail about your friend that you leave
unticked for `work` is never written in a work file, so it never reaches a work backup.
When you ask Claude to remember someone, it asks in which spaces they should be visible.

## Inbox

An idea that is not a task yet. You say: "note this: look at the new hosting prices".
Claude adds one dated line to `inbox.md`, without opening a task:

```
- 2026-10-17 — Look at the new hosting prices
- 2026-10-17 — Call the insurance (remind: 2026-10-22)
```

Each line is a **capture**. The [weekly review](weekly-review.md) empties the inbox: each
capture becomes a task, goes to an area or knowledge, or is dropped.

"Remind me on Thursday to call the insurance" is a capture with a date. From a week
before, and every day after, it comes up at the start of the session, until the weekly
review deals with it (or you mark it done in the panel).

## Rules

How you want Claude to work. You say: "from now on, answer in short sentences". Claude
saves it as a **rule**. Every session loads every rule, whole.

There are three levels:

- **common rules**, which come with the program: verify before advising, never push
  unless asked, never become root, never write secrets…
- **shared rules**, yours, for every space ("answer in short sentences");
- **space rules**, in the space's `rules/`, for what only applies there ("the database
  of the shop is never restarted on a Friday").

A rule that holds everywhere goes to the shared notes. Tell Claude "this holds
everywhere".

## Your profile

Who you are and how you like to work: your name, what you do, your language, how long
you like answers, how much Claude may do without asking. It is one file for every space,
`profile.md` in the configuration folder. The first-session interview writes it. To
change it, tell Claude.

## Repositories

What each of your git repositories is, one line each: "`~/code/blog` — The blog, built
with Hugo". It lives in one reference of the space, `knowledge/ref-repositories.md`, so
the backup keeps it. You write the line in the panel's Repositories page, or Claude
writes it for you.

## The index

`MEMORY.md` is the list Claude reads at the start of every session: your profile line,
the rules, the areas and the live tasks with their status and next step. The shared
references and lessons are listed by title and file. At the start, Claude gets it
shorter: tasks go without their summary, and a task shows its next step (its first 120
characters) only when it waits on you. Claude reads a task before working on it. The
program writes this file after every change: do not edit it by hand, it would be
rewritten.

## Documents and the archive

Two folders are never loaded:

- `docs/<task>/` keeps the long documents a task produces (a plan, an analysis), next to
  the notes and out of the project's repository.
- `archive/` keeps old things: long journals moved out, summaries of compacted
  conversations. Nothing there is ever deleted.

## Editing by hand

Every note is plain Markdown. Open it with any editor and change it. Afterwards, run
`augmented-brain check` in a folder of the space: it checks every note and rebuilds the
index.

## Next

- [What to say to Claude](phrases.md): the phrases for each kind of note.
- [Memory format](../spec/memory-format.md): every folder and header field.
