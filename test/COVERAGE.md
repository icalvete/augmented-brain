# What the tests cover

Every case the beta tests set out to cover, and what covers it. **Go** is `go test ./...`
(Linux, macOS and Windows in CI); the scripts run in CI where the last column says. A case
with **by hand** has no automated test: it is a step of the release rehearsal or of the
pass on a real Mac, written in [the Mac pass](#the-pass-on-a-real-mac).

When you add a command, a tool, a panel page or a way something can be missing, add its
row here with the test that covers it.

## Installing, updating, uninstalling

| Case | Covered by | Where |
|---|---|---|
| From nothing, with nothing installed | `scripts/install-check.sh`, `install-check.ps1` | CI: Ubuntu, macOS (Apple and Intel, with and without Homebrew), Windows |
| A second run only updates | `install-check.sh` step 3, `install-check.ps1` | CI |
| Stopped half way (Ctrl+C, a space that fails) | `internal/setup` `TestInitSpacesAllOrNothing` | Go |
| Update from the release published last | `install-check.sh` step 8, `install-check.ps1` step 5 | CI, real download from GitHub |
| A download that fails (504, 404, cut, checksum) | `cmd` `TestUpdateDownloadFails`, `TestInstallerDownloadFails`, `TestInstallerFallsBackToAPI`, `TestInstallPs1DownloadFails`; `internal/update` `TestDownload` | Go |
| Update with nothing newer, or GitHub out of reach | `cmd` `TestUpdateNothingNewer` | Go |
| Notes in an older format | `internal/format` tests, `scripts/compat-check.sh` | Go, CI |
| Uninstall, and install again | `install-check.sh` step 4, `install-check.ps1` step 4 | CI |
| `uninstall --config`, and install again | `install-check.sh` step 5, `cmd` `TestUninstallConfig` | CI, Go |
| `uninstall --all` | `install-check.sh` step 7, `cmd` `TestSaysThereIsNoBackup` | CI, Go |
| The desktop app's download stalls (Linux: gives up with its message, the install goes on) | `install-check.sh` step 1b, a local server that trickles | CI: Ubuntu |
| A home with spaces and accents | `install-check.sh` step 6 | CI |
| macOS: shells, version, Apple's developer tools | `scripts/install-check-mac.sh` | CI |
| macOS: downloaded with a browser (Gatekeeper) | `scripts/mac-check.py` (what macOS says of it, and that it still runs from Terminal); the Finder's dialog by hand | CI: macOS; Mac pass |
| Windows: PowerShell's policy, no winget | by hand | Windows VM rehearsal |
| The installer's Spanish on Windows | `cmd` `TestInstallPs1IsASCII` | Go |

## Commands

| Command | Covered by |
|---|---|
| `init` | `cmd` `TestInitDefaultSpaces`, `TestInitOneSpace`, `TestInitCustomAsks`, `TestInitRefusesAnUnknownLanguage`, `TestInitWarnsWithoutThePlugin`; `internal/setup` |
| `space`, `space add` | `cmd` `TestSpaceCommand`, `TestSpaceAdd`; `internal/setup` `TestAddSpaceErrors` |
| `check` | `cmd` `TestCheckUsesTheDefaultSpace`, `TestCheckWarnsOfMissingRepos`, `TestCheckFindsSecretsInDocs`, `TestCheckMakesNotesPrivate` |
| `doctor` | `internal/doctor` (12 tests), `cmd` `TestSaysThereIsNoBackup`, `TestConfigRestoreFixesABrokenConfig` |
| `import` | `internal/importer`, `cmd` `TestImportMissingSource` |
| `backup`, `backup setup`, `restore` | `internal/backup` (25 tests), `cmd` `TestBackupSpaceFlag`, `TestRestoreNamesTheSecrets` |
| `panel`, `panel remote`, `panel stop` | `internal/panel`, `cmd` `TestPanelRemoteExplains`, `TestExplainTailnet`; `scripts/panel-browser-check.py` |
| `remote install`, `uninstall`, `status` | `test/rehearsal/smoke.py`: on Linux with the real systemd and a pretend claude, install, a reboot of the VM, and the unit back with its PATH (claude's folder and the program's) and permission mode (rehearsal, Ubuntu VM); `internal/remote` (14 tests on Linux's systemd; `TestLaunchdInstallStatusUninstall`, `TestPlistText`, `TestLaunchdErrors` for macOS; `TestTasksInstallStatusUninstall`, `TestTaskText`, `TestTasksErrors` for Windows, all with fakes); `scripts/remote-check.py` for real with a pretend claude (CI: macOS and Windows; tried in the Windows VM, also with the real claude not signed in); a real connection from claude.ai on a Mac and on Windows by hand |
| Task sessions outlive a restart of remote access (Claude's daemon kept out of the service) | `internal/remote` `TestWatchKeepsTheDaemonOutside`, `TestWatchDoesNotRestartOverTheDaemon`, `TestInstallAndUninstallKeepTheDaemon`, `TestStartInItsOwnScope` (fakes); on 11-nov, by hand on a real Linux machine with throwaway user services: a detached process moved to `augmented-brain-kept-<pid>.scope` outlived a restart, and one started through `systemd-run --scope` inside a service outlived its stop |
| `remote watch` (a service whose session the server archived) | `internal/remote` `TestWatchInstalled`, `TestWatchTwoSteps`, `TestWatchLeavesStoppedServices`, `TestWatchOnlyOnLinux` (fakes); the signal (a server with no child) and the pointer measured on a real Linux machine from the debug logs of 8-nov and by hand on 11-nov; on 11-nov a session archived in claude.ai came back with a new one through the two steps, by hand on a real Linux machine (about ten minutes at the timer's pace) |
| `session start`, `stop`, `list` | `internal/remote` `TestStartRefuses`, `TestStartRunsAndStops`, `TestStartReportsFailure`, `TestStartFindsSessionWithoutID`, `TestStopEarlierUnit`, `TestFolderTrustedIsExact`; `internal/hook` `TestStartIgnoresTaskInEnvironment`, `TestFirstMessageNamesTheTask`, `TestEverySessionKnowsTheLauncher`, `TestSessionStartOwnOldServer`, `TestRecoveryOffer`; `internal/remote` `TestLaunchedTask`; `internal/session` `TestContextDueAndReview` (a task's session hears only its task); `test/rehearsal/smoke.py` after the reboot, with a pretend claude (`--bg` in its folder with the mode and a first message naming the task, no task in its environment, `list`, `stop` from any folder runs `claude rm`); tried by hand on Linux on 10-nov (a real session started on its own, showed up in Claude Desktop, and `stop` took it out of it) |
| `session restart` (one session, `--old`, `--when-idle`; what `update` does after updating) | `internal/remote` `TestSessionsKinds`, `TestOldSessions`, `TestRestartEachItsWay`, `TestRestartLooksAgain`, `TestRestartServiceOnce`, `TestRestartNotSelf`, `TestRestartFailureLeavesOthers`, `TestRestartKeepsOutsideFirst`, `TestFindSession`, `TestRestartWhenIdle`, `TestRestartDesktopServer`, `TestRestartOldDesktopServer` (a Claude Desktop session: only its augmented-brain mcp stops); `cmd` `TestAfterUpdate`; `claude respawn` keeping the conversation, tried by hand on Linux on 12-nov; remote services keep their session on restart, measured on 11-nov |
| A task's own worktree of a bare repository, and the sweep of those left behind | `internal/remote` `TestTaskWorktree`, `TestSweepWorktrees` (real git, pretend systemd) | CI: Linux, macOS |
| `config history`, `config restore` | `cmd` `TestConfigHistoryAndRestore`, `TestConfigRestoreFixesABrokenConfig`; `internal/confighistory` |
| `update` | see Installing |
| `uninstall` | see Installing |
| `mcp` | `internal/mcpserver` |
| `hook` | `internal/hook` (34 tests), `test/rehearsal/checks.py` |
| `version`, `help`, nothing, an unknown command, extra words | `cmd` `TestNoArguments`, `TestExtraWordsAreRefused`, `TestUsageListsEveryCommand` |
| Every command and flag documented | `cmd` `TestCommandsDocumented` |

## The guard

| Case | Covered by | Where |
|---|---|---|
| What it stops: push, root, secrets, production, untouchable hosts and repositories | `internal/guard` (one test file per subject), `internal/hook` `TestPreToolUse`; `checks.py` | Go, rehearsal |
| What it must let through: everyday commands, in the shapes of real sessions (reading git, `sh -n`, `go test`, `gh pr checks`, pipelines, heredocs, loops, `$((…))`) | `internal/hook` `TestInnocentCommands`, over `internal/hook/testdata/innocent-commands.txt` | Go, every pull request |

A false positive found by hand goes into that corpus with its fix. The corpus is made up
in the shapes of real commands: this repository is public, so no real paths or names.

## MCP tools

| Case | Covered by |
|---|---|
| Every tool, its use and its errors | `internal/mcpserver` `TestTools` and one test per tool; `internal/brain` |
| `review_done`, `secret_name_save` through the server | `TestReviewDoneThroughMCP`, `TestSecretNameSaveThroughMCP` |
| A required parameter left out | `TestMissingParameters` |
| A secret in a note is refused | `TestSecretRefusedInATask`, `internal/brain`, `checks.py` |
| Two sessions writing at once | `TestConcurrentCalls`, `TestFreshBothWritingAtOnce` |
| Phases: skipped, closing twice, reopening | `internal/brain` `TestPhases` |
| A parent with open subtasks | `internal/brain` `TestSubtasks` |
| `task_worktree`, and closing removes the task's worktrees (made, noted, with work, in use) | `TestTaskWorktreeTool`; `internal/remote` `TestNotedWorktrees` |
| Closing a task whose own session runs | `TestCloseStopsTheTaskSession`; `internal/remote` `TestStopWhenIdle`; tried by hand on 10-nov |
| A session Claude retired (no pid), Claude's daemon stopped, a session waiting for a permission, `stop` with nothing to close | `TestCloseStopsTheTaskSession`; `internal/remote` `TestRetiredSession`, `TestWakeRetiredSession` (`session start` brings it back with `claude respawn`), `TestAgentsWithoutAll`, `TestStopWakesDaemon`, `TestWaitingForPermissionIsBusy`, `TestStopEarlierUnit`; `cmd` `TestSessionStopSaysWhatHappened`; `claude agents` outputs and the daemon measured by hand on 11-nov (2.1.322) |
| A session that cannot work: no login or an expired one, a first answer that is an API error (as Claude Code writes it when the login ran out, measured on 11-nov, 2.1.322), a slow first answer, a session whose last answer failed (`list` says failed) | `internal/state` `TestLastAnswer`; `internal/remote` `TestStartNeedsLogin`, `TestStartFirstTurnFails`, `TestStartWaitsForFirstAnswer`, `TestLaunchesFailed`; `cmd` `TestSessionListFailed` |
| Every tool and parameter documented | `TestToolsDocumented` |

## The panel

| Case | Covered by | Where |
|---|---|---|
| Every page loads, no console error, fits a phone | `scripts/panel-browser-check.py` | CI: Chromium and WebKit |
| Every edit saves | `internal/panel` (one test per page) | Go |
| Configuration edits leave a version | `internal/panel` `TestPanelChangesLeaveVersions` | Go |
| The Guard page: what changed, the box, the warnings | `TestGuardPageShowsWhatChanged`, `TestGuardPageWarnsOnDangerousLists`, the browser check | Go, CI |
| Each page says what it is; hiding it is remembered | `TestIntroHideAndShow`, `TestIntroAnchors` (the guide's anchors exist) | Go |
| Going back to a version asks, and warns about the guard | `TestVersionsPage` | Go |
| Cut-off sessions from Today; Sessions out of the menu | `TestSessions` | Go |
| The Backup page: set up (own repository, GitHub), back up now, last copies | `TestBackupPage`, `TestBackupSetupWithoutGitHub`, the browser check | Go, CI |
| What each copy changed; going back to a copy (box and name, undone by going back again, local only) | `internal/backup` `TestCopiesChangesAndGoingBack`, `TestGoBackOnlyToACopy`, `TestGoBackRefusesNotesNotValid`, panel `TestBackupGoBack` | Go |
| From another device (access, the switch) | `TestRemoteAccess`, `TestRemoteSwitchInPlace` and the others | Go |
| Firefox | `panel-browser-check.py firefox` | by hand, before a release |
| Safari for real (every page, its stylesheet, a capture) | `scripts/mac-check.py`, through safaridriver | CI: macOS |

## When something is missing

| Case | Covered by | Where |
|---|---|---|
| No backup set up | `scripts/faults-check.py`, `cmd` `TestSaysThereIsNoBackup` | CI: Linux, macOS, Windows |
| No gh, gh not logged in | `faults-check.py`, `internal/backup` `TestGitHubRepo` | CI |
| No GitHub account (GitLab, a server) | `internal/backup` (bare repositories as remotes) | Go |
| No git | `faults-check.py` | CI |
| No network, a remote that is gone or hangs | `faults-check.py`, `internal/backup` `TestBackupGivesUpOnAHangingRemote`, `TestBackupDoesNotWaitForeverForAnother` | CI, Go |
| No Claude Code | `faults-check.py`, `cmd` `TestInitWarnsWithoutThePlugin` | CI, Go |
| No Tailscale | `faults-check.py`, `internal/tailnet` `TestMissing` | CI, Go |
| No keychain, or one that does not answer | `internal/keyring` `TestClassify`, `cmd` `TestRestoreNamesTheSecrets` | Go |
| A locked macOS keychain: no password dialog, never "missing" | `scripts/mac-check.py` (only the secret's attributes are read, which needs no unlocking) | CI: macOS |
| No systemd, no launchctl, no Task Scheduler | `faults-check.py` (Linux, macOS), `internal/remote` `TestInstallErrors`, `TestLaunchdErrors`, `TestTasksErrors` | CI, Go |
| A configuration that does not load | `faults-check.py`, `cmd` `TestConfigRestoreFixesABrokenConfig` | CI, Go |
| `guard.toml` that does not load | `internal/hook` `TestPreToolUseGuardFile` | Go |
| Notes that cannot be written | `faults-check.py` (Unix) | CI |
| A full disk | not covered: writes go through a temporary file and a rename, so a failed write leaves the old note | — |

## The documentation

| Case | Covered by |
|---|---|
| Links and anchors, Spanish pages link Spanish guides | `cmd` `TestDocLinks` |
| Every guide in both languages, same headings and examples | `cmd` `TestGuidesInBothLanguages` |
| Every guide in the sidebar and in both READMEs | `cmd` `TestGuidesInTheSidebar`, `TestReadmesLinkEveryGuide` |
| Every example is a real command with real flags | `cmd` `TestDocExamples` |
| The website's pages match the guides | CI, `privacy` job |
| What the guides claim is true | by hand: a page-by-page review against the code before a release |

## The pass on a real Mac

What only a person on a real Mac can see. There is no Mac to do it on for now (decided on
9-nov: macOS is covered by CI), so these are written as
[known limitations](../docs/guide/limitations.md#macos) until someone does:

1. Install with the terminal, answering the questions; then download the program with a
   browser, open it from the Finder and see that "Open Anyway" works (`mac-check.py`
   already checks what Gatekeeper says, and that it runs from Terminal).
2. Open a real Claude Code session: the guard stops a push, the session start loads.
3. Claude Desktop installed by the installer, and Tailscale with `panel remote`.
4. Switch the network off: `backup`, `update` and `panel remote` say so and leave nothing
   half done.
5. `augmented-brain remote install` with a signed-in Claude: the session shows up in
   claude.ai, and comes back after logging out and in again, and after a reboot and a
   login (`remote-check.py` covers the rest with a pretend claude).
6. Open the panel in Safari and look at the console: safaridriver cannot read it, so
   `mac-check.py` checks the pages load whole, not that the console is quiet.

A locked keychain and Safari loading every page are no longer here: `mac-check.py` checks
them on every pull request.
