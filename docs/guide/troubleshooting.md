# Troubleshooting

[Leer en español](troubleshooting.es.md)

Something is off. Start here: open a terminal and run

```bash
augmented-brain doctor
```

It checks the whole setup and says what is missing and how to fix it. Each line starts
with a mark: `✓` fine, `·` for your information, `!` a warning, `✗` something that
fails. Under a line that needs work, `→` says what to do.

If you have the panel open, its **Help** page has links to these answers and a button to
report a problem on GitHub, with the health check filled in and what is personal taken
out. See [The panel](panel.md#help).

## Problems and fixes

### Claude does not know my tasks, or skips the interview

- **Claude Code asked whether you trust the folder.** The first time you open Claude in
  a folder, Claude Code asks this before loading anything. Say yes in your work folders.
  Until then, augmented brain cannot load your notes.
- **The session started before the install.** Start a new session.
- **You are in another space.** Run `augmented-brain space` in that folder. Outside every
  space's folder, the default space is used. To make a folder count for a space, add it
  to the space's `work_folders` (see [Spaces](spaces.md)).

### The terminal cannot find a command

`command not found`, or "not recognized" on Windows. Close the terminal and open a new
one: the one you used does not see the new PATH yet.

### Claude Code stops every command and file change

The guard cannot run, so Claude Code stops everything instead of letting it through. The
program is missing or broken. Run `augmented-brain doctor`. Make sure `augmented-brain`
is on your PATH or in `~/.local/bin`. Installing again fixes it.

### Every PowerShell command is stopped

That is expected, on Windows. The guard reads the shell, not PowerShell, so it stops
PowerShell commands and Claude runs them in Git Bash instead.

### Claude asks before every aws or ssh command

The configuration cannot be read, so the guard does not know your lists and asks to be
safe. `augmented-brain doctor` names the problem. If it says "unknown settings", the
file has a line this version does not know: a typo, or a setting of a newer version. For
a newer version's setting, update with `augmented-brain update`. For a typo, or a change
that broke it, go back to a version that worked: `augmented-brain config history`, then
`augmented-brain config restore <number>`. Both work while the file is broken. See
[Configuration](configuration.md) and [Configuration versions](config-history.md).

### Claude cannot push, even when I ask

If it says a repository is **untouchable**, that is on purpose: only you can change it, in
the panel's Guard page (see [The guard](guard.md#repositories-claude-must-never-touch)).

The guard stops `git commit` and `git push` unless the repository is listed. Push it
yourself, or add the repository to `git_allowed_repos` (or `git_ask_repos`) and ask Claude
to use `git -C <repo>`. See [The guard](guard.md#let-claude-commit-in-a-repository).

### Claude asks before a push it used to do

The guard's lists changed outside the panel since you last accepted them (by hand, with a
command, or going back to a version). Until you accept them in the Guard page or start a
new session, a commit, push or merge they allow asks first. `doctor` says so. Open the
panel's Guard page, look at what changed and accept. See [The guard](guard.md#only-you-change-the-guards-lists).

### Claude needs root on a machine

Claude gives you the exact command to run yourself. For a migration that needs many
commands, add a dated exception, `root_ask = ["db01 until 2026-12-31"]`. Then root on
that machine asks you each time, until the date.

### augmented brain's tools fail after an update

A session that was open keeps the old program. Close and open it. `augmented-brain
doctor` says how many sessions still run the old version.

### The installer or the update cannot download

GitHub sometimes fails to hand out a release's files for a while. The installer and
`augmented-brain update` try three times, and also through GitHub's API, which often
keeps working when the usual address does not. If all of it fails, they say so and
nothing changes: the program you had stays. Try again in a while.

What `augmented-brain update` may end with:

- **"cannot reach GitHub to see the latest version"**: `update` asks for the latest
  version once, before downloading. Try again in a while.
- **"the program is still X, not Y"**: the installer ran but the program did not change.
  Nothing else changed; run `augmented-brain update` again later.
- **"The backup failed, so nothing was updated"**: fix the backup (see below) and update
  again, or run `augmented-brain update --force`.

See [Installing](install.md#updating).

### The install stopped half way

Run the install command again (or `augmented-brain init`): until it finishes, there is no
configuration, so it starts over. See [Installing](install.md).

### "Written by a newer augmented brain"

A newer version on another computer moved the notes to a new format. Update this
computer too: `augmented-brain update`. The older program refuses the notes instead of
breaking them.

### The backup failed

Claude tells you at the start of the next session. Run `augmented-brain backup` in a
folder of the space to see why:

- **No network, or no access.** Check you can reach the repository. On GitHub, log in
  with `gh auth login`.
- **"Stopped after 3m0s without an answer".** The network or the repository did not
  answer, so the backup gave up instead of waiting for good. Try again when the network
  works; the next backup does not wait behind this one.
- **"Another backup has been running for over 10m0s".** A backup that no longer
  answers is in the way. Try again later.
- **A secret in a note.** It names the file. Take the secret out, then back up again.
  `augmented-brain check` names those files too.
- **Two machines changed the same note.** It names the notes, and the folder where to
  merge them by hand. When each machine only added journal entries, they are joined by
  themselves.
- **A file over 50 MB.** It is left out, and the rest is backed up. Move it out of the
  notes folder.

### `doctor` says a space has no backup

If this computer is lost, so are that space's notes. Ask Claude "set up the backup" in a
folder of the space. Or run `augmented-brain backup setup <address of a private
repository>` there, and then `augmented-brain backup`. See [Backup and restore](backup.md).

### A note is not valid

`augmented-brain check` in a folder of the space lists each problem with its file. Fix
the header (between the `---` lines) by hand, and run `check` again. Until then, the
index is not rebuilt, but the tools keep working.

### A repository is "not on this computer"

A task names a repository in `repos:` that is in none of the space's work folders. It
was moved, renamed or deleted, or this computer never had it. Tell Claude where it is
now, or remove it from the task.

### Claude Code keeps a second memory

`doctor` warns that Claude Code keeps a memory of its own for your folders. It drifts
from the space's. Add `"autoMemoryEnabled": false` to the settings file `doctor` names.
What it holds stays, and you can [bring it in](import.md).

### Other users can read my notes

`doctor` warns about it. Run `augmented-brain check` in a folder of the space: it makes
the notes yours only.

### The panel does not open, or its address stopped working

On a computer with no screen, `augmented-brain panel` prints the address to open
yourself. The address carries a key that changes each time the panel starts, and the
panel stops after 30 minutes without use. Run `augmented-brain panel` again for a new
address.

### The panel from another device does not open

`doctor` says what is missing: Tailscale stopped, HTTPS off in your network, or an
address left published after the panel stopped (it gives the command to remove it). See
[The panel from another device](panel-remote.md).

### Remote access does not work

Run `augmented-brain remote status`: under each problem, `→` says what to do. Usually
Claude does not trust the folder yet (run `cd <folder> && claude` once), or linger is
off. `remote install` stops, changing nothing, on a machine without systemd's user
services (a container, WSL without systemd). See [Remote access](remote.md).

### A session was cut off

The window was closed by force, or the computer went down. The next session offers to
recover it. Say yes, or go to the panel's Sessions page. Claude Code keeps each
conversation for 30 days.

### The window uses a lot of memory

Claude says that the window uses a lot of memory, or `doctor` warns about a Claude window
of several GB. The process that holds the window's sessions has grown far past normal:
VS Code's extension host (the program inside VS Code that runs Claude and the other
extensions), or Claude itself in a terminal. It normally stays near 0.5 GB for days.
Now and then one grows without stopping, and the system ends up closing it, with every
Claude session of the window. The length of the conversation has nothing to do with it.

What to do, as soon as you see it:

1. Ask Claude to park the task ("park it"). Where you were and what comes next are kept.
2. Open a new VS Code window (or a new terminal session) and resume the task there.
3. Close the old window when you are done with it: that frees the memory.

Claude warns at 2 GB, then at 4 and 8 GB, once each. On Windows it does not look.

### After a restore, nothing asks before production

Backups do not carry the guard's lists. Open a Claude session: it offers to ask you for
them. Or add `production_aws_profiles` and `untouchable_hosts` to the configuration
yourself.

### After a restore, secrets "could not be checked"

The keychain could not be asked: a Linux machine without a desktop session, a locked
macOS keychain, or Windows, where it cannot be asked. The secrets may be there. Open a
desktop session or unlock the keychain, and look in the panel's Secrets page. Store those
that are missing with the command shown. See [Backup and restore](backup.md).

### Windows: accents look broken (`sesiÃ³n`)

Windows PowerShell 5.1 misreads what other programs write when it captures it. The text
is right: in Git Bash, or in a window, it shows well.

### macOS: the developer tools window did not open

Run `xcode-select --install`, then the install command again.

More problems of the installer, on each system: [Installing](install.md#if-something-goes-wrong).

## Questions

### Is it free?

augmented brain is: open source, MIT license, no account, no subscription. But it works
with Claude Code, which needs a paid Claude plan (Pro or above).

### Do my notes leave my computer?

augmented brain never sends them anywhere. But your assistant reads them to work, like
any file it opens, so they travel with the conversation to its provider. With a backup,
they also go to the private repository you chose.

### Can I edit the notes by hand?

Yes, they are Markdown files. Afterwards, run `augmented-brain check`.

### Which assistants does it work with?

Claude Code today. The tools are an MCP server, an open standard, so other assistants can
follow.

### What if I stop using it?

Your notes stay where they are, as plain files. There is nothing to export. See
[Installing](install.md#uninstalling).

## Still stuck

Tell us in [the project's issues](https://github.com/icalvete/augmented-brain/issues),
with what showed up on screen and what `augmented-brain doctor` says. The panel's Help
page fills that in for you.

What changed in each version is in the
[changelog](https://github.com/icalvete/augmented-brain/blob/main/CHANGELOG.md).
