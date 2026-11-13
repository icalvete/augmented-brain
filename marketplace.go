// Package augmentedbrain carries the Claude Code plugin inside the program: its
// marketplace file and the plugin folder, as they are in this repository. The program
// writes them to a folder of its own and has Claude Code take the plugin from there, so
// the hooks a person runs are always the ones of the program they installed, never
// whatever is on main today.
package augmentedbrain

import "embed"

// Marketplace is .claude-plugin/marketplace.json and plugin/, with its dot files.
//
//go:embed .claude-plugin/marketplace.json all:plugin
var Marketplace embed.FS
