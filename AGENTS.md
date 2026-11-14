# AGENTS.md

A handbook for AI coding agents working on this repository. People should start with
[README.md](README.md); how the pieces fit is in
[docs/guide/how-it-works.md](docs/guide/how-it-works.md). The detail behind every section
here is in [dev/](dev/): [map](dev/map.md), [checks](dev/checks.md),
[rules](dev/rules.md), [releases](dev/releases.md).

## What this is

augmented brain gives an AI assistant memory, a method and a guard: tasks, areas, people
and rules kept as Markdown files that belong to the user, loaded at the start of every
session, and a check on risky commands. One Go program (`augmented-brain`) does the work;
a Claude Code plugin plugs it in through hooks and an MCP server. It is public, MIT,
beta, and for people who are not necessarily technical: every change is judged by how
it feels to a newcomer.

## Before you touch anything

- **You probably run on the maintainer's machine, with augmented brain installed.** Never
  try the program against the real notes: give every manual run a configuration of its
  own, `AB_CONFIG_DIR=$(mktemp -d) ./augmented-brain init …`. Tests already use temp
  folders.
- **The guard and the installed plugin apply to you too.** A push, a pull request merge
  or a release asks the maintainer, and so does a write to the plugin's own files. The MCP
  tools and hooks in your session are the *installed* version's, not this checkout's: to
  try a change, build it and run `claude --plugin-dir plugin` with it first in `PATH`.
- **Code, comments, commit messages and `dev/` are in English.** Whatever a person reads
  in the program or the docs is in English and Spanish (rules below).
- **Read the package comment first.** Every package starts with `// Package …`, which
  says what it is for; [dev/map.md](dev/map.md) adds the reasons and the traps.

## Map

| Path | What |
|---|---|
| `cmd/augmented-brain/` | The commands. Their list, flags and hidden ones: `commands` in `commands.go`. `loadConfig` and `currentSpace` (main.go) give a command its configuration and space. |
| `internal/config` | `config.toml` and its sibling files (`guard.toml` with `reach`, `secrets.toml`, `session.toml` with `memory_trust`); edits keep the user's comments (`edit.go`). |
| `internal/memory` | Reads and checks notes (headers only); builds `MEMORY.md`; the one place for what a note looks like. |
| `internal/brain` | Everything that writes notes: tasks, areas, knowledge, people, captures, secrets' names. |
| `internal/mcpserver` | The MCP tools, thin layers over `brain`, one call at a time. |
| `internal/hook`, `internal/session` | Claude Code's hooks; what the assistant reads at the start of a session. |
| `internal/guard` | The command guard: a small shell parser and the policy. |
| `internal/reach` | Where augmented brain acts: everywhere, or only in the spaces (`reach`). |
| `internal/panel` | The local web panel; routes in `pages.go`, one file per page. |
| `internal/tailnet` | Tailscale, for `panel remote`. |
| `internal/remote` | `remote install` services and `session start \| stop \| restart \| list`. |
| `internal/watch` | `watch pr`: follows a pull request until there is news. |
| `internal/leftovers` | `clean`: what Claude Code leaves behind. |
| `internal/backup` | Backup and restore through a private git repository; the git runner. |
| `internal/format` | Format versions and their migrations. |
| `internal/confighistory` | The last versions of the configuration, and going back. |
| `internal/state` | What this machine knows of each space's sessions; Claude's transcripts. |
| `internal/fresh` | Which tasks changed outside a session since it last looked. |
| `internal/importer` | `import`: a memory of Claude Code into the spaces. |
| `internal/setup`, `internal/doctor`, `internal/claudecode`, `internal/update` | `init` and `space add`; the health check; the plugin in Claude Code; the new-version check. |
| `internal/repostat` | Whether a repository has work not committed or not pushed. |
| `internal/keyring` | Whether the system keychain holds a secret, never its value. |
| `internal/hostmem` | The memory of the program hosting a Claude session. |
| `internal/i18n` | Every message a person reads, English and Spanish side by side. |
| `internal/paths` | Paths, and `WriteAtomic`, the one way to replace a file. |
| `internal/filelock`, `internal/detach` | The cross-process lock; processes that outlive their parent. |
| `internal/rules`, `internal/templates` | The common rules and a new space's starter files. |
| `internal/version` | The program's version, set at build time. |
| `internal/doccheck`, `internal/stable`, `internal/portcheck`, `internal/testenv` | Test only: reference pages cover the program; the compatibility promise is frozen; patterns that fail on Windows; a plain test machine. |
| `plugin/` | The Claude Code plugin (hooks, MCP server, `/brain:panel`, and the help skill `/brain:help` with the guides inside); it travels inside the program (`marketplace.go`). |
| `install.sh`, `install.ps1` | The installers people run. |
| `docs/` | The website (GitHub Pages, from `main`), the guides in `docs/guide/` and the format spec. |
| `scripts/` | Checks, builders and release helpers. |
| `changelog.d/` | Changelog entries waiting for the next release. |
| `test/` | `COVERAGE.md`, the release rehearsal and the Ubuntu and Windows VMs. |
| `dev/` | Detail for agents and contributors, English only. |

## Recipes

- **A command or flag:** add it to `commands` in `commands.go` and to the usage text;
  write its section in `docs/guide/commands.md` and `commands.es.md` (its
  `augmented-brain help` text is generated from there); freeze it (below).
- **An MCP tool or parameter:** `internal/mcpserver/server.go`, its work in
  `internal/brain`, the tool list in `server_test.go`, `docs/guide/mcp-tools.md` and its
  twin; freeze it.
- **A setting:** its file in `internal/config` (a new file of settings rather than a new
  key in `config.toml`: earlier programs refuse unknown keys there),
  `docs/guide/configuration.md` and its twin; freeze it.
- **A panel page:** a file in `internal/panel`, its route in `pages.go`, a row in
  `test/COVERAGE.md`; it must load without console errors and fit a phone
  (`scripts/panel-browser-check.py`).
- **A guard rule:** its function in `rules`, a row in `pinned_test.go`, and whether it
  holds outside the spaces (`outside.go`, `outside_test.go`).
- **A message:** `internal/i18n`, in the file of its subject, both languages.
- **A change of note or configuration format:** the spec, a migration step, its test
  ([dev/rules.md](dev/rules.md)).

Freezing: `AB_STABLE_UPDATE=1 go test ./...` adds what is new to the `testdata/stable-*.json`
files; it never removes anything.

## Build and check

Go 1.26 or later (`go.mod`; an older `go` fetches it by itself).

```bash
gofmt -l .                     # must print nothing
go vet ./...
go test -race ./...
go test ./internal/guard/      # one package
go build ./cmd/augmented-brain
scripts/precheck.sh            # before every push
```

`scripts/precheck.sh` is what CI would say, in about a minute: format, vet and test
builds for Linux, Windows and macOS, the tests, the guard's coverage floor, privacy, the
generated guides, the changelog entries and the third-party notices. The `pre-push` hook
runs it (`git config core.hooksPath .githooks`). Never push without it.

| If you touch | Also run |
|---|---|
| paths, files, permissions, hooks, commands | `scripts/precheck.sh --windows` (the Windows VM) |
| what notes, backups or settings look like | `scripts/precheck.sh --full` (compat-check) |
| `install.sh`, `install.ps1`, the plugin | `scripts/install-check.sh` with `AB_CHECK_HOME=<throwaway>` |
| the panel's pages | `scripts/panel-browser-check.py chromium` |
| `remote install` | `scripts/remote-check.py` (CI, macOS and Windows) |
| what happens when a tool is missing | `python3 -I scripts/faults-check.py` |
| the guides in `docs/guide/` | `scripts/build-guides.py` |

CI runs only in pull requests and on `main`, and every job stops at 10 minutes. The
heavier checks, the VMs and the real-Claude runs are in [dev/checks.md](dev/checks.md).

## Done when

- [ ] `scripts/precheck.sh` passes, plus the row of the table above that fits.
- [ ] A change people notice has its pair `changelog.d/<branch>.en.md` and `.es.md`;
      otherwise a commit line `Changelog: none`.
- [ ] Every new text a person reads is in `internal/i18n`, or the doc and its `.es` twin.
- [ ] The reference page and the guide that tell it are updated, and regenerated.
- [ ] A new command, tool, page or gap has its row in `test/COVERAGE.md`.
- [ ] What the compatibility promise covers is frozen.
- [ ] A format change has its migration and test.
- [ ] A new or updated dependency regenerated `THIRD_PARTY_NOTICES.md`.

## Rules of the house

The full text, with the reasons, is in [dev/rules.md](dev/rules.md).

- **Nothing personal, no secrets.** The repo is public; `scripts/privacy-check.sh` runs
  as the pre-commit hook and in CI. Never bypass it. Made-up data in examples.
- **Two languages, always in step.** Everything a person reads exists in English and
  Spanish, and changing one changes the other. Only `dev/`, code and commits are English
  alone.
- **Changelog entries go in `changelog.d/`,** never in `CHANGELOG.md`.
- **The docs follow the code, and are checked as people read them.** Tests fail when the
  reference misses a command, flag, tool, parameter or setting, when a link is broken or
  a guide lacks its twin.
- **Generated files are not edited by hand:** the guides' HTML, `docs/sitemap.xml`,
  `docs/llms-full.txt`, `internal/i18n/cmdhelp.go` and the help skill's `plugin/skills/help/guides/` come
  from `scripts/build-guides.py`;
  `docs/img/*.svg` from `docs/img/diagrams.py`; `MEMORY.md` from the program.
- **The memory format is a contract.** Anything an earlier program would refuse (a new
  header field, note type or task status) raises the format version with a migration.
- **What people have keeps working.** A renamed or removed command, flag, setting or tool
  keeps working under its old name; the last two releases must read what this one writes.
- **Every guard rule says whether it holds outside the spaces** (`reach = "spaces"`).
- **A test package that runs git calls `testenv.CleanGit()`** in its `TestMain`.
- **Replace files with `paths.WriteAtomic`** and its variants; run git through the backup
  package's runner, which never waits for a password.
- **Hooks are short.** The assistant reads what they print: write it as instructions,
  under the budget. SessionEnd has 1.5 seconds in all; Stop runs after every answer.
- **The installer works with and without a terminal** (`AB_YES=1`).
- **Plain words for people.** Short sentences, one idea each.

## Commits and releases

- Small commits that stand on their own, with a message that says why.
- Work on a branch, open a pull request, merge when CI is green: `main` only takes
  pull requests whose CI is green and up to date, and Pages publishes `main` right away.
  No force push; tags `v*` never move.
- A release is the maintainer's gesture. An assistant prepares the branch (`VERSION`,
  `python3 -I scripts/changelog.py release <version>`), and runs the `release` workflow
  only when the maintainer asks. The workflow refuses a commit whose required checks, or
  the macOS ones, have not passed. Before it, the rehearsal in the VMs. The detail is in
  [dev/releases.md](dev/releases.md).
