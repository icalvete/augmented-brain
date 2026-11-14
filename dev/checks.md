<!-- Detail for AI coding agents and contributors. English only: see AGENTS.md. -->

# Checks, in detail

Go 1.26 or later (`go.mod`; an older `go` fetches it by itself).

```bash
gofmt -l .                 # must print nothing
go vet ./...
go test -race ./...
go build ./cmd/augmented-brain
```

`scripts/precheck.sh` runs all of that and more before every push (the `pre-push` hook,
with `git config core.hooksPath .githooks`): vet compiled for Linux, Windows and macOS,
the tests compiled for Windows and macOS, the guard's test coverage (`scripts/guard-coverage.sh`:
never below `internal/guard/coverage-floor`; raise the floor when a change raises it),
the privacy check, the generated guides, the changelog entries and the third-party
notices, in about a minute. It compiles every package through the tests but builds no
program: `go build` runs only with `--full`. `--full` adds `compat-check`. `--windows` also runs every
package's tests on Windows, in the VM of `test/vm/windows-11` (`scripts/test-windows.sh`,
about 3 minutes): use it for a change to paths, files, permissions, hooks or commands,
the kind that passed on Linux and failed on Windows in CI. `internal/portcheck` catches
the two such patterns a reader can see (permission bits checked without looking at
`runtime.GOOS`, JSON written by hand around a path) in every `go test`. CI runs once per change, in its
pull request and again on `main`: a branch with no pull request gets no CI, so this is
how it gets tried. Never push without it.

What each case is covered by, and what is left for a person on a real Mac, is in
[test/COVERAGE.md](../test/COVERAGE.md): a new command, tool, panel page or missing piece
adds its row there.

Known vulnerabilities: `vuln.yml` runs `go mod verify` and `govulncheck` (Linux, macOS and
Windows) on every change and every Monday, and fails on anything the code can reach. The
fix is usually a newer patch in the `toolchain` line of `go.mod` or a newer module;
Dependabot (`.github/dependabot.yml`) proposes both weekly. How people report a security
problem is in `SECURITY.md`.

Heavier checks, before a release or when you touch what they cover:

- `scripts/install-check.sh`: the installer from nothing, hands-off, and checks of the
  result; then uninstall and install again, `--config`, a home with spaces and accents,
  `--all`, and the update from the release published last. It installs Claude Code for whoever runs it, so use
  `AB_CHECK_HOME=<throwaway folder>` on a real machine. CI runs it on macOS and Ubuntu in
  every pull request that touches the installer, the plugin or the code (`install.yml`);
  the Intel Mac only on `main`.
- `scripts/compat-check.sh`: what this version writes (tasks, people, secrets' names, what a repository is,
  captures, rules, a backup), the last two releases must still read: their `doctor`,
  `check`, MCP tools, session start and restore. After an update, open Claude sessions
  keep the old program, and other machines may not be updated yet. The other way round,
  each of them makes a backup and the new program restores it. CI runs it in every
  pull request (`compat` in `ci.yml`). By hand: `COMPAT_DIR=<folder> scripts/compat-check.sh
  <new binary> [versions]`; it leaves the folder for you to delete.
- `scripts/install-check-mac.sh`: what only a Mac can get wrong in the installer (the PATH
  file for zsh and bash, macOS 13 or later, Apple's stub git), with fakes for `sw_vers`
  and `xcode-select`. CI runs it on every Mac row; on a Mac without Go, give it a built
  release with `AB_CHECK_RELEASE`.
- `scripts/install-check.ps1`: the same for Windows and `install.ps1`, with the update
  from the release published last. It
  installs Git, Claude Code and the program for whoever runs it: CI runs it on
  windows-latest (`windows.yml`), and so can the Windows VM from its `clean` snapshot.
- `scripts/remote-check.py`: `remote install`, `status`, `doctor` and `uninstall` for real,
  against launchd on macOS and the Task Scheduler on Windows, with a `claude` that only
  pretends (built by the script; it writes its arguments, folder and PATH to the
  service's log). CI runs it on both (`remote` in `ci.yml` and `windows.yml`). On a
  Windows machine without Go, `AB_REMOTE_CHECK_PREBUILT` names a folder with both
  programs built.
- `scripts/faults-check.py`: what the program does when something it uses is missing,
  one thing at a time in a throwaway home (Claude Code, gh, git, the network, Tailscale,
  systemd, a backup, a configuration that loads, write access to the notes): it must say
  so and leave nothing half done. Tools reach it through wrappers on a PATH of its own.
  CI runs it on Linux, macOS and Windows (`faults` in `ci.yml`); by hand, `python3 -I
  scripts/faults-check.py`, about a minute.
- `scripts/panel-browser-check.py`: the panel in real browser engines (Playwright for
  Python): every page loads with no console error and fits a phone's screen, and a few
  things a person does (a capture, opening a task, the Guard page's confirmations and
  what-changed list) work. CI runs it with Chromium and WebKit (`panel` in `ci.yml`); by
  hand, `pip install playwright && python -m playwright install chromium`, then
  `python3 -I scripts/panel-browser-check.py chromium`. Firefox once before a release.
- `scripts/mac-check.py`: what only a real Mac can tell, with no one at the screen: what
  Gatekeeper says of the program downloaded by a browser (and that it still runs from
  Terminal), the Secrets page with the keychain locked (no password dialog, never
  "missing"), and every page of the panel in Safari itself through `safaridriver`. CI runs
  it on macOS (`mac` in `ci.yml`), with `AB_MAC_CHECK_KEYCHAIN=1` so it may add a
  throwaway keychain; on your Mac, allow Safari's remote automation first (Develop menu).
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
  (the plugin goes inside the program) and run the installer with `AB_RELEASE_URL`.
- `test/vm/windows-11/Vagrantfile`: the same on Windows 11 (an evaluation edition,
  about 90 days). From the host, `vagrant winrm -s powershell -c …` runs commands inside
  and the guest reaches the host's 127.0.0.1 at 10.0.2.2, so a release served there with
  `python3 -m http.server` can be installed with `AB_RELEASE_URL`.
