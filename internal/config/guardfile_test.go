package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
)

func TestGuardFileMissingIsEmpty(t *testing.T) {
	g, err := LoadGuard(t.TempDir())
	if err != nil || len(g.UntouchableRepos) != 0 {
		t.Fatalf("missing file: %v %v", g, err)
	}
}

func TestGuardFileRoundTrip(t *testing.T) {
	dir := t.TempDir()
	// Paths of this system, resolved as Load resolves them (on Windows "/srv/core" gains a drive).
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	core, cli := filepath.Join(base, "core"), filepath.Join(base, "cli-*")
	in := Guard{UntouchableRepos: []string{core, core, cli}}
	if err := SaveGuard(dir, in); err != nil {
		t.Fatal(err)
	}
	g, err := LoadGuard(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.UntouchableRepos) != 2 || g.UntouchableRepos[0] != core || g.UntouchableRepos[1] != cli {
		t.Fatalf("got %v", g.UntouchableRepos)
	}
	if fi, _ := os.Stat(filepath.Join(dir, GuardFile)); runtime.GOOS != "windows" && fi.Mode().Perm() != 0o600 { // Windows has no such modes
		t.Fatalf("mode %v", fi.Mode())
	}
}

func TestGuardFileBrokenIsAnError(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, GuardFile), []byte("untouchable_repos = ["), 0o600)
	if _, err := LoadGuard(dir); err == nil {
		t.Fatal("a file that cannot be read must be an error")
	}
}

// Saving changes guard.toml one setting at a time: the user's comments, a setting this
// program does not know (one a newer program wrote) and the layout stay; only the
// changed setting is written again.
func TestSaveGuardKeepsTheRest(t *testing.T) {
	dir := t.TempDir()
	home, _ := os.UserHomeDir()
	old := strings.Join([]string{
		"# My notes: core is off limits until the audit.",
		"untouchable_repos = [",
		"  \"~/work/core\", # the audited one",
		"  \"~/work/odd]name\",",
		"]",
		"",
		"# Written by a newer program.",
		"future_setting = [\"keep\", \"me\"]",
		"reach = \"spaces\"",
		"",
	}, "\n")
	path := filepath.Join(dir, GuardFile)
	if err := os.WriteFile(path, []byte(old), 0o600); err != nil {
		t.Fatal(err)
	}
	g, err := LoadGuard(dir)
	if err != nil {
		t.Fatal(err)
	}
	g.UntouchableRepos = append(g.UntouchableRepos, filepath.Join(home, "work", "infra"))
	if err := SaveGuard(dir, g); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(path)
	got := string(b)
	for _, want := range []string{
		"# My notes: core is off limits until the audit.\nuntouchable_repos = [\n",
		"\"~/work/infra\",\n]\n\n# Written by a newer program.\nfuture_setting = [\"keep\", \"me\"]\nreach = \"spaces\"\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("lost %q:\n%s", want, got)
		}
	}
	g2, err := LoadGuard(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(g2.UntouchableRepos) != 3 || !g2.OnlySpaces() {
		t.Errorf("read back: %+v", g2)
	}
	var raw map[string]any
	if _, err := toml.Decode(got, &raw); err != nil || raw["future_setting"] == nil {
		t.Errorf("unknown setting gone: %v %v", raw, err)
	}

	// A setting not in the file yet is added with its comment; taking it out again
	// takes that comment too, and nothing else.
	if err := os.WriteFile(path, []byte("# mine\nuntouchable_repos = []\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := SaveGuard(dir, Guard{Reach: ReachMachine}); err != nil {
		t.Fatal(err)
	}
	b, _ = os.ReadFile(path)
	if !strings.HasPrefix(string(b), "# mine\nuntouchable_repos = [\n]\n# reach:") || !strings.HasSuffix(string(b), "reach = \"machine\"\n") {
		t.Errorf("reach added:\n%s", b)
	}
	if err := SaveGuard(dir, Guard{}); err != nil {
		t.Fatal(err)
	}
	if b, _ = os.ReadFile(path); string(b) != "# mine\nuntouchable_repos = [\n]\n" {
		t.Errorf("reach taken out:\n%q", b)
	}
}

// A change that would leave guard.toml unreadable never reaches the disk.
func TestSaveGuardRefusesABrokenResult(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, GuardFile)
	old := "untouchable_repos = []\nreach = \"machine\"\n"
	if err := os.WriteFile(path, []byte(old), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := SaveGuard(dir, Guard{Reach: "nowhere"}); err == nil {
		t.Error("an unknown reach saved")
	}
	if b, _ := os.ReadFile(path); string(b) != old {
		t.Errorf("file changed: %q", b)
	}
}
