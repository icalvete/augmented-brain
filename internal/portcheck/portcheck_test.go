package portcheck

import (
	"os"
	"path/filepath"
	"testing"
)

// The repository's tests have none of the patterns that fail on Windows.
func TestRepositoryTests(t *testing.T) {
	// The package's folder, where go test runs it (runtime.Caller would give the path
	// of the machine that built it, not of the one that runs it).
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(wd, "..", "..")
	found, err := Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	why := map[string]string{
		"R1": "checks permission bits without looking at runtime.GOOS: Windows has no such modes",
		"R2": "writes JSON by hand around a path: on Windows its backslashes break the JSON (json.Marshal it, or filepath.ToSlash)",
	}
	for _, f := range found {
		t.Errorf("%s:%d: %s %s (or mark the line `// windows-ok`)", f.File, f.Line, f.Rule, why[f.Rule])
	}
}

// The cases that failed on Windows in CI (commits 4908b2d and 1433aad) are caught, and
// what is safe is not.
func TestRules(t *testing.T) {
	cases := []struct {
		name, src string
		want      []string
	}{
		{"mode check, no GOOS (1433aad)", `package x
func TestA(t *testing.T) {
	if fi, _ := os.Stat(p); fi.Mode().Perm() != 0o600 { t.Fail() }
}`, []string{"R1"}},
		{"mode check behind GOOS", `package x
func TestA(t *testing.T) {
	if fi, _ := os.Stat(p); runtime.GOOS != "windows" && fi.Mode().Perm() != 0o600 { t.Fail() }
}`, nil},
		{"mode check in a test that skips Windows", `package x
func TestA(t *testing.T) {
	if runtime.GOOS == "windows" { t.Skip() }
	if fi, _ := os.Stat(p); fi.Mode().Perm() != 0o600 { t.Fail() }
}`, nil},
		{"marked windows-ok", `package x
func TestA(t *testing.T) {
	_ = fi.Mode().Perm() // windows-ok: only compared on Unix below
}`, nil},
		{"a symlink check is not a permission check", `package x
func TestA(t *testing.T) {
	if fi.Mode()&os.ModeSymlink == 0 { t.Fail() }
}`, nil},
		{"JSON by hand around a path (4908b2d)", "package x\nfunc f(dir string) string {\n\treturn `{\"projects\":{\"` + dir + `\":{\"ok\":true}}}`\n}", []string{"R2"}},
		{"JSON with the path through ToSlash", "package x\nfunc f(work string) string {\n\treturn `{\"cwd\":\"` + filepath.ToSlash(work) + `\"}`\n}", nil},
		{"JSON around something that is not a path", "package x\nfunc f(id string) string {\n\treturn `{\"session_id\":\"` + id + `\"}`\n}", nil},
		{"a join that is not JSON", "package x\nfunc f(dir string) string {\n\treturn \"cd \" + dir\n}", nil},
	}
	for _, c := range cases {
		got, err := File("x_test.go", []byte(c.src))
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		var rules []string
		for _, f := range got {
			rules = append(rules, f.Rule)
		}
		if len(rules) != len(c.want) || (len(rules) > 0 && rules[0] != c.want[0]) {
			t.Errorf("%s: got %v, want %v", c.name, rules, c.want)
		}
	}
}
