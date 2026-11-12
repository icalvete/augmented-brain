# Updating and uninstalling without losing anything

[Leer en español](update-and-uninstall.es.md)

Your notes are plain files that belong to you. Updating never touches what they say, and
uninstalling leaves them where they are unless you ask for the opposite. This page says
what each step does, so you can do it calmly.

## Updating

When a new version is out, Claude tells you at the start of a session. To update:

```bash
augmented-brain update
```

What it does, in order:

1. **Checks the installation** and shows what it finds, so a problem that was already
   there is not taken for one of the new version.
2. **Backs up every space that has a backup.** If one fails, it stops and updates nothing:
   that copy is your way back. With no backup at all, it says it goes ahead without one.
3. **Downloads and installs** the new version and the Claude Code plugin. If GitHub fails
   to hand out the files, it tries three times, also through GitHub's API; if all fails,
   it says so and nothing changes.
4. **Checks the new version is in place**, and restarts the idle Claude sessions that
   still run the old one, one at a time, with the same conversation: a task's session,
   those of the remote access, and those of Claude Desktop, which stay open while only
   their augmented brain restarts. A session that is answering is left as it is.
5. **Says what is left:** the sessions it could not restart (one answering, the one you
   ran it from, on Windows any of Claude Desktop), and which task each space has active.

For those left, ask the session to park its task, and close and open it. A session that
was open keeps the old program until it restarts; `/clear` is not enough. When the
session you are in is one of them, its start tells you so. Later on,
`augmented-brain session restart --old` restarts the ones that have gone idle, and
`augmented-brain update --no-restart` updates without restarting any.

### Before you update

- **Set up a backup** if you have none: say "set up the backup" to Claude. `augmented-brain
  doctor` says which spaces have none.
- **Update every computer you use.** An older version never breaks notes a newer one
  wrote: it refuses them and asks to be updated.

### When a version changes how notes are stored

Sometimes a new version stores notes in a new way. Then the first session moves them, after
keeping a full copy in the `migrations/` folder of the configuration, and Claude tells
you. Nothing is moved without that copy.

## Uninstalling

```bash
augmented-brain uninstall
```

It shows what it will remove and asks. There are three levels:

| Command | Removes | Keeps |
|---|---|---|
| `uninstall` | The program and the Claude Code plugin | Your notes and the configuration |
| `uninstall --config` | Also the configuration and your profile | Your notes |
| `uninstall --all` | Also every space's notes | Your backups |

- **To come back**, install again: with the configuration there, it finds everything as it
  was; without it, it makes the spaces again over your notes, losing none.
- **`--all` deletes your notes.** It shows each notes folder with how many notes it has,
  and warns about those with no backup: they will be gone for good. Answer no, set up the
  backup or copy the folders, and run it again.
- **Your backups are never deleted**, on GitHub or anywhere. It says which they are and how
  to delete one, if you want to.
- Claude Code, its desktop app and git stay.

On Windows the program file disappears a moment after the command ends: Windows does not
let a running program delete itself.

## Moving to a new computer

1. On the old one, make a last backup: `augmented-brain backup --all`.
2. On the new one, install augmented brain, then restore each space:
   `augmented-brain restore <address of its backup>`.
3. Give again what does not travel: the lists of the guard, and the secrets' values
   (`restore` names those its keychain lacks, with the command to store each).

See [Backup and restore](backup.md), and [Where everything lives](where-things-live.md) for
what travels and what does not.
