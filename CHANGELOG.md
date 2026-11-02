# Changelog

## Unreleased

From the second rehearsal of a real migration:

- **A restored machine is the same again.** The space that was the default one is the
  default again (before, restoring two spaces left none, and then no backup kept the
  profile). The guard's lists are still in no backup, since they may name work
  machines: `restore` and `doctor` now say so, until you give them again.
- **The installer refreshes the package lists before installing git** (`apt-get
  update`): with old lists, apt failed and the installation stopped.
- **`import` again puts each file where its note is.** A second `--apply` sent the
  archive and the documents of notes imported before to the space of `*`. A secret in a
  document or an archive file is named in the report too, and `check` looks at every
  file the backup takes, not only the notes.
- **The guard follows `cd`.** `cd ~ && rm -rf .` is never run, and `git -C` with a
  relative path counts from where the `cd` went. `rm -rf *` in a folder is no longer
  taken for the whole disk.
- **The guard looks inside `vagrant ssh -c`, `docker exec`, `podman exec` and `kubectl
  exec`**, as it does inside `ssh`: becoming root there is for you. And an Elasticsearch
  API key pasted alone, without `ApiKey` before it, is caught as a secret.
- **Flags go anywhere** (`import --map m folder --apply`), and an argument too many is
  an error instead of being ignored.

## 0.5.2-alpha

Released 2026-11-02.

From the first rehearsal of a real migration:

- **A secret key that ends in `/` or `+` is caught.** The guard let an AWS secret key
  through when its last character was not a letter or a digit (and a Google key ending
  in `-`). Writes and commands with one are now blocked like the rest.
- **The repositories you let Claude commit in count from every space**, like the
  production profiles: a repository listed in `personal` is let through from a `work`
  session too. A repository in both lists asks.
- **A push that rewrites the remote asks, always** (`--force`, `--delete`, `--mirror`,
  `+branch`), even in a repository Claude may push to. **`rm -r` of your whole home
  folder or of the whole disk is never run.**
- **Your rules reach every session.** When they do not fit whole at the start, each one
  goes as its title and summary, not only the path of its file.
- **`import` brings the rest:** the captures of `inbox.md` go to the inbox, and
  `docs/<task>/` is copied as it is. A note with a secret written in it is named in the
  report, and `check` names it too.
- `docs/` in the notes folder is never read as notes (a document with no header was
  reported as a broken note). The index of the shared notes lists their rules and
  knowledge, not areas and tasks they cannot have. `init` warns when the plugin did not
  install (with no git, say), instead of only saying it is ready. `augmented-brain panel`
  opens on the space of the folder it is run in.

## 0.5.1-alpha

Released 2026-11-02.

- **The panel shows your rules and the shared notes.** A new Rules section lists the
  rules of the space and the shared ones; Knowledge lists the shared references and
  lessons too. Each shared note is marked as such.

## 0.5.0-alpha

Released 2026-11-02.

- **Notes shared by every space.** What you learn that holds everywhere (how git
  behaves, how you like answers) is saved once, in `shared/` next to your profile, and
  every space sees it: its rules are loaded at the start of each session, its references
  and lessons are listed for Claude to open. Tell Claude "this holds everywhere" and it
  saves the note there. Only rules, references and lessons can be shared. They travel in
  the default space's backup, like the profile, and `import` can send notes there
  (`lesson-x = shared` in the map).
- **Claude can save a rule you give it** (`rule_new`), in the space or shared, instead
  of you writing the file by hand.
- **Your repositories stay clean.** A new common rule: a repository only gets what its
  team needs, through a branch. The plans, analyses and reports of a task go to `docs/`
  in the notes folder, next to the task, and are never loaded at the start.
- **Bring in the memory Claude Code already keeps.** `augmented-brain import <folder>`
  reads a memory written by Claude Code (or shaped like it) and shows where each note
  would go in your spaces, and what it could not decide: files that are not notes,
  notes about you, links that would cross spaces, ids already taken. Nothing is written
  until you run it again with `--apply`, and the folder you import from is only read.
  With several spaces, a small file says which space each area goes to, which notes
  go somewhere other than their area, and which go to the shared notes (`--map`).
- **The guard misses less.** `terraform`, `tofu`, `pulumi`, `eksctl` and `copilot` are
  checked against the production profiles like `aws`, the inherited one included. In
  `find` with several `-exec`, a blocked one is no longer hidden by an earlier one that
  only asks. `git -C a -C b` is read as git reads it, and a commit that names another
  repository (`--git-dir`, `--work-tree`, `GIT_DIR`) never counts as an allowed one.
- **A change to a task is saved whole or not at all.** A task update with a bad watch
  date used to save the rest of the change and then fail.
- **The help and the commands' errors speak your language**, as the rest already did,
  and so does the panel's warning about files left out of a backup.
- Smaller fixes: the configuration keeps its format version when rewritten; `import`
  refuses to write into notes of a newer version and prints a command that can be
  pasted; every small file the program rewrites (the inbox too) is replaced in one step;
  an unknown `AB_LANGUAGE` no longer stops the installer; `uninstall` no longer takes
  any error that says "not" as "already gone".

## 0.4.0-alpha

Released 2026-11-01.

- **Notes say which format they follow.** Each space's notes now carry their format
  version (`.augmented-brain/format`), and so does the configuration folder (in
  `config.toml`, from version 2 on). When a future release writes notes in a new way,
  the first session after updating moves them forward, after a full copy in
  `migrations/` of the configuration folder, and Claude tells you; `augmented-brain
  check` does the same. A program older than the notes refuses to open them instead of
  breaking them, and asks to be updated. Nothing changes for notes today: this is
  format 1.

- **Commit and push where you say, or after asking.** Besides `git_allowed_repos`, a
  space can list repositories in `git_ask_repos`: there, Claude's `git -C <repo>
  commit` or `push` asks you each time instead of being blocked. In both lists a git
  worktree of a listed repository counts as it, and the last part of a path may hold a
  `*` for one folder (`~/code/app-*`).

- **More kinds of secrets.** The guard, the backup check and the summaries now also
  know Google AI Studio keys, PostHog personal keys, Elasticsearch API keys written as
  `ApiKey …`, and Telegram bot tokens of any shape.

## 0.3.3-alpha

Released 2026-11-01.

- **Repositories that are no longer there.** A note's `repos:` names folders inside the
  space's work folder. When one is not there any more (it was moved, renamed or
  deleted), `augmented-brain check` and the panel's Health page now say so, the task's
  page in the panel shows it, and reading the task tells Claude to ask you where it is
  now. Nothing is changed by itself. Closed tasks are not checked.

## 0.3.2-alpha

Released 2026-11-01.

- **Sessions in the panel.** A new section lists your Claude sessions of each space on
  this computer. Those that did not end show what was said after the last journal
  entry, and you can keep it in the notes (in `archive/sessions/`, secrets taken out,
  with a line in the task's journal) or mark it as reviewed, without opening Claude.
  Those running now and those that ended show how much disk their transcripts take,
  and the transcript of an ended session can be deleted, after you confirm that it can
  no longer be resumed.

## 0.3.1-alpha

Released 2026-11-01.

- **The guard fails closed.** If `augmented-brain` cannot run when Claude is about to
  run a command or change a file (the program is missing from the PATH, crashes or
  takes more than 10 seconds), Claude Code now stops that action instead of letting it
  through unchecked. It uses Claude Code's `onFailure: "block"`, from 2.1.295; older
  versions ignore it and behave as before. Only the guard does this: the hooks that load
  the space, remind, keep the summary and back up never block your work.
  `augmented-brain doctor` now says which of the two problems it found: the guard
  cannot run (everything stops) or it runs but lets a risky command through.
  `augmented-brain uninstall` keeps the program when the plugin could not be removed
  completely, so Claude Code is never left with a guard it cannot run.
- **Commands left running are guarded too.** Claude Code's Monitor tool runs a command
  in the background to watch a log or a build; the guard now checks that command
  exactly as it checks Bash. The tool calls of subagents and workflows already went
  through the guard.
- The end-to-end check of the Stop reminder no longer fails when the assistant adds a
  line after its answer, such as "Nothing new happened on the task".

## 0.3.0-alpha

Released 2026-11-01.

- **Subtasks and steps.** A part of a task that waits on someone else or goes at its
  own pace can be a subtask: a task of its own with `parent:` in its header
  (`task_new` with `parent`). The index, `pending` and the panel list it under its
  parent; only one level, and the parent cannot be closed while a subtask is open.
  Smaller parts are steps: checkboxes (`- [ ] text`) in the task's Plan, added and
  ticked with the new `task_step` tool or from the panel, and counted in the index
  ("2/5 steps"). Each change of a step writes a short line in the journal.
- **The panel**: `augmented-brain panel`, or `/brain:panel` in Claude, opens a page in
  your browser to see and edit your spaces without the assistant: tasks (status, who
  they wait on, next step, journal, park, close, new), watch dates and reminders, the
  inbox, areas, knowledge, people and your profile, and each space's health (invalid
  notes, index, last backup, weekly review). It runs only on this computer, only while
  you use it (it stops by itself after 30 minutes without use, or with
  `augmented-brain panel stop`), and only for whoever has the address it prints: the
  address carries a key made for each run, and no cookie is set. It loads nothing from
  the internet. Every change goes through the same code and the same locks as Claude's
  tools, so the two never step on each other: saving a task saves only the fields you
  changed, and a step is ticked only if it still says what the page showed. Running the
  command several times at once starts one panel.

## 0.2.9-alpha

Released 2026-11-01.

- The production AWS profiles and the untouchable hosts are now for the whole machine.
  Before, they belonged to the space the first-session interview ran in (often
  `personal`, in the desktop app), so work sessions were not guarded. `safety_save` writes
  them at the top of the configuration; lists still kept inside a space keep working.
- The guard checks the AWS profile an `aws` command really uses when it gives none
  (`AWS_PROFILE`, `AWS_DEFAULT_PROFILE`, or `default`); `cdk`, `sam`, `serverless` and
  `aws-vault exec` with a production profile ask too; when several profiles could apply,
  any production one asks. `gh pr merge`, `gh release create`, `glab mr merge` (and
  `accept`), `gh repo sync <repo>` and similar are blocked like `git push`. What a pipe feeds a shell is
  checked (`echo 'sudo …' | bash`, a heredoc into `bash` or `ssh`). Host names match in
  any case and short or full either way; `sftp host` and `mosh` are covered.
- When the configuration cannot be read, the guard asks before every `aws`, `ssh`, `scp`,
  `rsync`, `sftp`, `mosh`, and `cdk`, `sam` or `serverless` with a profile, instead of
  letting them through unchecked.
  `augmented-brain doctor` runs the guard for real and says if it does not block, and
  shows the guard's lists.
- One note that is not valid no longer locks every tool: the others keep working, the
  broken file is never written over, and the assistant is told which file and how to
  fix it. A note saved with a byte order mark (some Windows editors) is valid.
- The backup: the profile travels only in the default space's backup, never in a work
  repository (a copy left by an older version is removed). `space add` sets the default
  space when there was none, so some backup always keeps the profile, and `doctor` warns
  when there are several spaces and no default one. The guard's lists are in no
  backup; after a restore, the next session offers to ask for them again. Captures in the inbox from
  two machines merge instead of stopping the backup. Leftovers of a crash and system
  files stay out (`.gitignore`), and so does any file over 50 MB, with a warning at the
  next session. A backup never runs at the same time as a write of the assistant or
  another backup. AWS's documented example keys no longer stop it.
- A restore that fails half way leaves the space it would replace as it was.
- Offline, sessions no longer wait for the new-version check every time.
- People whose names have no Latin letters no longer overwrite each other; a person is
  found by name or by one of their other names.
- `augmented-brain check` (and `space`, `backup`) outside every space's folder use the
  default space, as Claude does. Folders match in any case on macOS and Windows.
- Work is written down as it happens, so a session that ends badly loses little. The
  method now says to journal each decision or finished step at the moment. A new Stop
  hook reminds Claude, once, when a task is open and nothing was written for 20 minutes.
  After Claude Code compacts a conversation, its summary is kept in
  `archive/sessions/` (secrets taken out), the task's journal points at it, and Claude is
  asked to write down what was decided. A session that did not end normally is offered
  for recovery at the next start: the new `session_recover` tool reads what was said
  after the task's last journal entry from Claude Code's transcript on this machine.

## 0.2.8-alpha

Released 2026-10-19.

- `augmented-brain uninstall` leaves nothing of the plugin behind: it also removes what
  Claude Code keeps of it after uninstalling it (its cached copy, its data folder and the
  logs of its MCP server), which a real uninstall left on the machine.
- `augmented-brain uninstall --all` starts from scratch: the configuration, the profile and
  every space's notes go too, after showing what (with how many notes each folder has) and
  asking. A space's folder goes only when nothing of the user's is left in it. Backups on
  GitHub are not deleted: it says which and how. The install check on Linux and macOS ends
  with it and makes sure nothing is left.

## 0.2.7-alpha

Released 2026-10-19.

- Installing the desktop app no longer goes quiet or gets it wrong. Before asking for the
  password (Linux), the installer says it will not show as you type it; it asks for it up
  front, then shows a dot every few seconds while the app installs. Whether it worked is
  judged by the app being there, not by the exit code: on a real machine, apt failed over
  half-installed kernel packages after installing the app all right, and the installer
  said the app was not installed. On failure, only the last lines of the output show,
  with the file that has all of it. The same on macOS and on Windows (where it warns
  that Windows may ask for permission in a window), and for installing git with sudo.
- When a Claude app is already open, the installer says so (close it and open it again,
  so it loads augmented brain) instead of seeming to open one: a second Claude app only
  hands over to the one running and closes, so nothing appeared.

## 0.2.6-alpha

Released 2026-10-19.

- The installer, on Linux, macOS and Windows, checks a Claude Code that is already there:
  when it is older than Anthropic's stable release it says so and offers to update it with
  `claude update` (yes by default). An old one can lack what the plugin uses, like the
  end-of-session hook that starts the backup. If the update fails, it says how to do it
  later and goes on. The install checks cover an old and a current Claude Code.

## 0.2.5-alpha

Released 2026-10-18.

- At the end of the install, the message about the Claude app says what to do: open
  the Code tab and click + New, which starts in the personal space; for the work space,
  pick its folder with Choose…. It used to send people to choose the work folder.

## 0.2.4-alpha

Released 2026-10-18. The weekly review and reminders with a date, checked end to end
with a real Claude Code (30 of 30).

- The weekly review, which the inbox always promised: when the last one is a week old and
  there is something to go through, the session start offers it. The new
  `weekly_review` tool lists the inbox, tasks untouched for two weeks and the dates due
  this week; `review_done` keeps what was decided in `reviews/` and empties the
  processed captures from the inbox.
- Reminders: "remind me on Thursday" is a capture with a date (`capture` takes
  `remind_on`), and `task_update` can now add a watch date to a task or mark it done.
  Whatever is due today or tomorrow, or overdue, is said at the start of the session.
- The installer on Linux and macOS says that Claude Code needs a paid plan before
  installing it, as the Windows one already did, with a link to the plans, and asks
  whether you have one. A "no" does not stop it: Claude asks you to sign in the
  first time you open it. Without a terminal, or with `AB_YES=1`, it only informs.

## 0.2.3-alpha

- On a phone, the website's menu shows every link, wrapping under the name instead of
  hiding the sections, and the guard's commands wrap instead of being cut.
- A guide, "Installing" ("Instalación"), that says what the one install command does on
  Linux, macOS and Windows, where it puts things, what is different on each, and how to
  update and uninstall. The README, the website and "Getting started" give the Windows
  command too, and no longer say Windows is not supported. install.sh, run from Git Bash
  on Windows, points to install.ps1.
- The installer on a Mac: it says up front when macOS is older than 13 (Ventura), which
  Claude Code needs, instead of installing what would not start. Without Apple's
  developer tools it no longer takes the stub /usr/bin/git for git; it opens their
  installer and, with a terminal, waits for it and carries on, with no need to run it
  again. With bash, the PATH line goes to ~/.bash_profile (or the .bash_login or
  .profile already there), which the Mac's Terminal reads, not to ~/.bashrc.
- `scripts/install-check-mac.sh` checks those three on every Mac row of CI, and CI now
  has an Intel Mac too (macos-15-intel).
- The guard blocks commands for Claude Code's PowerShell tool, the main one on Windows,
  and asks the assistant to use Bash (Git Bash) instead. The guard reads only POSIX
  shell: on Windows, a `git push` through PowerShell went through unchecked.
- On Windows, uninstall removes the program too, and the console shows accents right.
- A Windows installer, `install.ps1`, and Windows archives in every release. On Windows,
  `update` runs it, and init says `cd …; claude`.
- The language is detected on Windows too (its display language), and `AB_LANGUAGE`
  wins over the locale.
- On Windows, the lock that keeps two writers apart now works, and the backup at the end
  of a session runs on its own.
- The plugin starts the program directly, with `~/.local/bin` added to the PATH it gets,
  instead of through `sh`, which Windows does not have: there the server did not start.
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
