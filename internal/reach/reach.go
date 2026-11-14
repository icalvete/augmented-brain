// Package reach says where augmented brain acts: in every Claude Code session of the
// computer (the default), or only in the folders of the spaces (guard.toml's reach =
// "spaces"). Every hook, the MCP server and doctor ask it the same way.
package reach

import (
	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/state"
)

// OnlySpaces says whether augmented brain keeps to the spaces' folders. It does only
// when guard.toml says so and the guard's lists are the ones the user last saw (the
// seal): reach can switch half the guard off, so a change nobody accepted, or one the
// guard cannot check, counts as the whole computer, the safe side.
func OnlySpaces(cfgDir string, c *config.Config) bool {
	if c == nil {
		return false
	}
	g, err := config.LoadGuard(cfgDir)
	if err != nil || !g.OnlySpaces() {
		return false
	}
	seal, err := state.ReadSeal(cfgDir)
	return err == nil && seal != "" && seal == c.GuardFingerprint(g)
}

// Outside says whether a session in dir is outside every space while augmented brain
// keeps to them: it then stays quiet there.
func Outside(cfgDir string, c *config.Config, dir string) bool {
	return OnlySpaces(cfgDir, c) && !c.InSpace(dir)
}
