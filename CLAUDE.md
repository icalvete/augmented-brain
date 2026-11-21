@AGENTS.md

## Claude Code

- To try a change to the plugin or the program without installing it:
  `scripts/try-plugin.sh` (`--prepare` to set it up without starting Claude). Not
  `claude --plugin-dir plugin` alone: its hooks and MCP server would run the installed
  program from `~/.local/bin`.
- The plugin's tools show up as `mcp__plugin_brain_augmented-brain__<tool>`.
