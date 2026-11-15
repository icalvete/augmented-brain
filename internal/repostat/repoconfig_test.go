package repostat

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// A repository's own configuration may name programs. Reading it must run none of
// them: a folder unpacked from an archive brings its .git/config, and the panel reads
// it just by being opened. Each case only writes an empty marker file.
func TestReadRunsNoProgramOfTheRepository(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git")
	}
	if runtime.GOOS == "windows" {
		t.Skip("the programs are shell commands; the code does not depend on the system")
	}
	cases := []struct {
		name  string
		setup func(t *testing.T, git func(...string), repo, marker string)
		attrs string
	}{
		{"core.fsmonitor", func(t *testing.T, git func(...string), repo, m string) {
			git("config", "core.fsmonitor", "touch "+m+"; false")
		}, ""},
		{"clean filter", func(t *testing.T, git func(...string), repo, m string) {
			git("config", "filter.x.clean", "touch "+m+"; cat")
		}, "* filter=x\n"},
		{"process filter", func(t *testing.T, git func(...string), repo, m string) {
			git("config", "filter.x.process", "touch "+m+"; false")
		}, "* filter=x\n"},
		{"required filter with dots in its name", func(t *testing.T, git func(...string), repo, m string) {
			git("config", "filter.a.b.clean", "touch "+m+"; cat")
			git("config", "filter.a.b.required", "true")
		}, "* filter=a.b\n"},
		{"filter in an included file", func(t *testing.T, git func(...string), repo, m string) {
			inc := filepath.Join(filepath.Dir(repo), "inc")
			if err := os.WriteFile(inc, []byte("[filter \"x\"]\n\tclean = touch "+m+"; cat\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			git("config", "include.path", inc)
		}, "* filter=x\n"},
		{"filter in the worktree's configuration", func(t *testing.T, git func(...string), repo, m string) {
			git("config", "extensions.worktreeConfig", "true")
			git("config", "--worktree", "filter.x.clean", "touch "+m+"; cat")
		}, "* filter=x\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			repo := filepath.Join(dir, "repo")
			marker := filepath.Join(dir, "RAN")
			git := newRepo(t, repo)
			c.setup(t, git, repo, marker)
			touchSameSize(t, repo, c.attrs)
			os.Remove(marker)
			st := Read(repo)
			if _, err := os.Stat(marker); err == nil {
				t.Fatalf("a program of the repository ran: %+v", st)
			}
			if st.Err != "" || st.Changed == 0 {
				t.Fatalf("the repository should read as changed: %+v", st)
			}
		})
	}
}

// A filter of the user's own configuration (git-lfs, say) still works.
func TestReadKeepsTheUsersFilters(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git")
	}
	if runtime.GOOS == "windows" {
		t.Skip("the programs are shell commands; the code does not depend on the system")
	}
	dir := t.TempDir()
	repo := filepath.Join(dir, "repo")
	marker := filepath.Join(dir, "RAN")
	global := filepath.Join(dir, "gitconfig")
	if err := os.WriteFile(global, []byte("[filter \"mine\"]\n\tclean = touch "+marker+"; cat\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", global)
	newRepo(t, repo)
	touchSameSize(t, repo, "* filter=mine\n")
	os.Remove(marker)
	Read(repo)
	if _, err := os.Stat(marker); err != nil {
		t.Fatal("the user's own filter should still run")
	}
}

// newRepo makes a repository with one committed file and returns a git runner for it.
func newRepo(t *testing.T, repo string) func(...string) {
	t.Helper()
	if err := os.MkdirAll(repo, 0o700); err != nil {
		t.Fatal(err)
	}
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	git("init", "-q")
	if err := os.WriteFile(filepath.Join(repo, "a.txt"), []byte("one\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	git("add", "a.txt")
	git("-c", "user.email=ana@example.com", "-c", "user.name=Ana", "commit", "-q", "-m", "x")
	return git
}

// touchSameSize writes the attributes and changes a.txt keeping its size, as after
// unpacking an archive: git has to read the file, through its filter, to compare it.
func touchSameSize(t *testing.T, repo, attrs string) {
	t.Helper()
	if attrs != "" {
		if err := os.WriteFile(filepath.Join(repo, ".gitattributes"), []byte(attrs), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(repo, "a.txt"), []byte("two\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}
