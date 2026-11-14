# Commands

[Leer en español](commands.es.md)

Every command of the program, with every option. Day to day you do not need them: you
talk to Claude. They are for setting up, checking and moving to a new machine.

Run them in a terminal. Most act on **the space of the folder you are in**. Outside every
space's folder, they use the default space. When there is no default space (you made a
single space with `init --space`, for example), they say so and do nothing: run them
from a folder of that space.

```bash
cd ~/augmented-brain/work
augmented-brain check
```

Options go before or after the other arguments, with one dash or two (`--all` or
`-all`). A command that gets an argument it does not take says so, and does nothing.
Each section gives the command's syntax: `<…>` is something you fill in, `[…]` can be
left out, and `|` separates choices.

Some commands are started for you, not by you: Claude Code starts `mcp` and `hook`,
`panel` starts `panel serve`, and on Linux a timer runs `remote watch`. They are here so
you know what they are.

Inside Claude Code there is one slash command: `/brain:panel` opens the panel, as
`augmented-brain panel` does.

## I want to…

| I want to… | Command |
|---|---|
| Set everything up the first time | [`init`](#augmented-brain-init) |
| Know which space this folder is in | [`space`](#augmented-brain-space) |
| Add a space | [`space add`](#augmented-brain-space-add) |
| Check my notes after editing them by hand | [`check`](#augmented-brain-check) |
| Find out what is wrong | [`doctor`](#augmented-brain-doctor) |
| Bring in a memory Claude Code already wrote | [`import`](#augmented-brain-import) |
| Back up a space now | [`backup`](#augmented-brain-backup) |
| Connect a space to its backup repository | [`backup setup`](#augmented-brain-backup-setup) |
| Bring a space back on a new computer | [`restore`](#augmented-brain-restore) |
| See and edit everything in the browser | [`panel`](#augmented-brain-panel) |
| Open the panel from my phone | [`panel remote`](#augmented-brain-panel-remote) |
| Reach Claude from my phone | [`remote install`](#augmented-brain-remote-install) |
| See how remote access stands | [`remote status`](#augmented-brain-remote-status) |
| Open a session of its own for a task | [`session start`](#augmented-brain-session-start) |
| Restart sessions after an update | [`session restart`](#augmented-brain-session-restart) |
| Be told when a pull request is done | [`watch pr`](#augmented-brain-watch-pr) |
| Go back to an earlier configuration | [`config restore`](#augmented-brain-config-restore) |
| Clean up what Claude Code left behind | [`clean`](#augmented-brain-clean) |
| Update to the latest version | [`update`](#augmented-brain-update) |
| Remove augmented brain | [`uninstall`](#augmented-brain-uninstall) |

What each exit code means is at the end, in [Exit codes](#exit-codes).

## augmented-brain init

Sets everything up: the spaces, the configuration and the Claude Code plugin. The
installer runs it for you the first time.

Syntax: `augmented-brain init [--custom] [--language en|es] [--space <name>]
[--work <folder>] [--memory <folder>] [--yes] [--no-plugin]`

```bash
augmented-brain init
```

With no options it asks nothing. It takes the language from the system and creates two
spaces in `~/augmented-brain`: `work` and `personal` (`trabajo` and `personal` in
Spanish), with `personal` as the default space. Then it installs the plugin. Without
Claude Code, it prints the two commands that install the plugin later.

If there is a configuration already, it stops and changes nothing. To add a space, use
`space add`. It writes the configuration last: if it stops half way, run it again.

- `--custom`: ask for the language, the space and its folders, to make one space your way.
- `--language`: the language, `en` or `es`. Without it, the system's (or `AB_LANGUAGE`).
- `--space`: the name of the one space to create (lowercase letters, digits and dashes,
  starting with a letter).
- `--work`: its work folder. Default: `~/augmented-brain/<space>`.
- `--memory`: its notes folder. Default: `~/augmented-brain/<space>/notes`, also when
  `--work` names another folder.
- `--yes`: do not ask; take the options given and the defaults.
- `--no-plugin`: do not install the Claude Code plugin.

`--custom`, `--space`, `--work` or `--memory` make a single space instead of the two, and
leave no default space: commands run outside its folders say so and do nothing. With a
second space (`space add`), there is a default space again.

```bash
augmented-brain init --space client --work ~/clients/acme --language en
```

That makes the space `client`, with its work folder in `~/clients/acme` and its notes in
`~/augmented-brain/client/notes`.

## augmented-brain space

Says which space this folder belongs to, and where its notes are.

Syntax: `augmented-brain space`

```bash
augmented-brain space
```

Outside every space's folder it names the default space. With no default space, it says
so, and lists the spaces.

## augmented-brain space add

Adds another space: a client, a side project, a volunteer job.

Syntax: `augmented-brain space add [<name>] [--space <name>] [--work <folder>]
[--memory <folder>] [--yes]`

```bash
augmented-brain space add client
```

The name goes as an argument (first or last) or with `--space`. It asks for the work
folder and the notes folder, suggesting `~/augmented-brain/<name>` and
`~/augmented-brain/<name>/notes`. If there was no default space yet, it sets one:
`personal` if a space has that name, or else the first of the spaces that were there
before, in alphabetical order.

- `--space`: the name of the new space.
- `--work`: its work folder.
- `--memory`: its notes folder. Default: `~/augmented-brain/<name>/notes`, also when
  `--work` names another folder.
- `--yes`: do not ask; take the options given and the defaults.

It has no `--language`. The language is the same for every space on this computer: the
`language` setting, which `init` sets. See [Configuration](configuration.md).

```bash
augmented-brain space add client --work ~/clients/acme --yes
```

See [Spaces](spaces.md).

## augmented-brain check

Checks the notes of this folder's space, and rebuilds its index when they are all valid.

Syntax: `augmented-brain check`

```bash
augmented-brain check
```

It also:

- moves the notes to a new format when a new version needs it, after a full copy;
- checks the shared notes, and rebuilds their index;
- names every file that holds a secret (the backup refuses it until the secret is out);
- names the repositories notes point at that are not on this computer;
- makes the notes readable by you only, if other users could read them;
- moves the list of secret names into `secrets.toml`, from `config.toml` where 0.6.0 kept it;
- moves the oldest journal entries of a task past 24 KB to `archive/journals/`, as the
  tools do when they write it (see [A long journal](tasks.md#a-long-journal)).

Run it after editing notes by hand. It ends with an error when a note is not valid.

## augmented-brain doctor

Says what is missing, and how to fix it.

Syntax: `augmented-brain doctor`

```bash
augmented-brain doctor
```

Each line starts with a mark: `✓` fine, `·` for your information, `!` a warning, `✗`
something that fails. Under a line that needs work, `→` says what to do. It checks:

- **Your setup:** the configuration, the profile and which space this folder is in;
  whether there is a default space; how many versions of the configuration are kept,
  and whether it changed outside them (by hand).
- **Your notes:** each space's notes folder, its notes and its index, and whether it has
  a backup; tasks too long to read in one go, which `check` shortens; how much of every
  session start your rules take, with the three longest (they go whole to every session,
  so a shorter rule costs less each time); notes that other users of the computer could
  read; a second memory that Claude Code keeps of its own for your folders.
- **Claude Code and the guard:** Claude Code, git and the plugin, and whether Claude Code
  is recent enough to stop everything when the guard fails (2.1.295 or later); the
  guard, for real, with a command that must be stopped; the guard's lists, the root
  exceptions in `root_ask` and those that have ended; the lists changed since you last
  accepted them, a repository both untouchable and allowed, and lists a restore left
  empty.
- **Sessions:** Claude sessions still running an older version after an update; a
  Claude window using a lot of memory (2 GB or more); background sessions that keep
  their process while nobody uses them (ended or archived, or opened by no task and idle
  for over 12 hours: archiving one in Claude Desktop does not stop it), each with the
  `claude stop` that ends it.
- **Leftovers:** worktrees Claude Code made whose session is over, and the scratch
  folders of sessions that ended (see `clean`).
- **From another device:** remote access, one line per space, once you set it up with
  `remote install` (a warning when its service is not running, Claude does not trust its
  folder yet, or, on Linux, linger is off); the panel from another device, once you have
  used it (Tailscale, HTTPS, and an address left published).

It ends with an error when something fails. When the configuration cannot be read, it
points to `config history` and `config restore`. See
[Troubleshooting](troubleshooting.md).

## augmented-brain import

Brings in a memory Claude Code already wrote. Without `--apply` it only shows what it
would do.

Syntax: `augmented-brain import <folder> [--map <file>] [--repos-base <folder>] [--apply]`

With a single space, everything goes there. With two or more (as `init` makes), it needs
`--map` to know where each note goes. A map with one line, `* = work`, sends everything
to `work`:

```bash
augmented-brain import ~/.claude/projects/<folder>/memory --map ~/import-map.txt
```

- `--apply`: write the notes. Without it, nothing is written.
- `--map`: a file that says which space each area or note goes to: lines `area-x =
  space`, `task-y = space`, `lesson-z = shared`, and `* = space` for the rest.
- `--repos-base`: the folder the `repos:` paths of the source are relative to.

It never writes over a note that exists, and only reads the source. See
[Bring in a memory](import.md).

## augmented-brain backup

Saves the latest changes of this folder's space to its private repository. It also runs
by itself when a Claude session ends.

Syntax: `augmented-brain backup [--all | --space <name>]`

```bash
augmented-brain backup
```

- `--all`: back up every space, not only this folder's.
- `--space`: back up this space, wherever you are: `--space work`.
- `--auto`: how the end of a session starts it, in the background. Quiet, and it never
  waits for a password. You do not need it.

It stops without pushing when a note holds a secret, or when another machine changed the
same note in another way, and says which. See [Backup and restore](backup.md).

## augmented-brain backup setup

Connects this folder's space to a private git repository. It does not copy anything yet:
run `augmented-brain backup` afterwards for the first copy (it says so).

Syntax: `augmented-brain backup setup <address>`

```bash
augmented-brain backup setup git@gitlab.com:lucia/brain-work.git
```

To create the repository in your GitHub account for you, ask Claude: "set up the backup".

## augmented-brain restore

Brings a space back from its backup, on a new machine.

Syntax: `augmented-brain restore <address> [--memory <folder>] [--work <folder>]`

```bash
augmented-brain restore https://github.com/lucia/augmented-brain-work.git
```

It replaces the empty space of the same name with yours. It leaves alone a space that
already has notes. It installs the Claude Code plugin if it is missing.

With the space come, when this machine has none of its own yet: the profile (only the
default space's backup carries it), the shared notes, and the names of your secrets
(never their values). Then it names what it did not bring: the guard's lists, the
repositories where Claude may commit, and the secrets this machine's keychain lacks,
each with the command to store it. When it cannot ask the keychain (there is none, or it
is locked), it names the secrets it could not check.

A folder the backup names is used only inside your home: not the home itself, not a
hidden folder at its top (`~/.config`), not the system's (`~/Library`, `~/AppData`) and
not inside a `.git`. Instead of such a folder it uses `~/augmented-brain/<space>`, and
says so.

- `--memory`: where to put the space's notes. Default: the same place inside the work
  folder as before, or `~/augmented-brain/<space>` when they were outside it.
- `--work`: the work folder that selects the space. Default: the one saved in the backup.

## augmented-brain panel

Opens the panel in your browser: today at a glance, and every page of your spaces to see
and edit.

Syntax: `augmented-brain panel [--no-browser]`

```bash
augmented-brain panel
```

It starts the panel in the background, or finds the one running, and opens the browser
on Today, with the space of this folder. It returns at once. The panel stops by itself
after 30 minutes without use. The address it prints opens the panel once, within 2
minutes; run by Claude, with the browser open, it prints none.

- `--no-browser`: only print the address, do not open the browser. `AB_NO_BROWSER=1` does
  the same.

See [The panel](panel.md).

## augmented-brain panel remote

Opens the panel for your other devices (another computer, your phone), through
[Tailscale](https://tailscale.com), a free app that joins your devices in a private
network of their own.

Syntax: `augmented-brain panel remote`

```bash
augmented-brain panel remote
```

It starts the panel and publishes it in your Tailscale network only. If the panel is
already open, that same panel answers through the network too, so the pages you have
open keep working. It prints the address and a QR code to open it from the phone.

- Only your own devices, signed in with the account this computer belongs to, can open
  it.
- The address opens it once, within 10 minutes. The pages stop working when the panel
  stops.
- Whatever is missing (Tailscale itself, HTTPS in your network, letting your user
  publish) it explains, with the step to take. It never installs or changes anything of
  Tailscale's for you.

See [The panel from another device](panel-remote.md).

## augmented-brain panel stop

Stops the panel now.

Syntax: `augmented-brain panel stop`

```bash
augmented-brain panel stop
```

It says whether a panel was running. The link at the bottom of every panel page does the
same.

## augmented-brain panel serve

The panel itself, running in the background. `augmented-brain panel` starts it; you never
run it by hand. It stops on its own after 30 minutes without use.

## augmented-brain remote

Keeps Claude reachable from another computer or your phone, also after a reboot. It is
three commands, `install`, `uninstall` and `status`, each with its own section below.
It uses a `systemd` user service on Linux, a LaunchAgent on macOS and a task of the Task
Scheduler on Windows. On Linux a fourth one, `watch`, is run by a timer, not by you.
Without a subcommand, or with another, `remote` prints its usage.
See [Remote access](remote.md).

## augmented-brain remote install

Sets remote access up, one service per space.

Syntax: `augmented-brain remote install`

```bash
augmented-brain remote install
```

For each space named in `remote.toml` it writes a service that runs `claude
remote-control`, and starts it. By default that is the default space, or the only space
when there is just one.

- **Where:** a `systemd` user service on Linux, a LaunchAgent in `~/Library/LaunchAgents`
  on macOS, a task `\augmented-brain\remote-<space>` of the Task Scheduler on Windows (it
  runs while you are logged in, with no window).
- **In which folder:** the space's first work folder, or the one `folders` names in
  `remote.toml`. Only a space with no work folder gets a folder of its own,
  `remote/<space>` in the configuration folder, which becomes its work folder.
- **What it needs:** `claude` (on your PATH, or where its installer puts it) and a folder
  that exists. The service gets a PATH with the folders of `claude` and
  `augmented-brain`, so its sessions find both after a reboot.
- **On Linux** it also installs the watch: a `systemd` timer,
  `augmented-brain-watch-remote`, that runs `augmented-brain remote watch` every two
  minutes (see below).

You can run it again at any time: it changes only what is different. It then says what
is still missing, and does none of it for you:

- Claude must **trust each folder**. It prints `cd <folder> && claude` for each one that
  is not trusted yet (on Windows, `cd "<folder>"; claude`): run it once and accept the
  question.
- On Linux, **linger** must be on for the services to start after a reboot with nobody
  logged in. It prints the command, `sudo loginctl enable-linger <you>`, for you to run.
  macOS and Windows have no linger: the service starts when you log in.

Without systemd's user services (a container, or WSL without systemd), launchctl on macOS
or the Task Scheduler on Windows, it says so and changes nothing. The settings are in
`remote.toml` (see [Configuration](configuration.md)).

## augmented-brain remote uninstall

Stops remote access and removes it.

Syntax: `augmented-brain remote uninstall`

```bash
augmented-brain remote uninstall
```

It stops and removes the services `remote install` wrote (on Linux, the watch first), and
sets `enabled = false` in `remote.toml`. The folders and their entries in `work_folders`
stay: it never deletes anything else.

## augmented-brain remote status

Says how remote access stands, and changes nothing.

Syntax: `augmented-brain remote status`

```bash
augmented-brain remote status
```

For each space: the session's name (what you pick in Claude Desktop or claude.ai),
whether the service is running and starts by itself, and whether Claude trusts its
folder. Then, on Linux, whether the watch (see `remote watch` below) is running and how
long ago it last looked, and whether linger is on; on macOS and Windows, that it starts
when you log in. Under each problem, `→` says what to do.

## augmented-brain remote watch

Brings a service back when the server archived its session. Linux only. Its timer runs
it every two minutes; you do not need to.

Syntax: `augmented-brain remote watch`

```bash
augmented-brain remote watch
```

What it does on each look:

- **A service with no session.** When the server archives a session (for example, you
  archived it in claude.ai), its Claude process ends and the service goes on running
  with no session: nobody can connect, and systemd sees nothing wrong. After two looks
  in a row like that, the watch restarts the service.
- **Still no session.** Then Claude was trying to reopen the archived session. The watch
  renames Claude's record of it (`bridge-pointer.json` becomes
  `bridge-pointer.json.archived`, in the folder's project in `~/.claude/projects`) and
  restarts the service again, so a new session starts: about ten minutes in all. The
  conversation of the archived session does not come back.
- **Task sessions kept safe.** Stopping a service stops everything started inside it.
  A task session started from a remote session can start Claude's background service
  there, which runs every task session. So the watch moves out of each service whatever
  its server did not start itself into a group of its own,
  `augmented-brain-kept-<number>`. `remote install` and `remote uninstall` do the same
  before they restart or stop a service; when it cannot be done, they stop and say so,
  and the watch does not restart that service.

What it does goes to the system journal
(`journalctl --user -u augmented-brain-watch-remote`).

It also wakes the task sessions that Claude's usage limit stopped. When your plan's
usage runs out, Claude Code refuses the turn and says when the limit resets. A session
you have open in a terminal goes on by itself at that hour; a task session runs in the
background and does not: it stays there, doing nothing, until someone writes to it. The
watch reads that hour from the session's last answer and, once it has passed, stops the
session and resumes it with a message that tells it to read its task and go on. It
comes back as the same session, with its conversation, in Claude Desktop and claude.ai.
It wakes one session on each look, so they come back two minutes apart and do not spend
the new usage all at once. Only sessions opened with `augmented-brain session start`
are woken, never any other. You are told twice, in the next session you use: when the
wait begins, with the hour, and when the last one is back. Each woken task gets a line
in its journal.

## augmented-brain session

Opens a session of its own for one task, so that from another computer you can work on
several tasks at once, each in its own session. It is four commands, `start`, `stop`,
`restart` and `list`, each with its own section below. `start`, `stop` and `list` work
on Linux only for now: they have been tried only there. `restart` works on Linux and
macOS. Without a subcommand, or with another, `session` prints its usage. Why and how to
use them: [A session per task](task-sessions.md).

## augmented-brain session start

Opens a session for a task, in the task's folder.

Syntax: `augmented-brain session start <task> [--folder <folder>]`

```bash
augmented-brain session start task-new-website
augmented-brain session start task-new-website --folder ~/projects/website
```

It starts a background Claude session of its own for the task (`claude --bg
--remote-control`), called `<name_prefix> · <task title>` (`name_prefix` is set in
`remote.toml`; by default, the computer's name). Within a minute it shows up in Claude
Desktop and claude.ai. It starts on its own: its first message has it read its task,
tell you where it stands, and get on with the next step.

- `--folder`: the folder to open the session in. It must lie inside the work folders of
  the task's space: a session elsewhere would not find its task.

**Which folder.** Without `--folder`, the first of the task's repositories found in the
space's work folders. A task that names no repository opens in the space's first work
folder, which several such tasks may share.

**A bare repository** (one with only git's history and no files to work on) gets a git
worktree for the task: a second working copy of the repository, in a folder next to it.
For the task `task-new-website` of `site` it is `site-new-website`, on a branch named
after the task. A new branch starts from origin's default branch as it is now: it
fetches first, and if origin cannot be reached it starts from what this computer last
knew of it, and says so. If an earlier session of the task left its branch, the worktree
picks it up again. Asked again, it uses the same worktree. The same goes for a bare
repository given with `--folder`. See
[Worktrees, for a bare repository](task-sessions.md#worktrees-for-a-bare-repository).

**Rules:**

- Run it from a folder of the task's space. The remote session you reach from another
  computer can run it for you ("open me a session for the website task").
- The folder must be trusted on its own: trusting a folder above it is not enough, so
  the first time the command says to open Claude once in that folder and accept the
  question. A git worktree is the exception: trusting its main repository is enough.
- One task per folder: a folder another task's session uses is refused. Give it another
  with `--folder` (a git worktree, for example).
- When Claude Code is not logged in, or its login has expired, nothing is opened. It
  waits up to a minute for the session's first answer; when that answer is an error
  (the login ran out, the service is overloaded), the session is closed. Either way it
  says what to do, such as `claude auth login`.
- If Claude stopped the task's session (it does that to one left unused for hours, and
  to every one when its background service restarts), `session start` brings it back,
  with its whole conversation and no new first message. Only if Claude refuses does a
  new session open and replace it.
- On a system where task sessions do not run yet, it says so before it makes anything.
- The session closes with `session stop`, or when the computer restarts. Sessions are
  not brought back after a reboot: ask for them again.

## augmented-brain session stop

Closes a task's session and takes it out of Claude Desktop and claude.ai. It works
from any folder.

Syntax: `augmented-brain session stop <task> [--when-idle]`

```bash
augmented-brain session stop task-new-website
```

- `--when-idle`: wait until the session has finished its answer, then close it. The
  command returns at once, and a process in the background waits, up to 30 minutes; a
  session waiting for a permission counts as answering, and one still answering by then
  stays open. Closing the task runs it this way by itself, so a session can close its own
  task without cutting its answer off. What it did goes to `state/session-stop.log` in
  the configuration folder.

**The task's worktree** (the one `session start` made, see above) goes too, with its
branch, as long as nothing would be lost. It stays, and the command says so, when it
has:

- changes not committed (files your git settings hide count too);
- commits not pushed, on the task's branch, even when the worktree was left on another
  commit;
- files git ignores that no build makes again (a `.env`, a key; `node_modules`, `build`,
  `dist` and the like do not count).

Sessions that end without `stop` (a reboot, a session closed from Claude Desktop) leave
their worktree. The next `session start` removes those with nothing of their own (not
one made in the last ten minutes), closing the task removes its own, and `doctor` only
says how many would go.

**Other cases:**

- Claude cannot tell whether the session still runs (its list cannot be read): it says
  so and changes nothing; the session stays on record, and its worktree too.
- A session Claude retired (one left unused for hours) is closed too. If Claude still
  cannot take it out of its lists, it says so and gives the command to do it later.
- A task with no session gets "nothing to close", never a silent success.

## augmented-brain session restart

Restarts a Claude session so it runs the version of augmented brain installed now, with
the same conversation. It works from any folder. Use it after an update, or right after
parking a task, so that its session starts afresh.

Syntax: `augmented-brain session restart <session> [--when-idle]`, or
`augmented-brain session restart --old` (with no session and no `--when-idle`).

```bash
augmented-brain session restart task-new-website
augmented-brain session restart --old
```

- `<session>`: its task (when `session start` opened it), its id, the first eight
  characters of its session id, or its name; `claude agents` lists them.
- `--old`: restart every idle session still running an earlier augmented brain, one at
  a time, and say what happened to each. `update` runs it by itself. Here only a task's
  own session (opened by `session start`) gets a new process: any other background
  session may have been archived in Claude Desktop, which Claude is not told, so only
  its augmented brain server stops.
- `--when-idle`: wait until the session has finished its answer, then restart it. This
  is how a session restarts itself after parking its task: the command returns at once,
  and a process in the background waits, up to 30 minutes. What it did goes to
  `state/session-restart.log` in the configuration folder.

Each kind of session restarts its own way:

- **A task's session** (`claude --bg`): Claude gives it a new process with the same
  conversation. One you archived, or that ended, is never brought back: only its
  augmented brain server stops, and the new version starts if you ever open it.
- **A session of the remote access** (`remote install`): its service restarts, and the
  same session comes back in claude.ai and Claude Desktop in about ten seconds. The
  service holds every session it serves, so it restarts only when all of them are idle.
  Anything else inside the service (Claude's background service, above all) is moved
  out first, so it does not stop with it. Linux only.
- **A session of Claude Desktop or a terminal:** its window stays open, and only its
  augmented brain server stops; Claude Code starts the new version the next time the
  session uses it. Linux and macOS. On Windows it is left as it is: close it and open it
  again yourself.

A session that is answering, or waiting for a permission, is never restarted: it is
looked at again right before. Nor is the session the command runs in: that one needs
`--when-idle`. One that fails leaves the others going.

## augmented-brain session list

Lists the task sessions open on this computer: the task, whether it runs, and its
folder.

Syntax: `augmented-brain session list`

```bash
augmented-brain session list
```

- **retired** means Claude stopped it, after hours unused or when its background service
  restarted. claude.ai shows it as disconnected, and `session start` with its task
  brings it back with its whole conversation.
- **failed** means its last answer was an error, such as an expired login: it does
  nothing until someone writes to it. The line gives the error and what to do.
- **waiting** means Claude's usage limit stopped it. The line gives the hour the limit
  resets, which is when the watch wakes it (see `remote watch` above).

## augmented-brain watch

Follows something that takes a while and says how it ended, so a session can leave it
running and get on with other work. For now it follows one thing, a pull request: see
`watch pr` below. Without a subcommand, `watch` prints its usage.

## augmented-brain watch pr

Follows a pull request on GitHub until there is something to tell, then ends on its own
and says how it went, in one line.

Syntax: `augmented-brain watch pr <number> [--task <task>] [--repo <owner/name>]
[--max <time>] [--every <time>]`

```bash
augmented-brain watch pr 104 --task task-my-change
```

It ends when the pull request is merged or closed, conflicts with its base branch (no
CI runs then), a required check fails, every required check has finished, or `--max`
passes.

- `--task`: write that line in the task's journal, and what happened as its next step.
  Every other session hears of it before its next answer, and the panel's Today shows
  it. The watching is then done by a process of its own, which goes on if the session
  that started it closes or restarts. Asked again for the same pull request and task,
  it waits for the one already running. The task must be in this folder's space.
- `--repo`: the repository, as `owner/name`. Default: the one of this folder.
- `--max`: the longest it watches. Default: `3h`.
- `--every`: how often it looks. Default: `30s`; `10s` at least.

`--max` and `--every` take a number with its unit: `s` seconds, `m` minutes, `h` hours,
as in `90m` or `1h30m`.

**What the line says.** The checks that are not required do not decide anything, but
the line names those still running or failed (five at most, each cut at 60 characters).
Their names go only in the journal, never in the next step, which every session reads
at the start: whoever writes the workflows chooses those names.

**When it gives up.** If GitHub lists no required check after 5 minutes, or `gh` fails
three times in a row or gives an answer it does not understand, it stops and says what
`gh` said.

It needs `gh` with your GitHub login. The assistant runs it in the background; its last
line asks it to tell you straight away. It ends with 0 when the checks passed or the
pull request was merged, and 1 otherwise.

## augmented-brain config

The versions of your configuration: two commands, `history` and `restore`, each with its
own section below. Without a subcommand, `config` prints its usage. See
[Configuration versions](config-history.md).

## augmented-brain config history

Lists the last versions of your configuration, newest first, and changes nothing.

Syntax: `augmented-brain config history [<number>]`

```bash
augmented-brain config history
augmented-brain config history 3
```

A version is a copy of `config.toml`, `guard.toml`, `remote.toml`, `session.toml`,
`profile.md` and `secrets.toml` as they were at one moment, with who changed them: the
panel, a command, the assistant (and which tool), going back to a version, or by hand.
The last 10 are kept, only on this computer. With a number, it shows what that version
changed from the one before it, line by line. It works even when `config.toml` is
broken.

## augmented-brain config restore

Goes back to one version of the configuration.

Syntax: `augmented-brain config restore <number>`

```bash
augmented-brain config restore 3
```

It shows what changes and writes it. The configuration you had is kept as a version
first, so going back can be undone the same way. A version that would leave a file the
program cannot read is refused, and nothing changes. It works even when `config.toml` is
broken, like `config history`.

If the guard's lists end up different from the ones you accepted, the guard asks before
every commit, push or merge it used to allow. It keeps asking until you accept the lists
in the panel's Guard page, or start a new session. A command can be run by the
assistant, so it never accepts them for you. Going back from the panel's Versions page
does accept them.

## augmented-brain clean

Removes what working with Claude Code leaves behind and nobody cleans up. It shows the
list first, and removes it only if you say yes.

Syntax: `augmented-brain clean [--yes]`

```bash
augmented-brain clean
```

It looks in every folder Claude Code was opened in, and in its scratch folder:

- **Worktrees Claude Code made** (in `.claude/worktrees`) whose session is over. One goes
  only when it has no change not committed, no commit that is not on the remote, and no
  file git ignores that no build makes again (a `.env`, a key): it is removed with
  `git worktree remove`, never by deleting the folder. Its branch goes with it when
  Claude Code named it (`worktree-<name>`); a branch of yours stays. A worktree with work
  of its own is only named, so you look at it.
- **Worktrees git remembers whose folder is gone**: `git worktree prune` forgets them.
  Their branches stay.
- **Scratch folders of sessions that ended** (`/tmp/claude-<number>` on Linux), untouched
  for over a day. Those of running sessions stay, and so does anything else there.

Before removing each thing it looks again, so a session that started meanwhile keeps its
folders. `doctor` says when there is something to clean.

- `--yes`: do not ask.

## augmented-brain update

Updates augmented brain and its Claude Code plugin.

Syntax: `augmented-brain update [--plugin] [--force] [--no-restart]`

```bash
augmented-brain update
```

It first asks GitHub for the latest version, once: if GitHub does not answer, it says so
and stops. If there is no newer version, it only updates the plugin. If there is one,
it runs the published installer, which replaces the program and then updates the
plugin. Around it, it does what you would otherwise do by hand:

- **Before:** the health check of `doctor` (what fails is shown, so it is not taken for a
  problem of the new version), and a backup of every space that has one. If a backup
  fails, it stops and updates nothing: that copy is the way back. With no backup at all,
  it says it goes ahead without a copy to go back to.
- **The installer:** if downloading it fails (three tries, also through GitHub's API),
  `update` ends with an error and nothing has changed. The installer tries the new
  program before it replaces the old one: if it does not start, the old one stays. If
  the program is not the new version afterwards, `update` says so and ends with an
  error.
- **After:** the idle Claude sessions still running the old version are restarted, one
  at a time, as `session restart --old` does: those of Claude Desktop stay open and only
  their augmented brain restarts. For those it cannot restart (one answering, or on
  Windows one of Claude Desktop), it says which task is active in each space, so you ask
  its session to park it, and then you close and open them.

Options:

- `--plugin`: update only the Claude Code plugin. The installer uses it.
- `--force`: update even when the backup before it fails.
- `--no-restart`: restart no session after updating; it only says which still run the
  old version.
- `--from <address>`: for tests: install the release at this address instead of
  GitHub's latest. It must be `https://`, `file://`, or `http://` to this computer or to
  the host of a test VM (`10.0.2.2`).

`update` ignores `AB_RELEASE_URL` and `AB_MARKETPLACE`, and says so: a program could set
them so that the update comes from somewhere else. Of the other `AB_` variables, it only
passes `AB_CONFIG_DIR` and `AB_INSTALL_DIR` on to the installer.

## augmented-brain uninstall

Removes the program and the Claude Code plugin. Your notes stay.

Syntax: `augmented-brain uninstall [--config | --all] [--yes]`

```bash
augmented-brain uninstall
```

It shows what it will remove and asks first. The remote access services and the
sessions opened for tasks go first; if one cannot be stopped, nothing else is removed.

- `--config`: also remove the configuration and the profile.
- `--all`: start from scratch: also the configuration, the profile and every space's
  notes. It shows each notes folder with how many notes it has, and warns about those
  with no backup, which will be gone for good. Only what augmented brain keeps in a notes
  folder goes; a folder that holds a work folder, or has no notes, stays whole. Your
  backups are not touched.
- `--yes`: do not ask.

See [Installing](install.md#uninstalling).

## augmented-brain mcp

Runs the MCP server: the tools Claude uses to keep your tasks and notes. **Claude Code
starts it**, through the plugin; you do not.

It talks over its input and output. It finds the space again on every call, from the
folder Claude works in. The tools are in [MCP tools](mcp-tools.md).

## augmented-brain hook

Answers a Claude Code hook: a call Claude Code makes at a given moment. **Claude Code
runs it**, through the plugin; you do not. It reads what Claude Code sends on its input.

Syntax: `augmented-brain hook <event> [<part>]`

```bash
augmented-brain hook session-start 1
```

A problem goes to its error output and the hook still ends with 0, so it never stops
Claude. The one exception is the guard (`pre-tool-use`): when it cannot give its answer,
it ends with 2, and Claude Code stops the command.

The events:

- `session-start`: a session starts. It hands Claude the profile, the rules, the index of
  tasks and what is due. It comes in up to nine parts, `session-start 1` to
  `session-start 9`, because Claude Code cuts each hook's text at 10,000 characters. Only
  the parts with something in them are sent.
- `pre-tool-use`: before every command and file change. This is the
  [guard](guard.md).
- `user-prompt-submit`: before Claude answers a message. It tells Claude which tasks
  changed outside its session since it last looked (another session closed one, the
  panel changed its next step), so it reads them again instead of trusting what it read
  earlier. It never holds up the message.
- `stop`: after each answer. It takes milliseconds, and:
  - reminds Claude to write in the journal of its session's task, once, after 20
    minutes without writing;
  - with `preferred_model` set (see [Configuration](configuration.md)), has Claude tell
    you when another model answered, once per session and model;
  - shows you, once, a change of the guard's lists or something the assistant saved for
    every session;
  - on Linux and macOS, looks at how much memory the window uses (VS Code's extension
    host, or Claude itself in a terminal), and has Claude warn you when it passes 2 GB,
    then 4 and 8 GB, once each (a fraction of a millisecond). See
    [Troubleshooting](troubleshooting.md#the-window-uses-a-lot-of-memory).
- `post-compact`: after Claude Code compacts a long conversation. It keeps the summary in
  `archive/sessions/` and points the task's journal at it.
- `session-end`: the session ends. It notes that it ended well, and starts the backup in
  the background.

## augmented-brain version

Prints the version of the program.

Syntax: `augmented-brain version`

```bash
augmented-brain version
```

`--version` and `-v` do the same.

## augmented-brain help

Prints the list of commands, in your language.

Syntax: `augmented-brain help`

```bash
augmented-brain help
```

`--help` and `-h` do the same. Running `augmented-brain` with nothing prints the list
too, as an error (exit code 2).

## Exit codes

Every command ends with one of these codes. A script can rely on them: they stay the
same from one version to the next (see [Compatibility](compatibility.md)). The messages
may change.

| Code | Meaning |
|---|---|
| `0` | It did what was asked. |
| `1` | Something failed, or did not end well. The message says what. |
| `2` | It was not run: the command line is wrong. |

A wrong command line is an unknown command or option, an argument missing or too many,
or a value it cannot use (`--language fr`).

A few commands give their codes a meaning of their own:

- `check` and `doctor`: `1` when a note is not valid, or something fails.
- `watch pr`: `0` only when the checks passed or the pull request was merged; `1` for
  everything else (closed, a conflict, a failed check, `--max` reached).
- `hook`: `0` even when something fails inside, so Claude is never stopped by it. The
  exception is the guard (`pre-tool-use`), which ends with `2` to stop the command when
  it cannot give its answer.
