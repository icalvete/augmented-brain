# Where everything lives, and how it is kept safe

[Leer en español](where-things-live.es.md)

augmented brain keeps everything in plain files on your computer. This page says where each
thing is, what keeps it safe, and what travels in the backup and what never does.

## Your notes

**Where:** one folder per space, inside its work folder: `~/augmented-brain/work/notes/`,
say (`notas` in Spanish). Tasks, areas, knowledge, people, the inbox, the index
`MEMORY.md`. Markdown files you can open and read.

**How they are kept safe:**

- **Only you can read them**, on Linux and macOS. The folder is private to your user;
  `augmented-brain check` closes it again if it was opened. On Windows the program does
  not change it: it keeps the permissions of the folder it is in.
- **No secrets inside.** Claude's tools refuse to write a password, a token or a key into a
  note.
- **Never half written.** Every change is checked first and written in one go, so a crash
  leaves the note as it was.
- **The backup**, if you set it up: the notes folder is a git repository, copied to a
  private repository of yours at the end of every session. See [Backup and
  restore](backup.md).

## The configuration

**Where:** one folder per computer, the configuration folder:
`~/.config/augmented-brain/` on Linux and macOS, `%APPDATA%\augmented-brain\` on Windows.

| File | What it holds |
|---|---|
| `config.toml` | The spaces and the guard's lists for the whole computer. |
| `guard.toml` | The repositories nothing may touch. |
| `secrets.toml` | The **names** of your secrets and what each is for. Never a value. |
| `session.toml` | Two choices of yours: the model your sessions should use, and whether Claude [asks before it saves a rule](guard.md#asking-before-claude-saves-a-rule), your profile or the guard's lists. |
| `remote.toml` | Remote access: which spaces have a door. |
| `profile.md` | Who you are and how you like to work. |
| `shared/` | The rules, references and lessons for every space. |

**How it is kept safe:**

- **The guard asks** before Claude writes any of these files.
- **Versions.** The last 10 versions are kept, with who made each change: going back
  takes one command, even if `config.toml` was left broken. See [Configuration
  versions](config-history.md). Only on this computer.
- **The seal.** The guard notices when its lists changed outside the panel, and asks
  before using them. See [What Claude may do without asking you](what-claude-may-do.md).

## What this computer knows

Also in the configuration folder, never backed up:

- `state/`: what each space's sessions did (each one's active task, a session that did
  not end well), the guard's seal, the Claude sessions running.
- `history/`: the configuration's versions.
- `panel.json`: where the panel listens while it runs, and its key. Claude is never let
  read it.
- `migrations/`: a full copy of the notes, kept before a new version moved them to its
  format.
- `marketplace/`: the Claude Code plugin, as the program carries it (below).

## Your secrets

The values stay in your system's keychain (or wherever you keep them): augmented brain
never stores them. Only their names are written down, in `secrets.toml`, so you and a new
machine know which secrets there are. The panel's **Secrets** page says which ones this
computer's keychain has.

## What the backup takes, and what it leaves

| Goes in the backup | Never goes |
|---|---|
| Every note of the space | The values of your secrets |
| The space's own settings: its folders and its lists | The lists for the whole computer: production profiles, untouchable machines and repositories, root exceptions |
| In the default space only: your profile, the shared notes and the names of your secrets | `guard.toml`, `session.toml` and `remote.toml`; the configuration's versions, `state/` and `panel.json` |

The lists for the whole computer stay out because they may name your work's machines.
The repositories where Claude may commit or push travel with the space, but a restore
does not use them, since someone could have changed the backup: it names them, and you add
them again if you still want to.

## The program and the plugin

- The program: `~/.local/bin/augmented-brain` (`augmented-brain.exe` on Windows).
- The Claude Code plugin travels inside the program: it writes it to `marketplace/` in
  the configuration folder, and Claude Code reads it from there. So its hooks are always
  the ones of the version you installed, and the guard asks before Claude changes them.

Uninstalling removes these two and leaves your notes. See [Updating and uninstalling
without losing anything](update-and-uninstall.md).
