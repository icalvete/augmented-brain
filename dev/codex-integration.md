# Codex integration

Plan B: when Claude's tokens run out, carry on with the same task in Codex.
Codex gets the same space at start (profile, rules, index) and enough tools to
take a task through its phases. The notes stay as they are; each client keeps
its own conversation. Claude Code is still the main client: nothing here
changes how its plugin works.

Experimental. `augmented-brain codex install` sets it up (`internal/codex`); the
user's guide is `docs/guide/codex.md`. The guard in Codex is
`hook pre-tool-use codex`: see `docs/guide/guard.md`, "In Codex", and
`scripts/codex-guard-check.py`.

## What there is

- **Start** — `augmented-brain hook session-start N codex` (or, as written by
  hand before, `AB_CODEX_START_HOOK=1 augmented-brain hook session-start N`),
  N from 1 to 9, as nine `SessionStart` hooks (`hook.CodexStartPart`). It
  gives the same parts as the Claude Code plugin, for the space of the folder
  alone: a work folder or the notes folder (`Config.SpaceOf`), never the
  default space. Part 1 adds a note on what differs from the rules, which are
  written for Claude Code: the guard, the tools there are, no SendMessage
  (`i18n/codex.go`; the guard's sentence is its own key, `codex.guard`: the
  guard runs only if the user registered `hook pre-tool-use codex`, and then
  it refuses what it would ask).
- **Tools** — `AB_MCP_TASKS_ONLY=1` in the MCP server's environment serves
  only `mcpserver.TasksOnlyTools`: where_am_i, pending, task_new, task_get,
  task_park, task_journal, task_update, task_step and task_section. A call
  that names another space is refused, and so is a folder of no space. It
  does not track which tasks changed outside the session (no `WithFresh`).
- **Probe** — `scripts/codex-check.py`, below.

What only Claude Code has stays out of the Codex start: its sessions and
transcripts (recovery, compaction), its model, Remote Control and
`session start`, old MCP servers, and the guard's notices, root exceptions
and seal, which belong to the sessions the guard protects.

## Measured

Codex CLI 0.170.0, Ubuntu 26.04, scripted local model.

- Codex cuts each hook's `additionalContext` at about **10,000 bytes of
  UTF-8**, keeping its start and end. Claude Code counts characters (UTF-16
  units). So the Codex parts are measured in bytes (`session.CodexBudget`,
  9,500); a part in Spanish that fits Claude's budget can lose its middle in
  Codex, and the probe has a rule that shows it.
- Nine `SessionStart` hooks of 9,000 bytes each arrive whole, in order.
- Codex's `SessionStart` input has `session_id`, `transcript_path`, `cwd`,
  `model`, `permission_mode` and `source`, the fields `hook.input` reads.
- The MCP server runs in the session's folder, so the space resolves as in
  Claude Code.
- A trusted `PreToolUse` hook that denies stops a shell command. Ask, a
  nonzero exit, bad JSON and a timeout let it run; an untrusted hook does
  not run at all.
- `codex exec` with an open stdin waits to read more of the prompt from it:
  run it with stdin closed.

## How `codex install` sets it up

A plugin, as for Claude Code, so its hooks have keys of their own and the user's
`hooks.json` is never touched. Everything in Codex's configuration is written by
Codex itself:

1. `codex.Write` puts a local marketplace in `<config>/codex-marketplace`:
   `.claude-plugin/marketplace.json` (Codex reads that name too) and the plugin
   `brain/` with `.codex-plugin/plugin.json`, `hooks/hooks.json` and `.mcp.json`.
   The hooks run the program by its full path (on Windows by its name); the
   guard's command exits 2 when sh says 127, so a missing program blocks
   instead of letting the command through. The MCP server has
   `AB_MCP_TASKS_ONLY=1`, `default_tools_approval_mode = "approve"` (otherwise
   Codex asks before every call) and `env_vars = ["AB_CONFIG_DIR"]` (Codex
   passes a server only a few variables).
2. `codex plugin marketplace add <dir>` and `codex plugin add brain@augmented-brain`.
   Adding again installs the new version in place.
3. `codex app-server` (JSON-RPC over stdio, `appserver.go`): `hooks/list` gives
   each hook's `key`, `currentHash` and `trustStatus` (`trusted`, `untrusted`,
   `modified`, `managed`); `config/batchWrite` writes
   `hooks.state."<key>".trusted_hash` with that hash and
   `features.unified_exec_tty = false`. It keeps the user's comments. The hook
   key has dots, so the key path quotes it; `replace` with `null` deletes one.
4. Checked again: all ten hooks `trusted`.

Measured with 0.170.0:

- Installing a plugin, from the CLI or the app server's `plugin/install`, does
  not trust its hooks. `codex plugin remove` leaves their trust behind:
  `Uninstall` removes it first (from `config/read`'s `hooks.state`).
- The hash depends only on the hook's content, not its place; it could not be
  reproduced outside Codex, hence the app server.
- A hook's key in the user's `hooks.json` is positional
  (`<path>:pre_tool_use:<group>:<hook>`): another hook put before it loses its
  trust. A plugin's are `brain@augmented-brain:hooks/hooks.json:…`.
- Codex adds `trusted_content_hash` next to `trusted_hash` by itself.

`update --plugin` (which the installer runs) installs again when the
marketplace folder exists and Codex still has the plugin, so a new program's
hooks are trusted again. `codex status` and `doctor` read the same three things:
the plugin installed and enabled (`plugin/list`), its hooks trusted
(`hooks/list`), `unified_exec_tty` false (`config/read`).

## The probe

```bash
go build -o /tmp/ab-codex-check ./cmd/augmented-brain
python3 -I scripts/codex-check.py /tmp/ab-codex-check
python3 -I scripts/codex-check.py --cross-client /tmp/ab-codex-check
```

It ends with `codex install` and `uninstall` for real, twice each, and Codex
running the plugin's hooks with real trust (no bypass). It needs `codex` (and,
with `--cross-client`, `claude`) on `PATH`, uses
throwaway homes and configuration, and no account. Not in CI: run it when
touching the Codex start, the tasks-only profile, `internal/codex` or after a
Codex update.

## Still open

- A rehearsal with real models, in the Codex desktop app as well as the CLI.
- Journal reminders, changes from other sessions and backup on close need a
  reliable link between Codex's hook events and its MCP process. In a probe,
  two desktop threads in one folder shared the MCP server's parent and got no
  session id in its environment.
