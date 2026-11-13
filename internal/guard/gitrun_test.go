package guard

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// The aliases gh and glab keep in their files are read as the commands they stand for
// (S-11 of Rev 29).
func TestSavedGhAliases(t *testing.T) {
	p := pinnedPolicy(t)
	write := func(rel, text string) {
		t.Helper()
		f := filepath.Join(p.Cwd, rel)
		if err := os.MkdirAll(filepath.Dir(f), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(f, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("GH_CONFIG_DIR", "")
	t.Setenv("GLAB_CONFIG_DIR", "")
	write(".config/gh/config.yml", "version: 1\n# a comment\naliases:\n    m: pr merge\n    co: pr checkout\n"+
		"    sh: '!gh pr merge 1'\n    \"q\": \"api graphql -f query='mutation{mergePullRequest(input:{}){clientMutationId}}'\"\n"+
		"    loop: '!gh loop'\neditor: vim\n")
	write(".config/glab-cli/aliases.yml", "ci: pipeline ci\nmm: mr merge\n")
	write("other/config.yml", "aliases:\n  x: pr merge\n")
	checkPinned(t, p, []pinned{
		{"gh m 1", Deny},
		{"gh -R o/r m 1", Deny},
		{"gh sh", Deny},
		{"gh q", Deny},
		{"gh co 1", Allow},
		{"gh pr list", Allow},
		{"gh nothing", Allow},
		{"gh loop", Ask}, // too deep: the user is asked
		{"glab mm 1", Deny},
		{"glab ci", Allow},
		{"GH_CONFIG_DIR=" + filepath.ToSlash(filepath.Join(p.Cwd, "other")) + " gh x 1", Deny},
		{"GH_CONFIG_DIR=" + filepath.ToSlash(filepath.Join(p.Cwd, "other")) + " gh m 1", Allow},
	})
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(p.Cwd, "xdg"))
	write("xdg/gh/config.yml", "aliases:\n  y: pr merge\n")
	write("xdg/glab-cli/aliases.yml", "y: mr merge\n")
	p.Env = map[string]string{"XDG_CONFIG_HOME": filepath.Join(p.Cwd, "xdg")}
	checkPinned(t, p, []pinned{{"gh y 1", Deny}, {"glab y 1", Deny}, {"gh m 1", Allow}})
	p.Env = map[string]string{"GLAB_CONFIG_DIR": filepath.Join(p.Cwd, "other")}
	write("other/aliases.yml", "z: mr merge\n")
	checkPinned(t, p, []pinned{{"glab z 1", Deny}})
}

// A push names a remote of the repository, read from its config, or somewhere else
// (S-09 of Rev 29).
func TestPushToNamedRemote(t *testing.T) {
	p := pinnedPolicy(t)
	mine := filepath.Join(p.Cwd, "repos", "mine")
	if err := os.MkdirAll(filepath.Join(mine, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	conf := "[core]\n\tbare = false\n[remote \"origin\"]\n\turl = git@github.com:me/mine.git\n" +
		"[remote \"team/fork\"]\n\turl = /tmp/x\n[remote.old]\n\turl = /tmp/y\n"
	if err := os.WriteFile(filepath.Join(mine, ".git", "config"), []byte(conf), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := remoteNames(mine); !slices.Equal(got, []string{"origin", "team/fork", "old"}) {
		t.Errorf("remoteNames = %q", got)
	}
	if got := remoteNames(filepath.Join(p.Cwd, "nothing")); got != nil {
		t.Errorf("remoteNames of nothing = %q", got)
	}
	// a folder named like a remote: the remote wins, as in git
	if err := os.MkdirAll(filepath.Join(mine, "origin"), 0o755); err != nil {
		t.Fatal(err)
	}
	checkPinned(t, p, []pinned{
		{"git -C ~/repos/mine pu_sh team/fork main", Allow},
		{"git -C ~/repos/mine pu_sh old", Allow},
		{"git -C ~/repos/mine pu_sh origin", Allow},
		{"git -C ~/repos/mine pu_sh team/other main", Ask},
		{"git -C ~/repos/mine pu_sh -- ../never", Deny},
		{"git -C ~/repos/mine pu_sh --repo=../never", Deny},
		{"git -C ~/repos/mine pu_sh ~/repos/mine", Allow},
	})
}

func TestSplitConfigParameters(t *testing.T) {
	for in, want := range map[string][]string{
		`'alias.p=push'`:                   {"alias.p=push"},
		`'alias.p'='push' 'a.b=c'`:         {"alias.p=push", "a.b=c"},
		`'it'\''s=x'`:                      {"it's=x"},
		"'a=b'\t'c=d'\n":                   {"a=b", "c=d"},
		`alias.p=push`:                     {"alias.p=push"},
		`'a=b c'  `:                        {"a=b c"},
		``:                                 nil,
		`'core.sshCommand=su do id' 'x=y'`: {"core.sshCommand=su do id", "x=y"},
	} {
		if got := splitConfigParameters(in); !slices.Equal(got, want) {
			t.Errorf("splitConfigParameters(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestYamlAlias(t *testing.T) {
	for _, c := range []struct {
		text, name, want string
		section, found   bool
	}{
		{"aliases:\n  m: pr merge\n", "m", "pr merge", true, true},
		{"aliases:\n  m: pr merge\nother:\n  n: x\n", "n", "", true, false},
		{"aliases:\n\tm: 'it''s'\n", "m", "it's", true, true},
		{"m: \"a\\\"b\"\n", "m", `a"b`, false, true},
		{"m: \"bad\\q\"\n", "m", `"bad\q"`, false, true},
		{"x: y\n", "m", "", false, false},
	} {
		got, ok := yamlAlias(strings.NewReader(c.text), c.section, c.name)
		if got != c.want || ok != c.found {
			t.Errorf("yamlAlias(%q, %q) = %q, %v; want %q, %v", c.text, c.name, got, ok, c.want, c.found)
		}
	}
}
