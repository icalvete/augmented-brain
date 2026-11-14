package panel

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/doctor"
)

// A fake AWS key, built so the file itself holds none.
var fakeKey = "AKIA" + "Z7Q3XK2M9PLW4RT8"

func redactorFixture(t *testing.T) (*redactor, string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	c := &config.Config{
		Language:              "en",
		UntouchableHosts:      []string{"db1.corp.example"},
		ProductionAWSProfiles: []string{"prod-admin"},
		RootAsk:               []string{"bastion.example until 2099-01-01", "jump.example until someday"},
		Spaces: map[string]*config.Space{
			"acme": {Name: "acme", Memory: filepath.Join(home, "acme-notes"), WorkFolders: []string{filepath.Join(home, "code", "acme")},
				Backup: "git@github.com:someone/acme-notes.git"},
		},
	}
	return newRedactor("en", c), home
}

func TestRedact(t *testing.T) {
	r, home := redactorFixture(t)
	for _, tc := range []struct{ in, want string }{
		{"Configuration: " + filepath.Join(home, ".config", "ab", "config.toml"), "Configuration: " + filepath.FromSlash("~/.config/ab/config.toml")},
		{"Space acme: 12 notes, valid", "Space [space]: 12 notes, valid"},
		{"Space acme-two stays", "Space acme-two stays"},
		{"notes in " + filepath.Join(home, "acme-notes", "tasks"), "notes in " + filepath.Join("[notes folder]", "tasks")},
		{"notes in ~/acme-notes", "notes in [notes folder]"},
		{"work in " + filepath.Join(home, "code", "acme"), "work in [work folder]"},
		{"write to jane.doe+x@mail.example.org now", "write to [e-mail] now"},
		{"never touch DB1.corp.example, ok", "never touch [machine], ok"},
		{"root on bastion.example until …", "root on [machine] until …"},
		{"profile prod-admin and prod-administrator", "profile [AWS profile] and prod-administrator"},
		{"backup to git@github.com:someone/acme-notes.git", "backup to [backup]"},
		{"acme acme", "[space] [space]"},
		{"jump.example until someday: not a date", "[machine] until someday: not a date"},
	} {
		if got := r.text(tc.in); got != tc.want {
			t.Errorf("%q\n got %q\nwant %q", tc.in, got, tc.want)
		}
	}
	got := r.text("first\nkey " + fakeKey + " here\nlast")
	if strings.Contains(got, fakeKey) || !strings.Contains(got, "left out") || !strings.HasPrefix(got, "first\n") || !strings.HasSuffix(got, "\nlast") {
		t.Errorf("a secret line stays: %q", got)
	}
}

func TestDiagnostics(t *testing.T) {
	r, home := redactorFixture(t)
	checks := []doctor.Check{
		{Status: doctor.OK, What: "Configuration: " + filepath.Join(home, "cfg", "config.toml")},
		{Status: doctor.Warning, What: "Space acme: index out of date", Fix: "run check in " + home},
		{Status: doctor.Failed, What: "Claude Code: not found"},
		{Status: doctor.Info, What: strings.Repeat("x", 1000)},
	}
	d := diagnostics("en", checks, r)
	for _, want := range []string{"augmented brain ", "System: ", "[ok]   Configuration: " + filepath.FromSlash("~/cfg/config.toml"), "[warn] Space [space]: index out of date", "[fail] Claude Code: not found"} {
		if !strings.Contains(d, want) {
			t.Errorf("no %q in\n%s", want, d)
		}
	}
	if strings.Contains(d, "run check") || strings.Contains(d, home) || strings.Contains(d, strings.Repeat("x", maxCheckText+1)) {
		t.Errorf("a fix, the home folder or a long line is in\n%s", d)
	}
}

func TestIssueURL(t *testing.T) {
	u := issueURL("o/r", "problem.yml", []issueField{{"title", "a & b"}, {"what", "line 1\nline 2 ñ #x"}}, "what", "[cut]", 8000)
	p, err := url.Parse(u)
	if err != nil || p.Host != "github.com" || p.Path != "/o/r/issues/new" {
		t.Fatalf("%s: %v", u, err)
	}
	q := p.Query()
	if q.Get("template") != "problem.yml" || q.Get("title") != "a & b" || q.Get("what") != "line 1\nline 2 ñ #x" {
		t.Errorf("%v", q)
	}
	if strings.ContainsAny(strings.TrimPrefix(u, "https://"), " \n#") {
		t.Errorf("not encoded: %s", u)
	}

	long := strings.Repeat("ñ line of the report\n", 2000)
	u = issueURL("o/r", "problem.yml", []issueField{{"title", "t"}, {"diagnostics", long}, {"after", "kept"}}, "diagnostics", "[cut]", 8000)
	if len(u) > 8000 || len(u) < 7900 {
		t.Errorf("length %d", len(u))
	}
	q, _ = url.ParseQuery(strings.SplitN(u, "?", 2)[1])
	d := q.Get("diagnostics")
	if !strings.HasSuffix(d, "\n\n[cut]") || !strings.HasPrefix(long, strings.TrimSuffix(d, "\n\n[cut]")) || q.Get("after") != "kept" || q.Get("title") != "t" {
		t.Errorf("cut wrong: %q … %q", d[:40], d[len(d)-40:])
	}
	if !strings.Contains(d, "ñ") || strings.ContainsRune(d, '�') {
		t.Error("a character was split")
	}
}

func TestHelpPage(t *testing.T) {
	s, cfg := fixture(t, "es")
	ran := 0
	s.diagnose = func(lang string) []doctor.Check {
		ran++
		return []doctor.Check{
			{Status: doctor.OK, What: "Configuración: " + filepath.Join(os.Getenv("HOME"), "cfg", "config.toml")},
			{Status: doctor.Warning, What: "Espacio work: <b>índice</b> viejo"},
			{Status: doctor.Info, What: "clave " + fakeKey},
		}
	}
	w := s.get(t, "/help?s=work")
	body := w.Body.String()
	if w.Code != 200 || ran != 0 {
		t.Fatalf("%d, doctor ran %d times on the page", w.Code, ran)
	}
	for _, want := range []string{"Ayuda", "https://icalvete.github.io/augmented-brain/es/#faq",
		"https://icalvete.github.io/augmented-brain/guide/getting-started.es.html", `rel="noopener noreferrer"`, `target="_blank"`,
		"https://github.com/sponsors/icalvete", `href="` + s.localBase() + `/help/report"`, "Avisa de un problema en GitHub",
		"https://github.com/icalvete/augmented-brain/issues/new?template=idea.yml", `href="` + s.localBase() + `/help?s=work"`} {
		if !strings.Contains(body, want) {
			t.Errorf("no %q", want)
		}
	}
	if strings.Contains(body, "template=problem.yml") || strings.Contains(body, "<textarea") {
		t.Error("a report on the page itself")
	}

	// The link runs the health check then, and sends the browser to GitHub's form.
	w = s.get(t, "/help/report")
	if w.Code != http.StatusSeeOther || ran != 1 {
		t.Fatalf("%d, doctor ran %d times", w.Code, ran)
	}
	loc, err := url.Parse(w.Header().Get("Location"))
	if err != nil || loc.Scheme != "https" || loc.Host != "github.com" || loc.Path != "/icalvete/augmented-brain/issues/new" {
		t.Fatalf("Location %q: %v", w.Header().Get("Location"), err)
	}
	if len(loc.String()) > maxIssueURL {
		t.Errorf("Location is %d long", len(loc.String()))
	}
	q := loc.Query()
	d := q.Get("diagnostics")
	if q.Get("template") != "problem.yml" || !q.Has("what") || q.Get("what") != "" {
		t.Errorf("query %v", q)
	}
	for _, want := range []string{"augmented brain ", "Sistema: ", "[ok]   Configuración: " + filepath.FromSlash("~/cfg/config.toml"),
		"[warn] Espacio [espacio]: <b>índice</b> viejo", "[se dejó fuera una línea"} {
		if !strings.Contains(d, want) {
			t.Errorf("no %q in\n%s", want, d)
		}
	}
	if strings.Contains(d, fakeKey) || strings.Contains(d, cfg) || strings.Contains(d, " work") {
		t.Errorf("not redacted:\n%s", d)
	}

	// Outside the session's address it answers nothing, and runs nothing.
	if w := s.do(t, req{path: "/help/report"}); w.Code != http.StatusForbidden || ran != 1 {
		t.Errorf("without the session: %d, doctor ran %d times", w.Code, ran)
	}
	if w := s.do(t, req{path: s.localBase() + "/help/report", host: "evil.example:45678"}); w.Code != http.StatusForbidden || ran != 1 {
		t.Errorf("another host: %d, doctor ran %d times", w.Code, ran)
	}
}

// The fields the page fills in are those of the issue forms, by id: GitHub fills in a
// form's field only when the address names its id.
func TestIssueTemplates(t *testing.T) {
	for file, ids := range map[string][]string{"problem.yml": {"what", "diagnostics"}, "idea.yml": {"idea"}} {
		data, err := os.ReadFile(filepath.Join("..", "..", ".github", "ISSUE_TEMPLATE", file))
		if err != nil {
			t.Fatal(err)
		}
		for _, id := range ids {
			if !strings.Contains(string(data), "id: "+id+"\n") {
				t.Errorf("%s has no field %s", file, id)
			}
		}
	}
}
