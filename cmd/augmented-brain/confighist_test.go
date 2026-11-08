package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/icalvete/augmented-brain/internal/config"
	"github.com/icalvete/augmented-brain/internal/confighistory"
	"github.com/icalvete/augmented-brain/internal/state"
)

// A change by hand becomes a version when a command runs; config history lists and
// compares them; config restore goes back without sealing the guard's lists, and says
// the guard will ask.
func TestConfigHistoryAndRestore(t *testing.T) {
	_, cfg := home(t)
	var out, errOut bytes.Buffer
	if code := run([]string{"config", "history"}, nil, &out, &errOut); code != 0 || !strings.Contains(out.String(), "Aún no hay versiones") {
		t.Fatalf("no versions yet: %d %q %q", code, out.String(), errOut.String())
	}
	c, _ := config.Load(cfg)
	g, _ := config.LoadGuard(cfg)
	if err := state.WriteSeal(cfg, c.GuardFingerprint(g)); err != nil {
		t.Fatal(err)
	}
	// The first version, kept by a command that changes nothing.
	run([]string{"space"}, nil, &out, &errOut)
	// By hand: an untouchable repository.
	if err := os.WriteFile(filepath.Join(cfg, config.GuardFile), []byte("untouchable_repos = [\"~/core\"]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	run([]string{"space"}, nil, &out, &errOut)
	vs, _ := confighistory.List(cfg)
	if len(vs) != 2 || vs[0].By != confighistory.ByOutside {
		t.Fatalf("versions: %+v", vs)
	}

	out.Reset()
	if code := run([]string{"config", "history"}, nil, &out, &errOut); code != 0 || !strings.Contains(out.String(), " 1  ") || !strings.Contains(out.String(), "a mano o con otro programa  · la actual") {
		t.Errorf("list: %d\n%s", code, out.String())
	}
	out.Reset()
	if code := run([]string{"config", "history", "1"}, nil, &out, &errOut); code != 0 || !strings.Contains(out.String(), `+ untouchable_repos = ["~/core"]`) {
		t.Errorf("what version 1 changed: %d\n%s", code, out.String())
	}
	out.Reset()
	errOut.Reset()
	if code := run([]string{"config", "history", "9"}, nil, &out, &errOut); code != 1 || !strings.Contains(errOut.String(), "no hay ninguna versión 9") {
		t.Errorf("a missing version: %d %q", code, errOut.String())
	}

	// Back to the sealed lists: nothing for the guard to ask.
	out.Reset()
	if code := run([]string{"config", "restore", "2"}, nil, &out, &errOut); code != 0 {
		t.Fatalf("restore: %d %q", code, errOut.String())
	}
	for _, want := range []string{"Vuelvo a la versión 2", "guard.toml", "Hecho."} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("restore says no %q:\n%s", want, out.String())
		}
	}
	if strings.Contains(out.String(), "el guardia pregunta") {
		t.Error("back to the sealed lists, yet it says the guard will ask")
	}
	if g, _ := config.LoadGuard(cfg); len(g.UntouchableRepos) != 0 {
		t.Errorf("not gone back: %v", g.UntouchableRepos)
	}
	// Forward to lists the user never sealed: not sealed by a command, and it says so.
	vs, _ = confighistory.List(cfg)
	n, _ := confighistory.Index(vs, vs[1].ID) // the version with ~/core, kept before going back
	out.Reset()
	if code := run([]string{"config", "restore", strconv.Itoa(n + 1)}, nil, &out, &errOut); code != 0 || !strings.Contains(out.String(), "el guardia pregunta") {
		t.Errorf("restore to unsealed lists: %d\n%s", code, out.String())
	}
	c, _ = config.Load(cfg)
	g, _ = config.LoadGuard(cfg)
	if len(g.UntouchableRepos) != 1 || state.ReadSeal(cfg) == c.GuardFingerprint(g) {
		t.Error("a command sealed the lists: the assistant could change the guard unseen")
	}
	out.Reset()
	if run([]string{"config", "restore", "1"}, nil, &out, &errOut); !strings.Contains(out.String(), "Es igual que la configuración actual") {
		t.Errorf("restoring the current one: %q", out.String())
	}
	if code := run([]string{"config"}, nil, &out, &errOut); code != 2 {
		t.Errorf("bare config: %d", code)
	}
}

// A config.toml that no longer loads is when going back matters most: config history
// still lists the versions (in the file's own language), config restore brings back one
// that works, and doctor points to them.
func TestConfigRestoreFixesABrokenConfig(t *testing.T) {
	_, cfg := home(t)
	var out, errOut bytes.Buffer
	run([]string{"space"}, nil, &out, &errOut) // the first version: one that works
	good, err := os.ReadFile(filepath.Join(cfg, config.FileName))
	if err != nil {
		t.Fatal(err)
	}
	broken := strings.Replace(string(good), "[spaces.", "[[spaces.", 1)
	if err := os.WriteFile(filepath.Join(cfg, config.FileName), []byte(broken), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := config.Load(cfg); err == nil {
		t.Fatal("the configuration still loads")
	}

	out.Reset()
	errOut.Reset()
	if code := run([]string{"doctor"}, nil, &out, &errOut); !strings.Contains(out.String()+errOut.String(), "config restore") {
		t.Errorf("doctor does not point to config restore: %d\n%s%s", code, out.String(), errOut.String())
	}

	out.Reset()
	errOut.Reset()
	if code := run([]string{"config", "history"}, nil, &out, &errOut); code != 0 || !strings.Contains(out.String(), " 1  ") || !strings.Contains(out.String(), "Versiones") {
		t.Fatalf("history with a broken config: %d %q %q", code, out.String(), errOut.String())
	}
	vs, _ := confighistory.List(cfg)
	n := strconv.Itoa(len(vs))
	out.Reset()
	errOut.Reset()
	if code := run([]string{"config", "restore", n}, nil, &out, &errOut); code != 0 {
		t.Fatalf("restore with a broken config: %d %q %q", code, out.String(), errOut.String())
	}
	if _, err := config.Load(cfg); err != nil {
		t.Errorf("still broken after restore: %v", err)
	}
}
