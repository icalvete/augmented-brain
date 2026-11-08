package panel

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/confighistory"
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

// The Guard page saves the preferred model in session.toml, and empties it.
func TestGuardPageModel(t *testing.T) {
	s, cfg := fixture(t, "es")
	if body := s.get(t, "/guard?s=work").Body.String(); !strings.Contains(body, "Modelo de tus sesiones") {
		t.Fatalf("no model section: %.200s", body)
	}
	if w := s.post(t, "/guard/model", url.Values{"s": {"work"}, "value": {"opus"}}); w.Code != http.StatusSeeOther {
		t.Fatalf("save: %d", w.Code)
	}
	if ss, err := config.LoadSession(cfg); err != nil || ss.PreferredModel != "opus" {
		t.Fatalf("saved: %+v %v", ss, err)
	}
	if body := s.get(t, "/guard?s=work").Body.String(); !strings.Contains(body, `value="opus"`) {
		t.Error("the page does not show the saved model")
	}
	if w := s.post(t, "/guard/model", url.Values{"s": {"work"}, "value": {"a\nb"}}); w.Code != http.StatusBadRequest {
		t.Errorf("a bad value: %d", w.Code)
	}
	if w := s.post(t, "/guard/model", url.Values{"s": {"work"}, "value": {""}}); w.Code != http.StatusSeeOther {
		t.Fatalf("empty: %d", w.Code)
	}
	if ss, _ := config.LoadSession(cfg); ss.PreferredModel != "" {
		t.Errorf("not emptied: %+v", ss)
	}
}

// When the lists changed outside the page, it shows what changed since the lists the
// user last accepted (the kept version whose fingerprint is the seal), and says so when
// that version is no longer kept.
func TestGuardPageShowsWhatChanged(t *testing.T) {
	s, cfg := fixture(t, "en")
	if err := config.SetTopList(cfg, "untouchable_hosts", []string{"old-host"}); err != nil {
		t.Fatal(err)
	}
	if _, err := confighistory.Record(cfg, confighistory.ByPanel, ""); err != nil {
		t.Fatal(err)
	}
	sealNow(t, cfg)
	// outside the page: a host swapped and a profile added
	if err := config.SetTopList(cfg, "untouchable_hosts", []string{"sneaky"}); err != nil {
		t.Fatal(err)
	}
	if err := config.SetTopList(cfg, "production_aws_profiles", []string{"prod"}); err != nil {
		t.Fatal(err)
	}
	body := s.get(t, "/guard?s=work").Body.String()
	for _, want := range []string{"What changed since you last saw them", `<span class="d-add">+ <code>sneaky</code>`, `<span class="d-del">- <code>old-host</code>`, `<span class="d-add">+ <code>prod</code>`} {
		if !strings.Contains(body, want) {
			i := strings.Index(body, "What changed")
			t.Errorf("missing %q in: %s", want, body[max(i, 0):min(len(body), max(i, 0)+700)])
		}
	}

	// no kept version has the sealed lists: it says so instead
	if err := os.RemoveAll(filepath.Join(cfg, "history")); err != nil {
		t.Fatal(err)
	}
	body = s.get(t, "/guard?s=work").Body.String()
	if !strings.Contains(body, "not among the kept versions") || strings.Contains(body, "What changed since") {
		t.Errorf("without the sealed version:\n%s", body[:min(len(body), 3000)])
	}
}

// The dangerous lists warn, and letting the assistant commit and push without asking
// needs the box ticked.
func TestGuardPageWarnsOnDangerousLists(t *testing.T) {
	s, cfg := fixture(t, "en")
	sealNow(t, cfg)
	body := s.get(t, "/guard?s=work").Body.String()
	for _, want := range []string{"commits and pushes without asking you each time", "may run commands as root", `name="confirm" value="yes" required`} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q", want)
		}
	}
	repo := filepath.Join(filepath.Dir(cfg), "work", "app")
	if w := s.post(t, "/guard/list", url.Values{"s": {"work"}, "list": {"allowed"}, "space": {"work"}, "op": {"add"}, "value": {repo}}); w.Code != http.StatusBadRequest {
		t.Errorf("allowed without the box: %d", w.Code)
	}
	if c, _ := config.Load(cfg); len(c.Spaces["work"].GitAllowedRepos) != 0 {
		t.Fatal("added without confirming")
	}
	if w := s.post(t, "/guard/list", url.Values{"s": {"work"}, "list": {"allowed"}, "space": {"work"}, "op": {"add"}, "value": {repo}, "confirm": {"yes"}}); w.Code != http.StatusSeeOther {
		t.Fatalf("allowed with the box: %d", w.Code)
	}
	if c, _ := config.Load(cfg); len(c.Spaces["work"].GitAllowedRepos) != 1 {
		t.Error("not added after confirming")
	}
}
