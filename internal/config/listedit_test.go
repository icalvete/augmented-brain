package config

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// A list the way people keep it by hand: many lines, several values on a line, a
// comment after a value and comments between values.
const handList = `# my settings
language = "en"
production_aws_profiles = ["prod", "root"]
untouchable_hosts = [
  "db01",  # the main one
  "db02",
]

[spaces.work]
memory = "~/notes"
work_folders = ["~/work"]
# the ask list, by hand
git_ask_repos = [
  "~/work/alpha",  # 2026-11-01, alpha (task-a)
  "~/work/beta", "~/work/gamma",
  "~/work/delta", "~/work/delta-*",
  # 2026-11-02, why epsilon
  "~/work/epsilon",
  "~/work/zeta"
]

[spaces.home]
memory = "~/home-notes"
`

func readFile(t *testing.T, dir string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, FileName))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// askList is what the file reads back as the work space's ask list.
func askList(t *testing.T, dir string) []string {
	t.Helper()
	c, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	return tildeAll(c.Spaces["work"].GitAskRepos)
}

func listHome(t *testing.T) {
	home, _ := filepath.EvalSymlinks(t.TempDir())
	t.Setenv("HOME", home)
	for _, d := range []string{"work", "notes", "home-notes"} {
		if err := os.MkdirAll(filepath.Join(home, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
}

// A list of many lines takes a value out, and only its own: the comment that was
// after it goes with its line, every other comment stays.
func TestListRemoveKeepsTheRest(t *testing.T) {
	listHome(t)
	dir := t.TempDir()
	write(t, dir, handList)
	cur := askList(t, dir)
	for _, gone := range []string{"~/work/alpha", "~/work/gamma", "~/work/zeta", "~/work/delta"} {
		out := slices.DeleteFunc(slices.Clone(cur), func(s string) bool { return s == gone })
		if err := SetSpaceList(dir, "work", "git_ask_repos", out); err != nil {
			t.Fatalf("taking out %s: %v", gone, err)
		}
		if got := askList(t, dir); !slices.Equal(got, out) {
			t.Fatalf("taking out %s: %v, want %v", gone, got, out)
		}
		cur = out
	}
	want := `git_ask_repos = [
  "~/work/beta",
  "~/work/delta-*",
  # 2026-11-02, why epsilon
  "~/work/epsilon",
]`
	if got := readFile(t, dir); !strings.Contains(got, want) {
		t.Errorf("the list now:\n%s\nwant it to hold:\n%s", got, want)
	}
	for _, keep := range []string{"# my settings", "# the ask list, by hand", `"db01",  # the main one`, "[spaces.home]"} {
		if !strings.Contains(readFile(t, dir), keep) {
			t.Errorf("%q is gone", keep)
		}
	}
}

// A value added goes right before the closing bracket, with the indent of the others
// and a comma for the last one; with a reason, the day and the reason after it.
func TestListAddBeforeTheBracket(t *testing.T) {
	listHome(t)
	dir := t.TempDir()
	write(t, dir, handList)
	home := os.Getenv("HOME")
	// Given in full, a value already there in "~" form is not added twice.
	out := append(askList(t, dir), filepath.Join(home, "work", "beta"), "~/work/eta")
	if err := SetSpaceListWhy(dir, "work", "git_ask_repos", out, "  eta,\nfor the demo "); err != nil {
		t.Fatal(err)
	}
	day := time.Now().Format("2006-01-02")
	want := "  \"~/work/zeta\",\n  \"~/work/eta\",  # " + day + ", eta, for the demo\n]\n\n[spaces.home]"
	got := readFile(t, dir)
	if !strings.Contains(got, want) {
		t.Errorf("the list now:\n%s\nwant it to end:\n%s", got, want)
	}
	if strings.Count(got, "work/beta") != 1 {
		t.Errorf("beta written twice:\n%s", got)
	}
	if l := askList(t, dir); l[len(l)-1] != "~/work/eta" || len(l) != 8 {
		t.Errorf("read back: %v", l)
	}
}

// The shapes a list can have: each edit reads back as the list asked for, and a list
// on one line stays on one line.
func TestListShapes(t *testing.T) {
	cases := []struct {
		name, before, why string
		values            []string
		after             string
	}{
		{"add to one line", `k = ["a", "b"]`, "", []string{"a", "b", "c"}, `k = ["a", "b", "c"]`},
		{"add to empty", `k = []`, "", []string{"a"}, `k = ["a"]`},
		{"add to empty with blank", `k = [ ]  # none yet`, "", []string{"a"}, `k = ["a"]  # none yet`},
		{"one line with trailing comma", `k = ["a",]`, "", []string{"a", "b"}, `k = ["a", "b"]`},
		{"remove first", `k = ["a", "b", "c"]`, "", []string{"b", "c"}, `k = ["b", "c"]`},
		{"remove last", `k = ["a", "b", "c"]`, "", []string{"a", "b"}, `k = ["a", "b"]`},
		{"remove only", `k = ["a"]`, "", nil, `k = []`},
		{"same list", "k = [\n  \"a\",\n]", "", []string{"a"}, "k = [\n  \"a\",\n]"},
		{"bracket after the last", "k = [\n  \"a\",\n  \"b\"]", "", []string{"a", "b", "c"}, "k = [\n  \"a\",\n  \"b\",\n  \"c\",\n  ]"},
		{"no comma on the last", "k = [\n    \"a\"\n]", "", []string{"a", "b"}, "k = [\n    \"a\",\n    \"b\",\n]"},
		{"first line holds values", "k = [\"a\",\n  \"b\",\n]", "", []string{"b"}, "k = [\n  \"b\",\n]"},
		{"remove on the closing line", "k = [\n  \"a\",\n  \"b\"]", "", []string{"a"}, "k = [\n  \"a\",\n  ]"},
		{"one line, with a reason", `k = ["a"]`, "why", []string{"a", "b"}, "k = [\"a\",\n  \"b\",  # DAY, why\n]"},
		{"literal strings", `k = ['a', 'b']`, "", []string{"b"}, `k = ['b']`},
		{"a comment with a bracket", "k = [\n  \"a\",  # see [x]\n]", "", []string{"a", "b"}, "k = [\n  \"a\",  # see [x]\n  \"b\",\n]"},
		{"a value with a bracket", `k = ["a]", "b"]`, "", []string{"b"}, `k = ["b"]`},
		{"not strings: one line again", "k = [\n  \"\"\"a\"\"\",\n]", "", []string{"a", "b"}, `k = ["a", "b"]`},
	}
	day := time.Now().Format("2006-01-02")
	for _, c := range cases {
		f := &lineEditor{lines: strings.Split(c.before+"\nnext = 1", "\n")}
		f.setList(0, "k", c.values, whyComment(c.why))
		got := strings.Join(f.lines, "\n")
		want := strings.ReplaceAll(c.after, "DAY", day) + "\nnext = 1"
		if got != want {
			t.Errorf("%s:\n%s\nwant:\n%s", c.name, got, want)
		}
	}
}

// A machine-wide list of many lines is edited in place too, and taking a setting out
// takes all its lines.
func TestTopListOfManyLines(t *testing.T) {
	listHome(t)
	dir := t.TempDir()
	write(t, dir, handList)
	if err := SetTopList(dir, "untouchable_hosts", []string{"db02", "db03"}); err != nil {
		t.Fatal(err)
	}
	if want := "untouchable_hosts = [\n  \"db02\",\n  \"db03\",\n]"; !strings.Contains(readFile(t, dir), want) {
		t.Errorf("hosts:\n%s", readFile(t, dir))
	}
	if err := RemoveTopKey(dir, "untouchable_hosts"); err != nil {
		t.Fatal(err)
	}
	if err := RemoveSpaceKey(dir, "work", "git_ask_repos"); err != nil {
		t.Fatal(err)
	}
	got := readFile(t, dir)
	if strings.Contains(got, "db0") || strings.Contains(got, "work/") {
		t.Errorf("lines left behind:\n%s", got)
	}
	if _, err := Load(dir); err != nil {
		t.Fatal(err)
	}
}

// A list not in the file yet, added with a reason, takes one value a line.
func TestNewListWithReason(t *testing.T) {
	listHome(t)
	dir := t.TempDir()
	write(t, dir, handList)
	if err := SetSpaceListWhy(dir, "home", "git_ask_repos", []string{"~/work/x"}, "x"); err != nil {
		t.Fatal(err)
	}
	day := time.Now().Format("2006-01-02")
	if want := "[spaces.home]\ngit_ask_repos = [\n  \"~/work/x\",  # " + day + ", x\n]"; !strings.Contains(readFile(t, dir), want) {
		t.Errorf("new list:\n%s", readFile(t, dir))
	}
}
