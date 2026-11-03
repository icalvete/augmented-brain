package panel

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/brain"
	"github.com/icalvete/augmented-brain/internal/config"
)

func saved(t *testing.T, w *httptest.ResponseRecorder, what string) string {
	t.Helper()
	if w.Code != http.StatusSeeOther {
		t.Fatalf("%s: %d %s", what, w.Code, w.Body)
	}
	return w.Header().Get("Location")
}

// idIn is the id a redirect to a note carries.
func idIn(loc string) string {
	return strings.SplitN(loc[strings.Index(loc, "id=")+3:], "&", 2)[0]
}

func openSpace(t *testing.T, cfg, space string) *brain.Brain {
	t.Helper()
	c, err := config.Load(cfg)
	if err != nil {
		t.Fatal(err)
	}
	b, err := brain.Open(c, c.Spaces[space])
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// A task's body is shown rendered: its essentials open, a [[link]] to a note that
// exists leads to it, and a note's HTML stays text.
func TestTaskBodyRendered(t *testing.T) {
	s, cfg := fixture(t, "en")
	id := idIn(saved(t, s.post(t, "/task/new", url.Values{"s": {"work"}, "title": {"Bake bread"}}), "new task"))
	saved(t, s.post(t, "/task/journal", url.Values{"s": {"work"}, "id": {id}, "entry": {"See [[area-general]] and **this** <b>bold</b>"}}), "journal")
	page := s.get(t, "/task?s=work&id="+id).Body.String()
	for _, want := range []string{`<details class="sec" open>`, `<strong>this</strong>`, `&lt;b&gt;bold&lt;/b&gt;`, `class="sec journal"`} {
		if !strings.Contains(page, want) {
			t.Errorf("task page lacks %q", want)
		}
	}
	if _, ok := openSpace(t, cfg, "work").Mem.Notes["area-general"]; ok && !strings.Contains(page, `href="note?s=work&amp;id=area-general"`) {
		t.Error("the link to the area does not lead to it")
	}
}

// Today gathers every space, and the repositories that hold work only here.
func TestToday(t *testing.T) {
	s, cfg := fixture(t, "en")
	saved(t, s.post(t, "/task/new", url.Values{"s": {"home"}, "title": {"Paint the fence"}}), "new task in home")
	page := s.get(t, "/today?s=work").Body.String()
	if !strings.Contains(page, "Space work") || !strings.Contains(page, "Space home") || !strings.Contains(page, "Paint the fence") {
		t.Fatalf("today does not show both spaces:\n%s", page)
	}
	if _, err := exec.LookPath("git"); err != nil {
		return
	}
	// A repository a task names, with a file not committed.
	repo := filepath.Join(filepath.Dir(cfg), "work", "shop")
	if out, err := exec.Command("git", "init", "-q", repo).CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	if err := os.WriteFile(filepath.Join(repo, "new.txt"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	id := idIn(saved(t, s.post(t, "/task/new", url.Values{"s": {"work"}, "title": {"Shop site"}}), "new task"))
	path := filepath.Join(filepath.Dir(cfg), "brain-work", "tasks", id+".md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	withRepos := strings.Replace(string(data), "\ntype: task\n", "\ntype: task\nrepos:\n  - shop\n  - not-here\n", 1)
	if err := os.WriteFile(path, []byte(withRepos), 0o600); err != nil {
		t.Fatal(err)
	}
	page = s.get(t, "/today?s=work").Body.String()
	if !strings.Contains(page, "1 file changed") || !strings.Contains(page, ">Shop site<") {
		t.Errorf("today does not show the repository:\n%s", page)
	}
}

// Search finds a note by any of its words, accents or not, and never shows its HTML.
func TestSearch(t *testing.T) {
	s, _ := fixture(t, "es")
	saved(t, s.post(t, "/task/new", url.Values{"s": {"work"}, "title": {"Revisión del horno"}, "summary": {"<i>pan</i> de centeno"}}), "new task")
	page := s.get(t, "/search?s=work&q=revision+centeno").Body.String()
	if !strings.Contains(page, "Revisión del horno") || strings.Contains(page, "<i>pan</i>") {
		t.Fatalf("search:\n%s", page)
	}
	if page := s.get(t, "/search?s=work&q=nada+de+esto").Body.String(); !strings.Contains(page, "Ninguna nota") {
		t.Error("an empty search says nothing")
	}
}

// People are added and changed from the panel, and their page shows what waits on them.
func TestPeople(t *testing.T) {
	s, cfg := fixture(t, "en")
	loc := saved(t, s.post(t, "/person", url.Values{"s": {"work"}, "name": {"Ana Ruiz"}, "aliases": {"Ana, Anita"}, "role": {"baker"}, "notes": {"Starts at six."}}), "new person")
	if idIn(loc) != "person-ana-ruiz" {
		t.Fatalf("went to %q", loc)
	}
	saved(t, s.post(t, "/person", url.Values{"s": {"work"}, "name": {"Ana Ruiz"}, "aliases": {"Ana"}, "role": {"head baker"}, "notes": {""}}), "change person, notes kept")
	p := openSpace(t, cfg, "work").Mem.Notes["person-ana-ruiz"]
	role, aliases := brain.PersonParts(p)
	if role != "head baker" || strings.Join(aliases, ",") != "Ana" || !strings.Contains(p.Body, "Starts at six.") {
		t.Fatalf("person: %q %q %q", role, aliases, p.Body)
	}
	id := idIn(saved(t, s.post(t, "/task/new", url.Values{"s": {"work"}, "title": {"Flour order"}}), "new task"))
	saved(t, s.post(t, "/task/update", url.Values{"s": {"work"}, "id": {id}, "waiting_on": {"Ana"}}), "wait on Ana")
	page := s.get(t, "/note?s=work&id=person-ana-ruiz").Body.String()
	if !strings.Contains(page, "Flour order") || !strings.Contains(page, `value="head baker"`) {
		t.Errorf("person page:\n%s", page)
	}
	// From the page's own form (replace=1), the notes are what it sends, empty too.
	saved(t, s.post(t, "/person", url.Values{"s": {"work"}, "name": {"Ana Ruiz"}, "aliases": {"Ana"}, "role": {"head baker"}, "notes": {""}, "replace": {"1"}}), "empty notes")
	if p := openSpace(t, cfg, "work").Mem.Notes["person-ana-ruiz"]; strings.TrimSpace(p.Body) != "" {
		t.Errorf("notes not emptied: %q", p.Body)
	}
	if tasks := s.get(t, "/tasks?s=work").Body.String(); !strings.Contains(tasks, `id=person-ana-ruiz">Ana</a>`) {
		t.Error("the task list does not lead from the wait to the person")
	}
	key := "AK" + "IA" + "QWERTYUIOPASDFGH" // built here, so this file holds no key-shaped text
	if w := s.post(t, "/person", url.Values{"s": {"work"}, "name": {"Leak"}, "notes": {"key " + key}}); w.Code != http.StatusBadRequest {
		t.Errorf("a person with a secret was saved: %d", w.Code)
	}
}

// The weekly review on a page: a capture becomes a task, another is dropped, and the
// record keeps both.
func TestReviewPage(t *testing.T) {
	s, cfg := fixture(t, "en")
	inbox := filepath.Join(filepath.Dir(cfg), "brain-work", "inbox.md")
	if err := os.WriteFile(inbox, []byte("- 2026-10-01 — Fix the oven door\n- 2026-10-02 — Old idea\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if page := s.get(t, "/review?s=work").Body.String(); !strings.Contains(page, "Fix the oven door") {
		t.Fatalf("review page:\n%s", page)
	}
	saved(t, s.post(t, "/review/capture", url.Values{"s": {"work"}, "date": {"2026-10-01"}, "text": {"Fix the oven door"}, "as": {"task"}}), "to task")
	saved(t, s.post(t, "/review/capture", url.Values{"s": {"work"}, "date": {"2026-10-02"}, "text": {"Old idea"}, "as": {"drop"}}), "drop")
	if w := s.post(t, "/review/capture", url.Values{"s": {"work"}, "date": {"2026-10-02"}, "text": {"Old idea"}, "as": {"drop"}}); w.Code != http.StatusBadRequest {
		t.Errorf("dropped twice: %d", w.Code)
	}
	if data, _ := os.ReadFile(inbox); strings.TrimSpace(string(data)) != "" {
		t.Errorf("inbox left:\n%s", data)
	}
	saved(t, s.post(t, "/review/done", url.Values{"s": {"work"}, "summary": {"Quiet week"}}), "done")
	b := openSpace(t, cfg, "work")
	if _, ok := b.Mem.Notes["task-fix-the-oven-door"]; !ok {
		t.Error("the capture did not become a task")
	}
	reviews := b.Mem.OfType("review")
	if len(reviews) != 1 || !strings.Contains(reviews[0].Body, "Quiet week") || !strings.Contains(reviews[0].Body, "Old idea") {
		t.Fatalf("review record: %+v", reviews)
	}
	if _, err := os.Stat(filepath.Join(cfg, "review-work.draft")); !os.IsNotExist(err) {
		t.Error("the draft outlived the review")
	}
}

// An open page can tell its note changed on disk; asking does not count as using the
// panel. An area's title and summary change from its page.
func TestStampAndArea(t *testing.T) {
	s, cfg := fixture(t, "en")
	b := openSpace(t, cfg, "work")
	var area string
	for _, a := range b.Mem.OfType("area") {
		area = a.Header.ID
	}
	if area == "" {
		t.Skip("no area in a new space")
	}
	page := s.get(t, "/note?s=work&id="+area).Body.String()
	i := strings.Index(page, `data-stamp="`)
	if i < 0 {
		t.Fatalf("no stamp on the page:\n%s", page)
	}
	before := strings.SplitN(page[i+len(`data-stamp="`):], `"`, 2)[0]
	last := s.last.Load()
	if got := s.get(t, "/stamp?s=work&id="+area).Body.String(); got != before {
		t.Errorf("stamp %q, page said %q", got, before)
	}
	if s.last.Load() != last {
		t.Error("asking for the stamp counted as using the panel")
	}
	saved(t, s.post(t, "/area/update", url.Values{"s": {"work"}, "id": {area}, "title": {"Bakery"}, "summary": {"Bread and cakes"}}), "area")
	if got := s.get(t, "/stamp?s=work&id="+area).Body.String(); got == before {
		t.Error("the stamp did not change with the note")
	}
	a := openSpace(t, cfg, "work").Mem.Notes[area]
	if a.Header.Title != "Bakery" || a.Header.Summary != "Bread and cakes" {
		t.Errorf("area: %+v", a.Header)
	}
	saved(t, s.post(t, "/area/update", url.Values{"s": {"work"}, "id": {area}, "title": {""}, "summary": {""}}), "area emptied")
	a = openSpace(t, cfg, "work").Mem.Notes[area]
	if a.Header.Title != "Bakery" || a.Header.Summary != "" {
		t.Errorf("area after emptying: %+v", a.Header)
	}
}

// The secrets page lists names, links the tasks that name them, and takes no value.
func TestSecretsPage(t *testing.T) {
	s, _ := fixture(t, "en")
	saved(t, s.post(t, "/secret", url.Values{"s": {"work"}, "service": {"telegram"}, "name": {"alerts-bot"}, "what": {"alerts by Telegram"}}), "add")
	saved(t, s.post(t, "/task/new", url.Values{"s": {"home"}, "title": {"Alerts"}, "summary": {"uses telegram/alerts-bot"}}), "task")
	if !namesKey("uses telegram/alerts-bot.", "telegram/alerts-bot") || namesKey("telegram/alerts-bot2", "telegram/alerts-bot") || namesKey("xtelegram/alerts-bot", "telegram/alerts-bot") {
		t.Error("namesKey")
	}
	page := s.get(t, "/secrets?s=work").Body.String()
	if !strings.Contains(page, "telegram/alerts-bot") || !strings.Contains(page, "alerts by Telegram") || !strings.Contains(page, ">Alerts</a>") {
		t.Fatalf("secrets page:\n%s", page)
	}
	key := "AK" + "IA" + "QWERTYUIOPASDFGH" // built here, so this file holds no key-shaped text
	if w := s.post(t, "/secret", url.Values{"s": {"work"}, "service": {"aws"}, "name": {"deploy"}, "what": {key}}); w.Code != http.StatusBadRequest {
		t.Errorf("a value was kept: %d", w.Code)
	}
	if w := s.post(t, "/secret/forget", url.Values{"s": {"work"}, "service": {"telegram"}, "name": {"alerts-bot"}}); w.Code != http.StatusBadRequest {
		t.Errorf("forgot without the tick: %d", w.Code)
	}
	saved(t, s.post(t, "/secret/forget", url.Values{"s": {"work"}, "service": {"telegram"}, "name": {"alerts-bot"}, "confirm": {"yes"}}), "forget")
	if page := s.get(t, "/secrets?s=work").Body.String(); strings.Contains(page, "<code>telegram/alerts-bot") {
		t.Error("still listed")
	}
}

// Found in review: an indented line, or a letter that folds into two, made the search
// snippet slice out of range, and the page then kept the notes locked.
func TestSnippetNeverPanics(t *testing.T) {
	for _, body := range []string{
		strings.Repeat(" ", 100) + strings.Repeat("x", 300) + "needle",
		strings.Repeat("İ", 300) + "needle",
		"\t\t" + strings.Repeat("á", 400) + " needle " + strings.Repeat("é", 400),
	} {
		got := snippet(body, []string{"needle"})
		if !strings.Contains(got, "needle") {
			t.Errorf("snippet lost the match: %q", got)
		}
	}
}

// Found in review: a second click on a capture made a second task.
func TestReviewCaptureOnce(t *testing.T) {
	s, cfg := fixture(t, "en")
	inbox := filepath.Join(filepath.Dir(cfg), "brain-work", "inbox.md")
	if err := os.WriteFile(inbox, []byte("- 2026-10-01 — Fix the oven door\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	form := url.Values{"s": {"work"}, "date": {"2026-10-01"}, "text": {"Fix the oven door"}, "as": {"task"}}
	saved(t, s.post(t, "/review/capture", form), "to task")
	if w := s.post(t, "/review/capture", form); w.Code != http.StatusBadRequest {
		t.Errorf("second click: %d", w.Code)
	}
	n := 0
	for _, task := range openSpace(t, cfg, "work").Mem.OfType("task") {
		if strings.HasPrefix(task.Header.ID, "task-fix-the-oven-door") {
			n++
		}
	}
	if n != 1 {
		t.Errorf("%d tasks from one capture", n)
	}
}

// Found in review: Today ran git in any folder a note named, an imported one say.
// Only the space's work folders count.
func TestTodayOnlyWorkFolders(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git")
	}
	s, cfg := fixture(t, "en")
	outside := filepath.Join(filepath.Dir(cfg), "elsewhere")
	if out, err := exec.Command("git", "init", "-q", outside).CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	if err := os.WriteFile(filepath.Join(outside, "f"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	id := idIn(saved(t, s.post(t, "/task/new", url.Values{"s": {"work"}, "title": {"Outside"}}), "new task"))
	path := filepath.Join(filepath.Dir(cfg), "brain-work", "tasks", id+".md")
	data, _ := os.ReadFile(path)
	with := strings.Replace(string(data), "\ntype: task\n", "\ntype: task\nrepos:\n  - "+outside+"\n  - ../elsewhere\n", 1)
	if err := os.WriteFile(path, []byte(with), 0o600); err != nil {
		t.Fatal(err)
	}
	if page := s.get(t, "/today?s=work").Body.String(); strings.Contains(page, "elsewhere") {
		t.Errorf("a repository outside the work folders was read:\n%s", page)
	}
}
