package panel

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/confighistory"
)

// What the coverage map of the beta tests found untested in the panel.

// The panel's root takes the browser to Today.
func TestRootGoesToToday(t *testing.T) {
	s, _ := fixture(t, "en")
	w := s.get(t, "/")
	if w.Code != http.StatusSeeOther || !strings.HasSuffix(w.Header().Get("Location"), "/today") {
		t.Errorf("root: %d %q", w.Code, w.Header().Get("Location"))
	}
}

// In the weekly review, a stalled task is kept (a journal line) or sent to someday.
func TestReviewTask(t *testing.T) {
	s, cfg := fixture(t, "en")
	if w := s.post(t, "/task/new", url.Values{"s": {"work"}, "title": {"Plant a garden"}}); w.Code != http.StatusSeeOther {
		t.Fatalf("new task: %d", w.Code)
	}
	task := filepath.Join(filepath.Dir(cfg), "brain-work", "tasks", "task-plant-a-garden.md")
	if w := s.post(t, "/review/task", url.Values{"s": {"work"}, "id": {"task-plant-a-garden"}, "as": {"keep"}}); w.Code != http.StatusSeeOther {
		t.Fatalf("keep: %d %s", w.Code, w.Body.String())
	}
	if b, _ := os.ReadFile(task); !strings.Contains(string(b), "## Journal") || strings.Count(string(b), "\n- 2") < 2 {
		t.Errorf("keep wrote no journal line:\n%s", b)
	}
	if w := s.post(t, "/review/task", url.Values{"s": {"work"}, "id": {"task-plant-a-garden"}, "as": {"someday"}}); w.Code != http.StatusSeeOther {
		t.Fatalf("someday: %d %s", w.Code, w.Body.String())
	}
	if b, _ := os.ReadFile(task); !strings.Contains(string(b), "status: someday") {
		t.Errorf("not someday:\n%s", b)
	}
	if w := s.post(t, "/review/task", url.Values{"s": {"work"}, "id": {"task-plant-a-garden"}}); w.Code != http.StatusBadRequest {
		t.Errorf("without as: %d", w.Code)
	}
}

// Every change to the configuration made from the panel is kept as a version: the
// profile, a secret's name, the guard's lists.
func TestPanelChangesLeaveVersions(t *testing.T) {
	s, cfg := fixture(t, "en")
	count := func() int {
		vs, err := confighistory.List(cfg)
		if err != nil {
			t.Fatal(err)
		}
		return len(vs)
	}
	for _, c := range []struct {
		what string
		path string
		form url.Values
	}{
		{"profile", "/profile", url.Values{"s": {"work"}, "name": {"Lucía"}, "about": {"- Baker"}}},
		{"secret", "/secret", url.Values{"s": {"work"}, "service": {"mail"}, "name": {"smtp"}, "what": {"sending mail"}}},
		{"guard host", "/guard/list", url.Values{"s": {"work"}, "list": {"hosts"}, "op": {"add"}, "value": {"db01"}}},
		{"guard model", "/guard/model", url.Values{"s": {"work"}, "value": {"opus"}}},
	} {
		before := count()
		if w := s.post(t, c.path, c.form); w.Code != http.StatusSeeOther {
			t.Fatalf("%s: %d %s", c.what, w.Code, w.Body.String())
		}
		if count() <= before {
			t.Errorf("%s left no version", c.what)
		}
	}
}
