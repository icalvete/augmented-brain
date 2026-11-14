package config

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/icalvete/augmented-brain/internal/i18n"
)

// Unknown is a setting of guard.toml or session.toml that this program does not know,
// so it ignores it. Those files are not refused for it, unlike config.toml: a newer
// program may add settings there, and an earlier one must keep working. But a
// misspelled name (untouchable_repo) protects nothing, so doctor and the session start
// say so.
type Unknown struct {
	File string // the file's name, e.g. "guard.toml"
	Key  string // the setting as written (its first part, for a table or a dotted key)
	Near string // a known setting it looks like a misspelling of, or ""
}

// UnknownSettings lists the settings of guard.toml and session.toml in cfgDir that this
// program does not know, file by file, in the order they are written. A missing file
// or one that cannot be read gives nothing here: LoadGuard and LoadSession report that.
func UnknownSettings(cfgDir string) []Unknown {
	var out []Unknown
	out = append(out, unknownIn(cfgDir, GuardFile, &Guard{})...)
	out = append(out, unknownIn(cfgDir, SessionFile, &Session{})...)
	return out
}

// UnknownByFile groups us by file, in order: for each file, its name and its unknown
// settings as a person reads them, with the likely right name when there is one.
func UnknownByFile(lang string, us []Unknown) [][2]string {
	var out [][2]string
	for _, u := range us {
		item := u.Key
		if u.Near != "" {
			item = i18n.T(lang, "config.unknown.near", u.Key, u.Near)
		}
		if n := len(out); n > 0 && out[n-1][0] == u.File {
			out[n-1][1] += ", " + item
			continue
		}
		out = append(out, [2]string{u.File, item})
	}
	return out
}

func unknownIn(cfgDir, name string, v any) []Unknown {
	data, err := os.ReadFile(filepath.Join(cfgDir, name))
	if err != nil {
		return nil
	}
	md, err := toml.Decode(string(data), v)
	if err != nil {
		return nil
	}
	known := tomlKeys(v)
	var out []Unknown
	var seen []string
	for _, k := range md.Undecoded() {
		if len(k) == 0 || slices.Contains(seen, k[0]) {
			continue
		}
		seen = append(seen, k[0])
		out = append(out, Unknown{File: name, Key: k[0], Near: nearest(k[0], known)})
	}
	return out
}

// tomlKeys are the setting names of the struct v points to, from its toml tags.
func tomlKeys(v any) []string {
	t := reflect.TypeOf(v).Elem()
	var keys []string
	for i := range t.NumField() {
		if name, _, _ := strings.Cut(t.Field(i).Tag.Get("toml"), ","); name != "" && name != "-" {
			keys = append(keys, name)
		}
	}
	return keys
}

// nearest is the known name that key is most likely a misspelling of: the same once
// case and - or _ are set aside, or at most two letters added, removed or changed.
// "" when none is that close.
func nearest(key string, known []string) string {
	norm := func(s string) string { return strings.ReplaceAll(strings.ToLower(s), "-", "_") }
	best, bestD := "", 3
	for _, k := range known {
		d := distance(norm(key), norm(k))
		if d < bestD {
			best, bestD = k, d
		}
	}
	return best
}

// distance is the Levenshtein distance between a and b, by bytes (setting names are
// ASCII).
func distance(a, b string) int {
	prev := make([]int, len(b)+1)
	cur := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[len(b)]
}
