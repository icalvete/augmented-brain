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
| `cmd/augmented-brain/` | The commands: `init`, `space add`, `doctor`, `check`, `backup`, `restore`, `update`, `uninstall`, `mcp`, `hook`. User-facing messages of `init` live in `init.go`. |
| `internal/config` | The configuration file: spaces, `default_space`. Edits keep the user's comments (`AddSpace`, `SetSpaceKey`, `SetSpaceList`, `SetDefaultSpace`). |
| `internal/memory` | Reads and checks notes (headers only), builds the index `MEMORY.md`. |
| `internal/brain` | Everything that writes notes: tasks, areas, knowledge, people, the profile, safety settings. |
| `internal/mcpserver` | The MCP tools, thin layers over `brain`. Calls run one at a time (mutex plus a file lock). |
| `internal/hook`, `internal/session` | The SessionStart, PreToolUse and SessionEnd hooks (SessionEnd starts the automatic backup); what the assistant reads at the start (budget: 9,500 UTF-16 units). |
| `internal/guard` | The command guard: a small shell parser and the policy (push, root, secrets, production profiles, untouchable hosts). |
| `internal/setup`, `internal/doctor`, `internal/backup`, `internal/claudecode`, `internal/update` | Creating spaces, the health check, backup and restore, installing the plugin, the new-version check. |
| `internal/i18n`, `internal/rules`, `internal/templates` | Messages, the common rules and the starter files, in English and Spanish. |
| `plugin/` | The Claude Code plugin: `hooks/hooks.json`, `.mcp.json`. `.claude-plugin/marketplace.json` makes this repo its marketplace. |
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
- `scripts/e2e-claude.sh`: the whole story (open, park, resume, close a task; the guard;
  the automatic backup; the first-session interview) against a real, isolated Claude
  Code. It spends usage of a logged-in account; see the script's header for the one-time
  login.
- `test/vm/ubuntu-desktop/Vagrantfile`: for what only a person can judge (the desktop app,
  signing in, the first-session interview). Restore the `clean` snapshot between tries.
  To test unpublished work there, upload a release built with `scripts/release-build.sh`
  plus the plugin, and run the installer with `AB_RELEASE_URL` and `AB_MARKETPLACE`.

## Rules of the house

- **Nothing personal, no secrets.** The repo is public. `scripts/privacy-check.sh` runs as
  the pre-commit hook (`git config core.hooksPath .githooks`) and in CI. Never bypass it.
- **Two languages, always in step.** Everything a person reads exists in English and
  Spanish: `internal/i18n`, the message maps in `cmd/`, `say` in `install.sh`, the
  templates, `README.md` and `README.es.md`, `docs/index.html` and `docs/es/index.html`,
  each guide and its `.es` twin. Change one, change the other.
- **Generated files are not edited by hand.** Change the source and regenerate:
  `docs/img/*.svg` from `docs/img/diagrams.py`; `docs/guide/how-it-works*.html` and
  `getting-started*.html` from their `.md` files with `scripts/build-guides.py`; a space's
  `MEMORY.md` is written by the program.
- **The memory format is a contract.** `docs/spec/memory-format.md` describes it; a change
  to folders, header fields or note types updates the spec, the index and their tests.
- **A new MCP tool** goes in `internal/mcpserver/server.go`, does its work in
  `internal/brain`, and is added to the tool list in `server_test.go`.
- **The assistant reads what the hooks print.** Keep the session text under the budget,
  write it as instructions to the assistant, and tell it to answer in the user's language.
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
