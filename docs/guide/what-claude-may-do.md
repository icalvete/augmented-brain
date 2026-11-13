# What Claude may do without asking you

[Leer en español](what-claude-may-do.es.md)

Claude works in your account, on your files. Two things decide what it may do on its own:
**Claude Code's permissions** and **augmented brain's guard**. This page puts them side by
side, so you know what happens without a question, what asks you first, and what never
happens.

## Two checks, one after the other

1. **The guard** looks at every command and file change before it runs. It stops some
   things, asks you about others, and lets the rest through. See [The guard](guard.md).
2. **Claude Code's permissions** then decide too: the mode you run it in (it asks before
   acting, by default) and the rules you added. Some modes have a check of their own that
   can stop an action the guard would let through.

Both must say yes. A yes from the guard never skips a question from Claude Code, and the
other way round.

If you keep augmented brain to your spaces, outside their folders the guard only
protects its own files, your home folder and what your lists name: there, the rest is
up to Claude Code's permissions alone. See
[Outside every space](spaces.md#outside-every-space).

## Never, whatever you say

- Becoming root, unless you open an exception for one machine until a date (below).
- Writing a secret in a command or a note: a password, a token, a key.
- Wiping your home folder or the whole disk.
- Anything in a repository you marked as **untouchable**: no commit, push or merge there,
  whatever the other lists say.

Claude tells you what it wanted and gives you the command, for you to run.

## It asks you each time

- **Commit, push or merge in a repository of the "ask" list.**
- **A push that rewrites what is on the remote** (`--force` and the like), even where
  Claude may push. Always, by design.
- **Merging with `--admin`**, which skips the branch's protection.
- **Production:** an `aws` command with a profile you marked as production, and `ssh` to a
  machine you marked as untouchable.
- **Root on a machine with an exception**, each command, until the exception's date.

## Without asking, because you said so

- **Commit and push in a repository of the "allowed" list**, when you asked for that
  change. A merge too, when the command names the repository: `cd ~/code/blog && gh pr
  merge 12`, or, already in that folder, `gh pr merge 12 -R owner/blog`.

Everything else Claude does on its own is reading, writing your notes, and running
ordinary commands: the guard lets them through, and Claude Code's mode decides.

## The lists are yours

The lists live in the configuration, and you change them in the panel's **Guard** page.
Claude cannot open that page, and the guard asks before Claude writes the configuration
files. Some pages warn on purpose: adding a repository where Claude commits and pushes
without asking needs a box ticked, and so on.

## First accept, then use

The guard keeps a **seal** of the lists you last accepted. If the lists change anywhere
else (by hand, a script, another program), what they newly allow does not count yet:

- every commit, push or merge they would let through **asks you first**;
- the Guard page shows **what changed** (what was added and what was removed, list by
  list) and a button to accept it;
- `augmented-brain doctor` says so too.

Accept in the panel, or start a new session, and the new lists apply. A command can be
run by Claude, so `augmented-brain config restore` never accepts lists for you.

## Root on one machine, for a while

For a job that needs it, add an exception with an end date in the Guard page: root on that
machine asks you for each command until the date, and is stopped again after it by
itself. Keep the date close.

## When you want to change something

- To let Claude commit in a repository, add it in the Guard page, to the "ask" list first.
- To have Claude Code ask less, change its own permissions, not the guard: the guard is
  about what must never happen.
- If a command is stopped and you think it should not be, Claude says why. Tell it what
  you wanted; it gives you the command to run yourself.
