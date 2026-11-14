# What Claude may do without asking you

[Leer en español](what-claude-may-do.es.md)

Claude works in your account, on your files. Two things decide what it may do on its own:
**Claude Code's permissions** and **augmented brain's guard**. This page puts them side by
side, so you know what is stopped, what asks you first, and what goes ahead without a
question. The whole story, command by command, is in [The guard](guard.md).

## Two checks, one after the other

1. **The guard** looks at every command and file change before it runs. It stops some
   things, asks you about others, and lets the rest through.
2. **Claude Code's permissions** then decide too: the mode you run it in (by default it
   asks before acting) and the rules you added. Some modes have a check of their own that
   can stop an action the guard would let through.

Both must say yes. A yes from the guard never skips a question from Claude Code. The
other way round holds too, with one exception: in the mode that skips Claude Code's
questions (`bypassPermissions`), a question from the guard may not reach you. Do not use
that mode with the guard; [why](guard.md#sessions-that-skip-claude-codes-questions).

## Stopped

The guard stops these. Claude tells you what it wanted and gives you the command, for you
to run.

- **A commit or a push in a repository you have not listed.** By default Claude commits
  and pushes nowhere: you open a repository to it (below).
- **Commit, push or merge in a repository you marked as untouchable**, whatever the other
  lists say. Other work there (reading, building, testing) goes on as anywhere else.
- **Becoming root** (the administrator of the machine), unless you open an exception for
  one machine until a date (below).
- **Writing a secret** in a command or a file: a password, a token, a key.
- **Wiping your home folder or the whole disk.**
- **PowerShell**, which the guard cannot read. Claude runs the same in Bash instead.

The first three are yours to change, with the lists; the last three are always stopped.

## It asks you each time

- **Commit, push or merge in a repository of the "ask" list.**
- **A push that overwrites what is on the server** (`--force` and the like), even where
  Claude may push. Always, by design.
- **A push to somewhere else**: an address that is not one of the repository's remotes.
- **Merging with `--admin`**, which skips the branch's protection.
- **Production:** an `aws` command with a profile you marked as production.
- **Untouchable machines:** `ssh` and the like to a machine you marked as untouchable.
- **Starting a release** (`gh workflow run release`): it publishes a version.
- **Root on a machine with an exception**, each command, until the exception's date.
- **A command the guard cannot read to the end**: a program given as a variable, a
  script it cannot open, `curl … | sh`.
- **Writing the guard's settings or Claude Code's**, since a line there could switch a
  check off.
- **Saving a rule, your profile or a guard list**, if you chose "ask first" (below).

## Without asking

- **Commit and push in a repository of the "allowed" list**, when you asked for that
  change. A merge too, when the command names the repository: `cd ~/code/blog && gh pr
  merge 12`, or, already in that folder, `gh pr merge 12 -R owner/blog`.
- **Saving a rule, your profile or a production profile or untouchable machine** it
  learned from you. Claude Code tells you in a line of its own what was saved; if you did
  not ask for it, tell Claude to undo it. This is the default; to be asked first instead,
  see [Asking before Claude saves a rule](guard.md#asking-before-claude-saves-a-rule).

Everything else Claude does on its own is reading, writing your notes, and running
ordinary commands: the guard lets them through, and Claude Code's mode decides.

## The lists are yours

The lists live in the configuration, and you change them in the panel's **Guard** page.
Claude cannot open that page, and the guard asks before Claude writes the configuration
files. Some pages warn on purpose: adding a repository where Claude commits and pushes
without asking needs a box ticked, and so on. Claude can only add to two of them, the
production profiles and the untouchable machines, which only makes the guard ask more.

## First accept, then use

The guard keeps a **seal** of the lists you last accepted. If the lists change anywhere
else (by hand, a script, another program), what they newly allow does not count yet:

- every commit, push or merge they would let through **asks you first**;
- the Guard page shows **what changed** (what was added and what was removed, list by
  list) and a button to accept it;
- `augmented-brain doctor` says so too.

Accept in the panel, or start a new session, and the new lists apply. When Claude only
adds a production profile or an untouchable machine to lists you had accepted, they stay
accepted: adding only makes the guard ask more.

`augmented-brain config restore` puts back an earlier version of the configuration, but
does not accept its lists for you: Claude could run that command too, so accepting stays
in the panel.

## Root on one machine, for a while

For a job that needs it, add an exception with an end date in the Guard page: root on that
machine asks you for each command until the date, and is stopped again after it by
itself. Keep the date close. More in
[The guard](guard.md#root-on-one-machine-for-a-while).

## When you want to change something

- To let Claude commit in a repository, add it in the Guard page, to the "ask" list first.
- To have Claude Code ask less, change its own permissions, not the guard: the guard is
  about what must never happen.
- If a command is stopped and you think it should not be, Claude says why. Tell it what
  you wanted; it gives you the command to run yourself.
