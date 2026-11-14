package panel

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/icalvete/augmented-brain/internal/brain"
)

// A search for a word many notes hold makes the snippets of the hits it shows only,
// not of every hit (P-11): with 3,000 notes it took seconds, with the notes lock held.
func TestSearchSnipsOnlyWhatItShows(t *testing.T) {
	s, cfg := fixture(t, "en")
	work := openSpace(t, cfg, "work")
	for i := range searchMax + 30 {
		if _, err := work.NewTask(brain.NewTaskInput{Title: fmt.Sprintf("Deploy number %d", i)}); err != nil {
			t.Fatal(err)
		}
	}
	calls := 0
	snippetOf = func(body string, words []string) string { calls++; return snippet(body, words) }
	t.Cleanup(func() { snippetOf = snippet })
	page := s.get(t, "/search?s=work&q=deploy").Body.String()
	if calls == 0 || calls > searchMax {
		t.Fatalf("%d snippets made for %d hits shown", calls, searchMax)
	}
	if n := strings.Count(page, "<mark>Deploy</mark> number"); n != searchMax {
		t.Errorf("%d hits shown, want %d", n, searchMax)
	}
}

// The stamp an open page asks for every 15 seconds does not wait for the notes lock,
// nor load the space (P-10), and says the same as the page.
func TestStampNeedsNoLock(t *testing.T) {
	s, cfg := fixture(t, "en")
	n, err := openSpace(t, cfg, "work").NewTask(brain.NewTaskInput{Title: "Stamp me"})
	if err != nil {
		t.Fatal(err)
	}
	page := s.get(t, "/task?s=work&id="+n.Header.ID).Body.String()
	i := strings.Index(page, `data-stamp="`)
	if i < 0 {
		t.Fatal("no stamp on the page")
	}
	want := strings.SplitN(page[i+len(`data-stamp="`):], `"`, 2)[0]
	release, err := s.writer.Lock()
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	got := make(chan string, 1)
	go func() { got <- s.get(t, "/stamp?s=work&id="+n.Header.ID).Body.String() }()
	select {
	case g := <-got:
		if g != want {
			t.Errorf("stamp %q, the page said %q", g, want)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the stamp waited for the notes lock")
	}
	for _, id := range []string{"", "..", "../config", "tasks/" + n.Header.ID, "task-nothing", ".hidden"} {
		if g := s.get(t, "/stamp?s=work&id="+id).Body.String(); g != "" {
			t.Errorf("stamp of %q = %q, want nothing", id, g)
		}
	}
	if r := s.get(t, "/stamp?s=nowhere&id="+n.Header.ID); r.Code != 404 {
		t.Errorf("stamp in a space that does not exist: %d", r.Code)
	}
}
