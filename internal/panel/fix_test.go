package panel

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/memory"
)

// Steps are corrected, moved and removed from the task's page, each by its number and
// the text the page showed; a title is changed with the task form.
func TestFixStepsAndTitle(t *testing.T) {
	s, cfg := fixture(t, "en")
	s.post(t, "/task/new", url.Values{"s": {"work"}, "title": {"Paint"}})
	id := "task-paint"
	for _, st := range []string{"Sand", "Prime", "Paint"} {
		s.post(t, "/task/step", url.Values{"s": {"work"}, "id": {id}, "step": {st}})
	}
	page := s.get(t, "/task?s=work&id="+id).Body.String()
	for _, want := range []string{`name="op" value="edit"`, `name="op" value="up"`, `name="op" value="remove"`, `name="title" value="Paint"`} {
		if !strings.Contains(page, want) {
			t.Errorf("the page lacks %s", want)
		}
	}
	steps := func() string {
		n, _ := openBrain(t, cfg).Task(id)
		var out []string
		for _, st := range memory.Steps(n.Body) {
			out = append(out, st.Text)
		}
		return strings.Join(out, "|")
	}
	post := func(v url.Values) *httptest.ResponseRecorder {
		v.Set("s", "work")
		v.Set("id", id)
		return s.post(t, "/task/step", v)
	}
	if w := post(url.Values{"n": {"2"}, "text": {"Prime"}, "op": {"edit"}, "edit": {"Prime twice"}}); w.Code != http.StatusSeeOther {
		t.Fatalf("edit: %d %s", w.Code, w.Body)
	}
	if w := post(url.Values{"n": {"2"}, "text": {"Prime twice"}, "op": {"up"}}); w.Code != http.StatusSeeOther {
		t.Fatalf("up: %d %s", w.Code, w.Body)
	}
	if got := steps(); got != "Prime twice|Sand|Paint" {
		t.Errorf("after edit and up: %s", got)
	}
	// The first step going up changes nothing, and is no error.
	if w := post(url.Values{"n": {"1"}, "text": {"Prime twice"}, "op": {"up"}}); w.Code != http.StatusSeeOther {
		t.Fatalf("up at the top: %d %s", w.Code, w.Body)
	}
	// A stale page touches no other step.
	if w := post(url.Values{"n": {"3"}, "text": {"Sand"}, "op": {"remove"}}); w.Code != http.StatusBadRequest {
		t.Fatalf("stale remove: %d", w.Code)
	}
	if w := post(url.Values{"n": {"3"}, "text": {"Paint"}, "op": {"remove"}}); w.Code != http.StatusSeeOther || !strings.Contains(w.Header().Get("Location"), "ok=task.step.remove") {
		t.Fatalf("remove: %d %s %s", w.Code, w.Header().Get("Location"), w.Body)
	}
	if got := steps(); got != "Prime twice|Sand" {
		t.Errorf("after remove: %s", got)
	}
	if w := post(url.Values{"n": {"1"}, "text": {"Prime twice"}, "op": {"edit"}, "edit": {" "}}); w.Code != http.StatusBadRequest {
		t.Errorf("empty edit: %d", w.Code)
	}

	// The title: only when changed, and never emptied.
	form := url.Values{"s": {"work"}, "id": {id}, "orig_title": {"Paint"}, "title": {"Paint the fence"}}
	if w := s.post(t, "/task/update", form); w.Code != http.StatusSeeOther {
		t.Fatalf("title: %d %s", w.Code, w.Body)
	}
	if n, _ := openBrain(t, cfg).Task(id); n.Header.Title != "Paint the fence" || !strings.Contains(n.Body, "Title changed: Paint → Paint the fence") {
		t.Errorf("title: %q\n%s", n.Header.Title, n.Body)
	}
	if w := s.post(t, "/task/update", url.Values{"s": {"work"}, "id": {id}, "orig_title": {"Paint the fence"}, "title": {""}}); w.Code != http.StatusBadRequest {
		t.Errorf("empty title: %d", w.Code)
	}
}

// A watch date is moved or taken out from the task's page; two on the same day stay
// apart.
func TestMoveAndDropWatch(t *testing.T) {
	s, cfg := fixture(t, "en")
	s.post(t, "/task/new", url.Values{"s": {"work"}, "title": {"Bank"}})
	id := "task-bank"
	for _, what := range []string{"Call", "Write"} {
		if w := s.post(t, "/watch/add", url.Values{"s": {"work"}, "id": {id}, "date": {"2026-12-01"}, "what": {what}}); w.Code != http.StatusSeeOther {
			t.Fatalf("add: %d %s", w.Code, w.Body)
		}
	}
	if page := s.get(t, "/task?s=work&id="+id).Body.String(); !strings.Contains(page, `/watch/move"`) || !strings.Contains(page, `/watch/drop"`) {
		t.Error("the page has no way to move or take out a date")
	}
	if w := s.post(t, "/watch/move", url.Values{"s": {"work"}, "id": {id}, "date": {"2026-12-01"}, "what": {"Write"}, "to": {"nope"}}); w.Code != http.StatusBadRequest {
		t.Errorf("bad date: %d", w.Code)
	}
	if w := s.post(t, "/watch/move", url.Values{"s": {"work"}, "id": {id}, "date": {"2026-12-01"}, "what": {"Write"}, "to": {"2026-12-09"}}); w.Code != http.StatusSeeOther {
		t.Fatalf("move: %d %s", w.Code, w.Body)
	}
	if w := s.post(t, "/watch/drop", url.Values{"s": {"work"}, "id": {id}, "date": {"2026-12-01"}, "what": {"Call"}}); w.Code != http.StatusSeeOther {
		t.Fatalf("drop: %d %s", w.Code, w.Body)
	}
	n, _ := openBrain(t, cfg).Task(id)
	if len(n.Header.Watch) != 1 || n.Header.Watch[0].Date != "2026-12-09" || n.Header.Watch[0].What != "Write" {
		t.Errorf("watch: %+v", n.Header.Watch)
	}
	if w := s.post(t, "/watch/drop", url.Values{"s": {"work"}, "id": {id}, "date": {"2026-12-01"}, "what": {"Call"}}); w.Code != http.StatusBadRequest {
		t.Errorf("drop again: %d", w.Code)
	}
}
