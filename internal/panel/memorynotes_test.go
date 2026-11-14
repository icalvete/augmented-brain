package panel

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/icalvete/augmented-brain/internal/config"
)

// Rules and knowledge are created, edited and retired from the panel, in the space or
// shared; the old version stays in archive/.
func TestRulesAndKnowledgeFromThePanel(t *testing.T) {
	s, cfg := fixture(t, "en")
	if page := s.get(t, "/rules?s=work").Body.String(); !strings.Contains(page, `/rule/new"`) || strings.Contains(page, `name="confirm"`) {
		t.Fatalf("rules page:\n%s", page)
	}
	if w := s.post(t, "/rule/new", url.Values{"s": {"work"}, "title": {"Short answers"}, "body": {"Keep answers short."}}); w.Code != http.StatusSeeOther {
		t.Fatalf("rule: %d %s", w.Code, w.Body)
	}
	if w := s.post(t, "/rule/new", url.Values{"s": {"work"}, "title": {"Plain words"}, "body": {"No jargon."}, "shared": {"1"}}); w.Code != http.StatusSeeOther {
		t.Fatalf("shared rule: %d %s", w.Code, w.Body)
	}
	if w := s.post(t, "/rule/new", url.Values{"s": {"work"}, "title": {"Empty"}, "body": {" "}}); w.Code != http.StatusBadRequest {
		t.Errorf("rule with no text: %d", w.Code)
	}
	if w := s.post(t, "/knowledge/new", url.Values{"s": {"work"}, "kind": {"lesson"}, "title": {"Backups need a test restore"}, "area": {"area-general"}}); w.Code != http.StatusSeeOther {
		t.Fatalf("lesson: %d %s", w.Code, w.Body)
	}
	b := openBrain(t, cfg)
	if _, ok := b.Mem.Notes["rule-short-answers"]; !ok {
		t.Fatalf("no rule in the space: %v", b.Mem.Notes)
	}
	shared, err := b.Shared()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := shared.Mem.Notes["rule-plain-words"]; !ok {
		t.Fatal("no shared rule")
	}
	if n := b.Mem.Notes["lesson-backups-need-a-test-restore"]; n == nil || n.Header.Area != "area-general" {
		t.Fatalf("lesson: %+v", n)
	}

	// Edit: the page carries when the note was written; a page shown before another
	// change does not overwrite it.
	page := s.get(t, "/note?s=work&id=rule-short-answers").Body.String()
	if !strings.Contains(page, `/note/update"`) || !strings.Contains(page, `/note/retire"`) {
		t.Fatalf("note page:\n%s", page)
	}
	stamp := stampIn(b.Mem, "rule-short-answers")
	edit := url.Values{"s": {"work"}, "id": {"rule-short-answers"}, "stamp": {stamp}, "title": {"Short answers"}, "body": {"Keep answers short, two lines at most."}}
	if w := s.post(t, "/note/update", edit); w.Code != http.StatusSeeOther {
		t.Fatalf("update: %d %s", w.Code, w.Body)
	}
	if n := openBrain(t, cfg).Mem.Notes["rule-short-answers"]; !strings.Contains(n.Body, "two lines at most") {
		t.Errorf("body: %s", n.Body)
	}
	archived, _ := filepath.Glob(filepath.Join(b.Mem.Dir, "archive", "rules", "rule-short-answers-*.md"))
	if len(archived) != 1 {
		t.Errorf("old version kept: %v", archived)
	}
	edit.Set("body", "Something else")
	if w := s.post(t, "/note/update", edit); w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "changed since") {
		t.Errorf("stale update: %d", w.Code)
	}
	// A shared rule is edited where it lives.
	sh := url.Values{"s": {"work"}, "id": {"rule-plain-words"}, "shared": {"1"}, "stamp": {stampIn(shared.Mem, "rule-plain-words")}, "title": {"Plain words"}, "body": {"No jargon, ever."}}
	if w := s.post(t, "/note/update", sh); w.Code != http.StatusSeeOther {
		t.Fatalf("shared update: %d %s", w.Code, w.Body)
	}

	// Retire: needs why; it moves to archive/ and the rule is gone from the memory.
	if w := s.post(t, "/note/retire", url.Values{"s": {"work"}, "id": {"rule-short-answers"}}); w.Code != http.StatusBadRequest {
		t.Errorf("retire with no reason: %d", w.Code)
	}
	if w := s.post(t, "/note/retire", url.Values{"s": {"work"}, "id": {"rule-short-answers"}, "reason": {"Says the same as plain words"}}); w.Code != http.StatusSeeOther {
		t.Fatalf("retire: %d %s", w.Code, w.Body)
	}
	if _, ok := openBrain(t, cfg).Mem.Notes["rule-short-answers"]; ok {
		t.Error("the retired rule is still in the memory")
	}
	if _, err := os.Stat(filepath.Join(b.Mem.Dir, "archive", "rules", "rule-short-answers.md")); err != nil {
		t.Error(err)
	}
	// A task is not a note these forms change.
	s.post(t, "/task/new", url.Values{"s": {"work"}, "title": {"Paint"}})
	if w := s.post(t, "/note/retire", url.Values{"s": {"work"}, "id": {"task-paint"}, "reason": {"x"}}); w.Code != http.StatusBadRequest {
		t.Errorf("retired a task: %d", w.Code)
	}
}

// With memory_trust = "ask", a rule is saved, changed or retired only once the user
// ticks the box that says yes; knowledge is not asked about.
func TestRuleAsksWhenTrustIsAsk(t *testing.T) {
	s, cfg := fixture(t, "en")
	if err := config.SaveSession(cfg, config.Session{MemoryTrust: config.TrustAsk}); err != nil {
		t.Fatal(err)
	}
	if page := s.get(t, "/rules?s=work").Body.String(); !strings.Contains(page, `name="confirm" value="1" required`) {
		t.Fatalf("no box to say yes:\n%s", page)
	}
	rule := url.Values{"s": {"work"}, "title": {"Short answers"}, "body": {"Keep answers short."}}
	if w := s.post(t, "/rule/new", rule); w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "Tick the box") {
		t.Fatalf("rule without yes: %d", w.Code)
	}
	if _, ok := openBrain(t, cfg).Mem.Notes["rule-short-answers"]; ok {
		t.Fatal("saved without yes")
	}
	rule.Set("confirm", "1")
	if w := s.post(t, "/rule/new", rule); w.Code != http.StatusSeeOther {
		t.Fatalf("rule with yes: %d %s", w.Code, w.Body)
	}
	if w := s.post(t, "/note/retire", url.Values{"s": {"work"}, "id": {"rule-short-answers"}, "reason": {"x"}}); w.Code != http.StatusBadRequest {
		t.Errorf("retire without yes: %d", w.Code)
	}
	if w := s.post(t, "/knowledge/new", url.Values{"s": {"work"}, "kind": {"reference"}, "title": {"How backups work"}}); w.Code != http.StatusSeeOther {
		t.Errorf("knowledge asked: %d %s", w.Code, w.Body)
	}
}

// Through the network (the remote panel, with its own ticket and form code) a rule is
// written too; the local form code does nothing there.
func TestRuleThroughTheNetwork(t *testing.T) {
	s, cfg := fixture(t, "en")
	const rhost, owner = "box.example.ts.net:8443", "lucia@example.com"
	s.remote.Store(&Remote{Host: rhost, Owner: owner})
	id, err := s.tickets.add("", RemoteTicketTTL, true, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if w := s.do(t, req{net: true, path: "/t/" + id, host: rhost, owner: owner}); w.Code != http.StatusSeeOther {
		t.Fatalf("ticket: %d", w.Code)
	}
	post := func(code string) int {
		return s.do(t, req{net: true, method: "POST", path: s.netBase() + "/rule/new", host: rhost, owner: owner, origin: "https://" + rhost,
			form: url.Values{"csrf": {code}, "s": {"work"}, "title": {"From the phone"}, "body": {"Written away from home."}}}).Code
	}
	if c := post(s.csrf); c == http.StatusSeeOther {
		t.Error("the local form code wrote through the network")
	}
	if c := post(s.netCSRF); c != http.StatusSeeOther {
		t.Fatalf("network form code: %d", c)
	}
	if _, ok := openBrain(t, cfg).Mem.Notes["rule-from-the-phone"]; !ok {
		t.Error("no rule written through the network")
	}
}
