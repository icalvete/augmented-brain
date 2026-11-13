// Package stable keeps what the compatibility promise covers (docs/guide/compatibility.md)
// from changing by accident: the MCP tools and their parameters, the commands and their
// flags, the settings of the configuration files. Each is frozen in a file under its
// package's testdata, and its test compares the program with it. Only tests use this
// package.
//
// What is frozen may only grow inside 1.x. Something missing from the program, or
// changed, is a break: the test fails, and the file is never rewritten to hide it. Something
// new fails too, until it is frozen with AB_STABLE_UPDATE=1, so that later removing it is
// caught.
package stable

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"testing"
)

// UpdateEnv, set to 1, adds what is new to the frozen file instead of failing. It never
// removes anything from it.
const UpdateEnv = "AB_STABLE_UPDATE"

// Surface is what the program offers: a name (a tool, a command, a file) with its
// attributes (its parameters, its flags, its settings), each one a string.
type Surface map[string][]string

// Check compares now with the file frozen at path. Every name and attribute frozen must
// still be in now. breaks are other breaks the caller found (a parameter that became
// required): each fails the test, and with any break the file is not updated.
func Check(t testing.TB, path string, now Surface, breaks ...string) {
	t.Helper()
	frozen := Read(t, path)
	for _, m := range Missing(frozen, now) {
		breaks = append(breaks, m+" is gone")
	}
	for _, b := range breaks {
		t.Errorf("%s: %s. The compatibility promise (docs/guide/compatibility.md) keeps it for every 1.x: keep the old name working, or this waits for 2.0", path, b)
	}
	added := Missing(now, frozen)
	if len(added) == 0 {
		return
	}
	if os.Getenv(UpdateEnv) != "1" || len(breaks) > 0 {
		for _, a := range added {
			t.Errorf("%s: new, not frozen yet: %s. Freeze it with %s=1 go test ./... (it then stays for every 1.x)", path, a, UpdateEnv)
		}
		return
	}
	for name, attrs := range now {
		for _, a := range attrs {
			if !slices.Contains(frozen[name], a) {
				frozen[name] = append(frozen[name], a)
			}
		}
		if _, ok := frozen[name]; !ok {
			frozen[name] = []string{}
		}
	}
	for name := range frozen {
		sort.Strings(frozen[name])
	}
	out, err := json.MarshalIndent(frozen, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(out, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("%s: froze %d new entries", path, len(added))
}

// Read is the surface frozen at path. A missing file is empty only while freezing
// (UpdateEnv): otherwise the test could pass by losing the file.
func Read(t testing.TB, path string) Surface {
	t.Helper()
	frozen := Surface{}
	data, err := os.ReadFile(path)
	switch {
	case err == nil:
		if err := json.Unmarshal(data, &frozen); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
	case os.IsNotExist(err) && os.Getenv(UpdateEnv) == "1":
	default:
		t.Fatal(err)
	}
	return frozen
}

// Missing is what a has that b lacks, sorted: a name, or a name and an attribute.
func Missing(a, b Surface) []string {
	var out []string
	for name, attrs := range a {
		battrs, ok := b[name]
		if !ok {
			out = append(out, fmt.Sprintf("%q", name))
			continue
		}
		for _, x := range attrs {
			if !slices.Contains(battrs, x) {
				out = append(out, fmt.Sprintf("%q of %q", x, name))
			}
		}
	}
	sort.Strings(out)
	return out
}
