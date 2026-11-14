# Updating and uninstalling without losing anything

[Leer en español](update-and-uninstall.es.md)

Your notes are plain files that belong to you. Updating never touches what they say, and
uninstalling leaves them where they are unless you ask for the opposite. This page says
what each step does, so you can do it calmly.

## Updating

When a new version is out, Claude tells you at the start of a session and offers to
update it. You can also do it yourself:

```bash
augmented-brain update
```

What it does, in order:

1. **Checks the installation** and shows what it finds, so a problem that was already
   there is not taken for one of the new version.
2. **Backs up every space that has a backup.** If one fails, it stops and updates nothing:
   that copy is your way back. `--force` updates anyway. With no backup at all, it says
   it goes ahead without one.
3. **Downloads and installs** the new version, with the Claude Code plugin that comes
   inside it. It asks GitHub for the latest version once: if GitHub does not answer, it
   says so and stops. The download is tried three times, also through GitHub's API. If
   all fails, or the program is not the new version afterwards, it says so and nothing
   changes: run it again later. When there is no newer version, it only updates the
   plugin.
4. **Restarts the idle Claude sessions** that still run the old version, one at a time,
   with the same conversation. A session that is answering is left as it is.
   - **A task's session** (opened by `session start`) gets a new process.
   - **A session of the remote access** (`remote install`): its service restarts. Only
     on Linux; on macOS and Windows it counts as one of Claude Desktop.
   - **A session of Claude Desktop or a terminal** stays open, and only its augmented
     brain restarts. On Windows it is left as it is.
5. **Says what is left:** the sessions it could not restart (one answering, the one you
   ran it from, on Windows any of Claude Desktop), and which task each space has active.

For those left, ask the session to park its task, and close and open it. A session that
was open keeps the old program until it restarts, and its augmented brain tools may fail;
`/clear` is not enough. `augmented-brain doctor` and the start of a new session say how
many still run the old version. Later on, `augmented-brain session restart --old`
restarts the ones that have gone idle.

Other ways:

- `augmented-brain update --no-restart` updates without restarting any session.
- `augmented-brain update --plugin` updates only the Claude Code plugin.
- Running the [install command](install.md) again also updates, without steps 1, 2, 4
  and 5.

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

It shows what it will remove and asks. First it stops what would keep running without
it: the [remote access](remote.md) services and the sessions opened for tasks (`session
start`). Without the plugin they would go on with no guard, and with no program left to
take them away. If one cannot be stopped, nothing else is removed: it tells you why, so
you can fix it and run it again.

Then there are three levels:

| Command | Removes | Keeps |
|---|---|---|
| `uninstall` | The program, the Claude Code plugin and what Claude Code keeps of it (its cache, its data and its server's logs) | Your notes and the configuration |
| `uninstall --config` | Also the configuration and your profile | Your notes |
| `uninstall --all` | Also every space's notes | Your backups |

- **To come back**, install again: with the configuration there, it finds everything as it
  was; without it, it makes the spaces again over your notes, losing none.
- **`--all` deletes your notes.** It shows each notes folder with how many notes it has,
  and warns about those with no backup: they will be gone for good. Answer no, set up the
  backup or copy the folders, and run it again.
- **It removes only what augmented brain keeps** in a notes folder (the note folders,
  `MEMORY.md`, `inbox.md`, `docs/`, `archive/`), and the folder only if nothing else is
  left. A notes folder that holds a work folder, your home folder, or a folder with no
  notes is left whole, and the plan says so. `--config` only removes a folder that holds
  a `config.toml`.
- **Your backups are never deleted**, on GitHub or anywhere. It says which they are and how
  to delete one, if you want to.
- Claude Code, its desktop app, git and the PATH entry stay: Claude Code lives in that
  same folder.

On Windows the program file disappears a moment after the command ends: Windows does not
let a running program delete itself.

## Moving to a new computer

Make a last backup on the old one (`augmented-brain backup --all`), install augmented
brain on the new one and restore each space from its backup, the default space too. The
steps, and what you set up again by hand, are in [Backup and restore](backup.md#on-a-new-machine).
What travels and what does not: [Where everything lives](where-things-live.md).
