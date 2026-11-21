# A session per task


Each task can have a Claude session of its own, in its own folder. You ask for it in
plain words, and it shows up in claude.ai and Claude Desktop within a minute, already
reading its task. With several, you work on several tasks at once, from the computer or
from your phone, and none of them steps on another.

## Where it works

| Command | Linux | macOS | Windows |
|---|---|---|---|
| `session start`, `stop`, `list` | yes | not yet | not yet |
| `session restart` | yes | yes | no |
| `watch pr`, `clean` | yes | yes | yes |

`session start`, `stop` and `list` have been tried only on Linux. On macOS and Windows
`session start` says it is not available, and makes nothing.

## Open a session for a task

Ask the session you are in: "open me a session for the website task". Every session on
Linux knows how, at the computer or [from another one](remote.md), because the start of
each session tells it. It runs

```bash
augmented-brain session start task-new-website
```

and a new session, named after the task, shows up in claude.ai and Claude Desktop. It
starts on its own: it reads its task, tells you in two or three lines where it stands,
and gets on with the next step. It hears only what is its task's: that task's dates, not
the rest of the space's reminders, the weekly review or sessions to recover, which come
up in your other sessions. When it stops (the task closed or parked, or waiting on you),
it tells the session that asked for it, in one line. With `--model sonnet` (or any
model `claude --model` takes) it starts with that model instead of Claude's default.

The command waits for the session's first answer, up to a minute. A session that cannot
work is not left open: if Claude Code is not logged in, nothing opens, and if the first
answer is an error (the login ran out, the service is overloaded), the session is
closed. Either way the command says what to do, such as `claude auth login`.

The new session does not get everything your terminal has. Claude starts its sessions
from one background program, which keeps the variables it started with for every later
session, of any task and any space. So it gets only what a session needs to work: your
home and user, `PATH`, language, the desktop's and the ssh agent's variables, proxies,
and Claude's own settings (`CLAUDE_*`, `ANTHROPIC_*`). A token you exported in a
terminal, such as `GH_TOKEN` or `AWS_*`, stays in that terminal: a tool in the session
uses its own login (`gh auth login`, an AWS profile).

## Which folder it opens in

The folder is the first of the task's repositories found in the space's work folders.
`--folder` gives another one, as long as it lies inside the space's work folders: a
session elsewhere would not find its task. A task that names no repository opens in the
space's first work folder.

```bash
augmented-brain session start task-new-website --folder ~/projects/website
```

Two rules keep it safe:

- **Claude must trust that folder.** A repository must be trusted on its own, so its
  own settings and hooks never run unasked: trusting a folder above it is not enough.
  A folder that is not a repository is trusted when a folder above it is, as Claude
  itself does. When Claude does not trust it yet, the command says what to do: on the computer (or through `ssh`), go to the folder, run
  `claude`, accept the question and close it with `/exit`. Then ask again.
- **One task per folder.** Two tasks in one folder would step on each other's changes,
  so a folder another task's session uses is refused. Give it another one with
  `--folder`, such as a worktree (below).

## Worktrees, for a bare repository

A **worktree** is a second working copy of a git repository: another folder, on another
branch, sharing the same history. A **bare repository** is one with history but no
working copy: some people keep each project that way and make a worktree for each piece
of work.

When the task's repository is bare, `session start` makes the task a worktree of its
own next to it: `site-new-website` for the task `task-new-website` of `site`, on a new
branch named after the task, from the default branch of origin. It fetches first, so
the branch starts from the code of now; if origin cannot be reached, it starts from what
this computer last knew, and says so. If an earlier session of the task left its branch,
the worktree takes that branch as it was. So two tasks of one repository never share a
folder. Asked again, it uses the same worktree. Claude keeps the trust of a
worktree under its main repository: trusting either one is enough.

The worktree goes away by itself when nothing would be lost: with `session stop`, when
the task is closed, or, for a session that ended some other way (a reboot, closing it
from Claude Desktop), at the next `session start`. One with changes not committed,
commits not pushed, or files git ignores that no build makes again (a `.env`, a key)
stays, and `augmented-brain doctor` names it.

## See them, and close them

```bash
augmented-brain session list
```

lists the task sessions open on this computer: the task, whether it runs, and its
folder.

- **retired**: Claude stops a session left unused for hours. claude.ai shows it as
  disconnected. `session start` with its task brings it back, with its whole
  conversation.
- **failed**: its last answer was an error, such as an expired login. The line says what
  to do.
- **waiting**: Claude's usage limit stopped it. The line says when the limit resets.
  With remote access installed (`augmented-brain remote install`), the session is woken
  then, by itself, and goes on with its task; you are told when the wait begins and
  when the sessions are back. Without it, write to the session once the limit resets.

```bash
augmented-brain session stop task-new-website
```

closes the session and takes it out of the lists of Claude Desktop and claude.ai,
worktree and all. The Claude app on a phone may keep showing it, offline, until you
archive it there.
Closing the task does the same, once the session has finished its answer. Parking it,
or sending it to someday, does too, but leaves its worktrees: the task goes on later,
and `session start` opens a new session for it. A session also ends when the computer
restarts, and does not come back by itself: ask for it again.

**Archiving in Claude Desktop does not close a session.** Claude keeps it in its list,
and may bring it back the next time it starts, as it does with every session it stopped
for being unused. Close it with `session stop`, park its task, or run `clean`.

**After Claude Code updates**, Claude Desktop may show a session twice for a while, one
of them offline (sometimes under "No folder"). It is the same session: claude.ai shows
it once, and Claude offers no way to remove the extra line from this computer.

## Restart a session after an update

After `augmented-brain update`, open sessions still run the program they started with.

```bash
augmented-brain session restart task-new-website
augmented-brain session restart --old
```

The first restarts one session, with the same conversation; `--old` updates every idle
one still running an earlier version without restarting it (only its augmented brain
starts again), and `update` runs it by itself. A session that is
answering is never cut off. It works on Linux and macOS; on Windows, close the session
and open it again yourself. Details in
[`session restart`](commands.md#augmented-brain-session-restart).

## Waiting on a pull request

A session that opened a pull request does not sit and wait for its checks. It leaves
this running in the background, with its task:

```bash
augmented-brain watch pr 104 --task task-new-website
```

When the checks pass or fail, the pull request is merged or closed, or it conflicts with
its base branch, it writes the result in the task, so your other sessions hear of it
even if this one is gone, and the session tells you at once. It never waits for ever:
3 hours at most. It needs `gh`, GitHub's command, with your login. Every option is in
[`watch pr`](commands.md#augmented-brain-watch-pr).

## Clean up what is left

Claude Code makes worktrees and scratch folders of its own, and leaves some behind; and
sessions whose task no longer goes on, or that someone else started and archived, stay
in its lists.

```bash
augmented-brain clean
```

shows what is left from sessions that are over, and removes it only if you say yes:
never a worktree with work of its own, which it only names. A session no task opened is
asked for one by one. `doctor` and the weekly review say when there is something to
clean. Details in [`clean`](commands.md#augmented-brain-clean).
