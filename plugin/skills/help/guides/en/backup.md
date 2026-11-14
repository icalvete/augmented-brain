# Backup and restore


Your laptop dies. On the new one, you install augmented brain, run one command, and open
Claude. It carries on where you left off: tasks, notes, rules and your profile.

That is the **backup**. The notes folder of each space is also a git repository (git keeps
every version of a folder and can copy it to a server). The backup pushes it to a private repository of yours, one per space.

## Set it up: ask Claude

Say "set up the backup". Claude offers it at the end of the first-session interview too.

It creates a private repository in your GitHub account, `augmented-brain-<space>`,
connects it and makes the first copy. For that it uses `gh`, GitHub's official command
line. If you do not have it, Claude tells you how to install it, and walks you through
the one step that needs your browser: logging in.

Rather somewhere else (GitLab, your company's git)? Do it yourself, in a folder of the
space:

```bash
augmented-brain backup setup git@gitlab.com:lucia/brain-work.git
```

Claude does not take an address for this: one it was handed by a page or a message it
read would send all your notes there. Use a **private** repository: your notes are in
it. And an address without a password: it is kept in the configuration and inside the
backup itself, so `backup setup` refuses one. Let git keep the login instead (for
GitHub, `gh auth setup-git`; or an ssh address like the one above). If an earlier version
kept one, `augmented-brain doctor` tells you how to take it out.

The panel's [Backup](panel.md#backup) page does the same with two buttons, and shows
each space's last copies and a **Back up now**.

Until a space has a backup, augmented brain says so: `augmented-brain doctor` names each
space with none, `update` warns when it goes ahead with no copy to go back to, and
`uninstall --all` warns that those notes will be gone for good.

## It runs by itself

From then on, **the backup runs every time a Claude session ends**, in the background.
You wait for nothing. A space that this session wrote in from another space (a task of
another space) is backed up then too. If it fails (no network, no access to the repository), Claude tells
you at the start of the next session.

You can also run it by hand:

```bash
augmented-brain backup          # this folder's space
augmented-brain backup --all    # every space
```

To turn the automatic backup off, use an environment variable: a setting your system
hands to every program it starts. Set `AB_NO_AUTO_BACKUP` to `1`. On Linux or macOS, add
this line to `~/.bashrc` or `~/.zshrc`:

```bash
export AB_NO_AUTO_BACKUP=1
```

On Windows, run `setx AB_NO_AUTO_BACKUP 1` once. Either way, it counts for Claude
sessions started after that, from a new terminal window. To turn the automatic backup on
again, take the line out (on Windows, `setx AB_NO_AUTO_BACKUP ""`).

## What you get

- **Several machines, one brain.** Before pushing, the backup fetches what your other
  machine did and merges it. A task where each machine only wrote in the journal, or
  where one moved old entries to `archive/journals/` while the other wrote, joins by
  itself.
- **Full history.** Every backup is a git commit, with the machine and the time. The
  panel shows what each one changed and goes back to it; with `git log` you see all of
  them.
- **Your place.** GitHub, GitLab or your own server: you pick the repository. augmented
  brain has no cloud.
- **One repository per space.** Work can go to your company's git, and personal to
  yours. Your profile only travels in the default space's backup, never in the work one.
- **No secrets go up.** Claude's tools refuse to write a password or a token into a note.
  If one gets in anyway (by hand, say), the backup stops and tells you which file.
- **Nothing gets overwritten.** If two machines changed the same note, the backup stops
  without touching anything and explains how to sort it out. When each machine only added
  journal entries to it, they are joined by themselves, in date order. The index is
  rebuilt by itself, and captures in the inbox from two machines are both kept.
- **No junk, no giants.** Leftovers of a crash and the files systems drop in folders
  (`.DS_Store`, `Thumbs.db`) stay out. A file over 50 MB stays out too (GitHub refuses
  them), and Claude tells you which one.

## On a new machine

1. Install augmented brain with the usual command. It creates the spaces, empty. If
   Claude opens at the end, close it without doing the interview: your profile comes in
   the backup.
2. If the repository is on GitHub, log in once with `gh auth login`, so git has access.
3. Bring each space back from its backup:
   ```bash
   augmented-brain restore https://github.com/lucia/augmented-brain-work.git
   ```
   Run it once per space, and do not leave out the one that was your default space (the
   installer calls it `personal`): your profile, your shared notes and the list of your
   secrets' names come only with it.

   Each `restore` brings the space's notes, tasks, rules and settings, readable by you
   only. If anything fails half way, everything stays as it was.
4. Open Claude. It carries on where you left off, and the backup keeps running by itself.

### Where the notes go

The installer already made a space with that name, still empty? Then your notes take its
place: they go into its notes folder, and its work folder stays. That is the usual case.
If that space already has notes of yours, `restore` stops and touches nothing.

A space this machine does not have keeps the work folders it had on the old machine.
Its notes go in the same place inside the first of them. If they were somewhere else,
they go to `~/augmented-brain/<space>`.

A backup cannot choose a place outside your home folder, or in its hidden or system
folders (`~/.config`, `~/Library`, `~/AppData`, any `.git`): your computer runs some of
what it finds there, and someone else may have changed the backup. Such a folder becomes
`~/augmented-brain/<space>`, and `restore` tells you. To choose the place yourself, use
`--memory` and `--work`.

### Then, by hand

The backup carries your notes, not this machine's setup. Do these again, if you used them:

- **The guard's lists**, the repositories where Claude may commit, and your secrets: see
  [What a restore does not bring](#what-a-restore-does-not-bring).
- **Which model you expect, and whether Claude asks before it saves a rule**: on the
  panel's [Guard](guard.md#asking-before-claude-saves-a-rule) page.
- **Remote access**: `augmented-brain remote install`. See [Remote access](remote.md).
- **The panel from another device**: set up Tailscale again, then run
  `augmented-brain panel remote`. See [The panel from another device](panel-remote.md).
- **Your code**: the backup holds your notes, not the repositories in your work folders.
  Clone them again.

## Go back to a copy

A note got broken, or deleted by mistake? Open the panel's **Backup** page. Each copy
shows **What it changed**: the notes it added, changed and removed, with their lines.

To have your notes as they were in one copy, use **Go back to this copy**:

1. A page shows what would change in your notes as they are now, note by note. Notes
   that no copy holds yet show up too, as removed.
2. Tick the box and type the space's name.
3. What you have now is kept first, as a new copy. Then your notes become those of the
   copy you picked, and that is kept as a copy too. The next backup sends it.

To undo it, go back to the copy made just before. Going back touches the notes of that
space only, never your configuration (that is [Versions](config-history.md)). It is done
from this computer's panel only: from another device the page only shows.

## What a restore does not bring

Some things are left out on purpose:

- **The guard's lists**: production AWS profiles, untouchable machines, repositories
  Claude must never touch, and root exceptions. They are kept for the whole computer and
  may name your work's machines, so they are in no backup. `restore` and `doctor` say
  so, and the next session offers to ask you for them again. Until then, nothing asks
  before touching production. A space set up by an older version may still keep lists
  of its own: those travel with it, and a restore uses them.
- **The repositories where Claude may commit and push** (`git_allowed_repos` and
  `git_ask_repos`). They travel with the space, but someone else may have changed the
  backup, so `restore` only names them, and you add them again if you still want to.
- **Your secrets' values.** Only their names travel. `restore` names the secrets this
  machine's keychain does not have yet, each with the command to store it, and those it
  could not check when the keychain does not answer, or on Windows, where it cannot be
  asked.
- **This machine's choices**: the model you expect and whether Claude asks before it
  saves a rule (`session.toml`), and remote access (`remote.toml`). Choose them again on
  the new machine.

## When it fails

Claude tells you at the start of the next session, with the reason. Running
`augmented-brain backup` in a folder of the space shows the problem. The usual ones are
in [Troubleshooting](troubleshooting.md#the-backup-failed).

It also tells you when a backup began and never ended (the computer went off half way),
and when the last good one is over 3 days old and your notes changed since: the
automatic backup is not running (say, `AB_NO_AUTO_BACKUP` was left set). `doctor` says
the same.

## Details

Your git settings for your code do not get in the way: the backup never signs its
commits, never runs your global hooks (`core.hooksPath`) and never opens a window asking
for a password. A credential helper that already has it (`gh`'s, your keychain's) still
works.

Backups go to the branch `main`, and `restore` reads them from there, even from a
repository whose default branch is another (one made with `git init --bare` on a
server, say).

A backup never hangs. Each step may take 3 minutes at most, and a download slower than
1 KB a second for a minute stops too. A backup waits at most 10 minutes for another one
to finish. Each time, it says why it stopped.
