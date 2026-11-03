# AGENTS.md

A handbook for AI coding agents working on this repository. People should start with
[README.md](README.md); how the pieces fit is in
[docs/guide/how-it-works.md](docs/guide/how-it-works.md).

## What this is

augmented brain gives an AI assistant memory, a method and a guard: tasks, areas, people
and rules kept as Markdown files that belong to the user, loaded at the start of every
session, and a check on risky commands. One Go program (`augmented-brain`) does the work;
a Claude Code plugin plugs it in through hooks and an MCP server. It is public, MIT,
alpha, and for people who are not necessarily technical: every change is judged by how
it feels to a newcomer.

## Map

| Path | What |
|---|---|
| `cmd/augmented-brain/` | The commands: `init`, `space add`, `doctor`, `check`, `import`, `backup`, `restore`, `panel`, `update`, `uninstall`, `mcp`, `hook`. `loadConfig` and `currentSpace` (main.go) are how a command gets the configuration and its space. |
| `internal/config` | The configuration file: spaces, `default_space`, the machine-wide guard lists. Edits keep the user's comments: every one goes through the line editor in `edit.go` (`AddSpace`, `SetSpaceKey`, `SetSpaceList`, `SetTopList`, `SetDefaultSpace`, `SetFormat`), which loads the file back before the change stays. |
| `internal/memory` | Reads and checks notes (headers only), builds the index `MEMORY.md`. The one place for what a note looks like: `Types` (folder and id prefix of each type, `PathFor`), the dated line `- date — text` (`dated.go`), `## ` headings (`Headings`), dates (`ParseDate`, `DaysBetween`), the header (`SplitHeader`). `LoadSpace` loads a space with the shared notes it may link to; `LoadShared` the shared notes alone (`<config>/shared/`: rules and knowledge for every space). |
| `internal/brain` | Everything that writes notes: tasks, areas, knowledge, people, the profile, safety settings, recovered sessions (`Recover`). A change copies the note, checks everything, then writes once (`Update` takes the watch dates too), so a failed change leaves nothing behind. |
| `internal/mcpserver` | The MCP tools, thin layers over `brain`. Calls run one at a time (`filelock.Writer`: a mutex plus the notes lock file). |
| `internal/panel` | The local web panel (`augmented-brain panel`): started detached on demand, 127.0.0.1 only, pages under `/s/<session>/` (derived from the run's token in `panel.json`; no cookie, since cookies of 127.0.0.1 reach every port), `/ping` answers an HMAC challenge so a stale `panel.json` cannot be impersonated, `panel.lock` serializes start and stop, Host/Origin/CSRF checks, strict CSP, stops after 30 idle minutes. Pages in `web/` (`go:embed`, no external requests); writes through `brain` under the same `filelock.Writer` as the MCP server. |
| `internal/filelock`, `internal/detach` | The cross-process lock and the shared `Writer`; starting a process that outlives its parent (the automatic backup, the panel). |
| `internal/hook`, `internal/session` | The SessionStart, PreToolUse, Stop, PostCompact and SessionEnd hooks (SessionEnd starts the automatic backup); what the assistant reads at the start, in `session.Parts` hooks of 9,500 UTF-16 units each (Claude Code cuts each hook's text at 10,000): part 1 alone does the work behind the notes. How the hooks ask the assistant to pass news on in the user's language is worded once, in `session/say.go`. |
| `internal/format` | The version of each space's notes (`.augmented-brain/format`) and of the configuration folder (`format =` in `config.toml`, never written while it is 1), and the migrations that move them forward at the start of a session or in `check`, after a full copy. Notes of a newer version are refused, never written. |
| `internal/importer` | `augmented-brain import`: a memory of Claude Code into the spaces. `MakePlan` decides (and reports what it cannot), `Apply` writes, never over a note that exists; the source is only read. |
| `internal/state` | What this machine knows about each space's sessions (active task, last write, sessions that did not end), in the configuration folder, never backed up; and the reader of Claude Code's transcripts for `session_recover`. |
| `internal/guard` | The command guard: a small shell parser and the policy (push, root, secrets, production profiles, untouchable hosts). |
| `internal/setup`, `internal/doctor`, `internal/backup`, `internal/claudecode`, `internal/update` | Creating spaces, the health check, backup and restore, installing the plugin, the new-version check. |
| `internal/i18n` | Every message a person reads, and every text the program writes into notes, in one catalogue: a file per subject (`doctor.go`, `commands.go`, `panel.go`, `import.go`, `init.go`, `notes.go`), each message with its English and Spanish side by side. `T` formats one; `All` gives both, to read back text written in either. |
| `internal/paths` | Paths and files: `~`, `Resolve`, `Within`, `Rename` (patient on Windows), `WriteAtomic`, the one way the program replaces a file, and `CopyTree`. |
| `internal/rules`, `internal/templates` | The common rules and the starter files, in English and Spanish. |
| `plugin/` | The Claude Code plugin: `hooks/hooks.json`, `.mcp.json`, `commands/panel.md` (`/brain:panel`). `.claude-plugin/marketplace.json` makes this repo its marketplace. |
| `install.sh` | The installer people run with `curl … \| sh`. |
| `docs/` | The website (GitHub Pages, served from `main`), the guides and the format spec. |
| `scripts/` | Checks, builders and release helpers (below). |
| `test/vm/ubuntu-desktop/` | A clean Ubuntu desktop VM (Vagrant + VirtualBox) to try it as a newcomer. |
| `test/vm/windows-11/` | A clean Windows 11 VM, the same way, for the Windows work (evaluation edition, about 90 days). |

## Build and check

Go 1.25 or later (`go.mod`; an older `go` fetches it by itself).

```bash
gofmt -l .                 # must print nothing
go vet ./...
go test -race ./...
go build ./cmd/augmented-brain
```

Heavier checks, before a release or when you touch what they cover:

- `scripts/install-check.sh`: the installer from nothing, hands-off, and 12 checks of the
  result (13 with the desktop app). It installs Claude Code for whoever runs it, so use
  `AB_CHECK_HOME=<throwaway folder>` on a real machine. CI runs it on macOS and Ubuntu on
  every push that touches the installer, the plugin or the code (`install.yml`).
- `scripts/install-check-mac.sh`: what only a Mac can get wrong in the installer (the PATH
  file for zsh and bash, macOS 13 or later, Apple's stub git), with fakes for `sw_vers`
  and `xcode-select`. CI runs it on every Mac row; on a Mac without Go, give it a built
  release with `AB_CHECK_RELEASE`.
- `scripts/install-check.ps1`: the same for Windows and `install.ps1`, 14 checks. It
  installs Git, Claude Code and the program for whoever runs it: CI runs it on
  windows-latest (`windows.yml`), and so can the Windows VM from its `clean` snapshot.
- `scripts/e2e-claude.sh`: the whole story (open, park, resume, close a task; the guard;
  the automatic backup; the first-session interview) against a real, isolated Claude
  Code. It spends usage of a logged-in account; see the script's header for the one-time
  login.
- `test/vm/ubuntu-desktop/Vagrantfile`: for what only a person can judge (the desktop app,
  signing in, the first-session interview). Restore the `clean` snapshot between tries.
  To test unpublished work there, upload a release built with `scripts/release-build.sh`
  plus the plugin, and run the installer with `AB_RELEASE_URL` and `AB_MARKETPLACE`.
- `test/vm/windows-11/Vagrantfile`: the same on Windows 11 (an evaluation edition,
  about 90 days). From the host, `vagrant winrm -s powershell -c …` runs commands inside
  and the guest reaches the host's 127.0.0.1 at 10.0.2.2, so a release served there with
  `python3 -m http.server` can be installed with `AB_RELEASE_URL`.

## Rules of the house

- **Nothing personal, no secrets.** The repo is public. `scripts/privacy-check.sh` runs as
  the pre-commit hook (`git config core.hooksPath .githooks`) and in CI. Never bypass it.
- **Two languages, always in step.** Everything a person reads exists in English and
  Spanish: a new message goes in `internal/i18n`, in the file of its subject, with both
  languages (`TestCatalog` checks they have the same placeholders); no package keeps a
  language table of its own. Outside Go: `say` in `install.sh`, the templates,
  `README.md` and `README.es.md`, `docs/index.html` and `docs/es/index.html`, each guide
  and its `.es` twin. Change one, change the other.
- **Generated files are not edited by hand.** Change the source and regenerate:
  `docs/img/*.svg` from `docs/img/diagrams.py`; `docs/guide/how-it-works*.html` and
  `getting-started*.html` from their `.md` files with `scripts/build-guides.py`; a space's
  `MEMORY.md` is written by the program.
- **The memory format is a contract.** `docs/spec/memory-format.md` describes it; a change
  to folders, header fields or note types updates the spec, the index and their tests.
  A change that older notes do not already follow adds a step to
  `format.NotesMigrations` (or `ConfigMigrations` for the configuration folder), which
  raises the format version by one: the step must be safe to repeat, and gets a test
  that runs it on notes written the old way.
- **Replace files with `paths.WriteAtomic`** (a crash never leaves half a file), and run
  git through the backup package's runner, which never waits for a password.
- **A new MCP tool** goes in `internal/mcpserver/server.go`, does its work in
  `internal/brain`, and is added to the tool list in `server_test.go`.
- **The assistant reads what the hooks print.** Keep the session text under the budget,
  write it as instructions to the assistant, and tell it to answer in the user's language.
- **SessionEnd has 1.5 seconds in all**, and a plugin's `timeout` does not raise it. Work
  there must take well under a second (a small state file), or run in a detached process,
  as the automatic backup does. Stop runs after every answer: keep it to milliseconds.
- **The installer must work without a terminal** (CI, `AB_YES=1`) and with one; its own
  messages go through `say`, and other tools' noise through `quiet`.
- **Plain words for people.** Short sentences, one idea each; a technical word is
  explained the first time. The user of this project is a newcomer.

## Commits and releases

- Small commits that stand on their own, with a message that says why. Nothing is pushed
  to `main` that the website should not show yet: Pages publishes `main` right away, so
  try work on a branch (CI's install check runs on any branch).
- A release: set `VERSION`, rename the `## Unreleased` section of `CHANGELOG.md` to
  `## <version>` with its date, commit, tag `v<version>`, push both. `release.yml` builds
  the archives, checksums and `install.sh` and publishes them; the installer always takes
  the latest release.
