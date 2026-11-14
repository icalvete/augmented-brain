# Spaces

[Leer en español](spaces.es.md)

You open Claude in `~/augmented-brain/work/website`. It sees your work tasks, your work
rules and the people from work. You open it in `~/augmented-brain/personal/blog`. Now it
sees your personal notes, and nothing from work.

That is a **space**: a full set of notes, rules and backup, kept apart from the others.
The installer creates two, `work` and `personal` (in Spanish, `trabajo` and `personal`).

## The folder picks the space

Each space has one or more **work folders**. When you open Claude inside one of them, you
are in that space. The space's notes live in a `notes/` folder inside it:

```
~/augmented-brain/
├── work/          a work folder: open Claude here for work
│   ├── notes/     the work notes
│   └── website/   a project of yours
└── personal/      and here for personal things
    └── notes/
```

The configuration file says which folders belong to which space:

```toml
# ~/.config/augmented-brain/config.toml
default_space = "personal"

[spaces.work]
memory = "~/augmented-brain/work/notes"
work_folders = ["~/augmented-brain/work", "~/code"]

[spaces.personal]
memory = "~/augmented-brain/personal/notes"
work_folders = ["~/augmented-brain/personal"]
```

`memory` is where the notes are. `work_folders` are the folders that pick the space.

To know which space a folder belongs to, run this in it:

```bash
augmented-brain space
```

## Outside every space

augmented brain works in every Claude Code session on this computer, whatever the
folder: its plugin is installed for your whole user. A folder that is in no work folder
is outside every space.

The desktop app starts each new session in a scratch folder of its own. That folder is
in no space. Then the **default space** is used, `personal` unless you change it, and
Claude tells you in one line, with how to switch to work.

The default space matters for one more thing: its backup carries your profile, the shared
notes and the names of your secrets (never their values). Set it with `default_space` in
the configuration. With a single space, it is that one.

What changes outside, next to a session in a space:

| | In a space | Outside every space |
|---|---|---|
| Profile, rules, tasks and the rest of the notes | The space's | The default space's, and Claude tells you |
| What Claude writes in your notes | Goes to the space | Goes to the default space |
| Tasks changed by another session | Claude hears of them | Those of the default space |
| Reminder to write in the task's journal | Yes | No |
| A session cut short, offered to recover | Yes | No |
| [The guard](guard.md) | All of it | All of it, the same |
| Backup when the session ends | The space's | The default space's |
| Warnings about the window's memory and the model | Yes | Yes |

The guard does not look at the folder: its lists hold for the whole computer. So in any
project, also one you never gave augmented brain, Claude cannot commit or push until you
[let it in that repository](guard.md#let-claude-commit-in-a-repository).

To bring a folder into a space, add it to the space's `work_folders` (next).

## Your projects live elsewhere?

If your code is in `~/code`, add that folder to the `work_folders` of the right space, as
in the example above. Then opening Claude in `~/code/shop` uses the work space.

## Add a space

For a client, a side project or a volunteer job:

```bash
augmented-brain space add client
```

It asks for the work folder and the notes folder, suggesting
`~/augmented-brain/client` and `~/augmented-brain/client/notes`. Press Enter to accept.
If there was no default space yet, it sets one. The options are in
[Commands](commands.md#augmented-brain-space-add).

Then two things:

- **Give it a backup.** Open Claude in the new work folder and say "set up the backup";
  each space has its own. See [Backup and restore](backup.md).
- **Work in it.** Open Claude in its work folder, or in a folder inside it, and the
  session uses the new space.

## What each space has, and what they share

Each space has its own:

- notes: tasks, areas, knowledge, people, inbox, reviews;
- rules, on top of the common ones;
- backup repository.

Every space shares:

- **your profile**: who you are and how you like to work. You are the same person at
  work and at home.
- **the shared notes**: rules, references and lessons that hold everywhere (below).
- **the guard's lists**: production AWS profiles, machines not to touch, the
  repositories where Claude may commit and those it must never touch, and the root
  exceptions. They hold on the whole machine, whichever space you write them from. A
  production profile saved from the work space is guarded in the personal one too, and a
  repository where Claude may commit is allowed from every space.

Spaces never read each other's notes on their own.

## A task of another space

Sometimes you decide, in one space, something about a task of another: in a personal
session you settle how a work task goes on. Say so, and Claude reads or writes that task
where it lives, in the other space. It can read it, list that space's live tasks to find
it, add to its journal, tick a step, and change its next step, status or who it waits
on.

- What Claude writes there ends in `(from personal)`, so whoever reads it in the other
  space knows where it was decided.
- Nothing is copied into this space's notes: what Claude reads stays in the
  conversation.
- Claude does it only when you ask, as with
  [sending a capture to another space](notes.md#sending-a-capture-to-another-space).
- Closing that task is done from a session of its own space, because closing also ends
  the task's session and its worktree.
- That space is backed up when this session ends, if it has a backup: it does not wait
  for a session of its own.

## Shared notes

Some things you learn hold everywhere: how git behaves, how you like answers, a lesson
about a tool. Saving them once per space would leave two copies that drift apart. So they
go once in the **shared notes**, next to your profile, in `shared/` inside the
configuration folder.

Only rules, references and lessons can be shared. Tasks and areas always belong
to one space. A person can be in several spaces, each with its own note and only what
you want visible there (see People in Notes).

You do not move files by hand. Tell Claude "this holds everywhere" and it saves the note
as shared. Every session loads the shared rules, and lists the shared references and
lessons for Claude to open when it needs them. They travel in the backup of the default
space, with your profile.

## Next

- [Tasks](tasks.md): what goes inside a space.
- [Configuration](configuration.md): every setting of a space.
