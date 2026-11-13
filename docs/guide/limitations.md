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
  gives you the command. When Claude tries PowerShell anyway, you see a red
  `PreToolUse:PowerShell hook error`: that is the guard stopping it, and Claude goes on
  in Bash. Nothing is wrong.
- **Installing over ssh asks nothing.** In an ssh session Windows says nobody is at the
  keyboard, so the installer cannot ask: it prints the commands to run instead, and winget
  often fails there. Install from PowerShell on the desktop, or with `AB_YES=1` to answer
  yes to everything.
- **Accents can look broken** (`sesiÃ³n`) in Windows PowerShell 5.1. The fix is in
  [Troubleshooting](troubleshooting.md).

## Claude Code's first run

The first time you run `claude`, Claude Code shows its own screens before augmented
brain says anything: a colour theme, your sign-in, and whether you trust the folder.
In your space, answer **Yes, I trust this folder**: Enter alone picks "No, exit". Ctrl-C
does not leave the theme screen; pick one with Enter. These screens belong to Claude
Code, and augmented brain cannot change them.

## Remote access and a session per task

`augmented-brain remote` (reaching your computer from claude.ai or Claude Desktop) works
on Linux, macOS and Windows. `augmented-brain session start` (a remote session for one
task) works on Linux only for now: on macOS and Windows it says so and changes nothing.

- **On a Mac and on Windows, remote access starts when you log in**, not before. After a
  restart, including one the computer does by itself (an update, a power cut), nobody
  can connect until you log in once. On Linux it can start with nobody logged in.
- **On Windows it runs only while you are logged in.** If you sign out, it stops.
- **On Windows, a folder with `%` in its name** may keep remote access from starting
  there. Pick another folder with `folders` in `remote.toml`.
- **A real connection has been tried on Linux only.** On a Mac and on Windows, install,
  status and uninstall are tested in CI with a pretend Claude; connecting from claude.ai
  has not been tried yet.
- A session opened with `session start` does not come back after a reboot: ask for it
  again. The space's own remote session does.
- Claude allows one remote session per folder, and the folder must be trusted on its
  own. `session start` tells you what to do the first time.

## macOS

Every pull request runs the tests on a Mac in CI, including the installer, Safari and a
locked keychain. A few things only a person at a real Mac can see, and nobody checks
them yet:

1. Opening the program downloaded with a browser from the Finder ("Open Anyway").
2. A real Claude Code session: the guard stops a push and the session start loads.
3. Claude Desktop installed by the installer, and Tailscale with `panel remote`.
4. With no network: `backup`, `update` and `panel remote` say so and leave nothing half
   done.
5. The panel in Safari with its console open, free of errors.

If one of them fails for you, no one has seen it yet: please report it.

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
  Such a program could also use the panel with a link it saw: the link works once, so
  your browser would say it was already used, and Claude Code tells you what the panel
  changed.
- **Git aliases in included files.** Your git aliases count as what they stand for
  (`git ci` for `git commit`) when they are in your usual git settings. An alias in a
  file those settings `include` is not seen.
- **Local commits from merge, revert and cherry-pick.** The guard stops `git commit`
  where Claude may not commit, but `git merge`, `git revert` and `git cherry-pick` also
  make commits on your computer. They are not published: a push is still stopped.
- **Rules Claude saves.** Claude saves a rule, your profile or a host for the guard
  without asking you first, and every later session loads them. A text Claude read (a
  web page, an email) could talk it into saving one. Nothing stops that, but you are
  told: Claude Code itself shows you, once, what was saved. If you did not ask for it,
  tell Claude to undo it, or look at the panel.
- **Questions in automatic modes.** When the guard asks, Claude Code shows you the
  question. What Claude Code does with it in its fully automatic modes is up to Claude
  Code, not augmented brain.

More in [The guard](guard.md).

## Tested by few people

augmented brain is beta. One person has used it every day on Ubuntu, and tried it on
macOS and Windows 11. What you have keeps working from one version to the next (see the
[disclaimer](https://github.com/icalvete/augmented-brain/blob/main/DISCLAIMER.md)), but
back up anything important.
