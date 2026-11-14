package config

import (
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"strings"
)

// GuardFingerprint is a hash of everything the guard answers to: the repositories it
// lets through, asks about or never allows, the root exceptions, the production
// profiles and the untouchable hosts. The order of a list does not change it, and
// nothing else of the configuration does. Where augmented brain acts (Guard.Reach) is
// in it only when it keeps to the spaces: the hash of the default is the one earlier
// programs compute, so they keep trusting the lists; with ReachSpaces they find the
// hash changed and ask before a commit, a push or a merge, the safe side.
func (c *Config) GuardFingerprint(g Guard) string {
	allowed, ask := c.GitLists()
	profiles, hosts := c.GuardLists()
	roots := append([]string{}, c.RootAsk...)
	h := sha256.New()
	for _, part := range []struct {
		name string
		list []string
	}{{"allowed", allowed}, {"ask", ask}, {"untouchable", g.UntouchableRepos},
		{"root", roots}, {"profiles", profiles}, {"hosts", hosts}} {
		l := slices.Sorted(slices.Values(unique(part.list)))
		h.Write([]byte(part.name + "\x00" + strings.Join(l, "\x00") + "\x01"))
	}
	if g.OnlySpaces() {
		h.Write([]byte("reach\x00" + ReachSpaces + "\x01"))
	}
	return hex.EncodeToString(h.Sum(nil))
}
