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
  GitHub, and its SHA-256 must match the release's list, or nothing is installed.

## Linux

**You need:** a 64-bit PC or Arm (amd64 or arm64), `curl` or `wget`, and `sha256sum` (all
usual distributions have them).

**Install:** open a terminal (on Ubuntu, `Ctrl + Alt + T`) and paste:

```bash
curl -fsSL https://github.com/icalvete/augmented-brain/releases/latest/download/install.sh | sh
```

**What it does:**

1. **git**, if missing, with your system's package manager: `apt` (Ubuntu, Debian), `dnf`
   (Fedora), `pacman` (Arch) or `zypper` (openSUSE). It asks for your password.
2. **Claude Code**, if missing, with Anthropic's own installer, into `~/.local/bin`. If
   you have it but it is behind, it offers to update it (`claude update`).
3. **The Claude desktop app**, if you want it, **only on Ubuntu and Debian**: from
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

```bash
augmented-brain update
```

It updates the program and the Claude Code plugin; your notes and settings stay. Claude
tells you when a new version is out. Running the install command again does the same.

## Uninstalling

```bash
augmented-brain uninstall
```

It removes the program, the Claude Code plugin and what Claude Code keeps of it (its cache
and its server's logs). **Your notes stay**, and so does the configuration unless you add
`--config`. To start from scratch, `--all` removes the configuration, your profile and
every space's notes too, after showing what and asking. Claude Code, its desktop app, git and the PATH
entry stay too: Claude Code lives in that same folder. On Windows the program file
disappears a moment after the command ends, because Windows does not let a running program
delete itself.

## If something goes wrong

- **The terminal cannot find a command** (`command not found`, or "not recognized" on
  Windows). Open a new terminal: the one you used does not see the new PATH yet.
- **Anything else after installing.** Run `augmented-brain doctor`: it says what is missing
  and how to fix it.
- **"This system is not supported yet".** It works on Linux, macOS and Windows, on 64-bit PCs
  and Arm.
- **macOS: "Claude Code needs macOS 13".** Update the Mac in System Settings → General →
  Software Update, then paste the command again.
- **macOS: the developer tools window did not open.** Run `xcode-select --install` and then
  the install command again.
- **Linux: no desktop app offered.** It is only for Ubuntu and Debian; use Claude in the
  terminal.
- **Windows: "winget is not recognized".** Install "App Installer" from the Microsoft Store,
  or install Git from [git-scm.com](https://git-scm.com/download/win), and run the command
  again.
- **Claude asks for a paid plan.** Claude Code needs Pro or above.
- **Something else.** Tell us in [the project's issues](https://github.com/icalvete/augmented-brain/issues),
  with what showed up on screen.
