# Changelog

## Unreleased

- **Looking for secrets is about 30 times faster.** `check`, every backup and every
  note written read the text for secrets at about 2.5 MB a second; with notes of a
  dozen MB, `check` took more than a second. Each kind of secret now has a quick test
  first, and its full search runs only when the test finds something like it: `check`
  takes about a tenth of a second or less. What is found is exactly the same, and a test
  compares both ways on thousands of texts.

## 0.21.0-beta

Released 2026-11-13.

- **The guard treats `doas` as root again, as its guide says.** `doas id` went through
  as if it were just `id`. Now `doas`, like `sudo`, is stopped unless it runs the
  command as another user (`doas -u deploy …`), and a test makes sure no other command
  with a rule of its own can be skipped the same way.
- **The backup only works in its own repository.** If your notes folder lost its `.git`
  (copied or restored by hand, or removed) and sits inside another git repository, such
  as your work folder, a backup used to commit and push in that other repository. Now it
  stops, touches nothing, and tells you to run `augmented-brain backup setup` again.
  Going back to an earlier copy does the same.
- **Claude no longer sets up a backup at an address it was given.** In a space with no
  backup, one call to `backup_setup` with an address sent every note there, and again at
  the end of each session; the address could come from a page or a message Claude read.
  Now the tool only creates the private repository in your GitHub account. A backup
  somewhere else you set up yourself, with `augmented-brain backup setup <url>` or the
  panel's Backup page.
- **No passwords in a backup's address.** `backup setup` refuses an address with a
  password or a token in it, since it was kept in the configuration and inside the
  backup itself. Let git keep the login (`gh auth setup-git`, or ssh). If you already
  have one, `doctor` tells you how to take it out, and backups no longer carry it.
- **Built with a Go that still gets security fixes.** Releases were built with Go
  1.25.0, which carries 46 known vulnerabilities in Go's own library that the program
  can reach (in its web panel, the network and file handling among them). They are now
  built with Go 1.26.11, and `golang.org/x/sys` moves to 0.50.0. Building from source
  needs Go 1.26 or later. Installing the next release closes them for you.
- **Vulnerabilities are watched.** With every change and every week, the CI checks for
  known vulnerabilities the program can reach, on Linux, macOS and Windows; Dependabot
  proposes newer modules and actions. `SECURITY.md` says how to report a security
  problem privately.

## 0.20.0-beta

Released 2026-11-12.

- **The guard stops an assistant from starting a release workflow.** `gh workflow run
  release` (or the same call through `gh api`) publishes a version, like `gh release
  create`, so it is for you. Other workflows, such as a deployment, still run. A
  workflow named only by its number asks first.
- **Long tasks read in one go again.** A journal kept growing until Claude could not read
  the task at once, and had to read it in pieces. Now, when a task passes 24 KB, its
  oldest journal entries move to a file in `archive/journals/`, and a line in the journal
  says where; the last 8 KB or so of entries stay. Nothing is lost. `check` does it for
  tasks already past it, `doctor` tells you when there are some, and the backup joins a
  task one machine shortened while the other wrote in it. A task still too long, written
  by hand, is read without its oldest entries, with a line saying where they are.
- **Work on a task of another space.** From a session of one space, Claude can now read
  and write a task of another when you ask: list that space's tasks to find it, add to
  its journal, tick a step, change its next step or status. Before, what you decided
  about a work task in a personal session had to be passed on by hand. What it writes
  there ends in `(from personal)`, nothing is copied into this space, and closing is left
  to a session of that space. That space is backed up when the session ends.
- **`augmented-brain clean` removes what working with Claude Code leaves behind.**
  Worktrees Claude Code made whose session is over, git's records of worktrees whose
  folder is gone, and the scratch folders of sessions that ended a day ago or more. It
  shows the list first and asks; a worktree with changes or commits of its own is only
  named. `doctor` says when there is something to clean.
- **`doctor` names background sessions nobody uses that keep running.** Archiving a
  session in Claude Desktop does not stop its process. `doctor` now names those that
  ended or were archived, and those no task opened that have been idle for over 12
  hours, each with the `claude stop` that ends it.

## 0.19.2-beta

Released 2026-11-12.

- **Updating no longer brings back sessions you archived.** In 0.19.1, `update` (and
  `session restart --old`) brought back background sessions you had archived in Claude
  Desktop or claude.ai, which keep their process. Now an archived or ended session is
  left where you put it: only its earlier augmented brain stops, so `doctor` no longer
  counts it. The same goes for any background session `session start` did not open
  for a task: archived in Claude Desktop, Claude is not told, so it cannot be sure.

## 0.19.1-beta

Released 2026-11-12.

- **A session can close itself.** `session stop <task> --when-idle` run from the
  task's own session waited for that session to finish its answer, and the session
  could not finish while the command ran: they waited for each other. Now the command
  returns at once and a process in the background waits and closes it.
- **A task's session tells you when it finishes.** What a background session writes
  stays there until you open it, so an ended CI or a parked task could go unnoticed.
  Its first message now has it also send you a notification then.

## 0.19.0-beta

Released 2026-11-12.

- **`watch pr --task` keeps watching when its session is gone.** The watching runs in
  a process of its own, so a session closed, restarted or retired by Claude no longer
  takes it along: the result still reaches the task, and your other sessions hear of it.
- **`remote status` names the watch.** On Linux it adds a line on the watch that brings
  remote access back when the server archives its session: whether it is running, how
  long ago it last looked, and what to do when it is stopped or its last look failed.
  `remote install` says it left it running, and `doctor` warns when it fails.
- **The installer no longer waits for ever on the desktop app.** If its download stalls,
  it gives up after 8 minutes on Linux (`AB_DESKTOP_TIMEOUT` changes that), says so, and
  finishes the install without the app; on macOS it gives up on a download that stays
  under 1 KB a second for a minute.
- **A task's session Claude stopped comes back with its conversation.** Claude stops a
  background session left unused for hours (and every one when its background service
  restarts), and claude.ai then shows it as disconnected, with no way to go on.
  `augmented-brain session start` with its task now brings it back (`claude respawn`):
  the same session, its whole conversation, connected again. `session list` calls it
  retired again (with Claude Code 2.1.322 it said "ended") and says how to bring it back.
- **Updating no longer asks you to close Claude Desktop.** `session restart`, `--old` and
  `update` now update a session of Claude Desktop or a terminal without closing it: only
  its augmented brain server stops, and Claude Code starts it again with the new program
  the next time the session uses it. Only while the session is idle; Linux and macOS
  (on Windows, close and open it as before). `doctor` says so too.
- **`augmented-brain watch pr N` waits on a pull request and says how it ended.** It
  ends when the pull request is merged or closed, a required check fails, every required
  check has passed, or 3 hours go by, and never waits for ever on an answer it does not
  understand. With `--task` it writes the result in the task, so your other sessions
  hear of it even if the session that started it is gone. Task sessions and the common
  rules now use it instead of loops of their own, which sometimes never told you.

## 0.18.0-beta

Released 2026-11-12.

- **Updating restarts the sessions still on the old program.** After an update,
  `augmented-brain update` restarts, one at a time, the idle Claude sessions that still
  run the old augmented brain, with the same conversation: a task's session (`claude
  respawn`) and those of the remote access (their service, only when all its sessions
  are idle, and after moving out whatever else is inside). A session that is answering
  is looked at again right before and left as it is. Claude Desktop's are left too, and
  it says why: close and open them yourself. `--no-restart` updates without restarting
  any. This works from the update after this version: the update to it still gives the
  old notice, and `augmented-brain session restart --old` does it by hand.
- **`augmented-brain session restart`** restarts one session (by its task, id or name),
  or every one still on an earlier program (`--old`). With `--when-idle` it waits until
  the session has finished its answer, so a session can restart itself right after
  parking its task.

## 0.17.4-beta

Released 2026-11-11.

- **Restarting remote access no longer stops your task sessions.** A task session
  started from a remote session could start Claude's background service inside the
  remote access service, and restarting that service (by hand, by `remote install` or by
  the watch) stopped every task session with it. Now `session start` starts it in a
  group of its own, and the watch, `remote install` and `remote uninstall` move out of
  the service whatever its server did not start before they restart or stop it. When
  they cannot, they say so and leave it running. Linux only.

## 0.17.3-beta

Released 2026-11-11.

- **A task's session hears only its task.** A session opened with `session start` got
  every notice of the space at its start: other tasks' dates, the weekly review,
  captures from other spaces, sessions to recover and other sessions to restart. Now it
  hears only its own task's dates. What concerns everything (a failed backup, a new
  version, the guard) still comes.
- **A session that runs an old program says so of itself.** After an update, the start
  of a session counted the open sessions still running the old program, and counted
  itself among the "others" when `/clear` had kept its old program. Now it tells you
  that this very session needs closing and opening again.

- **A task's session that cannot work says so.** When Claude Code's login had expired,
  `session start` opened the session anyway: its first answer was "Login reached its
  time limit", and `session list` still called it active. Now `session start` checks
  the login before opening anything, waits for the first answer (up to a minute), and
  when that answer is an error it closes the session and says what to do, such as
  `claude auth login`. `session list` shows a session whose last answer failed as
  **failed**, with the error.
- **Each session has its own active task.** The reminder to write in the journal, at
  the end of an answer, named the last task any session of the space had touched. With
  two windows on different tasks, one was asked to write in the other's journal. Now
  each session keeps its own task, and so does the line a compaction adds to the
  journal. Sessions opened before the update get no reminder until they are restarted.
- **Remote access comes back when the server archives its session.** Until now, when
  the session of a remote access service was archived (in claude.ai, say), the service
  went on running with nobody behind it, and nothing brought it back. On Linux a watch
  now looks every two minutes and restarts it, with a new session of the same name in
  about ten minutes. Run `augmented-brain remote install` again to get it: `remote
  status` and `doctor` say so.

## 0.17.2-beta

Released 2026-11-11.

- **See what each backup copy changed, and go back to one.** On the panel's Backup page,
  each copy shows the notes it added, changed and removed, with their lines. **Go back
  to this copy** shows first what would change, asks for a box and the space's name,
  and keeps what you had as a copy before, so it can be undone. Only from this
  computer's panel; from another device the page only shows. Fixing a broken or deleted
  note no longer needs git.
- **Closing a task's session works every time, and says what it did.** Claude retires
  a background session left unused for hours: it stops running but stays in Claude
  Desktop and claude.ai. `session list` called it active, `session stop` failed once
  Claude's background service had stopped too, and closing the task waited 30 minutes
  for an answer that never came. Now `session list` says **retired**, `session stop`
  starts Claude's service for a moment to take the session out (or says how to do it
  later), and closing the task closes a retired session at once. `session stop
  --when-idle` also stops sessions of earlier versions, and with no session it says
  "nothing to close" instead of saying nothing; when closing a task starts it, what it
  did goes to `state/session-stop.log`.

## 0.17.1-beta

Released 2026-11-10.

- **Every page of the panel says what it is.** A few words at the top of each page,
  with a link to its part of the guide, in your language. Pages that had none (Tasks,
  a task, a note, Search, Health) have them now. You can hide them on one page or on
  all, and the panel remembers it on this computer.
- **Going back to a version of the configuration asks first.** On the Versions page,
  **Go back to this one** now opens to say what it does, warns when the guard's lists
  would change (with what each would gain or lose), and needs a box ticked. The page
  also says that the names of your secrets are there, never their values, and that the
  versions never go in the backup.
- **Sessions that were cut off are dealt with from Today.** Each one now shows on Today
  with **Keep it in the notes** and **Mark as reviewed**. The Sessions page leaves the
  menu, which was long for a newcomer; Health links to it.
- **A Backup page in the panel.** For every space: whether its notes have a copy
  elsewhere (a space with none says so in red), the last copies, and **Back up now**. A
  space with none sets one up in two clicks, on your GitHub (with `gh`) or in a private
  repository you already have. Restoring stays a command, for a new computer, and the
  page shows it. From another device the page only shows.
- **Closing a task removes the worktrees its sessions made.** A new MCP tool,
  `task_worktree`, makes a task's git worktree in any repository of the space, or notes
  one made some other way; closing the task removes each one with nothing not committed
  or not pushed, and says why any stays. The common rules ask every session to use it.
  Before, a session that made worktrees by hand left them behind.
- **Every session knows it can open a session for a task.** On Linux, the start of
  each session now tells Claude about `augmented-brain session start` and how to use
  it. Before, only the command's help said so, and a session once told its user there
  was no way to do it.
- **A task's session starts on its own and closes completely.** `session start` now
  opens a background Claude session (`claude --bg`) whose first message has it read
  its task and get going: before, it waited until someone wrote to it. `session stop`
  also takes it out of Claude Desktop and claude.ai, and works from any folder. Two
  tasks still never share a folder. Sessions opened by an earlier version are still
  listed and closed.
- **A task's session tells the session that asked for it when it stops.** Its first
  message carries that session's address (from Claude Code), so it reports in one line
  when the task is closed or parked, or waits on you.
- **Closing a task closes its session.** When the task has a session of its own,
  `task_close` has it closed once it has finished its answer (`session stop
  --when-idle`, in the background), and taken out of Claude Desktop and claude.ai,
  with its worktree when nothing would be lost.
- **A task with no repository opens in the space's folder.** Before, `session start`
  asked for a folder of its own, trusted by Claude, which a newcomer has no way to
  find. Several such tasks may share it.
- **A bare repository given with `--folder` gets the task's own worktree.** Before, it
  reused one of the repository's worktrees, and the session could open on another
  branch's code.

## 0.17.0-beta

Released 2026-11-10.

- **augmented brain is now beta.** It is used every day and has been tried on Ubuntu,
  macOS and Windows 11, so it leaves the alpha stage: the README, the
  disclaimer, the website, the guides and the installer now say beta. Nothing else
  changes: what you have keeps working, as before. An installed 0.16 sees 0.17.0-beta
  as newer and offers the update as usual.

## 0.16.4-alpha

Released 2026-11-10.

- **The CI's browser check no longer hangs.** It installed WebKit's system packages
  from Ubuntu's mirror on every run, and four runs on 9 November ran out of time when
  the mirror slowed to a trickle. It now runs inside Playwright's own image, which
  already has them. Nothing changes in the program.

## 0.16.3-alpha

Released 2026-11-09.

- **The README says what a newcomer meets.** A test with someone new (an agent that only
  had the README, on clean Ubuntu and Windows machines) tripped on four things, now said
  where they happen: installing git (missing on a fresh Ubuntu) and the desktop app on
  Linux ask for your password, or on Windows for permission; the spaces take the
  language you choose (`trabajo` in Spanish); the command works in a new terminal, where
  `augmented-brain doctor` checks the install; and the first questions come after you
  write something. Windows PowerShell gets its own form of `cd … && claude`.
- **Known limitations: what a newcomer meets that is not ours.** Claude Code's own
  first-run screens (Enter alone on "trust this folder" exits), the red PowerShell hook
  error on Windows, and installing on Windows over ssh, where the installer cannot ask.
- **The installer only takes a real yes.** Its questions took any answer but "n" for a
  yes, so a stray word (a newcomer typed `cls` to see the screen) opened Claude. Now
  Enter, yes or no; anything else asks again. Installing Git and Claude Code on Windows,
  and Claude Code on Linux and macOS, shows dots while it works instead of a minute of
  silence. And when it changed the PATH, its last message says that the terminal you
  used does not find `augmented-brain` yet, also when it suggests the desktop app.
- **`doctor`** no longer says "1 versions kept".
- **A first session in plain words.** The first-session interview greeted a newcomer
  as "bienvenido" before knowing who they were, and asked a freelance designer about
  "AWS profiles that reach production". Now it greets without assuming a gender, keeps to
  words the user has used, and fits question 4's examples to the person (a live website,
  a client's files; AWS and servers only to someone technical). `doctor`'s backup hint
  says what a private repository is and that Claude helps create it.
- **A syntax check is no longer stopped.** `sh -n install.sh` (also `bash -n`, `-o
  noexec`) only reads the script to check how it is written and runs none of it, but the
  guard checked the script as if it ran, and stopped it when the same command had just
  written it or the script held something it refuses. Now it lets it through. `+n`
  after `-n`, or an interactive shell (`-i`), still counts as running the script.
- **Arithmetic no longer asks.** `echo $(( $(date +%s)-1 ))`, the usual way to time
  something, asked the user: the guard read `$((…))` as a command inside a subshell and
  took the expression for a program it could not know. Now only the commands inside it
  (`$(date +%s)`) are checked, and stopped when they should be. `systemd-run --version`
  and `--help`, which start nothing, are no longer taken for becoming root.
- **A net under false positives.** A corpus of everyday commands (reading git, `sh -n`,
  `go test`, `gh pr checks`, pipelines, heredocs, loops), in the shapes of real
  sessions, goes through the guard in every pull request: one that is stopped or asked
  about fails the tests. And the rehearsal before a release now reboots the Ubuntu VM
  after `remote install` and checks, with the real systemd, that the service comes back
  with its PATH and permission mode and that `session start` runs a session.
- **The installer keeps the promise too.** At the end of an install or an update it
  still said the program "may change"; it now says what the README says: it may have
  bugs, but what you have keeps working.

## 0.16.2-alpha

Released 2026-11-09.

- **A task of a bare repository gets a worktree of its own.** `session start` used to pick
  one of the repository's existing worktrees, so two tasks could fall in the same folder
  (Claude serves each folder once, so the second did not start) or a session could start
  on old code. It now fetches origin and makes `<repo>-<task>` on a new branch named
  after the task, from the default branch as it is now, and uses it again next time.
  `session stop` removes it with its branch when it holds nothing that is not on the
  remote, and keeps it otherwise. Sessions that end without `session stop` (a reboot, a
  session closed from Claude Desktop) are swept: the next `session start` and `doctor`
  remove their worktrees with nothing of their own, `task_close` removes the task's, and
  `doctor` warns about one that stays with work in it. Only the worktrees `session start`
  made are ever touched. Picking an existing worktree stays for `--folder` naming a bare
  repository.
- **`session start` picks the worktree of the default branch.** For a bare repository it
  took the first trusted worktree by name, and since 0.16.1 every worktree of a trusted
  repository counts as trusted, so a session could open on a leftover feature branch. It
  now prefers the worktree on the default branch, then one named after it
  (`site-main`). Its message and the guides no longer say the session starts by reading
  its task on its own: it waits for you, and reads its task first when you write.

## 0.16.1-alpha

Released 2026-11-09.

- **`session start` knows a trusted worktree.** Claude keeps its trust for a git
  worktree under the main repository, so after you accepted the question in a worktree
  `session start` still said the folder was not trusted and refused. It now looks there
  too.
- **A task whose repository is bare gets a session in one of its worktrees.**
  `session start` skipped a bare repository but then found no folder, so you had to
  give one with `--folder`. It now picks one of the repository's worktrees inside the
  space's work folders, one Claude already trusts first.

## 0.16.0-alpha

Released 2026-11-09.

- **Remote access on macOS and Windows.** `augmented-brain remote install`, `status` and
  `uninstall` now work there too: a LaunchAgent on macOS, a task of the Task Scheduler on
  Windows (no window, no administrator). On both it starts when you log in and again by
  itself if it stops; after a reboot, log in once. Its errors go to a log file that
  `status` names. `session start` stays Linux only.
- **A page of known limitations.** What augmented brain cannot do yet, or does only in
  part, each with what to do instead: only Claude Code, PowerShell on Windows, remote
  access, what is checked by hand on a Mac, and what the guard cannot see.
- **A remote session finds its programs after a reboot.** systemd does not read your
  shell's profile, so a session opened through `remote` found neither `claude` nor
  `augmented-brain`, and `session start` failed from it. `remote install` now writes a
  PATH into each service with the folders of both, `session start` passes it on, and both
  look for `claude` where its installer puts it when the PATH does not lead there.
  `remote status` and `doctor` say to run `remote install` again for a service written
  before. `session start` also skips a bare repository when it picks the task's folder:
  it has no files, and Claude does not trust it.
- **The guard sees a few more ways around it.** A script that a command writes and then
  runs in the same line (`echo '…' > x.sh && bash x.sh`) is checked as it will be, and
  when the guard cannot tell what was written into it (`curl -o x.sh … && sh x.sh`), it
  asks you. Your git aliases count as what they stand for (`git ci` for `git commit`),
  `env -S '…'` is read, and a container given the whole machine (`docker run
  --privileged`, `-v /:/host`, `--pid=host`) is treated as becoming root.
- **The website shows a week with it.** The demo on the home page is now one week:
  a task opened on Monday, a date to check that comes up on Tuesday in another session,
  an idea for the inbox and a push the guard stops on Wednesday, and the weekly review on
  Friday. Buttons jump to each day. The home page also tells about remote access, a
  session per task and the configuration's versions, with their commands, and no longer
  promises adapters for other assistants.

- **What you have keeps working.** The README, the DISCLAIMER and the website no longer
  say the format and commands may change: every change to how notes or settings are kept
  ships with the step that moves yours forward, and a renamed command, setting or tool
  keeps working under its old name.
- **With one space, saving a person no longer asks where.** The program already saved
  them in the only space, but the text Claude reads told it to always ask, so a first
  session asked "which spaces?" with only one and did not save the person. It now says
  that with one space there is nothing to ask.

## 0.15.0-alpha

Released 2026-11-08.

- **A session per task, from anywhere.** `augmented-brain session start <task>` opens a
  remote session of its own for a task: it shows up in Claude Desktop and claude.ai named
  after the task, in the task's folder, and starts by reading it. The session behind a
  space's door cannot open sessions itself, but it can run this, so from another computer
  you ask it "open me a session for the website task" and get one per task, also after a
  reboot, when only the doors come back. `session stop` and `session list` close and list
  them. The first time in a folder it tells you how to make Claude trust it: Claude's
  server only starts in a folder trusted on its own, and serves each folder once. Linux
  only, like `remote`.

## 0.14.0-alpha

Released 2026-11-08.

- **Extra words are said, never ignored.** `space`, `check`, `doctor`, `mcp`, `version` and
  `help` ignored what came after them, so a typo like `space ad x` just showed the space.
  Now they say what they did not expect, as every other command does. `panel` takes
  `-no-browser` with one dash too. On Windows, the installer's new messages about failed
  downloads showed broken accents; they no longer do.
- **An unreadable `guard.toml` asks before commits, pushes and merges.** The guides said
  so, but only `aws` and `ssh` asked: with `guard.toml` broken, a push to a repository
  Claude may push to went through, and the repositories nothing may touch were no longer
  protected. Now every commit, push or merge the other lists would let through asks you
  first, and says why, until the file is fixed.
- **Phases skipped are written down, and a closed task stays closed.** A task could jump
  phases, or be closed a second time, writing over the result it was closed with, and
  nothing said so. Now skipping phases is still allowed, and the journal names the phases
  skipped; closing a closed task is refused; and opening it again needs a reason
  (`reopen` in `task_update`), which goes to the journal. From the panel, choosing a new
  status for a closed task is that reason.
- **Missing backups are said out loud.** Nothing warned when a space had no backup.
  Now `doctor` says so for each space, with how to set it up; `update` says when it goes
  ahead with no copy to go back to; and `uninstall --all` warns that notes with no
  backup will be gone for good.
- **Merging from the repository's own folder works.** Claude Code drops a `cd` to the
  folder a session is already in before the guard sees the command, so `cd <repo> && gh
  pr merge` written there arrived as a bare `gh pr merge` and was stopped. The guard now
  also takes the repository named in the command itself (`-R owner/name`, or the pull
  request's address) when it is that folder's repository, and its message says so.
- **`remote install` without systemd changes nothing.** On a machine without systemd's
  user services (a container, WSL without systemd) it wrote the services and turned
  remote access on before systemd refused them, and `doctor` then reported it broken.
  Now it asks systemd first and stops there, saying why; and remote access is turned on
  only once systemd has taken the services.
- **The Guard page shows what changed, and warns where it matters.** When the guard's
  lists changed outside the page, it now shows what was added and removed, list by list,
  since the lists you last accepted, before you accept them. The list where Claude
  commits and pushes without asking carries a warning, and adding to it asks you to tick
  that you understand; removing an untouchable repository and adding a root exception
  say what they allow.
- **A keychain that cannot be asked is no longer "missing".** On Linux without a desktop
  session, `secret-tool` fails the same way as for a secret that is not there, so
  `restore` and the panel's Secrets page told you to store again secrets you had. Now a
  failure to ask is "could not check", on Linux and on macOS (a locked keychain), and
  `restore` names those secrets too, where before, with no keychain at all (Windows), it
  said nothing.
- **A backup no longer hangs on a network that half works.** git had no time limit, and
  a backup holds a lock while it runs: one left hanging (the one at the end of a
  session, in the background) made every later backup wait behind it, including the one
  `update` makes first. Now each git call of a backup or restore stops after 3 minutes
  without an answer (and over https, on a transfer slower than 1 KB a second for a
  minute), and a backup waits at most 10 minutes for another one; both say why.
- **Going back works when the configuration is broken.** `config history` and `config
  restore` refused to run when `config.toml` did not load, which is exactly when going
  back to a version helps. Now they work on the files as they are, and `doctor` points
  to them when the configuration is broken.
- **An install stopped half way can be run again.** `init` wrote the configuration
  after the first space and then added the second: stopped in between (Ctrl+C, a folder
  it could not make), it left a configuration without `personal`, and the installer,
  seeing a configuration, never ran `init` again. Now `init` makes every space's folders
  first and writes the configuration once, at the end.
- **An update whose download fails no longer looks done.** `augmented-brain update` piped
  the installer from GitHub straight into `sh`: when GitHub answered 504, as it did for
  over an hour on 8 November, `sh` got an empty script, the update "worked" and nothing
  had changed. Now it downloads the installer first, checks how it ended, and checks the
  program is the new version afterwards; if not, it says so and ends with an error. On
  Windows the installer now reports its failure to `update` too.
- **Downloads try again, also through GitHub's API.** The installers (`install.sh`,
  `install.ps1`) and `update` try each file three times, and when GitHub's usual address
  fails they take the same file through its API, which kept working on 8 November. If
  everything fails they say so, in your language, and install nothing.

## 0.13.0-alpha

Released 2026-11-08.

- **Your configuration keeps its last 10 versions.** A list of the guard emptied by
  mistake, a space pointing to the wrong folder: until now there was no way back. Every
  change of `config.toml`, `guard.toml`, `remote.toml`, `session.toml`, the profile or the
  names of your secrets is now kept as a version, with who made it (the panel, a command,
  the assistant and its tool, or by hand), only on this computer and never in the backup.
  The panel's new Versions page shows what each one changed and goes back to one (only
  from this computer: from another device it only shows); so do
  `augmented-brain config history` and `config restore`. Going back keeps the current
  configuration as a version first, and refuses a version the program could not read. A
  command never accepts new guard lists for you: the guard asks until you do. `doctor`
  says how many versions are kept.
- **A shorter session start.** Every session started with about 60,000 characters of
  memory; measured by taking pieces out one at a time and running the same requests, most
  of them changed nothing in the first answer. Now each live task in the start's index
  comes without its summary, and with its next step (its first 120 characters) only when
  it waits on you: Claude reads a task before working on it. The shared references and
  lessons are listed by title and file, without their summaries. `MEMORY.md` keeps every
  detail.

## 0.12.0-alpha

Released 2026-11-08.

- **"forgotten?" only on a repository with pending work.** The panel's Repositories page
  marked a clean repository "forgotten?" just because the tasks that name it were all
  parked. Now, as on Today, it is marked only when it holds changes not committed or
  commits not pushed.
- **Sessions hear what other sessions changed.** A session reads the task index once, when
  it starts; another session, the panel or a hand edit may close a task a minute later.
  Now, before Claude answers each message, augmented brain tells it which tasks changed
  outside its session since it last looked (status, next step, who it waits on; and any
  change in a task it is working on), so it reads them again instead of acting on what it
  read hours ago. A journal entry of another session in a task it never touched is not
  news, and its own writes never are. The same note comes at the end of any augmented
  brain tool's answer, so a long piece of work hears it too. It runs in under a
  millisecond, through a new hook (`user-prompt-submit`); a program older than the plugin
  just says nothing.
- **Send something to another space.** `capture` takes `space`: what was decided in one
  space but must be done in another goes to that space's inbox, marked with where it came
  from, and comes up at the start of every session there until it is done. A new tool,
  `capture_done`, takes one capture out of the inbox without waiting for the weekly
  review. The panel's inbox shows where each one came from.
- **Closing a task says which watch dates it keeps.** They still come up at the start of
  the session on their day, as before; now `task_close` says so.
- **`panel remote` no longer stops the panel you have open.** The running panel starts
  answering through your Tailscale network too, with the same key, so the pages open in
  the browser keep working. If publishing fails, it goes back to local only.
- **A remote access service that fails says why.** Its errors now go to the system
  journal (`journalctl --user -u augmented-brain-remote-<space>`). `remote status` and
  `doctor` tell you to run `augmented-brain remote install` again to update a service
  installed by an earlier version.

## 0.11.0-alpha

Released 2026-11-07.

- **Open the panel from your phone or another computer.** A new command,
  `augmented-brain panel remote`, publishes the panel in your own
  [Tailscale](https://tailscale.com) network and prints its address and a QR code. Only
  your devices reach it, and only your Tailscale account gets in; the address stops
  working when the panel stops, and a leftover one is taken back at the next start.
  Tailscale is optional: without it, the command explains what it is and how to get it,
  and so it does for HTTPS in your network and, on Linux, letting your user publish.
  `doctor` gets a line about it once you have used it. New guide: "The panel from
  another device". The QR code uses `rsc.io/qr` (BSD license).
- **The panel on a phone.** On a narrow screen the sidebar becomes one bar on top (logo,
  a menu button that says where you are, the space) and the sections open below it as a
  plain list, with the search; the profile and the footer go to the end of the page.
  The footer says when the panel is also open on your Tailscale devices.
## 0.10.0-alpha

Released 2026-11-07.

- **Remote access starts where you work.** Each space's session now starts in the space's
  first work folder, not in a new folder under the configuration. If you already use
  Claude there, it trusts it, so there is nothing more to accept. A new setting,
  `folders` in `remote.toml`, picks another folder inside the space. Only a space with no
  work folder still gets one of its own. Run `augmented-brain remote install` again to
  move a service you already have.
- **`update` does the checks around it.** Before a new version, it runs the health check
  (and shows what it finds) and backs up every space that has a backup; if a backup
  fails, nothing is updated (`--force` updates anyway). After, it says how many open Claude sessions still run the
  old version and which task each space has active, to park before restarting Claude.

## 0.9.0-alpha

Released 2026-11-06.

- **A name and a personality for your assistant.** The panel's Profile page has a new
  part, **Your assistant**, and `profile_save` two new fields, `assistant_name` and
  `assistant_personality`. Every session starts with them; they change how Claude talks,
  never the rules, the guard or its honesty. With a name, the emails Claude writes in
  your name are signed with it and, under it, a line the program adds and you cannot
  remove, such as "Ana's assistant (AI, Claude)". They live in a new section of
  `profile.md`; earlier versions keep it as text. Also, `profile_save` no longer wipes
  what it is not sent: a part left out keeps what it has, and an empty one is cleared.
- **Rules and knowledge can be changed and retired, and are never duplicated without
  anyone knowing.** Saving a rule, reference, lesson or area with a title that already
  exists used to create a second copy ("-2"), and both loaded; now nothing is created,
  and Claude is told the id. `rule_new` and `knowledge_new` take an `id` to change one
  in place; the version it replaces is kept in `archive/`. A new tool, `note_retire`,
  moves a rule, reference or lesson to `archive/` with the reason and what replaces it:
  it stays, but a retired rule no longer loads. No new field in the notes, so earlier
  versions read them as before.
- **Know which model is answering.** A new setting, `preferred_model` in a new file,
  `session.toml` (or the panel's Guard page): when a session answers with another model,
  Claude tells you, at the start if Claude Code names it and otherwise after an answer
  (in a new session, the second at the latest), once per session and model. It never changes the model itself. Empty by
  default: nothing is checked.
- **After a compaction, the summary is not taken as fact.** Claude used to be asked to
  write the journal from the summary; now it is asked to check before stating anything
  from before, and to tell you the session was compacted and suggest parking the task
  and going on in a new session.

- **Repositories Claude never touches, and a guard Claude cannot loosen.** A new list of
  **untouchable repositories** (`untouchable_repos` in a new file, `guard.toml`) where the
  guard denies commit, push and merging a pull request, whatever `git_allowed_repos` and
  `git_ask_repos` say. A new **Guard** page in the panel shows and changes every list the
  guard answers to (untouchable and allowed repositories, root exceptions with their end
  date, production profiles, untouchable machines), keeping your comments; removing an
  untouchable repository or adding a root exception asks you to type its name. To keep
  those lists yours: the guard now also asks before a write to `guard.toml`,
  `secrets.toml` and `remote.toml`; denies Claude reading `panel.json` (the panel's key)
  or calling the panel from a command; and keeps a seal of its lists, so when something
  other than the panel or a session start changed them, a push or merge that the lists
  allow asks first, and `doctor` and the Guard page say so until you accept them. The
  questions about a push or a merge now name the branch and where the remote leads (no
  passwords). `doctor` warns about an unreadable `guard.toml` and about a repository that
  is both untouchable and in another list. A session still running an earlier version
  does not enforce `guard.toml`. A script Claude writes in line can still edit the
  files: it is noticed, not prevented. The plugin's hook now also listens to `Read` and
  `Grep`: update the plugin with the program.

- **People shared across spaces, from the panel.** The same person can now be in several
  spaces (a client who is also a friend). Each space keeps its own note for them, with
  the same id, and it holds only what you want visible there; nothing changes on disk, so
  earlier versions read the files as before. The panel lists each person once, with a
  badge per space, and their page is one form: a role and other names per space, and a
  list of details with a box per space. A detail you do not tick for a space is never
  written there, so something personal never reaches a work file or its backup. Claude's
  `person_save` takes a new `spaces` parameter: with several spaces and none given, it
  asks you where to save instead of guessing. Two notes written today with the same id
  (one paragraph each) show as one card, each paragraph a detail ticked in its own space.

- **A warning before a window runs out of memory.** Now and then the process that holds
  a window's Claude sessions (VS Code's extension host, or Claude itself in a terminal)
  grows without stopping, from its usual half a gigabyte to many, until the system kills
  it with every session of the window. After each answer, the Stop hook now looks at its
  memory, and past 2 GB (then 4 and 8 GB, once each) Claude tells you and suggests
  parking the task and going on in a new window. `doctor` says it too, for every Claude
  window on the computer. On Linux and macOS; it adds a fraction of a millisecond. See
  Troubleshooting, "The window uses a lot of memory".
- The landing pages show the panel: a picture of its Today page, what it is for, and a
  link to its guide page, with a Panel link in the top menu.
- The panel says "1 aparcada" in Spanish, in the singular, instead of "1 aparcadas".
- **Remote access: Claude reachable from another computer, also after a reboot.**
  `augmented-brain remote install` sets up a service per space that keeps
  `claude remote-control` running, so you can connect to this machine from claude.ai/code
  or Claude Desktop (choose the environment `<computer name>-<space>`). Each session starts
  in a folder of its space, so its notes and rules are that space's. `remote status` and
  `doctor` show whether it runs, whether Claude trusts its folder, and whether "linger"
  is on; `remote uninstall` takes it away and deletes nothing else. Claude must trust each
  folder (open it once with `claude`) and linger needs one command as administrator:
  `install` prints both, it does not run them. Settings in the new `remote.toml`. Linux
  only for now. See Remote access.

## 0.8.0-alpha

Released 2026-11-04.

- **`space add` no longer takes `--language`.** It never did anything: the language is
  the same for every space on a computer, the `language` setting that `init` sets. A
  script that still passes it now gets an error that names it.
- A documentation site, in English and Spanish, in the website's `guide/`: twenty pages
  from getting started to troubleshooting, with a reference of every command and flag,
  MCP tool and `config.toml` setting. Tests fail when one of them is missing from the
  reference in either language.
- One list of commands, the one the help and those tests are checked against. Every
  command works as before.
- **Every message is in your language.** "Unknown settings", "written by a newer
  augmented brain", what is wrong in `config.toml`, a failed move to a new format, the
  `hook` command's usage and the MCP server's errors now come in Spanish too: in the
  configuration's `language`, or the system's while the configuration cannot be read.
- The guard's reasons, which Claude reads and you see when it asks, all follow your
  `language`. Until now only those about merging a pull request did, next to the others
  in English.
- The guard no longer takes the text given to `gh pr merge --body` or `--subject` (or
  `glab mr merge --message`) as part of the command: an address or `--admin` written
  there does not change what the merge is. `gh pr merge` written in a pull request's
  body or a commit message was already not a merge; tests now say so.
- **A hole in the guard closed.** `gh pr merge -m` (`--merge`, which takes no value)
  made the guard skip the next word, so `gh pr merge -m <another repository's pull
  request>` went through in a repository where merging is allowed. It is now refused,
  like any merge that names a repository other than the one it runs in.
- CI stops an older run when a newer push to the same branch or pull request
  supersedes it (never on `main`), and runs the installer's checks two at a time, so
  the macOS jobs no longer wait for runners until they are cancelled.
- `compat-check.sh` runs the earlier releases in English, so the words it looks for
  are there whatever the language of the machine.

## 0.7.0-alpha

Released 2026-11-03.

- The panel says "1 file changed" and "1 commit not pushed", not "1 files", "1 commits".
- The panel has a **Repositories** page, a map of your repositories: every git
  repository in the work folders of each space (up to three folders down), grouped by
  space, those with work pending first. Each one says its branch, where it pushes (with
  no user or password), files changed, commits not pushed, the open tasks that name it,
  and one line of yours on what it is, which you write right there.
- That line is kept in the space's notes, in the reference "Repositories"
  (`knowledge/ref-repositories.md`), so the backup keeps it, and earlier releases read
  it as any other reference. The new MCP tool `repo_describe` lets Claude write it too.
- Today's Repositories block links to the new page, and asks to describe a repository
  that has no line yet.
- The panel has a **Help** page: links to the website's questions and guides, in your
  language, and links to report a problem or propose an idea on GitHub. The problem
  link runs the health check when you click it and opens GitHub's form with the
  version, the system and that check filled in, with what is personal taken out (home
  folder, notes and work folders, e-mails, space names, untouchable machines,
  production AWS profiles, lines that look like a secret). Nothing is sent until you
  press GitHub's submit button, though the text reaches GitHub in the address as soon
  as the form opens.
- Issue forms on GitHub for a problem and an idea, and a link to the questions.
- **Merging a pull request is a push.** `gh pr merge`, `glab mr merge` and `glab mr accept`
  now go by `git_allowed_repos` and `git_ask_repos`, as `git push` does: let through in a
  repository Claude may push to, asked in one where it asks, blocked elsewhere as
  before. With `main` protected so that changes only go in through a pull request, a
  merge no longer has to be run by hand. gh and glab have no `-C`, so the command names
  the repository with a `cd`, as a push does with `git -C`: `cd <repo> && gh pr merge
  12`. The session's folder alone does not count. A repository
  the command names (`-R`, `GH_REPO`, the pull request's address) must be one of that
  repository's remotes, or the merge is blocked. `gh pr merge --admin`, which skips the
  branch's protection, always asks. Releases (`gh release create`, …) stay for you.

## 0.6.1-alpha

Released 2026-11-03.

From 0.6.0 on a real machine: programs before 0.6.0 refuse a setting of `config.toml`
they do not know, and 0.6.0 added one. Every Claude session open since before the
update kept the old program, and lost its augmented brain tools.

- The names of the secrets live in a file of their own, `secrets.toml`, which earlier
  programs never read. The list 0.6.0 left in `config.toml` moves there at the start of
  a session or with `check`, and a backup carries the file, as it does the profile.
- A configuration from a newer program, with settings this one does not know, says to
  update, not that the file is wrong.
- `doctor` and the start of a session say how many Claude sessions on this computer
  still run an earlier augmented brain, and to close and open them again. Each session's
  server notes its version when it starts; on Linux, those of earlier versions are found
  too.
- A restore names the secrets the new machine's keychain does not have, each with the
  command that stores it; the panel's Secrets page shows that command too.
- CI checks that the last two releases read what this one writes (`compat-check.sh`):
  their `doctor`, `check`, MCP tools, session start and restore. It would have caught
  0.6.0.

Also, from the review of 0.6.0:

- In a table, a `|` inside code no longer splits the cell.
- The panel can empty a person's notes and an area's summary.

## 0.6.0-alpha

Released 2026-11-03.

The panel, grown into the place to see everything at a glance:

- **Today** is its first page: every space at once, with what is due in the next week,
  the tasks waiting on you, the inbox, whether the weekly review is due and the sessions
  cut off. Below, **Repositories**: those your open tasks name in `repos:` that hold
  changes not committed or commits not pushed, marked "forgotten?" when every task that
  names them is parked or left for some day. Read only; git runs without taking locks and never asks for a
  password.
- **Notes read as text**, not as raw Markdown: headings, lists, tables, code and links,
  and `[[links]]` lead to their note. A task shows its essentials first, the rest folded,
  and its journal newest first. A small renderer of its own, with no dependency: a note
  can never put HTML or scripts in the page.
- **Tasks by area**, with filters (waiting on you, under way, parked, untouched, by
  person) and a mark on those untouched for two weeks.
- **People** can be added and changed in the panel. A person's page shows the tasks that
  wait on them and those that link to them; "waits on" leads to the person.
- **Search** every note of the space and the shared ones, with or without accents.
- **The weekly review** on a page: each capture becomes a task, a reference or a lesson,
  or is dropped; each untouched task goes on or waits for some day. What was decided is
  kept with the review. The inbox has a "make it a task" button.
- A note's page says when the note changed elsewhere since it opened. Areas can be
  renamed and their summary changed.
- **Secrets**, by name only: the secrets you keep in the system keychain, what each is
  for, which tasks name it, and, when you ask, whether this computer's keychain has it.
  Never a value: one is refused. Kept in `config.toml` as `secrets`; Claude adds one with
  the new MCP tool `secret_name_save`; the list travels in the default space's backup,
  and a restore brings it to a machine that has none.

Also:

- `import` names the knowledge that lists people (one section each), so each can be
  saved as a person: in the first real migration, people stayed as a reference and the
  panel's People was empty.

From the rehearsal of 0.5.10 on the real machine:

- The parts of the session start count only those with something: "part 3 of 6",
  never "of 9" with three that do not come, which made the assistant think some were
  lost.
- `doctor` warns when Claude Code keeps a memory of its own for a work folder (its auto
  memory, a second memory next to the space's), and says how to turn it off.
- The restore's note on repositories says some only asked before committing.

## 0.5.10-alpha

Released 2026-11-03.

From the rehearsal of 0.5.9:

- A restore leaves the notes readable by you only, whatever the umask; `check` does the
  same for notes written before 0.5.9, and `doctor` says when other users can read them.
- A restore names the repositories the backup let Claude commit in, which it does not
  bring back, and where to add them again; `doctor` and the first session say so too.

## 0.5.9-alpha

Released 2026-11-03.

From a deep review of the code (architecture, the guard, and the rest of the security):

- **The guard catches more mistakes.** A redirection next to a word (`git push>log`,
  `2>/dev/null sudo …`), `powershell -Command` or `cmd /c` from Bash, `git.exe`, a short
  script run with `bash x.sh` or `./x.sh`, `ssh root@…`, `rm -rf "$DIR/"*` with a variable
  that may be empty, `rm -rf ~/..`, `find ~ -delete`, `git submodule foreach`, `git
  subtree push`, `gh api` calls that merge, `ssh -J`, `bash -o pipefail -c …`, and
  sudoedit, runuser, systemd-run and machinectl. A command nested too deep asks instead
  of passing, and so does a program given as a variable. More token shapes are secrets
  (Stripe, npm, Hugging Face, Docker Hub, SendGrid, more GitLab ones).
- **Claude cannot switch the guard off on its own.** Changing augmented brain's
  configuration or Claude Code's settings asks you. `safety_save` only adds to the
  guard's lists, and `backup_setup` never moves an existing backup elsewhere.
- **A changed backup cannot reach outside the notes.** A backup holding a symbolic link
  is neither restored nor merged; a restore no longer brings the repositories Claude may
  commit in; git runs with only the usual protocols, and an address is never an option.
- Notes are written readable by you only (0600, in 0700 folders), and every replaced
  file goes through a temporary file with a name nobody can guess.
- install.sh runs nothing when its download is cut short; on Windows, Git's installer
  needs its checksum and the signature of Git for Windows itself.
- The session start: a plugin newer than the program gets an empty part instead of an
  error; part 1 waits for the notes at most 3 seconds, and the others wait for it;
  `check` takes the notes lock; a root exception that cannot be read is named; today's
  and the coming days' reminders keep their room among many overdue ones.

## 0.5.8-alpha

Released 2026-11-03.

From the rehearsal of 0.5.7:

- **A closed task's watch dates still come up.** Closing is when the checks for later
  are written down, so they are mentioned at the start of a session until they are
  marked done, as those of a live task. The task itself is no longer pending.
- "1 day overdue", not "1 days".

## 0.5.7-alpha

Released 2026-11-03.

From the rehearsal of 0.5.6:

- **Your rules arrive whole.** In real sessions the assistant never opened a rule's
  file before acting, so the details in it were lost (a `glab api` without its
  `--hostname`). The start now has five parts for your rules, nine parts in all, and
  each rule goes whole, without its `History` section. Only a rule that does not fit
  goes in one line.
- **No secrets through the tools.** A task, a journal entry, a capture, the profile or
  any note the assistant writes is refused when it holds a secret (an AWS key, a token),
  so it never lands in a note that the backup would then refuse. A note that already
  held one, as an import may leave, can still be changed, to fix it.
- **Two machines, the same task.** When each machine only added journal entries (dated
  lines) to the same note, the backup joins them by itself, in date order. Anything
  else still stops it, with a message in your language.
- The session start also mentions what is due in the next 7 days, after what is overdue.
- A task keeps up: `task_new` takes the goal and "done when" into the Definition, and
  parking or closing updates "where it is" and "next" in the Essentials.
- `doctor`: an ended root exception says to remove it or give it a new date.
- Windows: removing the panel's file waits for whoever is reading it.

## 0.5.6-alpha

Released 2026-11-03.

- **Temporary root exceptions.** `root_ask = ["db01 until 2026-10-31"]` in the
  configuration: until that date, a command as root on that machine (here, or through
  `ssh`) asks you instead of being refused, and the session start names the exceptions
  in force. The date is required; after it, root is refused again by itself, and
  `doctor` says the entry can go.
- The install guide explains Claude Code's question about trusting a folder, and broken
  accents in Windows PowerShell 5.1 (the text is right; PowerShell misreads it).

## 0.5.5-alpha

Released 2026-11-03.

From the rehearsal of 0.5.4:

- **The start of a session holds the index of a big space.** Claude Code cuts each hook's
  text at 10,000 characters, so a space with 43 tasks started with none of them listed.
  The plugin now loads the start in five parts, each within that limit: the heading with
  your profile and the common rules, the index in three parts, and your rules.
- A second `import` no longer counts the inbox captures again: when the inbox has them
  all, the plan lists `inbox.md` with the files the space already has.
- A backup stopped by a secret says so in your language, the kind of secret included.
- The guard's reason for `rm -rf /c/` (or `/`) speaks of a whole disk, not of the home
  folder.

## 0.5.4-alpha

Released 2026-11-03.

From the rehearsal on Windows 11:

- **The guard reads paths as Git Bash writes them** (`/c/Users/ana`), which is where Claude
  runs commands on Windows. `rm -rf /c/Users/<you>`, `rm -rf "$USERPROFILE"` and
  `rm -rf /c/` were let through; now they are never run. And `git -C /c/…` of a repository
  in `git_allowed_repos` counts as that repository.
- **The Windows installer gets Git even when winget is broken** (an old App Installer whose
  source does not open): it downloads Git for Windows from its own release, checks its
  checksum and signature, and installs it.
- A second `import` no longer copies the archive and `docs/` again: the files the space
  already has are listed as such, and the message about files that appeared after the plan
  is in your language.

## 0.5.3-alpha

Released 2026-11-02.

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
