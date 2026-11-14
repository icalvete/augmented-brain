# Installing


One command installs augmented brain and everything it needs, on Linux, macOS and
Windows. This page says what that command does on each system, where it puts things and
what is different on each one. New to the terminal? [Getting started](getting-started.md)
walks you through it.

## On every system

- **A paid Claude plan** (Pro or above: Pro, Max, Team or Enterprise), or an Anthropic API
  key. augmented brain is free, but it works with Claude Code, and the free plan does not include Claude
  Code. The installer says so before installing it. Plans: [claude.com/pricing](https://claude.com/pricing).
- **No administrator or root.** The program goes in your own folder. Only installing
  something that is missing, like git, may ask for your password.
- **The installer asks first.** The language (your system's is suggested), then each
  missing piece, with what it is for. Press Enter to accept what it suggests. To change the
  language later, set `language` in your [configuration](configuration.md#settings-for-the-whole-computer).
- **No GitHub account.** The program downloads from GitHub without one. The backup does
  use one, or another git server, and you set it up later: see [Backup and restore](backup.md).
- **A new installation ends the same way:** two spaces, `work` and `personal` (in Spanish,
  `trabajo` and `personal`), in `~/augmented-brain`, the Claude Code plugin installed, and
  the offer to open Claude. Run over one you already have, it keeps your spaces and
  settings, and updates the program and the plugin.
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
- In the app: **Code** tab, **+ New**. Without choosing a folder you work in your personal
  space; for work, **Choose…** and pick `~/augmented-brain/work`.
- To work without the app: `cd ~/augmented-brain/work && claude`.
- The first time you open Claude in a folder, it asks whether you trust it: say yes, or
  augmented brain cannot load your notes.

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

**You need:** **Windows 10 or 11**, 64-bit PC or Arm. The PowerShell that comes with
Windows is enough. **winget**, Windows' own installer, helps but is not required: Windows 11
and an up-to-date Windows 10 already have it (if it is missing, it comes with "App
Installer" from the Microsoft Store). Without it, Git comes from its own website; only the
desktop app needs winget.

**Install:** open **PowerShell** (Start menu, type `PowerShell`, Enter) and paste, with a
right click:

```powershell
irm https://github.com/icalvete/augmented-brain/releases/latest/download/install.ps1 | iex
```

**What it does:**

1. **Git for Windows**, if missing, with winget, or from Git's own website without it.
   **On Windows it is required**, not only for
   the history of your notes: it brings **Git Bash**, where Claude runs commands with the
   guard watching. Windows shows a permission window: click **Yes**.
2. **Claude Code**, if missing, with Anthropic's own installer, into
   `%USERPROFILE%\.local\bin`. If you have it but it is behind, it offers to update it
   (`claude update`).
3. **The Claude desktop app**, if you want it, with winget (without winget, it is not
   offered: get it from [claude.ai/download](https://claude.ai/download)).
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

It backs up your spaces first, updates the program and the Claude Code plugin, and
restarts the sessions still running the old version. Running the install command again
also updates. What each step does, and what to do with the sessions it leaves: [Updating
and uninstalling](update-and-uninstall.md#updating).

## Uninstalling

```bash
augmented-brain uninstall
```

It removes the program and the Claude Code plugin, after showing what it will remove and
asking. Your notes and the configuration stay; `--config` and `--all` go further. What
each level removes and keeps: [Updating and uninstalling](update-and-uninstall.md#uninstalling).

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

## Check a download

You do not need this to install. It is for whoever wants to be sure that what they
downloaded is what the project published, made from its public code.

Every release is built by GitHub from a commit of `main`, never on someone's computer,
and GitHub signs a record of it: which workflow built each file, from which commit.
With the [GitHub CLI](https://cli.github.com/) (`gh`), check any file of a release,
the installer included:

```bash
gh attestation verify augmented-brain_<version>_linux_amd64.tar.gz -R icalvete/augmented-brain
```

with the name of the file you downloaded.

It ends with `Verification succeeded!` and names the workflow and the commit. A file
changed after it was built fails.

`checksums.txt`, in every release, holds the SHA-256 of each archive and of both
installers. The installers check the archive against it; you can check any file too:

```bash
sha256sum --ignore-missing -c checksums.txt
```

The build is reproducible: the same commit, built with the Go version in `go.mod`, gives
the same bytes. `scripts/release-build.sh <version>` from a clone of that tag writes
the same `checksums.txt` as the release.
