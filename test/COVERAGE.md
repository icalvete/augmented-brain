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
| A home with spaces and accents | `install-check.sh` step 6 | CI |
| macOS: shells, version, Apple's developer tools | `scripts/install-check-mac.sh` | CI |
| macOS: downloaded with a browser (Gatekeeper) | by hand | Mac pass |
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
| `remote install`, `uninstall`, `status` | `internal/remote` (14 tests) |
| `session start`, `stop`, `list` | `internal/remote` `TestStartRefuses`, `TestStartRuns`, `TestStartReportsFailure`, `TestFolderTrustedIsExact`; `internal/hook` `TestLaunchedSessionReadsItsTask`; tried by hand on Linux (a real session opened in a trusted folder) |
| `config history`, `config restore` | `cmd` `TestConfigHistoryAndRestore`, `TestConfigRestoreFixesABrokenConfig`; `internal/confighistory` |
| `update` | see Installing |
| `uninstall` | see Installing |
| `mcp` | `internal/mcpserver` |
| `hook` | `internal/hook` (34 tests), `test/rehearsal/checks.py` |
| `version`, `help`, nothing, an unknown command, extra words | `cmd` `TestNoArguments`, `TestExtraWordsAreRefused`, `TestUsageListsEveryCommand` |
| Every command and flag documented | `cmd` `TestCommandsDocumented` |

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
| Every tool and parameter documented | `TestToolsDocumented` |

## The panel

| Case | Covered by | Where |
|---|---|---|
| Every page loads, no console error, fits a phone | `scripts/panel-browser-check.py` | CI: Chromium and WebKit |
| Every edit saves | `internal/panel` (one test per page) | Go |
| Configuration edits leave a version | `internal/panel` `TestPanelChangesLeaveVersions` | Go |
| The Guard page: what changed, the box, the warnings | `TestGuardPageShowsWhatChanged`, `TestGuardPageWarnsOnDangerousLists`, the browser check | Go, CI |
| From another device (access, the switch) | `TestRemoteAccess`, `TestRemoteSwitchInPlace` and the others | Go |
| Firefox | `panel-browser-check.py firefox` | by hand, before a release |
| Safari for real | by hand | Mac pass |

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
| A locked macOS keychain asking for its password | by hand | Mac pass |
| No systemd | `faults-check.py`, `internal/remote` `TestInstallErrors` | CI, Go |
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

What only a person on a real Mac can see, before a beta or a 1.0:

1. Install with the terminal, answering the questions; then download the program with a
   browser and see what Gatekeeper says, and that "Open Anyway" works.
2. Store a secret, lock the keychain, run `augmented-brain restore` and the panel's
   Secrets page: does a password dialog appear?
3. Open a real Claude Code session: the guard stops a push, the session start loads.
4. Claude Desktop installed by the installer, and Tailscale with `panel remote`.
5. Switch the network off: `backup`, `update` and `panel remote` say so and leave nothing
   half done.
6. Open the panel in Safari and look at the console.
