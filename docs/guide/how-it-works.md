# How it works

[Leer en español](how-it-works.es.md)

augmented brain is one small program and a folder of notes. This is how the pieces fit:

<picture>
  <source media="(max-width: 600px)" srcset="../img/how-layers-narrow.en.svg">
  <img alt="augmented brain in four layers" src="../img/how-layers.en.svg">
</picture>

## Four layers

| Layer | What it is | Where it lives |
|---|---|---|
| **3 · You and your assistant** | You talk to Claude Code as usual. No new commands to learn. | Your terminal |
| **2 · The plugin** | The doors Claude uses to reach the program: two hooks and an MCP server. | Inside Claude Code, which installs it |
| **1 · The program** | `augmented-brain`, a single file. It guards commands, keeps tasks, rebuilds the index and runs the backup. | `~/.local/bin/augmented-brain` |
| **0 · Your files** | Plain text: the configuration, your notes and, if you want, their backup. | `~/.config/augmented-brain/`, `~/augmented-brain/` and your git repo |

Put another way: **your notes are a notebook, the program keeps it in order and the
plugin is the door.** You only talk to Claude; Claude comes in through the door and the
program writes in the notebook.

Two words that come up a lot:

- **Hook**: a call Claude Code makes at a given moment. Here, when a session starts and
  before every command. Each time, Claude Code starts the program, asks it, and the
  program exits. Nothing keeps running in the background.
- **MCP** (Model Context Protocol): an open standard to give tools to an assistant. The
  program offers its own ("open a task", "park it") and Claude uses them. Since it is a
  standard, other assistants will be able to use the same ones.

## A session, step by step

<picture>
  <source media="(max-width: 600px)" srcset="../img/how-session-narrow.en.svg">
  <img alt="A session, step by step" src="../img/how-session.en.svg">
</picture>

**1. You open Claude.** The start hook looks at the folder you are in and picks your
space. Before you type anything, it hands Claude your profile, your rules and the list of
live tasks. That is how it knows where you left off.

**2. You say "park it".** Claude uses the `task_park` tool. The program writes down in the
task file where it stopped and what the next step is, and rebuilds the index. The next
session starts from there. Opening, resuming and closing work the same way.

**3. Claude is about to run a command.** The guard hook reads it first and decides:

- **Go ahead**: almost always. `ls`, `go test`, `aws --profile dev s3 ls`.
- **Ask you**: production, or a machine you marked as untouchable.
  `aws --profile prod …`, `ssh db01 …`.
- **Stop**: pushing code, root and secrets. `git push`, `sudo …`, a token in a command.

It also looks inside `ssh`, `bash -c` and loops. It is a net against slips, not a security
wall: read the [disclaimer](../../DISCLAIMER.md).

## Your files

Everything lives in `~/augmented-brain`, one folder per **space**. You work inside it,
and its notes go in `notes/`:

```
~/augmented-brain/
├── work/                 open Claude here for work
│   ├── notes/
│   │   ├── MEMORY.md     the index Claude reads at the start (written by the program)
│   │   ├── inbox.md      quick notes, to sort out later
│   │   ├── areas/        what you always look after: "the website", "the servers"
│   │   ├── tasks/        one file per task
│   │   ├── knowledge/    how things work and what you learned
│   │   ├── people/       who you deal with, so it knows who "Marcos" is
│   │   ├── rules/        rules for this space only
│   │   ├── reviews/      weekly reviews
│   │   └── archive/      old stuff; never loaded, never deleted
│   └── (your projects)
└── personal/             and here for personal things, with its own notes/
```

Your **profile**, who you are and how you like to work, is one for every space:
`~/.config/augmented-brain/profile.md`. You don't write it by hand: the first time you
open Claude, it asks six quick questions, saying why it asks each one. With your answers
it saves the profile, creates your areas, tells the guard what is production and which
machines must not be touched, and notes the people you work with. To change anything,
tell it.

A task is a text file. At the top, between the `---` lines, the data the program uses.
Below, what you and Claude write:

```markdown
---
id: task-add-a-contact-form
type: task
title: Add a contact form
area: area-website
status: parked
waiting_on: me
next: Write the plan
created: "2026-10-15"
updated: "2026-10-15"
---
## Essentials
Name, email and message, sent by email.

## Journal
- 2026-10-15 — Task opened.
- 2026-10-15 — Parked. Next: Write the plan
```

You can open and edit everything with any editor. The full format is in
[memory-format.md](../spec/memory-format.md).

## Work and personal, apart

A space is a full set of notes, rules and backup. The installer creates two, `work` and
`personal`. **The folder where you open Claude picks the space**, following your
configuration:

```toml
# ~/.config/augmented-brain/config.toml
default_space = "personal"

[spaces.work]
memory = "~/augmented-brain/work/notes"
work_folders = ["~/augmented-brain/work"]

[spaces.personal]
memory = "~/augmented-brain/personal/notes"
work_folders = ["~/augmented-brain/personal"]
```

In `~/augmented-brain/work/website`, Claude sees your work notes. In
`~/augmented-brain/personal/blog`, your personal ones. They never mix.

What if you open Claude in another folder? For example, the desktop app starts each new
session in a scratch folder of its own. Then the **default space**, `personal`, is used,
and Claude tells you in one line, with how to switch to work. You can change it in
`default_space`, or remove it.

Do your projects already live elsewhere, like `~/code`? Add that folder to the
`work_folders` of the right space. For a new space (a client, say):
`augmented-brain space add client`.

## The backup

The notes folder of each space is also a git repository. The backup pushes it to a
private repository of yours. Set it up once:

```bash
augmented-brain backup setup git@github.com:you/my-brain.git
```

And from then on, whenever you like (at the end of the day, say):

```bash
augmented-brain backup          # this space
augmented-brain backup --all    # every space
```

What you get:

- **A new machine in minutes.** `augmented-brain restore <url>` brings the whole space
  back: notes, tasks, rules, your profile and the space's settings. Open Claude and carry
  on where you left off.
- **Several machines, one brain.** Before pushing, the backup fetches what your other
  machine did and merges it.
- **Full history.** Every backup is a git commit, with the machine and the time. With
  `git log` you see what changed, and you can go back to any day.
- **Your place.** GitHub, GitLab or your own server: you pick the repository. augmented
  brain has no cloud.
- **One repository per space.** Work can go to your company's git, and personal to yours.
- **No secrets go up.** If a note holds something that looks like a password or a token,
  the backup stops and tells you which file.
- **Nothing gets overwritten.** If two machines changed the same note, it stops without
  touching anything and explains how to sort it out. The index, being generated, is
  rebuilt by itself.

It does not run by itself yet: it is a command you run.

## What it does not do

- **No server and no account.** If Claude is not open, nothing runs.
- **It sends data to no one.** Only the backup, if you set it up, talks to your repository.
- **It does not touch your Claude Code settings.** The plugin goes in and out with Claude
  Code's own commands.
- **Mind this: your assistant reads your notes**, like any file it opens. They travel with
  your conversations to its provider. Do not keep there what you would not share.

## Remove it

```bash
claude plugin uninstall brain@augmented-brain
claude plugin marketplace remove augmented-brain
rm ~/.local/bin/augmented-brain
rm -r ~/.config/augmented-brain        # configuration and profile
```

Your notes stay in `~/augmented-brain/`. They are yours.
