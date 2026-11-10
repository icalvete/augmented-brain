package panel

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/icalvete/augmented-brain/internal/state"
)

// A dead session is shown with what was said since the last journal entry, and kept in
// the notes on request; a reviewed one leaves the list; the transcript of an ended one
// is deleted only once the box is ticked.
func TestSessions(t *testing.T) {
	s, cfg := fixture(t, "en")
	mem := filepath.Join(filepath.Dir(cfg), "brain-work")
	w := s.post(t, "/task/new", url.Values{"s": {"work"}, "title": {"Bake bread"}})
	if w.Code != http.StatusSeeOther {
		t.Fatalf("new task: %d", w.Code)
	}
	task := "task-bake-bread"

	proj := t.TempDir()
	now := time.Now()
	old := now.Add(-2 * time.Hour)
	said := now.Add(time.Minute).UTC().Format(time.RFC3339Nano) // after today's journal entry, whatever the hour
	write := func(id, body string, mod time.Time) string {
		p := filepath.Join(proj, id+".jsonl")
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(p, mod, mod); err != nil {
			t.Fatal(err)
		}
		return p
	}
	key := "AKIA" + "ABCDEFGHIJKLMNOP" // split, so no secret scanner takes the test for a leak
	conv := fmt.Sprintf(`{"type":"user","timestamp":%q,"message":{"content":"use rye flour"}}`+"\n"+
		`{"type":"assistant","timestamp":%q,"message":{"content":[{"type":"text","text":"Rye it is; key %s"}]}}`+"\n", said, said, key)
	sessions := map[string]string{
		"dead1": write("dead1", conv, old),
		"dead2": write("dead2", conv, old),
		"live1": write("live1", conv, now),
		"done1": write("done1", conv, old),
	}
	for id, p := range sessions {
		if err := state.Started(cfg, "work", id, p, old); err != nil {
			t.Fatal(err)
		}
	}
	_ = state.Ended(cfg, "work", "done1", old)
	_ = state.Update(cfg, "work", now, func(x *state.Space) {
		for i := range x.Sessions {
			x.Sessions[i].Task = task
		}
	})

	page := s.get(t, "/sessions?s=work").Body.String()
	for _, want := range []string{"Did not end", "dead1", "use rye flour", "Running now", "live1", "Ended", "done1", "Bake bread"} {
		if !strings.Contains(page, want) {
			t.Errorf("the page lacks %q", want)
		}
	}

	if w := s.post(t, "/session/recover", url.Values{"s": {"work"}, "id": {"live1"}}); w.Code != http.StatusBadRequest {
		t.Errorf("recovering a running session: %d", w.Code)
	}
	if w := s.post(t, "/session/recover", url.Values{"s": {"work"}, "id": {"dead1"}}); w.Code != http.StatusSeeOther {
		t.Fatalf("recover: %d %s", w.Code, w.Body)
	}
	kept, _ := filepath.Glob(filepath.Join(mem, "archive", "sessions", "*-dead1-recovered.md"))
	if len(kept) != 1 {
		t.Fatalf("archive: %v", kept)
	}
	data, _ := os.ReadFile(kept[0])
	if !strings.Contains(string(data), "use rye flour") || strings.Contains(string(data), key) {
		t.Errorf("archived conversation (secrets must be out):\n%s", data)
	}
	n, err := openBrain(t, cfg).Task(task)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(n.Body, "archive/sessions/") {
		t.Errorf("no journal line points at it:\n%s", n.Body)
	}

	// Today offers the same two actions (dead1 is kept already), and comes back to itself.
	today := s.get(t, "/today?s=home").Body.String()
	if !strings.Contains(today, "1 session cut off") || strings.Count(today, `name="back" value="today"`) != 2 ||
		!strings.Contains(today, `<input type="hidden" name="s" value="work"><input type="hidden" name="id" value="dead2">`) {
		t.Error("Today does not offer to deal with the sessions that did not end, in their space")
	}
	w = s.post(t, "/session/reviewed", url.Values{"s": {"work"}, "id": {"dead2"}, "back": {"today"}})
	if w.Code != http.StatusSeeOther || !strings.Contains(w.Header().Get("Location"), "/today?") {
		t.Fatalf("reviewed from Today: %d %q", w.Code, w.Header().Get("Location"))
	}
	// Sessions is no longer in the menu: Health leads to it.
	if strings.Contains(today, `aria-current="page">Sessions</a>`) || strings.Contains(today, `/sessions?s=home">Sessions</a>`) {
		t.Error("Sessions is still in the menu")
	}
	if h := s.get(t, "/space?s=work").Body.String(); !strings.Contains(h, "/sessions?s=work") {
		t.Error("Health does not lead to the Sessions page")
	}
	for _, l := range state.Read(cfg, "work").List(now) {
		if l.Kind == state.Dead {
			t.Errorf("%s is still listed as not ended", l.ID)
		}
	}

	if w := s.post(t, "/session/delete", url.Values{"s": {"work"}, "id": {"done1"}}); w.Code != http.StatusBadRequest {
		t.Errorf("deleting without the box ticked: %d", w.Code)
	}
	if _, err := os.Stat(sessions["done1"]); err != nil {
		t.Fatal("deleted without the box ticked")
	}
	if w := s.post(t, "/session/delete", url.Values{"s": {"work"}, "id": {"live1"}, "confirm": {"yes"}}); w.Code != http.StatusBadRequest {
		t.Errorf("deleting a running session's transcript: %d", w.Code)
	}
	if w := s.post(t, "/session/delete", url.Values{"s": {"work"}, "id": {"done1"}, "confirm": {"yes"}}); w.Code != http.StatusSeeOther {
		t.Fatalf("delete: %d %s", w.Code, w.Body)
	}
	if _, err := os.Stat(sessions["done1"]); !os.IsNotExist(err) {
		t.Error("the transcript is still there")
	}
}
