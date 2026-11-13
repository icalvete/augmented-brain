package panel

import (
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/confighistory"
)

// A change in the panel keeps a version (and the configuration before it); the
// Versions page shows what it changed and goes back, sealing the guard's lists again.
// Through the network the page only shows.
func TestVersionsPage(t *testing.T) {
	s, cfg := fixture(t, "en")
	sealNow(t, cfg)
	repo := filepath.Join(filepath.Dir(cfg), "work", "core")
	if w := s.post(t, "/guard/list", url.Values{"s": {"work"}, "list": {"untouchable"}, "op": {"add"}, "value": {repo}}); w.Code != http.StatusSeeOther {
		t.Fatalf("add untouchable: %d", w.Code)
	}
	vs, err := confighistory.List(cfg)
	if err != nil || len(vs) != 2 {
		t.Fatalf("versions after a change in the panel: %d %v", len(vs), err)
	}
	if vs[0].By != confighistory.ByPanel || vs[0].Detail != "/guard/list" || vs[1].By != confighistory.ByFirst {
		t.Errorf("who: %+v / %+v", vs[0], vs[1])
	}

	page := s.get(t, "/versions?s=work").Body.String()
	for _, want := range []string{"Versions", "panel, page Guard", "current", "What it changed", "guard.toml", `class="d-add"`, "Go back to this one"} {
		if !strings.Contains(page, want) {
			t.Errorf("the page lacks %q", want)
		}
	}
	if strings.Count(page, "Go back to this one") != 2 { // the fold and its button, once
		t.Error("the current version can be gone back to")
	}
	// Going back would drop the repository from a guard's list: the page says so first.
	if !strings.Contains(page, "changes what the guard lets") || !strings.Contains(page, `<span class="d-del">- <code>~/work/core</code>`) {
		t.Error("the page does not warn that going back changes the guard's lists")
	}
	if w := s.post(t, "/versions/restore", url.Values{"s": {"work"}, "id": {vs[1].ID}}); w.Code != http.StatusBadRequest {
		t.Fatalf("restore without ticking the box: %d", w.Code)
	}
	if g, _ := config.LoadGuard(cfg); len(g.UntouchableRepos) != 1 {
		t.Fatal("a restore not confirmed went back")
	}

	if w := s.post(t, "/versions/restore", url.Values{"s": {"work"}, "id": {vs[1].ID}, "confirm": {"yes"}}); w.Code != http.StatusSeeOther {
		t.Fatalf("restore: %d %s", w.Code, w.Body.String())
	}
	g, _ := config.LoadGuard(cfg)
	if len(g.UntouchableRepos) != 0 {
		t.Errorf("not gone back: %v", g.UntouchableRepos)
	}
	c, _ := config.Load(cfg)
	if readSeal(t, cfg) != c.GuardFingerprint(g) {
		t.Error("going back in the panel did not seal the lists")
	}
	if after, _ := confighistory.List(cfg); len(after) != 3 || after[0].By != confighistory.ByRestore {
		t.Errorf("versions after going back: %+v", after)
	}
	if w := s.post(t, "/versions/restore", url.Values{"s": {"work"}, "id": {"../config"}, "confirm": {"yes"}}); w.Code != http.StatusBadRequest {
		t.Errorf("a made-up version: %d", w.Code)
	}

	// Through the network: shown, no button, and a post is refused.
	const rhost, owner = "box.example.ts.net:8443", "lucia@example.com"
	s.remote.Store(&Remote{Host: rhost, Owner: owner})
	w := s.do(t, req{path: s.base() + "/versions?s=work", host: rhost, owner: owner})
	if w.Code != http.StatusOK || strings.Contains(w.Body.String(), "Go back to this one") || !strings.Contains(w.Body.String(), "from another device") {
		t.Errorf("remote page: %d, button or note wrong", w.Code)
	}
	before, _ := confighistory.List(cfg)
	form := url.Values{"s": {"work"}, "id": {before[1].ID}, "confirm": {"yes"}, "csrf": {s.csrf}}
	if w := s.do(t, req{method: http.MethodPost, path: s.base() + "/versions/restore", host: rhost, owner: owner, origin: "https://" + rhost, form: form}); w.Code != http.StatusBadRequest {
		t.Errorf("remote restore: %d", w.Code)
	}
	if after, _ := confighistory.List(cfg); len(after) != len(before) {
		t.Error("a refused restore changed something")
	}
}
