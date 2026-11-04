# How it works inside

[Leer en español](how-it-works.es.md)

augmented brain is one small program and a folder of notes. This page is how the pieces
fit. To use it, you do not need any of this.

<picture>
  <source media="(max-width: 600px)" srcset="../img/how-layers-narrow.en.svg">
  <img alt="augmented brain in four layers" src="../img/how-layers.en.svg">
</picture>

## Four layers

| Layer | What it is | Where it lives |
|---|---|---|
| **3 · You and your assistant** | You talk to Claude Code as usual. No new commands to learn. | Your terminal or the Claude app |
| **2 · The plugin** | The doors Claude uses to reach the program: a few hooks and an MCP server. | Inside Claude Code, which installs it |
| **1 · The program** | `augmented-brain`, a single file. It guards commands, keeps tasks, rebuilds the index and runs the backup. | `~/.local/bin/augmented-brain` |
| **0 · Your files** | Plain text: the configuration, your notes and, if you want, their backup. | `~/.config/augmented-brain/`, `~/augmented-brain/` and your git repository |

Put another way: **your notes are a notebook, the program keeps it in order and the
plugin is the door.** You only talk to Claude. Claude comes in through the door, and the
program writes in the notebook.

Two words that come up a lot:

- **Hook**: a call Claude Code makes at a given moment. Here: when a session starts,
  before every command and file change, after each answer, when a long conversation is
  compacted, and when the session ends. Each time, Claude Code starts the program, asks
  it, and the program exits.
- **MCP** (Model Context Protocol): an open standard to give tools to an assistant. The
  program offers its own ("open a task", "park it") and Claude uses them. Since it is a
  standard, other assistants will be able to use the same ones.

The list of hooks is in [Commands](commands.md#augmented-brain-hook). The tools are in
[MCP tools](mcp-tools.md).

## A session, step by step

<picture>
  <source media="(max-width: 600px)" srcset="../img/how-session-narrow.en.svg">
  <img alt="A session, step by step" src="../img/how-session.en.svg">
</picture>

**1. You open Claude.** The start hook looks at the folder you are in and picks your
[space](spaces.md). Before you type anything, it hands Claude your profile, your rules
and the list of live tasks. It also says what is due soon, whether the weekly review is
due, and whether an earlier session was cut off. That is how Claude knows where you left
off.

**2. You say "park it".** Claude uses the `task_park` tool. The program writes in the
task file where it stopped and what comes next, and rebuilds the index. Opening,
resuming and closing work the same way. See [Tasks](tasks.md).

**3. Claude is about to run a command.** The guard hook reads it first, and lets it
through, asks you, or stops it. See [The guard](guard.md).

**4. Claude finishes an answer.** If a task is open and nothing was written in it for 20
minutes, a hook reminds Claude to write in its journal, once. The same hook looks at the
memory of the window: if it has grown far past normal, Claude tells you, so you can park
the task before the system closes the window. See
[Troubleshooting](troubleshooting.md#the-window-uses-a-lot-of-memory).

**5. You close the session.** The end hook notes that the session ended well. It starts
the [backup](backup.md) of the space in the background, if you set one up. You wait for
nothing. If the backup fails, Claude tells you at the start of the next session.

If a session did not end well (the window was closed by force, the computer went down),
the next one offers to recover it. Claude reads what you both said after the task's last
journal entry, from Claude Code's own record of the conversation on this computer (kept
30 days), and writes down what is missing.

## Where files live

Your notes, one folder per space:

```
~/augmented-brain/
├── work/                 a work folder: open Claude here for work
│   ├── notes/
│   │   ├── MEMORY.md     the index Claude reads at the start (written by the program)
│   │   ├── inbox.md      quick notes, to sort out later
│   │   ├── areas/        what you always look after
│   │   ├── tasks/        one file per task
│   │   ├── knowledge/    how things work and what you learned
│   │   ├── people/       who you deal with
│   │   ├── rules/        rules for this space only
│   │   ├── reviews/      weekly reviews
│   │   ├── docs/         long documents of a task; never loaded
│   │   └── archive/      old stuff; never loaded, never deleted
│   └── (your projects)
└── personal/             the same, for personal things
```

The configuration folder, `~/.config/augmented-brain/` (on Windows,
`%APPDATA%\augmented-brain`):

```
config.toml       the spaces and the guard's lists
secrets.toml      the names of your secrets, never their values
profile.md        who you are, for every space
shared/           rules, references and lessons for every space
state/            what this computer knows of each space's sessions; never backed up
migrations/       a full copy of the notes before a new version moves them
panel.json        where the panel listens, while it runs
```

Each file is described in [Configuration](configuration.md) and in the
[memory format](../spec/memory-format.md).

## What it does not do

- **No server and no account.** If Claude is not open, nothing runs, except the panel
  while you use it and the backup that the end of a session starts.
- **It sends your data to no one.** Only the backup, if you set it up, talks to your
  repository. Apart from that, once a day, it asks GitHub which version is the latest,
  sending nothing of yours. `AB_NO_UPDATE_CHECK=1` turns that off.
- **It does not change your Claude Code settings.** The plugin goes in and out with
  Claude Code's own commands, which the installer, `update` and `uninstall` run for you.
- **Mind this: your assistant reads your notes**, like any file it opens. They travel
  with your conversations to its provider. Do not keep there what you would not share.

## Moved from this page

This page used to hold more. Each part now has a page of its own:

- <a id="your-files"></a><a id="work-and-personal-apart"></a><a id="shared-notes"></a>Your files, spaces and shared notes: [Spaces](spaces.md) and [Areas, knowledge and the rest](notes.md).
- <a id="the-backup"></a><a id="on-a-new-machine"></a>The backup, and a new machine: [Backup and restore](backup.md).
- <a id="bring-in-a-memory-you-already-have"></a>Importing an existing memory: [Bring in a memory](import.md).
- <a id="the-panel"></a>The panel and every page in it: [The panel](panel.md).
- <a id="update-it"></a><a id="remove-it"></a>Updating and removing it: [Installing](install.md#updating).
