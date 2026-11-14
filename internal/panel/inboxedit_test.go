package panel

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A capture is corrected from the inbox page: its text and its reminder; its date
// stays, and a stale page finds nothing.
func TestInboxEdit(t *testing.T) {
	s, cfg := fixture(t, "en")
	inbox := filepath.Join(filepath.Dir(cfg), "brain-work", "inbox.md")
	if err := os.WriteFile(inbox, []byte("- 2026-10-01 — Call Ana (remind: 2026-10-05)\n- 2026-10-02 — Buy milk\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	page := s.get(t, "/inbox?s=work").Body.String()
	if !strings.Contains(page, `/inbox/edit"`) || !strings.Contains(page, `name="new" required maxlength="4000" value="Call Ana"`) || !strings.Contains(page, `name="remind_on" value="2026-10-05"`) {
		t.Fatalf("inbox page:\n%s", page)
	}
	if w := s.post(t, "/inbox/edit", url.Values{"s": {"work"}, "form": {"e0"}, "date": {"2026-10-01"}, "text": {"Call Ana"}, "new": {"Call Ana about the roof"}, "remind_on": {""}}); w.Code != http.StatusSeeOther {
		t.Fatalf("edit: %d %s", w.Code, w.Body)
	}
	if data, _ := os.ReadFile(inbox); string(data) != "- 2026-10-01 — Call Ana about the roof\n- 2026-10-02 — Buy milk\n" {
		t.Errorf("inbox:\n%s", data)
	}
	// The error goes next to the field.
	w := s.post(t, "/inbox/edit", url.Values{"s": {"work"}, "form": {"e1"}, "date": {"2026-10-02"}, "text": {"Buy milk"}, "new": {"Buy oat milk"}, "remind_on": {"soon"}})
	if w.Code != http.StatusBadRequest {
		t.Errorf("bad date: %d", w.Code)
	}
	if w := s.post(t, "/inbox/edit", url.Values{"s": {"work"}, "date": {"2026-10-01"}, "text": {"Call Ana"}, "new": {"x"}}); w.Code != http.StatusBadRequest {
		t.Errorf("stale page: %d", w.Code)
	}
	if w := s.post(t, "/inbox/edit", url.Values{"s": {"work"}, "date": {"2026-10-02"}, "text": {"Buy milk"}, "new": {" "}}); w.Code != http.StatusBadRequest {
		t.Errorf("empty: %d", w.Code)
	}
}
