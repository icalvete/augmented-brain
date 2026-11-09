# Known limitations

[Leer en español](limitations.es.md)

What augmented brain cannot do yet, or does only in part. Each line says what to do
instead. The list grows when something new turns up: if you find one, tell us from the
panel's Help page.

## Only Claude Code

augmented brain works with Claude Code: the terminal, the desktop app and the IDE
extensions. Other assistants could use the note tools over MCP, but the session start
and the guard rely on Claude Code's hooks, so they would get neither.

## Windows

- **Claude runs commands in Git Bash, not in PowerShell.** The guard reads the commands
  of a POSIX shell (Bash), not PowerShell, so it stops every PowerShell command, and
  Claude uses Git Bash instead. What only works in PowerShell, you run yourself: Claude
  gives you the command.
- **Accents can look broken** (`sesiÃ³n`) in Windows PowerShell 5.1. The fix is in
  [Troubleshooting](troubleshooting.md).

## Remote access and a session per task

`augmented-brain remote` (reaching your computer from claude.ai or Claude Desktop) and
`augmented-brain session start` (a remote session for one task) work on Linux only, with
systemd. On macOS and Windows they say so and change nothing.

- A session opened with `session start` does not come back after a reboot: ask for it
  again. The space's own remote session does.
- Claude allows one remote session per folder, and the folder must be trusted on its
  own. `session start` tells you what to do the first time.

## macOS

Every pull request runs the tests on a Mac in CI, including the installer, Safari and a
locked keychain. A few things only a person at a real Mac can see, and are checked by
hand before a release:

1. Opening the program downloaded with a browser from the Finder ("Open Anyway").
2. A real Claude Code session: the guard stops a push and the session start loads.
3. Claude Desktop installed by the installer, and Tailscale with `panel remote`.
4. With no network: `backup`, `update` and `panel remote` say so and leave nothing half
   done.
5. The panel in Safari with its console open, free of errors.

If one of them fails for you, it may not have been seen yet: please report it.

## The guard

The guard reads each command as text before Claude runs it. It catches the mistakes an
assistant makes; it is not a wall, and it does not replace permissions, backups or your
own review. What it cannot see:

- **Scripts written by another program.** If Claude writes a script with a program
  (Python, a build tool) and runs it later, the guard only sees what the script holds
  when it runs. A script written and run in the same command is checked; one written in
  a way the guard cannot read makes it ask you first.
- **Values known only when the command runs.** When a program or a folder comes from a
  variable or from another command's output, the guard cannot know it in advance: it
  asks you, but it cannot tell you exactly what will run.
- **Settings changed through a program.** The guard asks before Claude edits augmented
  brain's or Claude Code's settings with its file tools or common commands. A small
  program written on the spot (in Python, say) could still change them; the guard
  notices afterwards and asks before a commit or a push, but it does not prevent it.
- **Git aliases in included files.** Your git aliases count as what they stand for
  (`git ci` for `git commit`) when they are in your usual git settings. An alias in a
  file those settings `include` is not seen.
- **Local commits from merge, revert and cherry-pick.** The guard stops `git commit`
  where Claude may not commit, but `git merge`, `git revert` and `git cherry-pick` also
  make commits on your computer. They are not published: a push is still stopped.
- **Questions in automatic modes.** When the guard asks, Claude Code shows you the
  question. What Claude Code does with it in its fully automatic modes is up to Claude
  Code, not augmented brain.

More in [The guard](guard.md).

## Tested by few people

augmented brain is alpha. One person has used it every day on Ubuntu, and tried it on
macOS and Windows 11. What you have keeps working from one version to the next (see the
[disclaimer](https://github.com/icalvete/augmented-brain/blob/main/DISCLAIMER.md)), but
back up anything important.
