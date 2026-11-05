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
`space add`.

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
- each space's notes folder, its notes and its index;
- Claude Code, git and the plugin;
- the guard, for real, with a command that must be stopped;
- the guard's lists, the root exceptions in `root_ask`, and those that have ended;
- Claude sessions still running an older version after an update;
- a second memory that Claude Code keeps of its own for your folders;
- notes that other users of the computer could read;
- whether there is a default space, and whether a restore left the guard's lists empty;
- remote access, one line per space, when you set it up with `remote install`: a warning
  when its service is not running, Claude does not trust its folder yet, or linger is off.

It ends with an error when something fails. It takes no options. See
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

Connects this folder's space to a private git repository, and makes the first copy.

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
the secrets this machine's keychain lacks, each with the command to store it.

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
Linux only for now (it uses a `systemd` user service); on macOS and Windows they say it
is not available yet. Without a subcommand, or with another, `remote` prints its usage.
See [Remote access](remote.md).

## augmented-brain remote install

Sets remote access up, one service per space.

```bash
augmented-brain remote install
```

For each space named in `remote.toml` (by default, the default space) it creates a
folder inside the configuration folder, makes it a work folder of that space, writes a
`systemd` user service that runs `claude remote-control` there, and starts it. You can
run it again at any time: it changes only what is different.

It then says what is still missing, and does none of it for you:

- Claude must **trust each folder**. It prints `cd <folder> && claude` for each one that
  is not trusted yet: run it once and accept the question.
- **Linger** must be on for the services to start after a reboot with nobody logged in.
  It prints the command, `sudo loginctl enable-linger <you>`, for you to run.

It takes no options. The settings are in `remote.toml` (see [Configuration](configuration.md)).

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
the service is running and starts by itself, and whether Claude trusts its folder. Then
whether linger is on. Under each problem, `→` says what to do. It takes no options.

## augmented-brain update

Updates augmented brain and its Claude Code plugin.

```bash
augmented-brain update
```

If there is a newer version, it runs the published installer, which replaces the program
and then updates the plugin. If not, it only updates the plugin. Close and open Claude
afterwards, so the sessions use the new version.

- `--plugin`: update only the Claude Code plugin. The installer uses it.

## augmented-brain uninstall

Removes the program and the Claude Code plugin. Your notes stay.

```bash
augmented-brain uninstall
```

It shows what it will remove and asks first.

- `--config`: also remove the configuration and the profile.
- `--all`: start from scratch: also the configuration, the profile and every space's
  notes. It shows each notes folder with how many notes it has. Your backups are not
  touched.
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

`--help` and `-h` do the same. Running `augmented-brain` with nothing prints the list too.
