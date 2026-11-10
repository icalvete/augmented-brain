# Commands

[Leer en español](commands.es.md)

Every command of the program, with every option. Day to day you do not need them: you
talk to Claude. They are for setting up, checking and moving to a new machine.

Run them in a terminal. Most act on **the space of the folder you are in**. Outside every
space's folder, they use the default space.

```bash
cd ~/augmented-brain/work
augmented-brain check
```

Options go before or after the other arguments, with one dash or two (`--all` or
`-all`). A command that gets an argument it does not take says so, and does nothing.

Three commands are started by Claude Code, not by you: `mcp`, `hook` and `panel serve`.
They are here so you know what they are.

Inside Claude Code there is one slash command: `/brain:panel` opens the panel, as
`augmented-brain panel` does.

## augmented-brain init

Sets everything up: the spaces, the configuration and the Claude Code plugin. The
installer runs it for you the first time.

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
- `--memory`: its notes folder. Default: `notes` inside the work folder.
- `--yes`: do not ask; take the options given and the defaults.
- `--no-plugin`: do not install the Claude Code plugin.

`--space`, `--work` or `--memory` make a single space instead of the two.

```bash
augmented-brain init --space client --work ~/clients/acme --language en
```

## augmented-brain space

Says which space this folder belongs to, and where its notes are.

```bash
augmented-brain space
```

Outside every space's folder it names the default space. With no default space, it says
so, and lists the spaces. It takes no options.

## augmented-brain space add

Adds another space: a client, a side project, a volunteer job.

```bash
augmented-brain space add client
```

The name goes as an argument (first or last) or with `--space`. It asks for the work
folder and the notes folder, suggesting `~/augmented-brain/<name>` and its `notes`
folder. If there was no default space yet, it sets one: `personal` if a space has that
name, or else the space that was there first.

- `--space`: the name of the new space.
- `--work`: its work folder.
- `--memory`: its notes folder.
- `--yes`: do not ask; take the options given and the defaults.

It has no `--language`. The language is the same for every space on this computer: the
`language` setting, which `init` sets. See [Configuration](configuration.md).

```bash
augmented-brain space add client --work ~/clients/acme --yes
```

See [Spaces](spaces.md).

## augmented-brain check

Checks the notes of this folder's space, and rebuilds its index when they are all valid.

```bash
augmented-brain check
```

It also:

- moves the notes to a new format when a new version needs it, after a full copy;
- checks the shared notes, and rebuilds their index;
- names every file that holds a secret (the backup refuses it until the secret is out);
- names the repositories notes point at that are not on this computer;
- makes the notes readable by you only, if other users could read them;
- moves the list of secret names into `secrets.toml`, from `config.toml` where 0.6.0 kept it.

Run it after editing notes by hand. It ends with an error when a note is not valid. It
takes no options.

## augmented-brain doctor

Says what is missing, and how to fix it.

```bash
augmented-brain doctor
```

Each line starts with a mark: `✓` fine, `·` for your information, `!` a warning, `✗`
something that fails. Under a line that needs work, `→` says what to do. It checks:

- the configuration, the profile and which space this folder is in;
- each space's notes folder, its notes and its index, and whether it has a backup;
- Claude Code, git and the plugin;
- the guard, for real, with a command that must be stopped;
- the guard's lists, the root exceptions in `root_ask`, and those that have ended;
- Claude sessions still running an older version after an update;
- a second memory that Claude Code keeps of its own for your folders;
- notes that other users of the computer could read;
- whether there is a default space, and whether a restore left the guard's lists empty;
- remote access, one line per space, when you set it up with `remote install`: a warning
  when its service is not running, Claude does not trust its folder yet, or (on Linux)
  linger is off;
- how many versions of the configuration are kept, and whether it changed outside them
  (by hand);
- the guard's lists changed since you last accepted them, and a repository both
  untouchable and allowed;
- a Claude window using a lot of memory (2 GB or more);
- the panel from another device, once you have used it: Tailscale, HTTPS, and an address
  left published.

It ends with an error when something fails. When the configuration cannot be read, it
points to `config history` and `config restore`. It takes no options. See
[Troubleshooting](troubleshooting.md).

## augmented-brain import

Brings in a memory Claude Code already wrote. Without `--apply` it only shows what it
would do.

```bash
augmented-brain import ~/.claude/projects/<folder>/memory
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

```bash
augmented-brain backup setup git@gitlab.com:lucia/brain-work.git
```

It takes one argument, the repository's address, and no options. To create the
repository in your GitHub account for you, ask Claude: "set up the backup".

## augmented-brain restore

Brings a space back from its backup, on a new machine.

```bash
augmented-brain restore https://github.com/lucia/augmented-brain-work.git
```

It replaces the empty space of the same name with yours. It leaves alone a space that
already has notes. It installs the Claude Code plugin if it is missing. Then it names
what it did not bring: the guard's lists, the repositories where Claude may commit, and
the secrets this machine's keychain lacks, each with the command to store it. When it
cannot ask the keychain (there is none, or it is locked), it names the secrets it could
not check.

- `--memory`: where to put the space's notes. Default: where they were, inside the work
  folder.
- `--work`: the work folder that selects the space. Default: the one saved in the backup.

## augmented-brain panel

Opens the panel in your browser: today at a glance, and every page of your spaces to see
and edit.

```bash
augmented-brain panel
```

It starts the panel in the background, or finds the one running, and opens the browser
on Today, with the space of this folder. It returns at once. The panel stops by itself
after 30 minutes without use.

- `--no-browser`: only print the address, do not open the browser. `AB_NO_BROWSER=1` does
  the same.

See [The panel](panel.md).

## augmented-brain panel remote

Opens the panel for your other devices (another computer, your phone), through
[Tailscale](https://tailscale.com), a free app that joins your devices in a private
network of their own.

```bash
augmented-brain panel remote
```

It starts the panel and publishes it in your Tailscale network only. If the panel is
already open, that same panel answers through the network too, so the pages you have
open keep working. It prints the address and a QR code to open it from the phone. Only
your own devices, signed in with the account this computer belongs to, can open it; the address stops working when the
panel stops. Whatever is missing (Tailscale itself, HTTPS in your network, letting your
user publish) it explains, with the step to take: it never installs or changes anything
of Tailscale's for you. See [The panel from another device](panel-remote.md).

## augmented-brain panel stop

Stops the panel now.

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
Scheduler on Windows. Without a subcommand, or with another, `remote` prints its usage.
See [Remote access](remote.md).

## augmented-brain remote install

Sets remote access up, one service per space.

```bash
augmented-brain remote install
```

For each space named in `remote.toml` (by default, the default space) it writes a service
that runs `claude remote-control` in the space's first work folder (or the one `folders`
names in `remote.toml`), and starts it: a `systemd` user service on Linux, a LaunchAgent
in `~/Library/LaunchAgents` on macOS, a task `\augmented-brain\remote-<space>` of the
Task Scheduler on Windows (it runs while you are logged in, with no window). Only a space with no work
folder gets a folder of its own, `remote/<space>` in the configuration folder, which
becomes its work folder. It needs `claude` (on your PATH, or where its installer puts it)
and a folder that exists. The service gets a PATH with the folders of `claude` and
`augmented-brain`, so its sessions find both after a reboot. You can run it again at any
time: it changes only what is different.

It then says what is still missing, and does none of it for you:

- Claude must **trust each folder**. It prints `cd <folder> && claude` for each one that
  is not trusted yet (on Windows, `cd "<folder>"; claude`): run it once and accept the
  question.
- On Linux, **linger** must be on for the services to start after a reboot with nobody
  logged in. It prints the command, `sudo loginctl enable-linger <you>`, for you to run.
  macOS and Windows have no linger: the service starts when you log in.

Without systemd's user services (a container, or WSL without systemd), launchctl on macOS
or the Task Scheduler on Windows, it says so and changes nothing. It takes no options. The settings are in `remote.toml` (see [Configuration](configuration.md)).

## augmented-brain remote uninstall

Stops remote access and removes it.

```bash
augmented-brain remote uninstall
```

It stops and removes the services `remote install` wrote, and sets `enabled = false` in
`remote.toml`. The folders and their entries in `work_folders` stay: it never deletes
anything else. It takes no options.

## augmented-brain remote status

Says how remote access stands, and changes nothing.

```bash
augmented-brain remote status
```

For each space: the session's name (what you pick in Claude Desktop or claude.ai), whether
the service is running and starts by itself, and whether Claude trusts its folder. Then,
on Linux, whether linger is on; on macOS and Windows, that it starts when you log in. Under each problem, `→` says what to do. It takes no options.

## augmented-brain session

Opens a remote session of its own for one task, so that from another computer you can
work on several tasks at once, each in its own session. It is three commands, `start`,
`stop` and `list`, each with its own section below. Linux only for now: it has
been tried only there. Without a subcommand, or with another, `session` prints its usage. See
[Remote access](remote.md#a-session-per-task).

## augmented-brain session start

Opens a session for a task, in the task's folder.

```bash
augmented-brain session start task-new-website
augmented-brain session start task-new-website --folder ~/projects/website
```

It starts a background Claude session of its own for the task (`claude --bg
--remote-control`), named after the task's title. Within a minute it shows up in Claude
Desktop and claude.ai. It starts on its own: its first message has it read its task,
tell you where it stands, and get on with the next step. When another Claude session
asked for it, the first message also has it tell that session, in one line, when it
stops: the task closed or parked, or waiting on you.
The folder is the first of the task's repositories found in the space's work folders,
or the one `--folder` gives.

A bare repository has no files to work on, so the task gets a git worktree of its own
next to it: `site-new-website` for the task `task-new-website` of `site`, on a new branch
named after the task, from the default branch of origin as it is now (it fetches first).
Two tasks of one repository never share a folder, and a session never starts on old
code. Asked again, it uses the same worktree. The same goes for a bare repository given
with `--folder`.

Run it from a folder of the task's space: the remote
session you reach from another computer can run it for you ("open me a session for
the website task"). Every session on Linux knows how.

Two rules keep it safe. The folder must be trusted on its own: trusting a folder above
it is not enough, so the first time the command says to open Claude once in that folder
and accept the question. Claude itself would not ask, and a repository's own settings
and hooks must not run unasked. A git worktree is the exception: Claude keeps the answer
under its main repository, so trusting either one is enough. And one task per folder:
two tasks in one folder would step on each other's changes, so a folder another task's
session uses is refused; give it another with `--folder` (a git worktree, for example).

The session closes with `session stop`, or when the computer restarts. Launched
sessions are not brought back after a reboot: ask for them again.

## augmented-brain session stop

Closes a task's session and takes it out of Claude Desktop and claude.ai. It works
from any folder.

```bash
augmented-brain session stop task-new-website
```

When `session start` made a worktree for the task, `stop` also removes it with its
branch, as long as nothing would be lost: with changes not committed, or commits not
pushed, it keeps it and says so. Closing the task closes its session too, once it has
finished its answer, worktree and all. Sessions that end without `stop` (a reboot, a
session closed from Claude Desktop) leave theirs: the next `session start` and `doctor`
remove those with nothing of their own, closing the task removes its own, and `doctor` warns
about any that stays with work in it.

- `--when-idle`: wait until the session has finished its answer, then close it. Closing
  the task runs it by itself in the background: the session closing its own task is
  often still answering, and stopping it there would cut that answer off. It waits up
  to 30 minutes.

## augmented-brain session list

Lists the task sessions open on this computer: the task, whether it runs, and its
folder.

```bash
augmented-brain session list
```

## augmented-brain config

The versions of your configuration: two commands, `history` and `restore`, each with its
own section below. Without a subcommand, `config` prints its usage. See
[Configuration versions](config-history.md).

## augmented-brain config history

Lists the last versions of your configuration, newest first, and changes nothing.

```bash
augmented-brain config history
augmented-brain config history 3
```

A version is a copy of `config.toml`, `guard.toml`, `remote.toml`, `session.toml`,
`profile.md` and `secrets.toml` as they were at one moment, with who changed them: the
panel, a command, the assistant (and which tool), going back to a version, or by hand.
The last 10 are kept, only on this computer. With a number, it shows what that version
changed from the one before it, line by line. It takes no options.

## augmented-brain config restore

Goes back to one version of the configuration.

```bash
augmented-brain config restore 3
```

It shows what changes and writes it. The configuration you had is kept as a version
first, so going back can be undone the same way. A version that would leave a file the
program cannot read is refused, and nothing changes. If the guard's lists end up
different from the ones you accepted, the guard asks before every commit, push or merge
it used to allow. It keeps asking until you accept the lists in the panel's Guard page,
or start a new session. A command can be run by the assistant, so it never accepts them
for you. Going back from
the panel's Versions page does accept them. It works even when `config.toml` is broken,
like `config history`. It takes no options.

## augmented-brain update

Updates augmented brain and its Claude Code plugin.

```bash
augmented-brain update
```

If there is a newer version, it runs the published installer, which replaces the program
and then updates the plugin. If not, it only updates the plugin. It first asks GitHub
for the latest version, once: if GitHub does not answer, it says so and stops. If the
download of the installer fails (three tries, also through GitHub's API) or the program is not the new version
afterwards, it says so, ends with an error and changes nothing. Around a new version it
does what you would otherwise do by hand:

- **Before:** the health check of `doctor` (what fails is shown, so it is not taken
  for a problem of the new version), and a backup of every space that has one. If a
  backup fails, it stops and updates nothing: that copy is the way back. With no backup
  at all, it says it goes ahead without a copy to go back to.
- **After:** how many open Claude sessions still run the old version, and which task is
  active in each space, so you ask its session to park it before restarting. Then close
  and open Claude, so the sessions use the new version.

- `--plugin`: update only the Claude Code plugin. The installer uses it.
- `--force`: update even when the backup before it fails.

## augmented-brain uninstall

Removes the program and the Claude Code plugin. Your notes stay.

```bash
augmented-brain uninstall
```

It shows what it will remove and asks first.

- `--config`: also remove the configuration and the profile.
- `--all`: start from scratch: also the configuration, the profile and every space's
  notes. It shows each notes folder with how many notes it has, and warns about those
  with no backup, which will be gone for good. Your backups are not touched.
- `--yes`: do not ask.

See [Installing](install.md#uninstalling).

## augmented-brain mcp

Runs the MCP server: the tools Claude uses to keep your tasks and notes. **Claude Code
starts it**, through the plugin; you do not.

It talks over its input and output. It finds the space again on every call, from the
folder Claude works in. The tools are in [MCP tools](mcp-tools.md). It takes no options.

## augmented-brain hook

Answers a Claude Code hook: a call Claude Code makes at a given moment. **Claude Code
runs it**, through the plugin; you do not. It reads what Claude Code sends on its input,
and never makes Claude fail: a problem goes to its error output.

```bash
augmented-brain hook session-start 1
```

The events:

- `session-start`: a session starts. It hands Claude the profile, the rules, the index of
  tasks and what is due. It comes in up to nine parts, `session-start 1` to
  `session-start 9`, because Claude Code cuts each hook's text at 10,000 characters. Only
  the parts with something in them are sent.
- `pre-tool-use`: before every command and file change. This is the
  [guard](guard.md). If it cannot run, Claude Code stops the command.
- `user-prompt-submit`: before Claude answers a message. It tells Claude which tasks
  changed outside its session since it last looked (another session closed one, the
  panel changed its next step), so it reads them again instead of trusting what it read
  earlier. It never holds up the message.
- `stop`: after each answer. It reminds Claude to write in the task's journal, once,
  after 20 minutes without writing. It also looks at how much memory the window uses
  (VS Code's extension host, or Claude itself in a terminal), and has Claude warn you
  when it passes 2 GB, then 4 and 8 GB, once each. On Linux and macOS; it takes a
  fraction of a millisecond. See
  [Troubleshooting](troubleshooting.md#the-window-uses-a-lot-of-memory).
- `post-compact`: after Claude Code compacts a long conversation. It keeps the summary in
  `archive/sessions/` and points the task's journal at it.
- `session-end`: the session ends. It notes that it ended well, and starts the backup in
  the background.

## augmented-brain version

Prints the version of the program.

```bash
augmented-brain version
```

`--version` and `-v` do the same.

## augmented-brain help

Prints the list of commands, in your language.

```bash
augmented-brain help
```

`--help` and `-h` do the same. Running `augmented-brain` with nothing prints the list
too, as an error (exit code 2).
