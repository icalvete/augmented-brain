# What stays stable

[Leer en español](compatibility.es.md)

augmented brain numbers its versions the usual way: **1.x** means every version that
starts with 1. From 1.0 on, nothing on this page breaks inside 1.x. If something here
ever has to break, that is version **2.0**, and the changelog says what changes and how
to move.

The beta versions (0.x) already follow these rules, but they are not a promise yet.

## What is promised

- **Your notes.** The format the [memory format](../spec/memory-format.md) describes:
  the folders, the types of note, the fields of the header and what each one means. When
  a 1.x writes something in a new way, it brings the step that moves your notes forward,
  and runs it by itself after a full copy. Notes of any beta (0.x) move forward the same
  way.
- **Your configuration.** `config.toml`, `secrets.toml`, `guard.toml`, `session.toml`,
  `remote.toml` and `profile.md`: every setting in the
  [configuration reference](configuration.md) keeps its name and what it does.
- **The commands.** Every command and option in the [command reference](commands.md)
  stays and does the same. So do the exit codes that page gives (for example, `watch pr`
  ends with 0 or 1).
- **The assistant's tools.** Every MCP tool in the [tool reference](mcp-tools.md) keeps
  its name, its parameters and what it does. A version may add tools, or parameters you
  can leave out; never a new parameter a call must give. This matters because your rules
  and notes name these tools.
- **The environment variables** in [Configuration](configuration.md#environment-variables)
  (`AB_CONFIG_DIR`, `AB_LANGUAGE`, `AB_NO_AUTO_BACKUP`…) and those of the installer.
- **Backups.** A backup made with any 1.x restores with any later 1.x.
- **Installing and updating.** The install line on the front page keeps working, and
  `augmented-brain update` takes any 1.x to the latest one.

**A name that changes keeps working.** When a command, an option, a setting, a tool or a
parameter gets a new name, the old one still works until 2.0 and says what replaced it.
Nothing on the list goes away inside 1.x.

**An earlier version never damages your notes.** With two computers, or with a session
still open after an update, an older program may read notes a newer one wrote. The two
versions before the newest read them; if the format changed, they refuse them and ask
you to update, without writing anything.

## What is not promised

- **The words.** The program's messages, what the tools answer, what the assistant
  reads when a session starts, and the wording of the common rules change to explain
  things better. Do not read them from a script; use the exit codes.
- **The panel.** Its pages, how they look and their addresses.
- **The inside.** The folders `state/`, `history/`, `migrations/` and `marketplace/` of
  the configuration folder, the hidden commands (`panel serve`, `remote watch`), what
  `hook` and `mcp` print (the names stay: the plugin calls them), and the Go code (it is
  not a library).
- **What the guard lets through.** It may get stricter in any version: a command that
  passes today may ask tomorrow. Your lists (`git_allowed_repos` and the rest) keep
  working.
- **The systems and versions of other programs.** A 1.x may need a newer Claude Code,
  Git or operating system when older ones stop working; the changelog says so.
- **Variables for testing** that the documentation does not give.

## One exception: security

If fixing a security problem means breaking something on the list, it is fixed in a 1.x,
and the changelog says so first, with what you have to do.

## How it is checked

What can be checked is frozen in files inside the repository, and the tests fail when a
version takes away or changes something in them: the MCP tools with their parameters and
which ones a call must give, the commands with their options and the events of `hook`,
and the settings of each configuration file. Adding is fine. For the notes, every change
of format ships with its step and a test that runs it on notes written the old way, and
in every change the two versions before it must read everything the new one writes.
