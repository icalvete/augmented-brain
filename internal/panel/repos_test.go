package panel

import (
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/setup"
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
	// A folder whose name would be markup if it were not escaped. Windows allows no < >
	// in a name: there, & and ' have to be escaped instead.
	odd, oddShown := "<b>odd</b>", "&lt;b&gt;odd&lt;/b&gt;"
	if runtime.GOOS == "windows" {
		odd, oddShown = "a&b'odd", "a&amp;b&#39;odd"
	}
	gitInit(t, filepath.Join(work, odd))
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
		"example.com/ana/zeta", "1 file changed", "~/work/group/alpha",
		`value="First &lt;script&gt;alert(1)&lt;/script&gt;"`,
		"Ship &lt;zeta&gt;", oddShown, `placeholder="describe it"`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("the page lacks %q", want)
		}
	}
	for _, bad := range []string{"s3cret", "ana@", "<script>", odd, "node_modules", "elsewhere", "brain-work"} {
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

// A work folder reached through a link (macOS's /var is /private/var; or one the user
// linked into their home) still shows its repositories and keeps what is said of them.
// A link inside it to a repository elsewhere is followed neither by the search nor for
// a task that names it.
func TestReposThroughLinks(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git")
	}
	if runtime.GOOS == "windows" {
		t.Skip("symbolic links need privileges on Windows")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	real := filepath.Join(home, "disk", "work")
	gitInit(t, filepath.Join(real, "site"))
	gitInit(t, filepath.Join(home, "outside-repo"))
	work := filepath.Join(home, "work")
	if err := os.Symlink(real, work); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(home, "outside-repo"), filepath.Join(real, "linked")); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(home, "cfg")
	if _, err := setup.Init(cfg, setup.Options{Language: "en", Space: "work", WorkFolder: work, Memory: "~/brain", Today: "2026-10-15"}); err != nil {
		t.Fatal(err)
	}
	s, err := New(cfg, port)
	if err != nil {
		t.Fatal(err)
	}
	id := idIn(saved(t, s.post(t, "/task/new", url.Values{"s": {"work"}, "title": {"Linked"}}), "new task"))
	path := filepath.Join(home, "brain", "tasks", id+".md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	with := strings.Replace(string(data), "\ntype: task\n", "\ntype: task\nrepos:\n  - linked\n  - site\n", 1)
	if err := os.WriteFile(path, []byte(with), 0o600); err != nil {
		t.Fatal(err)
	}
	saved(t, s.post(t, "/repo/describe", url.Values{"s": {"work"}, "repo": {filepath.Join(work, "site")}, "text": {"The site"}}), "describe")
	page := s.get(t, "/repos?s=work").Body.String()
	for _, want := range []string{">site ", `value="The site"`, ">Linked<"} {
		if !strings.Contains(page, want) {
			t.Errorf("the page lacks %q", want)
		}
	}
	for _, bad := range []string{"outside-repo", ">linked "} {
		if strings.Contains(page, bad) {
			t.Errorf("the page shows %q: a link led out of the work folder", bad)
		}
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

// "forgotten?" marks work only on this computer that nobody is on: a repository named
// only by parked tasks gets it if it holds pending changes, never if it is clean.
func TestReposForgottenOnlyWithPending(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git")
	}
	s, cfg := fixture(t, "en")
	work := filepath.Join(filepath.Dir(cfg), "work")
	gitInit(t, filepath.Join(work, "dirty"))
	if err := os.WriteFile(filepath.Join(work, "dirty", "new.txt"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitInit(t, filepath.Join(work, "clean"))

	id := idIn(saved(t, s.post(t, "/task/new", url.Values{"s": {"work"}, "title": {"Old work"}}), "new task"))
	path := filepath.Join(filepath.Dir(cfg), "brain-work", "tasks", id+".md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	with := strings.Replace(string(data), "\ntype: task\n", "\ntype: task\nrepos:\n  - dirty\n  - clean\n", 1)
	if err := os.WriteFile(path, []byte(with), 0o600); err != nil {
		t.Fatal(err)
	}
	saved(t, s.post(t, "/task/park", url.Values{"s": {"work"}, "id": {id}, "next": {"Later"}}), "park")

	page := s.get(t, "/repos?s=work").Body.String()
	card := func(name string) string {
		for _, c := range strings.Split(page, `<li class="card`) {
			if strings.Contains(c, `<div class="title">`+name+" ") {
				return c
			}
		}
		t.Fatalf("no card for %s", name)
		return ""
	}
	if !strings.Contains(card("dirty"), "forgotten?") {
		t.Error("a repository with pending changes and only parked tasks is not marked")
	}
	if strings.Contains(card("clean"), "forgotten?") {
		t.Error("a clean repository is marked forgotten")
	}
}
