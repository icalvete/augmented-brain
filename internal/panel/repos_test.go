package panel

import (
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func gitInit(t *testing.T, dir string, args ...string) {
	t.Helper()
	if out, err := exec.Command("git", "init", "-q", dir).CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	if len(args) > 0 {
		if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("%v %s", err, out)
		}
	}
}

// The Repositories page lists every repository of each space's work folders, pending
// ones first, with the remote cleaned, the tasks that name it and what it is; what is
// described there is kept in the notes; and every text is escaped.
func TestReposPage(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git")
	}
	s, cfg := fixture(t, "en")
	home := filepath.Dir(cfg)
	work := filepath.Join(home, "work")
	gitInit(t, filepath.Join(work, "zeta"), "remote", "add", "origin", "https://ana:s3cret@example.com/ana/zeta.git")
	if err := os.WriteFile(filepath.Join(work, "zeta", "new.txt"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitInit(t, filepath.Join(work, "group", "alpha"))
	gitInit(t, filepath.Join(work, "node_modules", "dep"))
	gitInit(t, filepath.Join(work, "<b>odd</b>"))
	gitInit(t, filepath.Join(home, "home", "garden"))
	gitInit(t, filepath.Join(home, "elsewhere")) // in no work folder

	id := idIn(saved(t, s.post(t, "/task/new", url.Values{"s": {"work"}, "title": {"Ship <zeta>"}}), "new task"))
	path := filepath.Join(home, "brain-work", "tasks", id+".md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	with := strings.Replace(string(data), "\ntype: task\n", "\ntype: task\nrepos:\n  - zeta\n", 1)
	if err := os.WriteFile(path, []byte(with), 0o600); err != nil {
		t.Fatal(err)
	}

	saved(t, s.post(t, "/repo/describe", url.Values{"s": {"work"}, "repo": {"~/work/group/alpha"}, "text": {"First <script>alert(1)</script>"}}), "describe")
	if got := openSpace(t, cfg, "work").RepoDescriptionOf(filepath.Join(work, "group", "alpha")); got != "First <script>alert(1)</script>" {
		t.Errorf("the description was not kept: %q", got)
	}

	page := s.get(t, "/repos?s=work").Body.String()
	for _, want := range []string{
		"Space work", "Space home", ">garden ", ">alpha ",
		"example.com/ana/zeta", "1 files changed", "~/work/group/alpha",
		`value="First &lt;script&gt;alert(1)&lt;/script&gt;"`,
		"Ship &lt;zeta&gt;", "&lt;b&gt;odd&lt;/b&gt;", `placeholder="describe it"`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("the page lacks %q", want)
		}
	}
	for _, bad := range []string{"s3cret", "ana@", "<script>", "<b>odd", "node_modules", "elsewhere", "brain-work"} {
		if strings.Contains(page, bad) {
			t.Errorf("the page shows %q", bad)
		}
	}
	// Pending first: zeta, with a file changed, comes before alpha.
	if strings.Index(page, ">zeta ") > strings.Index(page, ">alpha ") {
		t.Error("the repository with pending work is not first")
	}

	// Today keeps only what is pending, links to the page and asks to describe it.
	today := s.get(t, "/today?s=work").Body.String()
	for _, want := range []string{"See all repositories", "describe it"} {
		if !strings.Contains(today, want) {
			t.Errorf("today lacks %q", want)
		}
	}
	if strings.Contains(today, ">alpha ") {
		t.Error("today shows a repository with nothing pending")
	}

	// A folder outside the work folders cannot be described from the panel either.
	if w := s.post(t, "/repo/describe", url.Values{"s": {"work"}, "repo": {filepath.Join(home, "elsewhere")}, "text": {"no"}}); w.Code != 400 {
		t.Errorf("describing a folder outside: %d", w.Code)
	}
}

// The Spanish page reads in Spanish.
func TestReposPageSpanish(t *testing.T) {
	s, _ := fixture(t, "es")
	page := s.get(t, "/repos?s=work").Body.String()
	if !strings.Contains(page, "Repositorios") || !strings.Contains(page, "No hay ningún repositorio git") {
		t.Errorf("the page is not in Spanish:\n%s", page)
	}
}
