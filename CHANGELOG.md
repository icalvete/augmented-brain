# Changelog

## Unreleased

- The guard blocks commands for Claude Code's PowerShell tool, the main one on Windows,
  and asks the assistant to use Bash (Git Bash) instead. The guard reads only POSIX
  shell: on Windows, a `git push` through PowerShell went through unchecked.
- A "Getting started" guide for people who have never opened a terminal: what you need,
  opening the terminal, the installer's questions, signing in, the interview, phrases for
  the first week, the words you will hear and what to do when something fails.
- The website, the README and the guide say up front that Claude Code needs a paid
  Claude plan (Pro or above). "Is it free?" said only that augmented brain is.
- The guides have a Copy button on every code block.
- The home page plays a short recreated conversation, two days of work in forty
  seconds: resuming a task, parking it, picking it up in a new session and the guard
  asking before production. It has a pause button, and it shows still with reduced
  motion or without JavaScript. The alpha notice is centered.

## 0.2.2-alpha

Released 2026-10-16. Easier to keep up to date, back up and remove. Checked with the
installer on macOS and Ubuntu, and end to end with a real Claude Code (26 of 26).

- `augmented-brain update`: updates the program, through the published installer, and
  then the Claude Code plugin, which until now stayed at the version first installed.
  Running the installer again also updates the plugin.
- The backup runs by itself: when a Claude session ends, a new hook backs up its space
  in the background, if it has a backup set up. git never waits for a password there;
  if it fails, the next session tells you, with git's reason. Backups run one at a time.
  `augmented-brain backup --space NAME` backs up a space from anywhere.
  `AB_NO_AUTO_BACKUP=1` turns it off.
- Set up the backup by asking Claude: the new `backup_setup` tool creates a private
  repository `augmented-brain-<space>` in the user's GitHub account with `gh`, lets git
  use gh's login, and makes the first copy; without gh, it returns the steps for Claude to
  guide the user through. The first-session interview ends by offering it.
- `restore` works on a freshly installed machine: it replaces a space that is still as the
  installer left it (and a blank profile) instead of refusing; a space with the user's
  notes is never replaced. The guide explains restoring step by step.
- `augmented-brain uninstall`: removes the Claude Code plugin and the program, after
  asking (`--yes` does not ask). The notes always stay, and the configuration too unless
  `--config`; Claude Code, its app, git and the PATH line (Claude Code lives in the same
  folder) are left alone. Installing again finds everything as it was: the installer now
  installs the plugin when it is missing instead of trying to update it.
- At most once a day, the first session tells you when a newer version is out, and
  Claude offers to run the update for you. `AB_NO_UPDATE_CHECK=1` turns it off.

## 0.2.1-alpha

Released 2026-10-16. Tried by a person on a real Mac (macOS 26, Apple M1, no Homebrew):
one command installed Claude Code, the desktop app and augmented brain, and the first
session started in the personal space with the interview, in Spanish.

- On a Mac without Homebrew, the usual one, the installer now installs the Claude
  desktop app too: the latest version from Anthropic's update feed (the one Homebrew
  reads), with its signature checked, into /Applications (or ~/Applications).
- The install check on GitHub runs on a Mac without Homebrew as well.

## 0.2.0-alpha

Released 2026-10-15. **Alpha:** a first run that a newcomer can follow, found by trying
it on a clean Ubuntu desktop. Tested by a person on Ubuntu 24.04; on macOS, the install
is tested hands-off on GitHub's machines, and the visual part (the desktop app, signing
in) is still to be tried by a person.

- The installer asks the language first (the system's is the default; `AB_LANGUAGE`
  answers it), and everything after follows it: its messages, the spaces' names, the
  program and the first-session interview. The interview then follows the language the
  user writes in.
- The installer checks first for git and Claude Code and, if one is missing, gives the
  exact command to install it. It adds `~/.local/bin` to the PATH in the shell's startup
  file, runs `init`, and ends with one next step.
- `init` asks nothing: the language comes from the system, and it creates two spaces,
  `work` and `personal` (`trabajo` and `personal` in Spanish), each in
  `~/augmented-brain/<space>` with its notes inside. `init --custom` asks, as before.
- A work folder that does not exist is created instead of refused.
- The installer offers to install what is missing (git with the system's package
  manager, Claude Code with its official installer, and the desktop app), showing only a
  line while it works and a ✓ when done (their own output only if they fail), and, at
  the end, to open Claude in
  the work space, with the right PATH: one command from nothing to the first session.
- A guided first session: while the profile is blank, Claude asks six quick questions,
  saying why it asks each one, and with the answers saves the profile (`profile_save`),
  creates areas, sets the guard (`safety_save`: production AWS profiles and hosts not to
  touch) and remembers people (`person_save`).
- A default space, `default_space` (set to `personal` by `init`): a session outside every
  work folder, such as the scratch folder where the Claude desktop app starts a new
  session, uses it, and Claude says so in one line. Without one, Claude says at once that
  no memory is loaded and which folders to open.
- People: a new note type in `people/`, listed in the index with who they are and the
  names the user calls them, so "Marcos" leads to the right person.
- `doctor` fails when git, Claude Code or the plugin is missing, with the command to
  fix it, and shows what is only news (a blank profile, a folder outside your spaces)
  with `·` instead of a warning.
- `restore` puts the notes back in the same place inside the work folder, installs the
  plugin when it is missing, and ends with the next step.
- The language is read from `LC_ALL`, `LC_MESSAGES` and `LANG`, in that order.

## 0.1.0-alpha

Released 2026-10-15. **Alpha:** the first usable version, tested on Linux only. It may
have bugs and its format may change; read [DISCLAIMER.md](DISCLAIMER.md).

- Memory format v0.1 (`docs/spec/memory-format.md`): a folder per type, one profile for
  the person plus an optional one per space; importing other formats is planned for 0.2.
- Go core: `internal/config` (spaces, chosen by the folder you work in) and
  `internal/memory` (reads and checks every note's header; the body can be in any language).
- `augmented-brain version`, `space` and `check`.
- `augmented-brain init` and `space add`: create the configuration, the profile and the
  spaces from templates in English or Spanish. Nothing is overwritten, and the
  assistant's own settings are not touched.
- `augmented-brain doctor`: says what is missing, and how to fix it.
- The session index, `MEMORY.md`, is generated: live tasks grouped by area.
- MCP server (`augmented-brain mcp`), for any assistant that speaks MCP: open, update,
  journal, park and close tasks; see what is pending; capture to the inbox; create
  areas, references and lessons. Calls run one at a time, also across sessions.
- Claude Code plugin `brain` (marketplace in this repository): loads the space's profile,
  rules and index when a session starts, guards commands and file writes (no commit or
  push, no root, no secrets; production AWS profiles and untouchable hosts ask first),
  and registers the MCP server. Checked end to end with a clean Claude Code: open a
  task, record a decision, park it, resume it in a new session, close it.
- `init` offers to install the Claude Code plugin; `doctor` checks it.
- Backup and restore: `augmented-brain backup setup <url>`, `backup` (or `backup --all`)
  and `restore <url>`. Each space goes to its own private repository with its
  configuration and the profile; a backup is refused while a note holds a secret.
- Releases: `install.sh` (Linux and macOS, no root, checksum verified) and archives for
  amd64 and arm64, built and published by GitHub Actions on a version tag.
- A code review before this release: the guard understands compound commands and what
  shells read; parking never loses a journal; config edits keep comments; backups merge
  across two machines. `scripts/e2e-claude.sh` checks the whole story with a real Claude.
- The website: a product page in English and Spanish, with a disclaimer and this alpha notice.

## 0.0.1

First public snapshot: the project's goal, design and brand. Nothing to install yet.

- README in English and Spanish.
- Draft of the memory format (`docs/spec/memory-format.md`).
- Brand: logo, symbol, icons, README header, social card and palette, with the scripts
  that build them (`assets/brand/`) and a sheet that shows them in place (`docs/brand.html`).
  `logo-dark` and `symbol-dark` add a light border for dark backgrounds. The README header
  is the logo on a white rounded card, so it reads on GitHub's light and dark themes.
- `scripts/privacy-check.sh` and a pre-commit hook that block personal data and secrets.
- MIT license.
