<!-- Detail for AI coding agents and contributors. English only: see AGENTS.md. -->

# Commits and releases, in detail

- Small commits that stand on their own, with a message that says why. Nothing is pushed
  to `main` that the website should not show yet: Pages publishes `main` right away, so
  try work on a branch, with `scripts/precheck.sh` and then a pull request (CI runs in
  pull requests, not on bare branches). Every CI job stops at 10 minutes, so a stuck
  runner fails instead of waiting for hours.
- Before a release, the rehearsal in the VMs: `test/rehearsal/rehearse.sh` (Ubuntu and
  Windows, from `clean`, about 5 minutes each), and again with `--update` (it installs the
  release published last first, then this one over it) to rehearse the update; once the
  release is out, `--update-from <version>` names the one before it. A new version is installed on a real machine
  only after it. What a release brings that its checks do not cover yet goes into
  `test/rehearsal/checks.py` with it.
- `main` takes changes only through a pull request whose CI is green and up to date with
  `main` (a GitHub ruleset; no force push, no deletion, no one bypasses it). Tags `v*`
  cannot be moved or deleted once pushed.
- A release: on a branch, set `VERSION`, run `python3 -I scripts/changelog.py release
  <version>` (it writes `## <version>` with its date, `Released <date>.` and `Publicada el
  <date>.`, in `CHANGELOG.md` and `CHANGELOG.es.md` with every entry of `changelog.d/`,
  and empties the folder, then regenerates the guides, whose sidebar shows the version last released), commit, open a pull request and merge it when CI is green.
  `release.yml` refuses a release while `changelog.d/` still holds entries. Publishing is then the maintainer's one gesture: Run workflow on
  `release` in GitHub's Actions tab (or `gh workflow run release -R icalvete/augmented-brain`).
  `release.yml` takes the version from `VERSION` on `main` and the commit that last set it
  or changed its section of the changelog (`scripts/release-commit.sh`: what merged
  later waits for the next version), refuses one already tagged, a tag on a commit not
  on `main`, or a commit where any required check, or a macOS one, has not passed
  (`scripts/release-checks.sh`); then a read-only job with no cache builds the archives,
  the installers and `checksums.txt` (reproducible: the same commit and Go give the same
  bytes), and another, in the environment `release`, attests their provenance and
  publishes them; the installer always takes the latest release. Two settings are the
  maintainer's, in GitHub: immutable releases, and actions pinned to a full commit SHA
  (every workflow already pins them; `zizmor` and `actionlint` check the workflows in
  `ci.yml`). An
  assistant prepares the branch and may start that workflow only when the maintainer
  asks for it: the guard asks the maintainer each time (pushing a tag `v<version>` by
  hand still works too).
