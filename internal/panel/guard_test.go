package panel

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/state"
)

func sealNow(t *testing.T, cfg string) {
	t.Helper()
	c, err := config.Load(cfg)
	if err != nil {
		t.Fatal(err)
	}
	g, err := config.LoadGuard(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := state.WriteSeal(cfg, c.GuardFingerprint(g)); err != nil {
		t.Fatal(err)
	}
}

func TestGuardPageEditsTheLists(t *testing.T) {
	s, cfg := fixture(t, "es")
	home := filepath.Dir(cfg)
	// the user's comment in config.toml must survive every change
	data, _ := os.ReadFile(filepath.Join(cfg, config.FileName))
	if err := os.WriteFile(filepath.Join(cfg, config.FileName), append([]byte("# mi comentario\n"), data...), 0o600); err != nil {
		t.Fatal(err)
	}
	sealNow(t, cfg)
	redirected := func(w interface{ Result() *http.Response }, what string) {
		t.Helper()
		if r := w.Result(); r.StatusCode != http.StatusSeeOther {
			t.Fatalf("%s: %d", what, r.StatusCode)
		}
	}
	if body := s.get(t, "/guard?s=work").Body.String(); !strings.Contains(body, "Repositorios intocables") {
		t.Fatalf("the page does not show: %.200s", body)
	}

	repo := filepath.Join(home, "work", "core")
	redirected(s.post(t, "/guard/list", url.Values{"s": {"work"}, "list": {"untouchable"}, "op": {"add"}, "value": {repo}}), "add untouchable")
	g, err := config.LoadGuard(cfg)
	if err != nil || len(g.UntouchableRepos) != 1 {
		t.Fatalf("guard.toml: %v %v", g, err)
	}
	// saved from the panel: sealed, so the guard sees no outside change
	c, _ := config.Load(cfg)
	if state.ReadSeal(cfg) != c.GuardFingerprint(g) {
		t.Error("a change made in the panel was not sealed")
	}
	// removing needs the folder's name
	if w := s.post(t, "/guard/list", url.Values{"s": {"work"}, "list": {"untouchable"}, "op": {"remove"}, "value": {g.UntouchableRepos[0]}, "confirm": {"nope"}}); w.Code != http.StatusBadRequest {
		t.Errorf("remove without the name: %d", w.Code)
	}
	if g, _ = config.LoadGuard(cfg); len(g.UntouchableRepos) != 1 {
		t.Fatal("removed without confirming")
	}
	redirected(s.post(t, "/guard/list", url.Values{"s": {"work"}, "list": {"untouchable"}, "op": {"remove"}, "value": {g.UntouchableRepos[0]}, "confirm": {"core"}}), "remove untouchable")
	if g, _ = config.LoadGuard(cfg); len(g.UntouchableRepos) != 0 {
		t.Errorf("not removed: %v", g.UntouchableRepos)
	}

	redirected(s.post(t, "/guard/list", url.Values{"s": {"work"}, "list": {"ask"}, "space": {"work"}, "op": {"add"}, "value": {repo}}), "add ask")
	redirected(s.post(t, "/guard/list", url.Values{"s": {"work"}, "list": {"hosts"}, "op": {"add"}, "value": {"db01"}}), "add host")
	redirected(s.post(t, "/guard/list", url.Values{"s": {"work"}, "list": {"profiles"}, "op": {"add"}, "value": {"prod"}}), "add profile")
	c, _ = config.Load(cfg)
	if _, ask := c.GitLists(); len(ask) != 1 || len(c.UntouchableHosts) != 1 || len(c.ProductionAWSProfiles) != 1 {
		t.Errorf("lists: ask %v hosts %v profiles %v", ask, c.UntouchableHosts, c.ProductionAWSProfiles)
	}

	// a root exception needs the machine's name typed again, and a date that reads
	if w := s.post(t, "/guard/root", url.Values{"s": {"work"}, "op": {"add"}, "host": {"m1"}, "until": {"2099-01-01"}, "confirm": {"m2"}}); w.Code != http.StatusBadRequest {
		t.Errorf("root without the name: %d", w.Code)
	}
	if w := s.post(t, "/guard/root", url.Values{"s": {"work"}, "op": {"add"}, "host": {"m1"}, "until": {"someday"}, "confirm": {"m1"}}); w.Code != http.StatusBadRequest {
		t.Errorf("root with a bad date: %d", w.Code)
	}
	redirected(s.post(t, "/guard/root", url.Values{"s": {"work"}, "op": {"add"}, "host": {"m1"}, "until": {"2099-01-01"}, "confirm": {"m1"}}), "add root")
	if c, _ = config.Load(cfg); len(c.RootAsk) != 1 || c.RootAsk[0] != "m1 until 2099-01-01" {
		t.Errorf("root_ask: %v", c.RootAsk)
	}
	redirected(s.post(t, "/guard/root", url.Values{"s": {"work"}, "op": {"remove"}, "value": {"m1 until 2099-01-01"}}), "remove root")
	if c, _ = config.Load(cfg); len(c.RootAsk) != 0 {
		t.Errorf("root_ask after removing: %v", c.RootAsk)
	}
	if data, _ := os.ReadFile(filepath.Join(cfg, config.FileName)); !strings.Contains(string(data), "# mi comentario") {
		t.Error("the user's comment was lost")
	}
}

func TestGuardPageSaysOutsideChangesAndAccepts(t *testing.T) {
	s, cfg := fixture(t, "en")
	sealNow(t, cfg)
	// someone else adds a host: the page says so, and a save from the page does not seal it
	if err := config.SetTopList(cfg, "untouchable_hosts", []string{"sneaky"}); err != nil {
		t.Fatal(err)
	}
	if body := s.get(t, "/guard?s=work").Body.String(); !strings.Contains(body, "not the ones you last saw") {
		t.Fatalf("no warning: %.300s", body)
	}
	before := state.ReadSeal(cfg)
	if w := s.post(t, "/guard/list", url.Values{"s": {"work"}, "list": {"profiles"}, "op": {"add"}, "value": {"prod"}}); w.Code != http.StatusSeeOther {
		t.Fatalf("save: %d", w.Code)
	}
	if state.ReadSeal(cfg) != before {
		t.Error("a save sealed lists that something else had changed")
	}
	if w := s.post(t, "/guard/accept", url.Values{"s": {"work"}}); w.Code != http.StatusSeeOther {
		t.Fatalf("accept: %d", w.Code)
	}
	if body := s.get(t, "/guard?s=work").Body.String(); strings.Contains(body, "not the ones you last saw") {
		t.Error("still warns after accepting")
	}
}
