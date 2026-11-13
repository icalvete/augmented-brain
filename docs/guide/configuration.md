# Configuration

[Leer en español](configuration.es.md)

The installer writes the configuration for you, and Claude adds to it when you ask
("`shop-prod` is production"). You only open it to change something by hand: a new work
folder, a repository where Claude may commit, a root exception.

## Where it is

One folder per computer, the **configuration folder**:

- Linux and macOS: `~/.config/augmented-brain/` (or `$XDG_CONFIG_HOME/augmented-brain/`).
- Windows: `%APPDATA%\augmented-brain\`.
- Anywhere else, if you set `AB_CONFIG_DIR`.

Inside it, `config.toml` holds the settings. It is a **TOML** file: `name = value` lines,
and a `[section]` heading for each space. Lines starting with `#` are comments; the
program keeps them when it changes the file.

## An example

```toml
# ~/.config/augmented-brain/config.toml
language = "en"
default_space = "personal"
production_aws_profiles = ["shop-prod", "shop-prod-eu"]
untouchable_hosts = ["db01", "backup.example.com"]
root_ask = ["web02 until 2026-12-31"]

[spaces.work]
memory = "~/augmented-brain/work/notes"
work_folders = ["~/augmented-brain/work", "~/code"]
backup = "git@gitlab.com:lucia/brain-work.git"
git_allowed_repos = ["~/code/blog", "~/code/blog-*"]
git_ask_repos = ["~/code/shop"]

[spaces.personal]
memory = "~/augmented-brain/personal/notes"
work_folders = ["~/augmented-brain/personal"]
backup = "https://github.com/lucia/augmented-brain-personal.git"
```

Paths may start with `~`, your home folder. The program reads the file every time it
runs, so a change counts at once.

## Settings for the whole computer

These go at the top, before any `[spaces.…]` heading.

- `language`: the language the program speaks to you, `en` or `es`: its messages and the
  guard's reasons. Default `en`.
- `default_space`: the space used outside every work folder, such as the scratch folder
  the desktop app opens a new session in. Its backup carries your profile and the shared
  notes. Optional; with two spaces or more, `init` and `space add` set it.
- `production_aws_profiles`: AWS profiles that reach production. The guard asks you
  before any command that uses one, in every space. See [The guard](guard.md).
- `untouchable_hosts`: machines not to touch without asking. The guard asks before
  `ssh`, `scp`, `rsync`, `sftp` and `mosh` to them, in every space. Names match ignoring
  case, and a short name matches the full one (`db01` and `db01.example.com`).
- `root_ask`: temporary exceptions for root, one machine each, written `"<host> until
  <date>"`. The date is a day (`2026-12-31`, which lasts to its end, in UTC) or a time
  (`2026-12-31T23:59Z`). Until then, root on that machine asks you instead of being
  stopped. The date is required.
- `format`: the version of the configuration folder. The program writes it when a new
  version needs it; never change it by hand. Without it, it is 1.

## Settings of each space

Each space is a section, `[spaces.<name>]`. Together they are the `spaces` of the file.
The name is lowercase letters, digits and dashes, starting with a letter.

- `memory`: the space's notes folder. Required.
- `work_folders`: the folders that pick this space when you open Claude in them, or in a
  folder inside them. See [Spaces](spaces.md).
- `backup`: the address of the space's private git repository. `backup setup` writes it.
  Without it, the space is not backed up.
- `git_allowed_repos`: repositories where Claude may commit and push, written with
  `git -C <repo>`. The last part of a path may hold a `*`, which matches one folder. A git
  worktree of a listed repository counts as it.
- `git_ask_repos`: the same, but Claude asks you each time. A repository in both lists
  asks.
- `production_aws_profiles` and `untouchable_hosts`: older versions kept these per space.
  They still work there, and the guard joins them with the top-level lists. New ones go
  at the top.

The repository lists of every space count together: a repository listed in one space is
let through from a session in any of them.

## What Claude may change

Claude never edits this file on its own: the guard asks you first. Two tools add to it,
and only add:

- `safety_save` adds production profiles and untouchable machines.
- `backup_setup` writes a space's `backup`, only for the repository it creates in your
  GitHub account, and never moves an existing backup elsewhere.

To take anything out, edit the file yourself.

## When it cannot be read

If the file has a mistake, the program says which line. A setting it does not know is an
error too, named in the message. If the file was written by a newer version, the program
says to update instead. A file with a mistake is told in the system's language (or
`AB_LANGUAGE`), since its own `language` cannot be read. Everything else is told in the
file's `language`: an unknown setting, a newer version, a space with no `memory`, a
`default_space` that is not a space. If `language` itself is wrong, the system's.

While the file cannot be read, the guard does not know your lists. So it asks you before
every `aws`, `ssh`, `scp`, `rsync`, `sftp` and `mosh`. `augmented-brain doctor` tells you
what is wrong.

To go back to a version that worked, run `augmented-brain config history` and then
`augmented-brain config restore <number>`: both work while the file is broken. See
[Configuration versions](config-history.md).

## `secrets.toml`

Next to `config.toml`, `secrets.toml` names the secrets you keep in your system
keychain, never their values:

```toml
secrets = ["telegram/alerts-bot: alerts to my phone", "smtp/newsletter: sending the newsletter"]
```

Each entry is `service/name: what it is for`. The service and the name are single words,
as the keychain knows them. Claude adds to it with the tool `secret_name_save`, and the
panel's Secrets page lists it. A value is refused. It is a file of its own because
versions before 0.6.0 refuse a setting of `config.toml` they do not know. 0.6.0 kept the
list in `config.toml` as `secrets`; `check` and the session start move it here.

## `guard.toml`

Next to `config.toml`, `guard.toml` holds what the guard never allows:

```toml
untouchable_repos = ["~/work/core", "~/work/cli-*"]
```

- `untouchable_repos`: repositories where Claude never commits, pushes or merges,
  whatever `git_allowed_repos` and `git_ask_repos` say. For the whole computer. Same form
  as those lists (`~`, a `*` in the last part, worktrees count).

The panel's Guard page writes it. If the file is missing, nothing is untouchable; if it
cannot be read, the guard asks before every commit and push. It is a file of its own for
the same reason as `secrets.toml`: versions before this one refuse a setting of
`config.toml` they do not know, and ignore a file they do not know. A session that still
runs such a version does not enforce the list. It is not carried by backups.

## `session.toml`

Next to `config.toml`, `session.toml` holds how you want your sessions:

```toml
preferred_model = "opus"
```

- `preferred_model`: the model you expect to answer (`opus`, `sonnet`, `haiku` or a full
  name such as `claude-opus-5-5`). When a session answers with another model, Claude
  tells you. It checks at the start, if Claude Code says which model it starts with. If
  not, it checks after an answer: in a new session, the second one at the latest. It
  checks again if the model changes. Once per session and model. It never changes the
  model itself: you do, with `/model` or the model picker.

The panel's Guard page writes it, and the guard asks before Claude writes it. If the file
is missing or the setting is empty, nothing is checked. It is a file of its own for the
same reason as `guard.toml`. It is not carried by backups.

## `remote.toml`

Next to `config.toml`, `remote.toml` holds the settings of [remote access](remote.md),
which keeps Claude reachable from another computer. `augmented-brain remote install`
creates it. It is a file of its own because versions before this one refuse a setting of
`config.toml` they do not know, and they must keep reading your configuration.

```toml
enabled = true
spaces = ["personal", "work"]
name_prefix = "my-laptop"
permission_mode = "default"
folders = { work = "~/code/clients" }
```

All are optional:

- `enabled`: whether remote access is set up. `remote install` writes `true` and
  `remote uninstall` writes `false`. Only when it is `true` does `doctor` check the
  services; a file it cannot read is a warning either way.
- `spaces`: the spaces that get a session, each in a service of its own. Default: the
  `default_space`, or the only space when there is one.
- `name_prefix`: sessions are called `<name_prefix>-<space>`, such as `my-laptop-personal`.
  One plain word: letters, digits, dot, dash and underscore. Default: the computer's name.
- `permission_mode`: how much Claude may do without asking. One of `default`,
  `acceptEdits`, `plan`, `auto`, `dontAsk` and `bypassPermissions`. Default: `default`,
  where Claude asks as usual and the guard stays in force. **Do not use
  `bypassPermissions`**: it lets whoever connects run anything, with no questions, and
  leaves the computer open.
- `folders`: where a space's session starts, when it is not the space's first work folder
  (the default). The folder must exist and lie inside a work folder of that space, so the
  session still loads that space.

Run `augmented-brain remote install` again after changing it, so the services pick it up.

## Other files in the folder

- `profile.md`: your profile, for every space.
- `shared/`: the shared notes (rules, references and lessons for every space).
- `state/`: what this computer knows of each space's sessions. Never backed up.
- `migrations/`: a full copy of notes before a new version moved them.
- `panel.json` and `panel.lock`: where the panel listens while it runs, and its lock.
  The guard never lets Claude read `panel.json`: it holds the panel's key.
- `remote/<space>/`: the folder where the remote access service of a space starts, only for
  a space with no work folder of its own. It is made a work folder of that space.
- `review-<space>.draft`: a weekly review started in the panel and not finished yet.
- `history/`: the last 10 versions of your configuration, with who changed them. Only on
  this computer, never backed up. See [Configuration versions](config-history.md).

## Environment variables

Set these in your shell to change how the program behaves:

- `AB_CONFIG_DIR`: use this folder as the configuration folder.
- `AB_LANGUAGE`: `en` or `es`, the language before there is a configuration, or while it
  cannot be read (the installer and `init` use it).
- `AB_NO_AUTO_BACKUP=1`: do not back up when a session ends.
- `AB_NO_UPDATE_CHECK=1`: do not ask GitHub, once a day, whether there is a new version.
- `AB_NO_BROWSER=1`: `augmented-brain panel` prints the address instead of opening the
  browser.

The installer reads a few more:

- `AB_YES=1`: answer yes to every question.
- `AB_NO_DESKTOP=1`: do not offer the Claude desktop app.
- `AB_DESKTOP_TIMEOUT`: on Linux, how many seconds the desktop app's download may take
  before the installer gives up and goes on without it. Default 480 (8 minutes).
- `AB_INSTALL_DIR`: where to put the program. Default `~/.local/bin`.
- `AB_NO_INIT=1`: only install the program, without setting anything up.
- `AB_VERSION`, `AB_RELEASE_URL`, `AB_API_URL`, `AB_RETRY_DELAY`, `AB_MARKETPLACE` and
  `AB_DESKTOP_URL`: for testing: which version, where the release files are, the GitHub
  API to fall back on, the wait between tries, a folder or repository to take the plugin
  from instead of the one inside the program, and where the desktop app's apt repository
  is.

## Claude Code's own settings

One setting of Claude Code matters here. If Claude Code keeps a memory of its own for your
spaces' folders, it drifts from the space's. `augmented-brain doctor` tells you, and the
fix is `"autoMemoryEnabled": false` in Claude Code's settings, in the file `doctor`
names.

The full format of every file is in the [memory format](../spec/memory-format.md).
