# Changelog

## Unreleased

- Memory format v0.1 (`docs/spec/memory-format.md`): a folder per type, one profile for
  the person plus an optional one per space; importing other formats is planned for 0.2.
- Go core: `internal/config` (spaces, chosen by the folder you work in) and
  `internal/memory` (reads and checks every note's header; the body can be in any language).
- `augmented-brain version`, `space` and `check`.
- `augmented-brain init` and `space add`: create the configuration, the profile and the
  spaces from templates in English or Spanish. Nothing is overwritten, and the
  assistant's own settings are not touched.
- `augmented-brain doctor`: says what is missing, and how to fix it.
- The session index, `MEMORY.md`, is generated: live tasks grouped by area.
- MCP server (`augmented-brain mcp`), for any assistant that speaks MCP: open, update,
  journal, park and close tasks; see what is pending; capture to the inbox; create
  areas, references and lessons. Calls run one at a time, also across sessions.
- Claude Code plugin `brain` (marketplace in this repository): loads the space's profile,
  rules and index when a session starts, guards commands and file writes (no commit or
  push, no root, no secrets; production AWS profiles and untouchable hosts ask first),
  and registers the MCP server. Checked end to end with a clean Claude Code: open a
  task, record a decision, park it, resume it in a new session, close it.
- `init` offers to install the Claude Code plugin; `doctor` checks it.
- Backup and restore: `augmented-brain backup setup <url>`, `backup` (or `backup --all`)
  and `restore <url>`. Each space goes to its own private repository with its
  configuration and the profile; a backup is refused while a note holds a secret.
- Releases: `install.sh` (Linux and macOS, no root, checksum verified) and archives for
  amd64 and arm64, built and published by GitHub Actions on a version tag.

## 0.0.1

First public snapshot: the project's goal, design and brand. Nothing to install yet.

- README in English and Spanish.
- Draft of the memory format (`docs/spec/memory-format.md`).
- Brand: logo, symbol, icons, README header, social card and palette, with the scripts
  that build them (`assets/brand/`) and a sheet that shows them in place (`docs/brand.html`).
  `logo-dark` and `symbol-dark` add a light border for dark backgrounds. The README header
  is the logo on a white rounded card, so it reads on GitHub's light and dark themes.
- `scripts/privacy-check.sh` and a pre-commit hook that block personal data and secrets.
- MIT license.
