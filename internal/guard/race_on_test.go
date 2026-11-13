//go:build race

package guard

// raceEnabled is true when the tests run with -race, which makes the long comparisons
// about twenty times slower: they then try fewer texts.
const raceEnabled = true
