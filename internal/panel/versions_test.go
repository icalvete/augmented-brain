package panel

import (
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/confighistory"
	"github.com/icalvete/augmented-brain/internal/state"
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
	if strings.Count(page, "Go back to this one") != 1 {
		t.Error("the current version can be gone back to")
	}

	if w := s.post(t, "/versions/restore", url.Values{"s": {"work"}, "id": {vs[1].ID}}); w.Code != http.StatusSeeOther {
		t.Fatalf("restore: %d %s", w.Code, w.Body.String())
	}
	g, _ := config.LoadGuard(cfg)
	if len(g.UntouchableRepos) != 0 {
		t.Errorf("not gone back: %v", g.UntouchableRepos)
	}
	c, _ := config.Load(cfg)
	if state.ReadSeal(cfg) != c.GuardFingerprint(g) {
		t.Error("going back in the panel did not seal the lists")
	}
	if after, _ := confighistory.List(cfg); len(after) != 3 || after[0].By != confighistory.ByRestore {
		t.Errorf("versions after going back: %+v", after)
	}
	if w := s.post(t, "/versions/restore", url.Values{"s": {"work"}, "id": {"../config"}}); w.Code != http.StatusBadRequest {
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
	form := url.Values{"s": {"work"}, "id": {before[1].ID}, "csrf": {s.csrf}}
	if w := s.do(t, req{method: http.MethodPost, path: s.base() + "/versions/restore", host: rhost, owner: owner, origin: "https://" + rhost, form: form}); w.Code != http.StatusBadRequest {
		t.Errorf("remote restore: %d", w.Code)
	}
	if after, _ := confighistory.List(cfg); len(after) != len(before) {
		t.Error("a refused restore changed something")
	}
}
