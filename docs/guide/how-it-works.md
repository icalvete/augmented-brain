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
| **2 · The plugin** | The doors Claude uses to reach the program: a few hooks, an MCP server, and the guides, so Claude can answer your questions about augmented brain. | Inside Claude Code, which installs it |
| **1 · The program** | `augmented-brain`, a single file. It guards commands, keeps tasks, rebuilds the index and runs the backup. | `~/.local/bin/augmented-brain` |
| **0 · Your files** | Plain text: the configuration, your notes and, if you want, their backup. | `~/.config/augmented-brain/`, `~/augmented-brain/` and your git repository |

Put another way: **your notes are a notebook, the program keeps it in order and the
plugin is the door.** You only talk to Claude. Claude comes in through the door, and the
program writes in the notebook.

Two words that come up a lot:

- **Hook**: a call Claude Code makes at a given moment. Here: when a session starts,
  before Claude answers each message (to tell it which tasks changed in another session),
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
off. A session opened for a task (`session start`) hears only that task's dates.

**2. You say "park it".** Claude uses the `task_park` tool. The program writes in the
task file where it stopped and what comes next, and rebuilds the index. Opening,
resuming and closing work the same way. See [Tasks](tasks.md).

**3. Claude is about to run a command.** The guard hook reads it first, and lets it
through, asks you, or stops it. See [The guard](guard.md).

**4. Claude finishes an answer.** If the session has a task open and nothing was written
in it for 20 minutes, a hook reminds Claude to write in its journal, once. Each session
has its own task, so two windows never get each other's reminder. On Linux and macOS,
the same hook also looks at the memory of the window: if it has grown far past normal,
Claude tells you, so you can park the task before the system closes the window. See
[Troubleshooting](troubleshooting.md#the-window-uses-a-lot-of-memory).

**5. You close the session.** The end hook notes that the session ended well. It starts
the [backup](backup.md) of the space in the background, if you set one up. You wait for
nothing. If the network does not answer, the backup gives up after 3 minutes instead of
hanging. If the backup fails, Claude tells you at the start of the next session.

If a session did not end well (the window was closed by force, the computer went down),
the next one offers to recover it. Claude reads what you both said after the task's last
journal entry, from Claude Code's own record of the conversation on this computer (kept
30 days), and writes down what is missing.

## Where your notes live

<a id="where-files-live"></a>Your notes, one folder per space:

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

Everything else (the configuration folder, what this computer keeps for itself, the
program) and what travels in the backup is in
[Where everything lives](where-things-live.md).

## What it does not do

- **No server and no account.** If Claude is not open, nothing runs, except:
  - the panel, while you use it;
  - the backup that the end of a session starts;
  - a task's own session, started with `augmented-brain session start`, until it is
    stopped or the task is closed;
  - `augmented-brain watch pr`, which waits for a pull request's checks, three hours at
    most;
  - if you set it up, [remote access](remote.md), which keeps running so you can reach
    your sessions from another device.
- **It sends your data to no one.** Only the backup, if you set it up, talks to your
  repository. Apart from that, once a day, it asks GitHub which version is the latest,
  sending nothing of yours. `AB_NO_UPDATE_CHECK=1` turns that off.
- **It does not change your Claude Code settings.** The plugin goes in and out with
  Claude Code's own commands, which the installer, `update` and `uninstall` run for you.
- **As it comes, it does not stay inside your spaces.** It works in every Claude Code
  session on this computer. Outside your spaces it uses the default space, and the guard
  watches the same. On the panel's Guard page you can keep it to your spaces instead.
  See [Outside every space](spaces.md#outside-every-space).
- **Mind this: your assistant reads your notes**, like any file it opens. They travel
  with your conversations to its provider. Do not keep there what you would not share.

<a id="your-files"></a><a id="work-and-personal-apart"></a><a id="shared-notes"></a><a id="the-backup"></a><a id="on-a-new-machine"></a><a id="bring-in-a-memory-you-already-have"></a><a id="the-panel"></a><a id="update-it"></a><a id="remove-it"></a>
