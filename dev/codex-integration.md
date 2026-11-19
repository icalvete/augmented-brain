# Codex integration

Plan B: when Claude's tokens run out, carry on with the same task in Codex.
Codex gets the same space at start (profile, rules, index) and enough tools to
take a task through its phases. The notes stay as they are; each client keeps
its own conversation. Claude Code is still the main client: nothing here
changes how its plugin works.

Experimental and wired by hand for now. A command to set it up is a separate
piece of work. The guard in Codex is `hook pre-tool-use codex`, registered by
hand: see the guide (`docs/guide/guard.md`, "In Codex") and
`scripts/codex-guard-check.py`.

## What there is

- **Start** — `AB_CODEX_START_HOOK=1 augmented-brain hook session-start N`,
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

## Setting it up by hand

In `$CODEX_HOME/config.toml` (the probe writes the same):

```toml
[mcp_servers.augmented-brain]
command = "augmented-brain"
args = ["mcp"]
enabled_tools = ["where_am_i", "pending", "task_new", "task_get", "task_park",
  "task_journal", "task_update", "task_step", "task_section"]

[mcp_servers.augmented-brain.env]
AB_MCP_TASKS_ONLY = "1"
```

In `$CODEX_HOME/hooks.json`, nine `SessionStart` commands,
`env AB_CODEX_START_HOOK=1 augmented-brain hook session-start N` for N from
1 to 9. Codex does not run a hook it does not trust: trust them in Codex
first.

## The probe

```bash
go build -o /tmp/ab-codex-check ./cmd/augmented-brain
python3 -I scripts/codex-check.py /tmp/ab-codex-check
python3 -I scripts/codex-check.py --cross-client /tmp/ab-codex-check
```

It needs `codex` (and, with `--cross-client`, `claude`) on `PATH`, uses
throwaway homes and configuration, and no account. Not in CI: run it when
touching the Codex start, the tasks-only profile or after a Codex update.

## Still open

- A rehearsal with real models, in the Codex desktop app as well as the CLI.
- Journal reminders, changes from other sessions and backup on close need a
  reliable link between Codex's hook events and its MCP process. In a probe,
  two desktop threads in one folder shared the MCP server's parent and got no
  session id in its environment.
