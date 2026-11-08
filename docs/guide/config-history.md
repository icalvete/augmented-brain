# Configuration versions

[Leer en español](config-history.es.md)

Your configuration is what augmented brain knows about this computer and about you:
your spaces, the guard's lists, remote access, your profile and the names of your
secrets. A wrong change there (a list emptied by mistake, a space pointing to the wrong
folder) used to have no way back. Now every change is kept as a **version**, and you can
see what changed and go back.

## What is kept

A version is a copy of these files, in the configuration folder, as they were at one
moment: `config.toml`, `guard.toml`, `remote.toml`, `session.toml`, `profile.md` and
`secrets.toml` (only the names of your secrets, never a value). It also says who made the
change: the panel, a command, the assistant (and with which tool), going back to an
earlier version, or by hand.

- A new version is kept every time the configuration changes. A change made by hand, or
  by an older program, is kept the next time augmented brain looks: when a session
  starts, a command runs, the assistant uses a tool or you save something in the panel.
- The last **10** are kept. When there is one more, the oldest goes.
- They stay **only on this computer**, in the `history` folder of the configuration
  folder. They never go into the backup: the guard's lists may name work machines, and
  the backup leaves them out on purpose.
- Your notes are not here. Tasks, rules, knowledge and people (shared ones too) have
  their own history in the [backup](backup.md).

## In the panel

The panel's **Versions** page lists them, newest first, with when and who. Under each
one, **What it changed** shows the lines that changed from the version before: green
what was added, red what was taken away. **Go back to this one** makes it the
configuration again.

Going back keeps the configuration you had as a version first, so it can be undone the
same way. A version that would leave a file the program cannot read is refused, and
nothing changes. If the guard's lists change, going back from the panel counts as you
accepting them, as any change on the [Guard](guard.md) page does.

From another device ([the panel from another device](panel-remote.md)) the page only
shows the versions: going back is only from the panel on this computer, because it can
change what the guard allows.

## With commands

```bash
augmented-brain config history
augmented-brain config history 3
augmented-brain config restore 3
```

`config history` lists the versions; with a number, it shows what that one changed.
`config restore` goes back to one. A command can be run by the assistant, so going back
this way never accepts new guard lists for you: until you accept them in the panel's
Guard page, or start a new session, the guard asks before every commit, push or merge it
used to allow. See [Commands](commands.md#augmented-brain-config-history).

Both work even when `config.toml` is broken and nothing else will start: that is when
going back helps most. `augmented-brain doctor` then says so, and points to them.

`augmented-brain doctor` has a line with how many versions are kept, and says when the
configuration changed after the newest one.
