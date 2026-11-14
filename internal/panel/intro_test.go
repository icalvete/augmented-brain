package panel

import (
	"html"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Every page starts with what it is and a link to its part of the guide, in the
// person's language; hiding it, on one page or all, is remembered on this machine,
// and it comes back with one click.
func TestIntroHideAndShow(t *testing.T) {
	s, cfg := fixture(t, "en")
	pages := map[string]string{ // path → key
		"/today": "today", "/tasks": "tasks", "/reminders": "reminders", "/inbox": "inbox",
		"/review": "review", "/areas": "areas", "/knowledge": "knowledge", "/rules": "rules",
		"/people": "people", "/search": "search", "/repos": "repos", "/secrets": "secrets",
		"/sessions": "sessions", "/profile": "profile", "/space": "space", "/guard": "guard",
		"/versions": "versions", "/help": "help",
	}
	for path, key := range pages {
		w := s.get(t, path+"?s=work")
		body := w.Body.String()
		if w.Code != http.StatusOK || !strings.Contains(body, `class="intro"`) ||
			!strings.Contains(body, `name="key" value="`+key+`"`) || !strings.Contains(body, "/guide/panel.html#") {
			t.Errorf("%s: no introduction with its guide (%d)", path, w.Code)
		}
	}

	w := s.post(t, "/hints", url.Values{"do": {"hide"}, "key": {"tasks"}, "back": {"/tasks?s=work&f=mine"}})
	if w.Code != http.StatusSeeOther || w.Header().Get("Location") != s.localBase()+"/tasks?s=work&f=mine" {
		t.Fatalf("hide: %d %q", w.Code, w.Header().Get("Location"))
	}
	if b := s.get(t, "/tasks?s=work").Body.String(); strings.Contains(b, `class="intro"`) || !strings.Contains(b, `value="show"`) {
		t.Error("the hidden introduction is still there, or cannot be shown again")
	}
	if b := s.get(t, "/today?s=work").Body.String(); !strings.Contains(b, `class="intro"`) {
		t.Error("hiding one page's introduction hid another's")
	}
	if _, err := os.Stat(filepath.Join(cfg, "state", hintsFile)); err != nil {
		t.Errorf("not remembered: %v", err)
	}

	s.post(t, "/hints", url.Values{"do": {"hideall"}, "key": {"today"}, "back": {"/today?s=work"}})
	if b := s.get(t, "/help?s=work").Body.String(); strings.Contains(b, `class="intro"`) || !strings.Contains(b, `value="showall"`) {
		t.Error("hide all left an introduction")
	}
	s.post(t, "/hints", url.Values{"do": {"showall"}, "back": {"/help"}})
	if b := s.get(t, "/tasks?s=work").Body.String(); !strings.Contains(b, `class="intro"`) {
		t.Error("show all did not bring them back")
	}

	// A new panel (another run, another port) remembers.
	s.post(t, "/hints", url.Values{"do": {"hide"}, "key": {"guard"}, "back": {"/guard"}})
	s2, err := New(cfg, port)
	if err != nil {
		t.Fatal(err)
	}
	if b := s2.get(t, "/guard?s=work").Body.String(); strings.Contains(b, `class="intro"`) {
		t.Error("another run of the panel forgot what was hidden")
	}
}

func TestIntroRefusesTheUnknown(t *testing.T) {
	s, _ := fixture(t, "en")
	for _, f := range []url.Values{
		{"do": {"hide"}, "key": {"../../x"}},
		{"do": {"explode"}, "key": {"tasks"}},
	} {
		if w := s.post(t, "/hints", f); w.Code != http.StatusBadRequest {
			t.Errorf("%v: %d", f, w.Code)
		}
	}
	// Going back never leaves the panel.
	for _, back := range []string{"https://evil.example/", "//evil.example/x", "/\\evil.example", "javascript:alert(1)", ""} {
		if got := backTo(back); got != "/today" {
			t.Errorf("backTo(%q) = %q", back, got)
		}
	}
	if got := backTo("/task?id=task-a&s=work"); got != "/task?id=task-a&s=work" {
		t.Errorf("backTo kept %q", got)
	}
}

func TestIntroSpanish(t *testing.T) {
	s, _ := fixture(t, "es")
	b := s.get(t, "/inbox?s=work").Body.String()
	if !strings.Contains(html.UnescapeString(b), "Ocultar esta ayuda") || !strings.Contains(b, "/guide/panel.es.html#buz") {
		t.Error("the Spanish panel does not link the Spanish guide")
	}
}

// The anchors the introductions link to exist in the built guide, in both languages.
func TestIntroAnchors(t *testing.T) {
	read := func(name string) string {
		b, err := os.ReadFile(filepath.Join("..", "..", "docs", "guide", name))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	en, es := read("panel.html"), read("panel.es.html")
	for key, a := range guideAnchors {
		if !strings.Contains(en, `id="`+a[0]+`"`) {
			t.Errorf("%s: no #%s in panel.html", key, a[0])
		}
		if !strings.Contains(es, `id="`+a[1]+`"`) {
			t.Errorf("%s: no #%s in panel.es.html", key, a[1])
		}
	}
}
