# Installing

[Leer en español](install.es.md)

One command installs augmented brain and everything it needs, on Linux, macOS and
Windows. This page says what that command does on each system, where it puts things and
what is different on each one. New to the terminal? [Getting started](getting-started.md)
walks you through it.

## On every system

- **A paid Claude plan** (Pro, Max, Team or Enterprise) or an Anthropic API key. augmented
  brain is free, but it works with Claude Code, and the free plan does not include Claude
  Code. The installer says so before installing it. Plans: [claude.com/pricing](https://claude.com/pricing).
- **No administrator or root.** The program goes in your own folder. Only installing
  something that is missing, like git, may ask for your password.
- **The installer asks first.** The language, then each missing piece, with what it is for.
  Press Enter to accept what it suggests.
- **It always ends the same way:** two spaces, `work` and `personal` (in Spanish, `trabajo`
  and `personal`), in `~/augmented-brain`, the Claude Code plugin installed, and the offer to
  open Claude.
- **Every download is checked.** The program comes from the project's latest release on
  GitHub, and its SHA-256 must match the release's list, or nothing is installed. When
  GitHub fails to hand it out, the installer tries three times, also through GitHub's
  API; if all fails, it says so and installs nothing.

## Linux

**You need:** a 64-bit PC or Arm (amd64 or arm64), `curl` or `wget`, and `sha256sum` or
`shasum` (all usual distributions have them).

**Install:** open a terminal (on Ubuntu, `Ctrl + Alt + T`) and paste:

```bash
curl -fsSL https://github.com/icalvete/augmented-brain/releases/latest/download/install.sh | sh
```

**What it does:**

1. **git**, if missing, with your system's package manager: `apt` (Ubuntu, Debian), `dnf`
   (Fedora), `pacman` (Arch) or `zypper` (openSUSE). It asks for your password.
2. **Claude Code**, if missing, with Anthropic's own installer, into `~/.local/bin`. If
   you have it but it is behind, it offers to update it (`claude update`).
3. **The Claude desktop app**, if you want it, **only on systems with `apt`** (Ubuntu,
   Debian and their family): from
   Anthropic's apt repository, which it adds for you; it asks for your password. On other
   distributions you use Claude in the terminal.
4. **The program**, into `~/.local/bin/augmented-brain`.
5. **The PATH:** it adds `~/.local/bin` to `~/.bashrc` with bash, `~/.zshrc` with zsh, or
   `~/.profile` otherwise, unless it is already there.
6. **The setup**, and the offer to open Claude.

**Where things go:** the program and Claude Code in `~/.local/bin`; the configuration in
`~/.config/augmented-brain` (or `$XDG_CONFIG_HOME/augmented-brain`); your notes in
`~/augmented-brain/<space>/notes`.

**Good to know:**

- After installing, **open a new terminal**: the one you used does not see the new PATH yet.
- To work without the app: `cd ~/augmented-brain/work && claude`.

## macOS

**You need:** **macOS 13 (Ventura) or later**, which Claude Code needs; see the Apple menu →
About This Mac. Apple silicon and Intel Macs both work. Homebrew is not needed.

**Install:** open the Terminal (`Cmd + Space`, type `Terminal`, Enter) and paste:

```bash
curl -fsSL https://github.com/icalvete/augmented-brain/releases/latest/download/install.sh | sh
```

**What it does:**

1. **It checks the macOS version first.** Older than 13, it says so and stops, before
   installing anything that would not start.
2. **git**, if missing, comes with **Apple's developer tools**. A Mac has a `git` command
   even without them, but it only asks you to install them: the installer looks past it.
   A window opens: click **Install** and accept. It takes a few minutes; the installer waits
   and carries on by itself.
3. **Claude Code**, if missing, with Anthropic's own installer, into `~/.local/bin`. If
   you have it but it is behind, it offers to update it (`claude update`).
4. **The Claude desktop app**, if you want it: with Homebrew when you have it, otherwise
   straight from Anthropic, with its signature checked. It goes to `/Applications`, or to
   `~/Applications` if you cannot write there.
5. **The program**, into `~/.local/bin/augmented-brain`.
6. **The PATH:** with zsh, the Mac's usual shell, in `~/.zshrc`. With bash, in
   `~/.bash_profile`, or in `~/.bash_login` or `~/.profile` if you already have one of them:
   the Mac's Terminal reads those, not `~/.bashrc`.
7. **The setup**, and the offer to open the Claude app.

**Where things go:** as on Linux: `~/.local/bin`, `~/.config/augmented-brain` and
`~/augmented-brain`.

**Good to know:**

- After installing, **open a new Terminal window**.
- In the app: **Code** tab, **+ New**. Without choosing a folder you work in your personal
  space; for work, **Choose…** and pick `~/augmented-brain/work`.

## Windows

**You need:** **Windows 10 or 11**, 64-bit PC or Arm. **winget**, Windows' own installer,
which Windows 11 and an up-to-date Windows 10 already have (if it is missing, it comes with
"App Installer" from the Microsoft Store). The PowerShell that comes with Windows is enough.

**Install:** open **PowerShell** (Start menu, type `PowerShell`, Enter) and paste, with a
right click:

```powershell
irm https://github.com/icalvete/augmented-brain/releases/latest/download/install.ps1 | iex
```

**What it does:**

1. **Git for Windows**, if missing, with winget. **On Windows it is required**, not only for
   the history of your notes: it brings **Git Bash**, where Claude runs commands with the
   guard watching. Windows shows a permission window: click **Yes**.
2. **Claude Code**, if missing, with Anthropic's own installer, into
   `%USERPROFILE%\.local\bin`. If you have it but it is behind, it offers to update it
   (`claude update`).
3. **The Claude desktop app**, if you want it, with winget.
4. **The program**, into `%USERPROFILE%\.local\bin\augmented-brain.exe`.
5. **The PATH:** it adds that folder to your user's PATH. Claude Code's own installer leaves
   that to you; this one does it.
6. **The setup**, and the offer to open the Claude app.

**Where things go:** the program and Claude Code in `%USERPROFILE%\.local\bin`; the
configuration in `%APPDATA%\augmented-brain`; your notes in
`%USERPROFILE%\augmented-brain\<space>\notes`.

**Good to know:**

- After installing, **open a new PowerShell window**: the old one, and the ones already
  open, do not see the new PATH.
- **Claude runs commands in Bash, not in PowerShell.** The guard reads the shell, not
  PowerShell, so it stops every PowerShell command and Claude uses Git Bash instead. You
  will see that block in Claude's messages: it is expected.
- **The app may ask to update itself** the first time ("needs the latest desktop app"):
  winget does not always have the very latest one. Click **Update**.
- To work without the app: `cd ~\augmented-brain\work; claude` (Windows PowerShell has no
  `&&`).

## Updating

When a new version is out, Claude tells you at the start of a session and offers to
update it. You can also do it yourself:

```bash
augmented-brain update
```

It updates the program and the Claude Code plugin. Before, it checks the installation
(and shows what fails) and backs up every space that has a backup; if a backup fails,
it updates nothing (`--force` updates anyway). If no space has a backup, it says that it
goes ahead with no copy to go back to. After, it restarts the idle sessions that still run the old version, with the same
conversation, and says which are left and which task each space has active: ask those
sessions to park their task. Running the install command again also
updates, without those steps. `augmented-brain update --plugin` updates only the plugin.

It first asks GitHub for the latest version, once: if GitHub does not answer, it says so
and stops. The download is then tried three times, also through GitHub's API. If it still
fails, or the program is not the new version afterwards, it says so and ends with an
error: nothing changed, run it again later.

Then close and open the sessions it could not restart (one that was answering, or on
Windows those of Claude Desktop). A session that was open keeps the old program until it restarts,
and its augmented brain tools may fail. `augmented-brain doctor` and the start of a new
session tell you how many sessions still run the old version.

Your notes and settings stay as they are. Sometimes a new version stores notes in a new
way. Then the first session moves them to the new way, after keeping a full copy in the
`migrations/` folder of the configuration, and Claude tells you. An older version never
breaks notes a newer one wrote: it refuses them and asks to be updated. So update every
computer you use.

## Uninstalling

```bash
augmented-brain uninstall
```

It shows what it will remove and asks. It removes the program, the Claude Code plugin and
what Claude Code keeps of it (its cache, its data and its server's logs).

- **First, what would keep running without it:** the [remote access](remote.md) services
  and the sessions opened for tasks (`session start`). Without the plugin they would go
  on with no guard, and with no program left to take them away. If one cannot be
  stopped, nothing else is removed: it tells you why, so you can fix it and run it again.

- **Your notes stay** in `~/augmented-brain/`: they are yours. If you install it again,
  you find everything as it was.
- **The configuration stays** too. `--config` removes it, with your profile.
- **To start from scratch**, `--all` also removes every space's notes. First it shows each
  folder with how many notes it has, and asks. It removes only what augmented brain keeps
  there (the note folders, `MEMORY.md`, `inbox.md`, `docs/`, `archive/`), and the folder
  only if nothing else is left. Anything of yours inside a space's folder stays. A notes
  folder that holds a work folder, your home folder, or a folder with no notes is left
  whole, and the plan says so. `--config` only removes a folder that holds a
  `config.toml`. Your backup on GitHub is not deleted: it says which one it is and how to delete
  it. If a space has no backup, it warns you that its notes will be gone for good. To keep
  them, answer no and set up the backup first.
- Claude Code, its desktop app, git and the PATH entry stay: Claude Code lives in that
  same folder.

On Windows the program file disappears a moment after the command ends, because Windows
does not let a running program delete itself.

All the commands and their options: [Commands](commands.md).

## If something goes wrong

- **The terminal cannot find a command** (`command not found`, or "not recognized" on
  Windows). Open a new terminal: the one you used does not see the new PATH yet.
- **Anything else after installing.** Run `augmented-brain doctor`: it says what is missing
  and how to fix it. It also warns you when a space has no backup, when Claude Code keeps
  a second memory of its own for your spaces' folders, and when other users of the
  computer could read your notes. If the configuration is broken, it tells you how to go
  back to a version that worked.
- **"This system is not supported yet".** It works on Linux, macOS and Windows, on 64-bit PCs
  and Arm.
- **macOS: "Claude Code needs macOS 13".** Update the Mac in System Settings → General →
  Software Update, then paste the command again.
- **macOS: the developer tools window did not open.** Run `xcode-select --install` and then
  the install command again.
- **Linux: no desktop app offered.** It is only for systems with `apt` (Ubuntu, Debian
  and their family); use Claude in the terminal.
- **"cannot download … from GitHub".** GitHub is having trouble. Nothing was installed:
  paste the command again in a while.
- **You stopped the installer half way** (Ctrl+C, a closed window). Paste the command
  again: it sets everything up from the start.
- **Windows: "Git was not installed".** The installer tries winget first and then Git's own
  download. If both fail, install Git from [git-scm.com](https://git-scm.com/download/win)
  and run the command again. The desktop app needs winget: update "App Installer" from the
  Microsoft Store.
- **Claude asks whether you trust the folder.** The first time you open Claude in a
  folder, Claude Code asks this before loading anything. Say yes in your work folder: until
  then, augmented brain cannot load your notes.
- **Windows: accents look broken (`sesiÃ³n`).** Windows PowerShell 5.1 misreads what
  other programs write when it captures it (a redirection, a remote session). The text is
  right: in Git Bash, or in a window, it shows well.
- **Claude asks for a paid plan.** Claude Code needs Pro or above.
- **Something else.** See [Troubleshooting](troubleshooting.md), or tell us in
  [the project's issues](https://github.com/icalvete/augmented-brain/issues), with what
  showed up on screen.
