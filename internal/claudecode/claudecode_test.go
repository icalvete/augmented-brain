package claudecode

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMain points the configuration folder at a throwaway one: Install, Update and
// Uninstall write and remove the plugin's folder inside it.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "claudecode-test")
	if err != nil {
		panic(err)
	}
	os.Setenv("AB_CONFIG_DIR", dir)
	os.Unsetenv("AB_MARKETPLACE")
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

type fake struct {
	calls []string
	out   map[string]string
	fail  map[string]bool
}

func (f *fake) run(name string, args ...string) (string, error) {
	cmd := name + " " + strings.Join(args, " ")
	f.calls = append(f.calls, cmd)
	for prefix, out := range f.out {
		if strings.HasPrefix(cmd, prefix) {
			if f.fail[prefix] {
				return out, errors.New("exit status 1")
			}
			return out, nil
		}
	}
	return "", nil
}

func TestInstall(t *testing.T) {
	t.Setenv("AB_MARKETPLACE", "/src/augmented-brain")
	f := &fake{}
	if err := Install(f.run); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"claude plugin marketplace add /src/augmented-brain",
		"claude plugin install brain@augmented-brain --scope user",
	}
	if strings.Join(f.calls, "\n") != strings.Join(want, "\n") {
		t.Errorf("calls:\n%s", strings.Join(f.calls, "\n"))
	}

	// Repeating is fine.
	f = &fake{out: map[string]string{"claude plugin marketplace add": "already installed"},
		fail: map[string]bool{"claude plugin marketplace add": true}}
	if err := Install(f.run); err != nil {
		t.Errorf("a marketplace already added must not fail: %v", err)
	}

	// A real failure is reported with Claude Code's own message.
	f = &fake{out: map[string]string{"claude plugin install": "network is unreachable"},
		fail: map[string]bool{"claude plugin install": true}}
	if err := Install(f.run); err == nil || !strings.Contains(err.Error(), "network is unreachable") {
		t.Errorf("err = %v", err)
	}
}

func TestInstalled(t *testing.T) {
	f := &fake{out: map[string]string{"claude plugin list": "Installed plugins:\n  ❯ brain@augmented-brain\n"}}
	if ok, err := Installed(f.run); !ok || err != nil {
		t.Errorf("installed = %v, %v", ok, err)
	}
	f = &fake{out: map[string]string{"claude plugin list": "No plugins installed"}}
	if ok, _ := Installed(f.run); ok {
		t.Error("reported installed when it is not")
	}
}

// With no AB_MARKETPLACE, the marketplace is the folder the program writes, in the
// configuration folder: never the repository's main.
func TestMarketplaceIsLocal(t *testing.T) {
	cfg := t.TempDir()
	t.Setenv("AB_CONFIG_DIR", cfg)
	f := &fake{}
	if err := Install(f.run); err != nil {
		t.Fatal(err)
	}
	dir := LocalDir(cfg)
	if want := "claude plugin marketplace add " + dir; f.calls[0] != want {
		t.Errorf("first call = %q, want %q", f.calls[0], want)
	}
	if strings.Contains(strings.Join(f.calls, "\n"), GitHubMarketplace) {
		t.Errorf("the plugin still comes from GitHub: %v", f.calls)
	}
	for _, name := range []string{".claude-plugin/marketplace.json", "plugin/hooks/hooks.json", "plugin/hooks/missing.sh", "plugin/.mcp.json", "plugin/.claude-plugin/plugin.json", "plugin/commands/panel.md"} {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(name))); err != nil {
			t.Errorf("%s not written: %v", name, err)
		}
	}
}

func TestUpdate(t *testing.T) {
	t.Setenv("AB_MARKETPLACE", "/src/augmented-brain")
	f := &fake{}
	if err := Update(f.run); err != nil {
		t.Fatal(err)
	}
	// The add comes first: it moves a marketplace that follows GitHub to the program's
	// folder, keeping the plugin installed (a remove would uninstall it and lose its data).
	want := "claude plugin marketplace add /src/augmented-brain\nclaude plugin marketplace update augmented-brain\nclaude plugin update brain@augmented-brain"
	if got := strings.Join(f.calls, "\n"); got != want {
		t.Errorf("calls:\n%s", got)
	}
	f = &fake{out: map[string]string{"claude plugin update": "plugin not found"},
		fail: map[string]bool{"claude plugin update": true}}
	if err := Update(f.run); err == nil || !strings.Contains(err.Error(), "plugin not found") {
		t.Errorf("err = %v", err)
	}
}

func TestUninstall(t *testing.T) {
	cfg := t.TempDir()
	t.Setenv("AB_CONFIG_DIR", cfg)
	if err := WriteLocal(LocalDir(cfg), "1.2.3", "linux"); err != nil {
		t.Fatal(err)
	}
	f := &fake{}
	if err := Uninstall(f.run); err != nil {
		t.Fatal(err)
	}
	want := "claude plugin uninstall brain@augmented-brain\nclaude plugin marketplace remove augmented-brain"
	if got := strings.Join(f.calls, "\n"); got != want {
		t.Errorf("calls:\n%s", got)
	}
	if _, err := os.Stat(LocalDir(cfg)); !os.IsNotExist(err) {
		t.Errorf("the plugin's folder is left: %v", err)
	}
	// Already gone: fine.
	f = &fake{out: map[string]string{"claude plugin uninstall": "plugin not installed"},
		fail: map[string]bool{"claude plugin uninstall": true}}
	if err := Uninstall(f.run); err != nil {
		t.Errorf("a plugin already gone must not fail: %v", err)
	}
	// Any other failure is one, even with "not" somewhere in it.
	f = &fake{out: map[string]string{"claude plugin uninstall": "cannot write the settings"},
		fail: map[string]bool{"claude plugin uninstall": true}}
	if err := Uninstall(f.run); err == nil {
		t.Error("a real failure was taken as already gone")
	}
}
