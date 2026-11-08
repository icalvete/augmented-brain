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
| `cmd/augmented-brain/` | The commands: `init`, `space add`, `doctor`, `check`, `import`, `backup`, `restore`, `panel`, `remote`, `config history`, `config restore`, `update`, `uninstall`, `version`, `mcp`, `hook`. `loadConfig` and `currentSpace` (main.go) are how a command gets the configuration and its space. |
| `internal/config` | The configuration file: spaces, `default_space`, the machine-wide guard lists (`root_ask` parsed in `rootask.go`), `SecretsFile` (`secrets.toml`: the names of the user's secrets, a file of its own because earlier programs refuse an unknown setting of `config.toml`), `GuardFile` (`guard.toml`: `untouchable_repos`, same reason; `LoadGuard`, `SaveGuard`; remote access follows it with `remote.toml`, read by `internal/remote`), `SessionFile` (`session.toml`: `preferred_model`, same reason; `LoadSession`, `SaveSession`, `ModelMatches`) and `GuardFingerprint` (a hash of every list the guard answers to, sealed in `state/guard.seal`). `Load` says "update" when a newer format holds settings it does not know. Edits keep the user's comments: every one goes through the line editor in `edit.go` (`AddSpace`, `SetSpaceKey`, `SetSpaceList`, `SetTopList`, `SetDefaultSpace`, `SetFormat`), which loads the file back before the change stays. |
| `internal/memory` | Reads and checks notes (headers only), builds the index `MEMORY.md`. The one place for what a note looks like: `Types` (folder and id prefix of each type, `PathFor`), the dated line `- date — text` (`dated.go`), `## ` headings (`Headings`), dates (`ParseDate`, `DaysBetween`), the header (`SplitHeader`). `LoadSpace` loads a space with the shared notes it may link to; `LoadShared` the shared notes alone (`<config>/shared/`: rules and knowledge for every space). |
| `internal/brain` | Everything that writes notes: tasks, areas, knowledge, people, the profile, safety settings, recovered sessions (`Recover`), the list of secret names (`secrets.go`: names only, a value is refused), what each repository is (`repos.go`: the reference `ref-repositories`, a line per repository), captures sent to another space's inbox (`CaptureTo`: the line ends `(from <space>)`, before any reminder, and is read as an origin only for a space of the configuration; `Arrived`, `InboxDoneEntry` for `capture_done`). A change copies the note, checks everything, then writes once (`Update` takes the watch dates too), so a failed change leaves nothing behind. |
| `internal/mcpserver` | The MCP tools, thin layers over `brain`. Calls run one at a time (`filelock.Writer`: a mutex plus the notes lock file). |
| `internal/panel` | The local web panel (`augmented-brain panel`): started detached on demand, 127.0.0.1 only (with `panel remote`, also through the user's Tailscale network: `Remote` in `panel.json` and in the server, which then also accepts exactly that Host and only with `Tailscale-User-Login` equal to the machine's owner, the origin being `https://` there; the server takes the address back when it stops, and `StartMode` clears a leftover through `UnpublishLeftover` whenever no panel answers; `GoRemote` has a panel already running switch in place, `POST /remote` with its token and never through the network, so its key and the open pages stay, and falls back to starting again for an earlier panel; `GoLocal` undoes it when publishing fails), pages under `/s/<session>/` (derived from the run's token in `panel.json`; no cookie, since cookies of 127.0.0.1 reach every port), `/ping` answers an HMAC challenge so a stale `panel.json` cannot be impersonated, `panel.lock` serializes start and stop, Host/Origin/CSRF checks, strict CSP, stops after 30 idle minutes (`/stamp`, which an open note's page asks now and then to tell whether the note changed on disk, does not count as use). Routes in `pages.go`; one file per page: `today.go` (every space at once, and the repositories of open tasks with work not committed or not pushed), `tasks.go`, `notes.go` (areas, knowledge, rules, people, any note), `people.go`, `search.go`, `review.go` (the weekly review; its draft lives in the configuration folder until finished), `secrets.go`, `sessions.go`, `reminders.go`, `inbox.go`, `profile.go`, `space.go` (Health), `repos.go` (Repositories: every git repository in each space's work folders, found by `repostat.Find`, with what is pending, the tasks that name it and the user's line on what it is), `versions.go` (Versions: the configuration's versions, what each changed, going back; local only), `guard.go` (Guard: every list the guard answers to, edited in place with comments kept, sealed with `state.WriteSeal` unless something else had changed them; the one way to change `guard.toml`), `help.go` (Help: links to the website, and a problem report or an idea for GitHub's issue forms in `.github/ISSUE_TEMPLATE`, filled in by field id; `GET /help/report`, a plain link (a form could not redirect to github.com under the CSP's `form-action`), runs `doctor` without holding the notes lock, redacts it with the version and the system, and answers 303 to the problem form with it filled in, under an 8,000-character cap; the idea link goes straight to its form). `markdown.go` renders notes: a small renderer of its own that escapes every piece of text and writes only its own tags, so a note cannot put HTML or scripts in the page. Pages in `web/` (`go:embed`, no external requests; light only, on purpose); writes through `brain` under the same `filelock.Writer` as the MCP server. |
| `internal/tailnet` | Tailscale for the remote panel (`panel remote`): `Status` (`tailscale status --json`: the machine's name, its owner's login, whether the network gives it HTTPS), `Serve`/`Unserve` (`tailscale serve --bg --https=8443` to `http://127.0.0.1:<port>`, and `off`), only ever its own entry: port `Port` (8443) with a single proxy to 127.0.0.1; anything else there is `ErrPortTaken` and never touched, never `serve reset`. Errors the command explains: `ErrMissing`, `ErrStopped`, `ErrNoHTTPS`, `ErrOperator` (Linux: `sudo tailscale set --operator=<user>`, printed, never run). `MarkUsed`/`Used` (`state/panel-remote`): only then does `doctor` add its line. All through `Tailnet.Run`, so tests never run tailscale; on macOS it also finds the app's program. |
| `internal/repostat` | Whether a repository on this computer holds changes not committed or commits not pushed, for the panel's Today. Read only: git with no optional locks, no password prompt, no filesystem monitor, 4 seconds at most. A branch never pushed counts only when the repository has a remote. `find.go`: the repositories inside a folder (a few levels down, no links followed, capped) and a remote's address with no credentials (`CleanRemote`). |
| `internal/remote` | `augmented-brain remote install \| uninstall \| status`: one `systemd --user` service per space (`augmented-brain-remote-<space>.service`) that keeps `claude remote-control` running, so the machine can be reached from claude.ai or Claude Desktop after a reboot. Each service starts in the space's first work folder, or the one `folders` in `remote.toml` names (it must lie inside a work folder of the space); only a space with no work folder gets `<config>/remote/<space>` (`Service.Own`), which `install` creates and adds through `config.SetSpaceList`. Either way the session's space comes from its folder as always. `System` holds everything outside the configuration folder (command runner, `LookPath`, hostname, GOOS, the unit folder, Claude's `.claude.json`, read only for `hasTrustDialogAccepted`), so tests use fakes and temp folders and never run claude, systemctl or loginctl. Settings (`enabled`, `spaces`, `name_prefix`, `permission_mode`, `folders`) are in `remote.toml`, a file of its own for the same reason as `secrets.toml`. Units carry a marker line: uninstall removes only those. Their stderr goes to the journal (rate-limited; claude remote-control writes its errors there and redraws on stdout, which goes nowhere); `Outdated` flags a unit of an earlier program, and `status`/`doctor` say to install again. `install` never runs `loginctl enable-linger` (root): it prints the command. Linux only; macOS (launchd) is pending. `doctor` gets a line per space when `enabled`. |
| `internal/keyring` | Whether the system keychain holds a secret, never its value: `secret-tool` on Linux, `security` on macOS, unknown elsewhere. What the command prints is discarded. |
| `internal/fresh` | Which tasks changed outside a session since it last looked. A "seen" file per Claude process (`state/seen/pid-<n>.json`, named after the process, not the session: after /clear the hooks get a new session id but the MCP server keeps the old one): for every task its modification time, size, SHA-256 and what the index says of it. `HookKey` (CLAUDE_PID) and `ServerKey` (the MCP server's parent, when CLAUDECODE is set) name the same file; without them nothing is tracked. `Baseline` (SessionStart part 1, after the migration and before the index is written), `Check` (UserPromptSubmit, and the MCP server before each tool: changes from outside), `Absorb` (the MCP server after each tool, still under the notes lock: the tool's own writes, never reported back), `Text` (the note, at most 8 lines). Reported: a change of what the index says of any task, any change of a task the session read or wrote, new and gone tasks; a change of format, in one line. |
| `internal/confighistory` | The last versions of the configuration (`Keep`, 10), only on this machine and never in the backup: `config.toml`, `guard.toml`, `remote.toml`, `session.toml`, `profile.md`, `secrets.toml` as they were, with who changed them (`ByOutside`, `ByCommand`, `ByTool`, `ByPanel`, `ByRestore`), in `<config>/history/<id>.json`. `Record` keeps one only when something changed; it runs before and after every command that may change the configuration (`keepsHistory` in main.go), MCP tool and panel write, and at the session start, so a change by hand is its own version. `Restore` keeps the current one first, checks the result (`Check`) and puts it back if it fails; it never seals the guard's lists: the panel's Versions page does (local only; through the network it only shows), `config restore` does not, so a command the assistant runs stays one the guard asks about. `Compare` is the line diff the page and `config history N` show. |
| `internal/filelock`, `internal/detach` | The cross-process lock and the shared `Writer`; starting a process that outlives its parent (the automatic backup, the panel). |
| `internal/hook`, `internal/session` | The SessionStart, PreToolUse, UserPromptSubmit, Stop, PostCompact and SessionEnd hooks (UserPromptSubmit tells the assistant which tasks changed outside its session, package fresh, waiting at most `FreshWait` for the notes lock; its command in `hooks.json` ends in `|| true`, because the plugin can reach a machine before the program and an unknown hook exits 2, which there would throw the user's message away; the guard's never does) (SessionEnd starts the automatic backup); what the assistant reads at the start, in up to `session.Parts` (9) hooks of 9,500 UTF-16 units each (Claude Code cuts each hook's text at 10,000); only the parts that have something are sent and counted ("part 2 of 3"): part 1 alone does the work behind the notes. How the hooks ask the assistant to pass news on in the user's language is worded once, in `session/say.go`. Stop also warns about memory: `internal/hostmem` walks up from the hook to `claude`, takes its parent when that is VS Code's extension host (claude itself otherwise), reads its resident memory (`/proc` on Linux, one `ps` on macOS, nothing on Windows), and past 2, 4 and 8 GB has the assistant warn the user, once per host (pid and start time) and step; the marks are `state/memory.json`, and `doctor` warns about every Claude host past 2 GB. With `preferred_model` set, SessionStart (when Claude Code sends `model`) and Stop (the model of the last answer, read from the end of the transcript by `state.LastModel`) have the assistant tell the user when another model answers, once per session and model (`state/model.json`, which also records what the last start said). `AB_MEMORY_WARN_MB` is a test aid, not a setting: it moves the first step (the others at twice and four times it) to try the warning on a window of normal size. |
| `internal/format` | The version of each space's notes (`.augmented-brain/format`) and of the configuration folder (`format =` in `config.toml`, never written while it is 1), and the migrations that move them forward at the start of a session or in `check`, after a full copy. Notes of a newer version are refused, never written. |
| `internal/importer` | `augmented-brain import`: a memory of Claude Code into the spaces. `MakePlan` decides (and reports what it cannot), `Apply` writes, never over a note that exists; the source is only read. |
| `internal/state` | What this machine knows about each space's sessions (active task, last write, sessions that did not end), in the configuration folder, never backed up; the reader of Claude Code's transcripts for `session_recover`; and the MCP servers running (`servers/<pid>`, with their version), so `doctor` and the session start can say which open sessions still run an earlier program after an update (on Linux also those of versions that kept no note: their program was replaced on disk). |
| `internal/guard` | The command guard: a small shell parser and the policy (push, root and its dated exceptions in `root_ask`, secrets, production profiles, untouchable hosts). `repoVerdict` is what the repository lists say (the untouchable list first: it wins), for a push and for merging a pull request (`merge.go`: the folder a `cd` in the command names, never the session's alone, and the remotes in its `.git/config`, read without running git)). `panelaccess.go` denies what reaches the panel's key (`ReadFile` for Read and Grep, `panelAccess` for commands) and `ConfigChanged` (the lists are not the sealed ones) turns an allowed commit, push or merge into a question; the hook feeds both (`hiddenFiles`, `guardSealNote`). |
| `internal/setup`, `internal/doctor`, `internal/backup`, `internal/claudecode`, `internal/update` | Creating spaces, the health check, backup and restore, installing the plugin, the new-version check. |
| `internal/i18n` | Every message a person reads, and every text the program writes into notes, in one catalogue: a file per subject (`doctor.go`, `commands.go`, `panel.go`, `import.go`, `init.go`, `notes.go`, `guard.go`, `help.go`, `hook.go`: what the hooks tell the assistant in the configuration's language), each message with its English and Spanish side by side. `T` formats one; `All` gives both, to read back text written in either. |
| `internal/paths` | Paths and files: `~`, `Resolve`, `Within`, `Rename` (patient on Windows), `WriteAtomic`, the one way the program replaces a file, and `CopyTree`. |
| `internal/rules`, `internal/templates` | The common rules and the starter files, in English and Spanish. |
| `plugin/` | The Claude Code plugin: `hooks/hooks.json`, `.mcp.json`, `commands/panel.md` (`/brain:panel`). `.claude-plugin/marketplace.json` makes this repo its marketplace. |
| `install.sh` | The installer people run with `curl … \| sh`. |
| `internal/portcheck` | Used only by tests: reads the repository's tests for what passes on Linux and fails on Windows (`.Perm()` in a test that never looks at `runtime.GOOS`; JSON joined by hand around a path). A line that is fine says `// windows-ok`. |
| `internal/doccheck` | Used only by tests: whether the reference pages (`docs/guide/commands`, `mcp-tools`, `configuration`, each with its `.es` twin) cover every command and flag (`commands` in `cmd/augmented-brain/commands.go`), MCP tool and parameter, and `config.toml` setting. `AB_DOCS_DIR` points the tests at other pages. |
| `docs/` | The website (GitHub Pages, served from `main`): the landing pages, the documentation in `docs/guide/` (Markdown sources, rendered with a sidebar by `scripts/build-guides.py`, which also writes `docs/llms-full.txt`) and the format spec. |
| `scripts/` | Checks, builders and release helpers (below). |
| `test/rehearsal/` | The rehearsal before a release: `rehearse.sh` builds the release from the checkout, serves it to the VMs, installs or updates from a snapshot (`--from`, `clean` by default) and runs `checks.py` there (the program, the guard, notes through MCP, the hooks, backup and restore; `--sessions` adds a real Claude), all with made-up data. Logs in `logs/`, not committed. |
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

`scripts/precheck.sh` runs all of that and more before every push (the `pre-push` hook,
with `git config core.hooksPath .githooks`): vet compiled for Linux, Windows and macOS,
the tests compiled for Windows and macOS, the privacy check and the generated guides,
in well under a minute. `--full` adds `compat-check`. `--windows` also runs every
package's tests on Windows, in the VM of `test/vm/windows-11` (`scripts/test-windows.sh`,
about 3 minutes): use it for a change to paths, files, permissions, hooks or commands,
the kind that passed on Linux and failed on Windows in CI. `internal/portcheck` catches
the two such patterns a reader can see (permission bits checked without looking at
`runtime.GOOS`, JSON written by hand around a path) in every `go test`. CI runs once per change, in its
pull request and again on `main`: a branch with no pull request gets no CI, so this is
how it gets tried. Never push without it.

Heavier checks, before a release or when you touch what they cover:

- `scripts/install-check.sh`: the installer from nothing, hands-off, and 12 checks of the
  result (13 with the desktop app). It installs Claude Code for whoever runs it, so use
  `AB_CHECK_HOME=<throwaway folder>` on a real machine. CI runs it on macOS and Ubuntu in
  every pull request that touches the installer, the plugin or the code (`install.yml`);
  the Intel Mac only on `main`.
- `scripts/compat-check.sh`: what this version writes (tasks, people, secrets' names, what a repository is,
  captures, rules, a backup), the last two releases must still read: their `doctor`,
  `check`, MCP tools, session start and restore. After an update, open Claude sessions
  keep the old program, and other machines may not be updated yet. CI runs it in every
  pull request (`compat` in `ci.yml`). By hand: `COMPAT_DIR=<folder> scripts/compat-check.sh
  <new binary> [versions]`; it leaves the folder for you to delete.
- `scripts/install-check-mac.sh`: what only a Mac can get wrong in the installer (the PATH
  file for zsh and bash, macOS 13 or later, Apple's stub git), with fakes for `sw_vers`
  and `xcode-select`. CI runs it on every Mac row; on a Mac without Go, give it a built
  release with `AB_CHECK_RELEASE`.
- `scripts/install-check.ps1`: the same for Windows and `install.ps1`, 14 checks. It
  installs Git, Claude Code and the program for whoever runs it: CI runs it on
  windows-latest (`windows.yml`), and so can the Windows VM from its `clean` snapshot.
- `scripts/faults-check.py`: what the program does when something it uses is missing,
  one thing at a time in a throwaway home (Claude Code, gh, git, the network, Tailscale,
  systemd, a backup, a configuration that loads, write access to the notes): it must say
  so and leave nothing half done. Tools reach it through wrappers on a PATH of its own.
  CI runs it on Linux, macOS and Windows (`faults` in `ci.yml`); by hand, `python3 -I
  scripts/faults-check.py`, about a minute.
- `scripts/e2e-fresh.py`: several real, isolated Claude sessions on one space: a task
  closed by another session is heard before the next answer, own writes are not, /clear
  keeps it working, a capture sent from another space comes up there. Same login and
  cost as `e2e-claude.sh`.
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
  Its denylist of the maintainer's personal data lives outside the repo; CI checks it by
  keyed fingerprints (`scripts/privacy-hashed.py`, `scripts/privacy-hashes`, the key in the
  `AB_PRIVACY_KEY` secret). After changing the denylist, run `scripts/privacy-hashed.py build`
  and commit the fingerprints.
- **Two languages, always in step.** Everything a person reads exists in English and
  Spanish: a new message goes in `internal/i18n`, in the file of its subject, with both
  languages (`TestCatalog` checks they have the same placeholders); no package keeps a
  language table of its own. Outside Go: `say` in `install.sh`, the templates,
  `README.md` and `README.es.md`, `CHANGELOG.md` and `CHANGELOG.es.md`,
  `docs/index.html` and `docs/es/index.html`, each guide and its `.es` twin. Change one,
  change the other.
- **The docs follow the code.** A new command or flag goes in `commands` (`cmd/augmented-brain/commands.go`) and in the usage text; a new command, flag, MCP tool, tool parameter or `config.toml` setting also goes in its reference page, in both languages, or `go test` fails (`TestCommandsDocumented`, `TestToolsDocumented`, `TestSettingsDocumented`). A change people notice also updates the guide that tells it.
- **Generated files are not edited by hand.** Change the source and regenerate:
  `docs/img/*.svg` from `docs/img/diagrams.py`; every `docs/guide/*.html`,
  `docs/sitemap.xml` and `docs/llms-full.txt` from the `.md` sources with
  `scripts/build-guides.py` (its `GROUPS` list is the sidebar: a new page goes there); a space's
  `MEMORY.md` is written by the program.
- **The memory format is a contract.** `docs/spec/memory-format.md` describes it; a change
  to folders, header fields or note types updates the spec, the index and their tests.
  A change that older notes do not already follow adds a step to
  `format.NotesMigrations` (or `ConfigMigrations` for the configuration folder), which
  raises the format version by one: the step must be safe to repeat, and gets a test
  that runs it on notes written the old way.
- **A test package that runs git calls `testenv.CleanGit()` first in its `TestMain`.**
  Git hands its hooks `GIT_DIR` and friends: without this, tests run from the pre-push
  hook make their throwaway repositories inside the real one.
- **A new or updated dependency regenerates `THIRD_PARTY_NOTICES.md`** with
  `scripts/third-party-notices.py` (precheck fails otherwise): its license travels in the
  release archives next to `LICENSE`.
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
  try work on a branch, with `scripts/precheck.sh` and then a pull request (CI runs in
  pull requests, not on bare branches). Every CI job stops at 10 minutes, so a stuck
  runner fails instead of waiting for hours.
- Before a release, the rehearsal in the VMs: `test/rehearsal/rehearse.sh` (Ubuntu and
  Windows, from `clean`, about 5 minutes each), and again `--from` a snapshot with the
  last release to rehearse the update. A new version is installed on a real machine
  only after it. What a release brings that its checks do not cover yet goes into
  `test/rehearsal/checks.py` with it.
- `main` takes changes only through a pull request whose CI is green and up to date with
  `main` (a GitHub ruleset; no force push, no deletion, no one bypasses it). Tags `v*`
  cannot be moved or deleted once pushed.
- A release: on a branch, set `VERSION`, rename the `## Unreleased` section of
  `CHANGELOG.md` and `CHANGELOG.es.md` to `## <version>` with its date (`Released <date>.`,
  `Publicada el <date>.`), commit, open a pull request and merge it
  when CI is green; then tag `v<version>` on that commit of `main` and push the tag. `release.yml` builds
  the archives, checksums and `install.sh` and publishes them; the installer always takes
  the latest release.
