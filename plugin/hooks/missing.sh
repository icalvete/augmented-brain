#!/bin/sh
# missing.sh — what the plugin's hooks do when the augmented-brain program is not
# installed (sh answered 127: command not found). Someone may install the plugin from
# Claude Code's plugin menu without the program; without this, every hook fails with
# "augmented-brain: not found" and the assistant is never told why.
#
#   missing.sh session-start <part>    part 1 tells the user and the assistant; the rest say nothing
#   missing.sh <other event>           says nothing
#
# The guard (PreToolUse) never comes here: if it cannot run, Claude Code stops the
# command, as it should.

[ "$1" = session-start ] && [ "$2" = 1 ] || exit 0

unix='curl -fsSL https://github.com/icalvete/augmented-brain/releases/latest/download/install.sh | sh'
win='irm https://github.com/icalvete/augmented-brain/releases/latest/download/install.ps1 | iex'

case "${LC_ALL:-${LC_MESSAGES:-${LANG:-}}}" in
es*)
  user="augmented brain: falta el programa augmented-brain, así que el plugin no puede hacer nada todavía. Instálalo en una terminal (Linux y macOS: $unix · Windows, en PowerShell: $win) y vuelve a abrir Claude."
  ;;
*)
  user="augmented brain: the augmented-brain program is missing, so the plugin cannot do anything yet. Install it in a terminal (Linux and macOS: $unix · Windows, in PowerShell: $win) and open Claude again."
  ;;
esac

assistant="The augmented brain plugin is installed, but the augmented-brain program it runs is not: there is no memory to load, its tools (mcp__plugin_brain_augmented-brain__*) will not work, and its guard cannot check commands. Before anything else, tell the user, in their language: augmented brain is missing its program; to install it, they paste one command in a terminal of their own, outside Claude (Linux and macOS: $unix ; Windows, in PowerShell: $win), then open Claude again. Show the command for their system in its own code block. Do not try to run the installer yourself."

printf '{"systemMessage":"%s","hookSpecificOutput":{"hookEventName":"SessionStart","additionalContext":"%s"}}\n' "$user" "$assistant"
