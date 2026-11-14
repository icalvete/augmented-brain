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

This page says what you see and what to do. What augmented brain cannot do at all is in
[Known limitations](limitations.md). The problems are grouped by subject, the most
common first in each group.

## Claude and your notes

### Claude does not know my tasks, or skips the interview

- **The session started before the install.** Start a new session.
- **Claude Code asked whether you trust the folder.** The first time you open Claude in
  a folder, Claude Code asks this before loading anything. Say yes in your work folders.
  Until then, augmented brain cannot load your notes.
- **You are in another space.** Run `augmented-brain space` in that folder. Outside every
  space's folder, the default space is used. To make a folder count for a space, add it
  to the space's `work_folders` (see [Spaces](spaces.md)).
- **The plugin is not installed.** `doctor` says "Claude Code plugin: not installed".
  The plugin is the piece that plugs augmented brain into Claude Code. Run
  `augmented-brain update --plugin`, then start a new session.
- **The notes folder does not exist.** `doctor` says "memory folder" and the error next
  to a space. It was moved or deleted, or it is on a disk that is not mounted. Put it
  back, or fix the path in the configuration, as `doctor` says.
- **There is no default space.** `doctor` says "no default space". Then a session
  outside every space's folder loads nothing. Add the line `doctor` gives to
  `config.toml`.
- **The configuration cannot be read.** `doctor` says "configuration:" and the error.
  Without it, no session loads anything. Go back to a version that worked: see
  [Claude asks before every aws or ssh command](#claude-asks-before-every-aws-or-ssh-command).

### A note is not valid

`augmented-brain check` in a folder of the space lists each problem with its file. Until
it is fixed, that note is left out: Claude does not see it, and it is not in the index.
The other notes work as usual. Fix the header (between the `---` lines) by hand, and run
`check` again. [Editing notes by hand](editing-notes.md) says what a header may hold.

When `check` says a field "is not one this version knows", a newer version, on another
computer or in a session started after an update, most likely wrote it. Update with
`augmented-brain update` and do not remove the field. If it is a typo of a field you
know, fix it.

A file that cannot be read (its permissions, or a broken link left by a sync) is listed
the same way: fix its permissions, or move it out of the notes folder. A file whose name
starts with a dot, such as the lock file Emacs leaves while you edit a note, is skipped.

### "Written by a newer augmented brain"

A newer version on another computer moved the notes to a new format. Update this
computer too: `augmented-brain update`. The older program refuses the notes instead of
breaking them.

### A session was cut off

The window was closed by force, or the computer went down. The next session offers to
recover it. Say yes, or use the panel: Today lists it. Claude Code keeps each
conversation for 30 days.

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

## Installing and updating

### The terminal cannot find a command

`command not found`, or "not recognized" on Windows. Close the terminal and open a new
one. The PATH is the list of folders where the terminal looks for programs; the
installer added one to it, and a terminal opened before does not see it yet.

### augmented brain's tools fail after an update

A session that was open keeps the old program. `augmented-brain session restart --old`
restarts the idle ones, with the same conversation; one of Claude Desktop stays open and
only its augmented brain restarts (on Windows, close and open it). `augmented-brain
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
- **"The backup failed, so nothing was updated"**: fix the backup (see
  [The backup failed](#the-backup-failed)) and update again. `augmented-brain update
  --force` updates anyway, but then there is no fresh copy of your notes to go back to
  if the new version goes wrong.

See [Installing](install.md#updating).

### The install stopped half way

Run the install command again (or `augmented-brain init`): until it finishes, there is no
configuration, so it starts over. See [Installing](install.md).

### macOS: the developer tools window did not open

Run `xcode-select --install`, then the install command again.

### Windows: accents look broken (`sesiÃ³n`)

Windows PowerShell 5.1 misreads what other programs write when it captures it. The text
is right: in Git Bash, or in a window, it shows well.

More problems of the installer, on each system: [Installing](install.md#if-something-goes-wrong).

## The guard

### Claude cannot push, even when I ask

The guard stops `git commit` and `git push` unless the repository is in one of its
lists. Push it yourself, or add the repository in the panel's **Guard** page, under "The
assistant may commit and push" or "The assistant asks you each time", then ask Claude to
use `git -C <repo>`. See
[The guard](guard.md#let-claude-commit-in-a-repository).

If it says a repository is **untouchable**, that is on purpose: only you can change it,
in the same Guard page (see
[The guard](guard.md#repositories-claude-must-never-touch)).

### Claude asks before a push it used to do

The guard's lists changed outside the panel since you last accepted them (by hand, with a
command, or going back to a version). Until you accept them in the Guard page or start a
new session, a commit, push or merge they allow asks first. `doctor` says so. Open the
panel's Guard page, look at what changed and accept. See [The guard](guard.md#only-you-change-the-guards-lists).

### Claude asks before every aws or ssh command

The configuration cannot be read, so the guard does not know your lists and asks to be
safe. `augmented-brain doctor` names the problem. If it says "unknown settings", the
file has a line this version does not know: a typo, or a setting of a newer version. For
a newer version's setting, update with `augmented-brain update`. For a typo, or a change
that broke it, go back to a version that worked: `augmented-brain config history`, then
`augmented-brain config restore <number>`. Both work while the file is broken. See
[Configuration](configuration.md) and [Configuration versions](config-history.md).

### "Settings this version does not know" in `guard.toml` or `session.toml`

`doctor` and the start of a session say so when one of these files has a line the
program does not know. The line is ignored: a misspelled `untouchable_repo` protects no
repository, and a misspelled `memory_trust` leaves Claude saving without asking. The
message gives the right name when it looks like a typo. Correct the name in the file.
If a newer version wrote that line, update with `augmented-brain update`. See
[Configuration](configuration.md#guardtoml).

### Claude Code stops every command and file change

The guard cannot run, so Claude Code stops everything instead of letting it through. The
program is missing or broken. Run `augmented-brain doctor`. Make sure `augmented-brain`
is on your PATH or in `~/.local/bin`. Installing again fixes it.

This needs Claude Code 2.1.295 or later. An older one lets every command through
unchecked when the guard cannot run, and `doctor` warns about it: update Claude Code
with `claude update`.

### Every PowerShell command is stopped

That is expected, on Windows. The guard reads the shell, not PowerShell, so it stops
PowerShell commands and Claude runs them in Git Bash instead.

### Claude needs root on a machine

Claude gives you the exact command to run yourself. For a migration that needs many
commands, add a dated exception in the panel's Guard page, or by hand in `config.toml`:
`root_ask = ["db01 until 2026-12-31"]`. Then root on that machine asks you each time,
until the date. See [The guard](guard.md#root-on-one-machine-for-a-while).

## Saving rules

Some things every later session follows: a rule, your profile, a machine for the guard.
The setting `memory_trust` in `session.toml` decides whether Claude asks you before
saving them. See [The guard](guard.md#asking-before-claude-saves-a-rule).

### Claude no longer asks before saving a rule

- **A new machine, or after a restore.** `session.toml` is not carried by backups, so
  the setting is back to telling you after saving. Choose "Ask me first" again in the
  panel's Guard page.
- **The setting is misspelled.** A line of `session.toml` with a name the program does
  not know is ignored. `doctor` names it, with the right name. See
  ["Settings this version does not know"](#settings-this-version-does-not-know-in-guardtoml-or-sessiontoml).

### Claude says it cannot save a rule

The session skips Claude Code's permission questions (`bypassPermissions`), so there is
no one to ask, and with "ask" the save is refused. Claude tells you what it wanted to
save. Start a session in a normal mode to save it, or choose "Save and tell me" in the
Guard page.

## Backup and restore

### The backup failed

Claude tells you at the start of the next session. Run `augmented-brain backup` in a
folder of the space to see why:

- **No network, or no access.** Check you can reach the repository. On GitHub, log in
  with `gh auth login`.
- **"stopped after 3m0s without an answer".** The network or the repository did not
  answer, so the backup gave up instead of waiting for good. Try again when the network
  works; the next backup does not wait behind this one.
- **"another backup has been running for over 10m0s".** A backup that no longer
  answers is in the way. Try again later.
- **A secret in a note.** It names the file. Take the secret out, then back up again.
  `augmented-brain check` names those files too.
- **Two machines changed the same note.** It names the notes, and the folder where to
  merge them by hand. When each machine only added journal entries, they are joined by
  themselves.
- **A file over 50 MB.** It is left out, and the rest is backed up. Move it out of the
  notes folder.

### `doctor` warns about a backup

- **"Began … and never ended".** The computer went off, or the backup was stopped half
  way. Run `augmented-brain backup` in a folder of the space: it backs up now, or says
  what stops it.
- **"The automatic backup is not running".** The notes changed after the last backup
  that went through. Run `augmented-brain backup` the same way. If `AB_NO_AUTO_BACKUP`
  is set in your environment, the automatic backup is off.
- **"Holds a password or a token".** The repository's address has your login written in
  it, so it is kept in the configuration and inside the backup. Let git keep the login
  instead (on GitHub, `gh auth setup-git`, or an ssh address), set the backup up again
  with the clean address, and change that password or token: earlier copies still hold
  it.

### `doctor` says a space has no backup

If this computer is lost, so are that space's notes. Ask Claude "set up the backup" in a
folder of the space. Or run `augmented-brain backup setup <address of a private
repository>` there, and then `augmented-brain backup`. See [Backup and restore](backup.md).

### After a restore, nothing asks before production, or Claude cannot push

Backups do not carry the guard's lists. Open a Claude session: it offers to ask you for
the production profiles and the untouchable machines. The repositories where Claude may
commit and push (`git_allowed_repos` and `git_ask_repos`) do not come back either:
`restore` names them, and you add them again in the panel's Guard page if you still want
them. See [Backup and restore](backup.md#what-a-restore-does-not-bring).

### After a restore, secrets "could not be checked"

The keychain could not be asked: a Linux machine without a desktop session, a locked
macOS keychain, or Windows, where it cannot be asked. The secrets may be there. Open a
desktop session or unlock the keychain, and look in the panel's Secrets page. Store those
that are missing with the command shown. See [Backup and restore](backup.md).

## The panel and remote access

### The panel does not open, or its address stopped working

On a computer with no screen, `augmented-brain panel` prints the address to open
yourself. That address works once, within 2 minutes. The pages it opens carry a key
that changes each time the panel starts, and the panel stops after 30 minutes without
use. Run `augmented-brain panel` again for a new address. If your browser says "This
link was already used" and you did not open it, something else did: look at the
panel's Guard page.

### The panel from another device does not open

`doctor` says what is missing: Tailscale stopped, HTTPS off in your network, or an
address left published after the panel stopped (it gives the command to remove it). See
[The panel from another device](panel-remote.md).

### Remote access does not work

Run `augmented-brain remote status`: under each problem, `→` says what to do. The usual
causes:

- **Claude does not trust the folder yet.** Run `cd <folder> && claude` once and say yes.
- **The computer restarted and nobody has logged in.** On a Mac and on Windows, remote
  access starts when you log in. On Windows it also stops when you sign out.
- **Linux: "linger" is off.** Linger is the setting that lets your services run with
  nobody logged in. Without it, remote access stops when you log out and does not start
  after a reboot. `remote install` prints the command to turn it on; an administrator
  runs it.
- **Linux without systemd's user services** (a container, WSL without systemd).
  `remote install` stops there, changing nothing.

See [Remote access](remote.md).

### Two remote sessions with the same name

When the door of a space moves to another folder (a new version starts it in your work
folder, say), Claude's side keeps the old session too, with the same name, in another
group of the list. Nothing answers in the old one: what you write there is lost. Use the
one in the group named after the new folder, and archive the old one in claude.ai or
Claude Desktop.

## The computer

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

### Old worktrees and scratch folders pile up

`doctor` says that something was left over from working with Claude Code. When Claude
Code works in a worktree (a second working copy of a repository, in `.claude/worktrees`),
the worktree stays after the session ends. Its sessions also leave scratch folders in
the system's temporary folder, which a computer that is rarely restarted never empties.

Run `augmented-brain clean`. It shows what it would remove and asks first. A worktree with
changes, commits or ignored files (a `.env`) of its own is never removed: it only names
it, so you look at it.

## Questions

### Is it free?

augmented brain is: open source, MIT license, no account, no subscription. But it works
with Claude Code, which needs a paid Claude plan (Pro or above).

### Do my notes leave my computer?

augmented brain never sends them anywhere, except to your own devices if you open the
panel from them ([`panel remote`](panel-remote.md)). But your assistant reads them to
work, like any file it opens, so they travel with the conversation to its provider. With
a backup, they also go to the private repository you chose.

### Can I edit the notes by hand?

Yes, they are Markdown files. [Editing notes by hand](editing-notes.md) says what to keep
in mind. Afterwards, run `augmented-brain check`.

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
