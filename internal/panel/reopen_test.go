package panel

import (
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/remote"
)

// A closed task opens again from its own page, as the guide says: the page has the
// form, and sending it puts the task in the phase chosen, with its journal line.
func TestReopenFromTheTaskPage(t *testing.T) {
	s, cfg := fixture(t, "en")
	oldSys := newRemoteSystem
	t.Cleanup(func() { newRemoteSystem = oldSys })
	newRemoteSystem = func() (remote.System, error) { return remote.System{}, errors.New("no claude here") }
	id := idIn(saved(t, s.post(t, "/task/new", url.Values{"s": {"work"}, "title": {"Fence"}}), "new task"))
	saved(t, s.post(t, "/task/close", url.Values{"s": {"work"}, "id": {id}, "result": {"Painted"}}), "close")

	page := s.get(t, "/task?s=work&id="+id).Body.String()
	if strings.Contains(page, `id="steps"`) {
		t.Error("a closed task with no steps still shows the Steps heading")
	}
	m := regexp.MustCompile(`(?s)<summary>Open it again</summary>\s*<form method="post" action="[^"]*/task/update".*?</form>`).FindString(page)
	if m == "" {
		t.Fatalf("no form to reopen the task:\n%s", page)
	}
	if strings.Contains(m, `<option value="closed"`) {
		t.Error("the reopen form offers closed")
	}
	form := url.Values{}
	for _, h := range regexp.MustCompile(`<input type="hidden" name="([^"]+)" value="([^"]*)">`).FindAllStringSubmatch(m, -1) {
		form.Set(h[1], h[2])
	}
	if form.Get("id") != id || form.Get("orig_status") != "closed" {
		t.Fatalf("hidden fields: %v", form)
	}
	// What the browser sends of the visible fields: waits on, filled in; next, typed.
	w := regexp.MustCompile(`name="waiting_on" value="([^"]*)"`).FindStringSubmatch(m)
	if w == nil || w[1] != "me" {
		t.Fatalf("waits on is not filled in: %v", w)
	}
	form.Set("waiting_on", w[1])
	form.Set("status", "execution")
	form.Set("why", "the paint peeled")
	// Without what comes next it cannot open: the page comes back with the error in
	// the form and what was typed kept.
	code, body, _ := s.postFrom(t, "/task/update", "/task?s=work&id="+id, form)
	if code != http.StatusBadRequest || len(ferr.FindAllString(body, -1)) != 1 || !strings.Contains(body, `value="the paint peeled"`) {
		t.Fatalf("reopen with no next: %d\n%s", code, body)
	}
	form.Set("next", "Sand it first")
	saved(t, s.post(t, "/task/update", form), "reopen")

	n := openSpace(t, cfg, "work").Mem.Notes[id]
	if n.Header.Status != "execution" || n.Header.Next != "Sand it first" || n.Header.WaitingOn != "me" {
		t.Fatalf("header %+v", n.Header)
	}
	if !strings.Contains(n.Body, "Reopened (now execution): from the panel: the paint peeled") {
		t.Errorf("journal:\n%s", n.Body)
	}
	page = s.get(t, "/task?s=work&id="+id).Body.String()
	if strings.Contains(page, "Open it again") || !strings.Contains(page, `id="steps"`) {
		t.Error("the reopened task's page is not a live task's")
	}
}

// A space that asks for nothing says so on Today, instead of a bare heading.
func TestTodayEmptySpace(t *testing.T) {
	s, _ := fixture(t, "en")
	page := s.get(t, "/today?s=work").Body.String()
	if !strings.Contains(page, "Nothing here asks for you today.") {
		t.Errorf("an empty space shows no line:\n%s", page)
	}
	if strings.Contains(page, `<ul class="pills">`+"\n  </ul>") {
		t.Error("an empty list of pills is still written")
	}
}
