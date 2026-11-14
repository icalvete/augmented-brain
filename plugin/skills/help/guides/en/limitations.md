# Known limitations


What augmented brain cannot do yet, or does only in part, and what to do instead when
there is something to do. When something goes wrong and you want to fix it, see
[Troubleshooting](troubleshooting.md). The list grows when something new turns up: if you
find one, tell us from the panel's Help page.

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
- **No warning about the window's memory.** On Linux and macOS, Claude warns when the
  window that holds your sessions grows past 2 GB. On Windows it does not look: if
  Claude gets slow, park the task and open a new window.
- **The keychain cannot be asked.** After a restore, augmented brain cannot tell which
  of your secrets the Windows keychain already holds. Look in the panel's Secrets page
  and store those that are missing.

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

It works on a Mac, and every change is tested on Macs in CI: the installer, the guard,
the panel in Safari and a locked keychain. But no person has used it on a real Mac yet,
so what only someone at the screen would notice has not been looked at: opening the
program from the Finder ("Open Anyway"), a real Claude session, Claude Desktop, or
Tailscale with `panel remote`. If something fails for you, no one has seen it yet:
please report it.

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
  where Claude may not commit, but `git merge`, `git pull`, `git revert`,
  `git cherry-pick` and `git am` also make commits on your computer. In an untouchable
  repository they are stopped too; elsewhere they go through. They are not published:
  a push is still stopped.
- **Rules Claude saves.** By default, Claude saves a rule, your profile or a host for
  the guard without asking you first, and every later session loads them. A text Claude
  read (a web page, an email) could talk it into saving one. You are told: Claude Code
  shows you, once, what was saved. If you did not ask for it, tell Claude to undo it, or
  look at the panel. To be asked first, choose "Ask me first" in the panel's Guard page
  (see [The guard](guard.md#asking-before-claude-saves-a-rule)).
- **No rules in sessions that skip questions.** With "Ask me first", a session that
  skips Claude Code's permission questions (`bypassPermissions`) has no one to ask, so
  Claude cannot save rules, your profile or hosts for the guard there. It tells you what
  it wanted to save.
- **Only in your spaces: a script that moves into one.** When augmented brain keeps to
  your spaces, a command run outside them that goes into one (`cd ~/work/app`,
  `git -C ~/work/app`) is checked with every rule. A script that changes folder inside
  itself is not: run from outside, `bash build.sh` that moves into a space is checked as
  outside. See [Outside every space](spaces.md#outside-every-space).
- **Only in your spaces, with an earlier version still open.** A Claude session started
  before an update keeps the earlier program, which does not know this choice. Its start
  accepts the guard's lists as that program counts them, so until a session of the new
  version starts again, augmented brain acts on the whole computer. It is the safe side,
  and Claude Code tells you each time the lists are accepted.
- **Questions in automatic modes.** When the guard asks, Claude Code shows you the
  question. What Claude Code does with it in its fully automatic modes is up to Claude
  Code, not augmented brain.

More in [The guard](guard.md).

## Elsewhere

- **`watch pr` follows pull requests on GitHub only.** It uses GitHub's own tool, `gh`.
  A merge request on GitLab or another forge is not followed: look at it there.
- **A file over 50 MB is left out of the backup.** The rest is backed up, and the
  backup says which file. Keep big files outside the notes folder.

## Tested by few people

augmented brain is beta. One person has used it every day on Ubuntu, and tried it on
Windows 11; on macOS it is tested in CI only. Every version so far has kept what you
have working, and from 1.0 on that is a promise (see
[What stays stable](compatibility.md)). Back up anything important.
