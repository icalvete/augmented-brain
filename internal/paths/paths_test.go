package paths

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// The macOS case, reproduced anywhere: a folder reached through a symlink, and paths
// below it that do not exist yet, must resolve to the same real place.
func TestResolveThroughASymlink(t *testing.T) {
	root := t.TempDir()
	real := filepath.Join(root, "private", "var")
	if err := os.MkdirAll(real, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "var")
	if err := os.Symlink(real, link); err != nil {
		if runtime.GOOS == "windows" {
			t.Skip("symlinks need Developer Mode or an administrator on Windows:", err)
		}
		t.Fatal(err)
	}
	realRoot, _ := filepath.EvalSymlinks(root)

	exists, _ := Resolve(link)
	missing, _ := Resolve(filepath.Join(link, "work", "project", "src"))
	if exists != filepath.Join(realRoot, "private", "var") {
		t.Errorf("existing: %s", exists)
	}
	if missing != filepath.Join(realRoot, "private", "var", "work", "project", "src") {
		t.Errorf("missing: %s", missing)
	}
	if !Within(missing, exists) {
		t.Error("a missing folder under a symlinked one is not inside it")
	}

	t.Setenv("HOME", link)
	t.Setenv("USERPROFILE", link) // where Windows takes the home folder from
	if got := Tilde(filepath.Join(link, "brain")); got != "~/brain" {
		t.Errorf("Tilde through the link = %s", got)
	}
	if got := Tilde(filepath.Join(real, "brain")); got != "~/brain" {
		t.Errorf("Tilde of the real path = %s", got)
	}
	if got, _ := Resolve("~/x"); got != filepath.Join(realRoot, "private", "var", "x") {
		t.Errorf("~ = %s", got)
	}
}

func TestSame(t *testing.T) {
	a := filepath.Join(t.TempDir(), "notes")
	if !Same(a, a) || Same(a, filepath.Join(a, "sub")) || Same(a, a+"x") {
		t.Error("Same: wrong answer for the same, a child and a sibling folder")
	}
	if runtime.GOOS == "windows" && !Same(a, strings.ToUpper(a)) {
		t.Error("on Windows, the same folder in other case must be the same")
	}
}

// On macOS and Windows a folder written in another case is the same folder.
func TestWithinIgnoresCaseWhereTheSystemDoes(t *testing.T) {
	old := foldCase
	t.Cleanup(func() { foldCase = old })
	folder := filepath.Join(string(filepath.Separator)+"Users", "Ana", "Notes")
	inside := filepath.Join(string(filepath.Separator)+"users", "ana", "notes", "tasks")
	foldCase = true
	if !Within(inside, folder) || !Same(strings.ToUpper(folder), folder) {
		t.Error("case-insensitive systems: not matched")
	}
	if runtime.GOOS == "windows" {
		return // filepath.Rel itself ignores case on Windows
	}
	foldCase = false
	if Within(inside, folder) {
		t.Error("case-sensitive systems: matched")
	}
}

// A Resolver gives Resolve's answer for every shape of path: folders and files reached
// through links (absolute, relative, chained), broken links, missing parts below a link,
// the home folder, and the same path asked twice. On Windows the last part may differ
// in case only, so there the paths are compared with Same.
func TestResolverMatchesResolve(t *testing.T) {
	root := t.TempDir()
	mk := func(p string) {
		if err := os.MkdirAll(filepath.Join(root, p), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	file := func(p string) {
		if err := os.WriteFile(filepath.Join(root, p), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	link := func(target, name string) {
		if err := os.Symlink(target, filepath.Join(root, name)); err != nil {
			if runtime.GOOS == "windows" {
				t.Skip("symlinks need Developer Mode or an administrator on Windows:", err)
			}
			t.Fatal(err)
		}
	}
	mk("real/plugin/skills")
	file("real/plugin/skills/a.md")
	file("real/top.md")
	link(filepath.Join(root, "real"), "abs")              // a folder, absolute target
	link("real", "rel")                                   // a folder, relative target
	link("rel", "chain")                                  // a link to a link
	link(filepath.Join("..", "top.md"), "real/plugin/up") // a file, relative with ..
	link(filepath.Join(root, "nowhere"), "broken")        // a link to nothing
	mk("other")
	link(filepath.Join("..", "real", "plugin"), "other/plug")

	t.Setenv("HOME", root)
	var cases []string
	for _, base := range []string{"real", "abs", "rel", "chain", "broken", "other/plug", "missing"} {
		for _, rest := range []string{"", "plugin", "plugin/skills/a.md", "plugin/up", "plugin/skills/new/deep.md", "top.md", "skills/a.md"} {
			cases = append(cases, filepath.Join(root, base, filepath.FromSlash(rest)))
		}
	}
	cases = append(cases, "~", "~/abs/plugin", "~/broken/x", "/", filepath.Join(root, "abs", "..", "real", "top.md"))

	var r Resolver
	for round := 0; round < 2; round++ { // the second round answers from memory
		for _, c := range cases {
			want, werr := Resolve(c)
			got, gerr := r.Resolve(c)
			if (werr == nil) != (gerr == nil) {
				t.Fatalf("%s: errors differ: %v, %v", c, werr, gerr)
			}
			if got != want && !(runtime.GOOS == "windows" && Same(got, want)) {
				t.Errorf("%s: Resolver %q, Resolve %q", c, got, want)
			}
		}
	}
}

// What the Resolver saves: a hundred files of a missing folder ten levels deep resolve
// that folder once, not a hundred times, each from the top.
func BenchmarkResolverManyFiles(b *testing.B) {
	dir := filepath.Join(b.TempDir(), "a", "b", "c", "d", "e", "f", "g", "h", "i", "j")
	for i := 0; i < b.N; i++ {
		var r Resolver
		for n := 0; n < 100; n++ {
			if _, err := r.Resolve(filepath.Join(dir, strings.Repeat("x", n%7+1)+".md")); err != nil {
				b.Fatal(err)
			}
		}
	}
}
